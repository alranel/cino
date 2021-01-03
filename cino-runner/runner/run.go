package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/alranel/cino/lib"
	"github.com/otiai10/copy"
	"go.bug.st/serial"
	"gopkg.in/ini.v1"
)

// testMsg represents a message coming from a board running a test sketch,
// encoded as a single-line JSON object.
type testMsg struct {
	Plan   int
	Result bool
	Expr   string
	File   string
	Line   int
	Fatal  bool
	Done   bool
}

// Run runs a test package on the given board
func RunTest(test *Test, devices []Device) error {
	// Make sure things are consistent.
	if len(devices) != len(test.Sketches) {
		return fmt.Errorf("number of assigned devices (%d) does not match the number of sketches defined in test (%d)",
			len(devices), len(test.Sketches))
	}

	// Prepare utilities for log generation
	outputChan := make(chan string)
	done := make(chan bool)
	appendOutput := func(sketchIdx int, s string) {
		if len(test.Sketches) > 1 && sketchIdx > -1 {
			s = fmt.Sprintf("[%s] %s", test.Sketches[sketchIdx].Dir, s)
		}
		outputChan <- s
	}
	go func() {
		for {
			s, more := <-outputChan
			if more {
				fmt.Print(s)
				test.Output += s
			} else {
				done <- true
				return
			}
		}
	}()

	appendOutput(-1, fmt.Sprintf("Test requires %d devices\n", len(test.Sketches)))

	// Prepare CLI wrapper
	runCLI := func(i int, args ...string) error {
		appendOutput(i, fmt.Sprintf("arduino-cli %s\n", strings.Join(args, " ")))
		cmd := exec.Command("arduino-cli", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			appendOutput(i, fmt.Sprintf("%s", out))
			return err
		}
		return nil
	}

	// Compile sketches and upload
	var wg sync.WaitGroup
	errs := make(chan error, len(test.Sketches))
	success := true
	for i, sketch := range test.Sketches {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			device := &devices[i]
			appendOutput(i, fmt.Sprintf("Device %d: %s on %s\n", i, device.FQBN, device.Port))
			test.DeviceFQBNs = append(test.DeviceFQBNs, device.FQBN)

			// Check if device exists
			if _, err := os.Stat(device.Port); os.IsNotExist(err) {
				errs <- fmt.Errorf(("Error: device %s does not exist\n"), device.Port)
				return
			}

			// Prepare a vanilla arduino-cli environment
			var cliDir, cliConfigFile string
			var err error
			defer os.RemoveAll(cliDir)
			{
				cliDir, err = ioutil.TempDir("/tmp", ".arduino-cli")
				if err != nil {
					errs <- err
					return
				}

				cliConfigFile = filepath.Join(cliDir, "config.yml")
				cmds := [][]string{
					{"config", "init", "--dest-file", cliConfigFile},
					{"--config-file", cliConfigFile, "config", "set", "directories.data", filepath.Join(cliDir, "data")},
					{"--config-file", cliConfigFile, "config", "set", "directories.downloads", filepath.Join(cliDir, "downloads")},
					{"--config-file", cliConfigFile, "config", "set", "directories.user", filepath.Join(cliDir, "user")},
					{"--config-file", cliConfigFile, "config", "set", "library.enable_unsafe_install", "true"},
					{"--config-file", cliConfigFile, "update"},
				}
				for _, cmd := range cmds {
					if err := runCLI(i, cmd...); err != nil {
						errs <- err
						return
					}
				}
			}

			// Install the needed core.
			if idx := strings.LastIndex(device.FQBN, ":"); idx != -1 {
				core := device.FQBN[:idx]
				if err := runCLI(i, "--config-file", cliConfigFile, "core", "install", core); err != nil {
					errs <- err
					return
				}
			}

			// Install the required libraries, as declared in the cino.yml file.
			for _, lib := range sketch.Libraries {
				if err := runCLI(i, "--config-file", cliConfigFile, "lib", "install", lib); err != nil {
					appendOutput(i, fmt.Sprintf("Error installing library: %s\n%s\n", lib, err.Error()))
					// Do not stop the process; it will probably fail during compilation
					return
				}
			}

			if test.PackageType == Library {
				// Install the library that we want to test
				/*
					// This does not work because of arduino-cli bug: https://github.com/arduino/arduino-cli/issues/1120
					if err := runCLI(i, "--config-file", cliConfigFile, "lib", "install", "--git-url", test.PackagePath); err != nil {
						errs <- err
						return
					}
				*/

				f, err := ini.Load(filepath.Join(test.PackagePath, "library.properties"))
				if err != nil {
					errs <- err
					return
				}
				libDir := filepath.Join(cliDir, "user/libraries", f.Section("").Key("name").String())
				os.Mkdir(libDir, os.ModePerm)
				err = copy.Copy(test.PackagePath, libDir)
				if err != nil {
					errs <- err
					return
				}
			}

			if test.PackageType == Core {
				// TODO: arduino-cli does not provide a way to install a core from a local path
			}

			// Write cino.h to a temporary file so that we can include it during compilation.
			// This can be removed when cino is available through the Library Manager.
			cinoLibDir, err := writeCinoH(test.Path)
			if err != nil {
				errs <- err
				return
			}
			defer os.RemoveAll(cinoLibDir)

			// Compile
			sketchPath := filepath.Join(test.Path, sketch.Dir)
			err = runCLI(i,
				"--config-file", cliConfigFile,
				"compile",
				"-b", device.FQBN,
				"--libraries", cinoLibDir,
				sketchPath)
			if err != nil {
				appendOutput(i, err.Error())
				success = false
				return
			}

			// Upload
			err = runCLI(i,
				"--config-file", cliConfigFile,
				"upload",
				"-b", device.FQBN,
				"-p", device.Port,
				sketchPath)
			if err != nil {
				errs <- err
				return
			}
		}(i)
	}
	wg.Wait()

	select {
	case err := <-errs:
		return err
	default:
	}

	if success == false {
		test.Status = "failure"
		return nil
	}

	// Connect to the boards
	serialPorts := make([]serial.Port, len(test.Sketches))
	for i := range test.Sketches {
		appendOutput(i, fmt.Sprintf("Connecting to %s\n", devices[i].Port))

		// Wait until port exists (it may be temporarily unavailable because of board reset)
		for {
			if _, err := os.Stat(devices[i].Port); !os.IsNotExist(err) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		var err error
		serialPorts[i], err = serial.Open(devices[i].Port, &serial.Mode{BaudRate: 9600})
		if err != nil {
			return err
		}
	}

	// Parse output coming from the boards
	for i := range test.Sketches {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			serialPort := serialPorts[i]
			defer serialPort.Close()

			testPlanDeclared := false
			plannedTests := -1
			totalTests := 0
			failedTests := 0
			r := bufio.NewReaderSize(serialPort, 256)
			for {
				rawLine, err := readln(r, 5*time.Second)
				if err == io.EOF {
					break
				} else if err != nil {
					errs <- err
					return
				}

				// Skip non-JSON lines
				if rawLine[0] != '{' {
					continue
				}

				// Parse line
				var line testMsg
				err = json.Unmarshal(rawLine, &line)
				if err != nil {
					errs <- err
					return
				}

				// Read message
				if line.Plan > 0 {
					// Line is a test plan declaration
					if testPlanDeclared == true {
						appendOutput(i, "Error: duplicate TEST_PLAN() directive\n")
						break
					}
					testPlanDeclared = true
					plannedTests = line.Plan
				} else if line.Expr != "" {
					// Line is a test result
					if testPlanDeclared == false {
						// A test was run before the test plan was declared
						appendOutput(i, "Error: no test plan declared\n")
						break
					}

					totalTests++
					if line.Result == true {
						appendOutput(i, fmt.Sprintf("PASS: %s:%d: %s\n", line.File, line.Line, line.Expr))
					} else {
						appendOutput(i, fmt.Sprintf("FAIL: %s:%d: %s\n", line.File, line.Line, line.Expr))
						failedTests++
					}
				}

				if line.Done || line.Fatal {
					break
				}
			}

			// Check the test results
			if plannedTests != -1 && plannedTests != totalTests {
				appendOutput(i, fmt.Sprintf("Error: expected %d tests but run %d\n", plannedTests, totalTests))
			}

			if failedTests > 0 {
				test.Status = "failure"
			} else {
				test.Status = "success"
			}

			appendOutput(i, "Test result: "+test.Status+"\n")
		}(i)
	}

	// Wait for all threads to finish
	wg.Wait()

	// Check if threads emitted errors
	select {
	case err := <-errs:
		return err
	default:
	}

	// Wait for all output to be written to test.Output
	close(outputChan)
	<-done

	return nil
}

func writeCinoH(dir string) (string, error) {
	cinoLibDir, err := ioutil.TempDir(dir, ".cino")
	if err != nil {
		return "", err
	}

	os.Mkdir(filepath.Join(cinoLibDir, "src"), os.ModePerm)

	cinoh := []byte(`
#ifndef CINO_H
#define CINO_H

#define TEST_PLAN(n)            \
Serial.begin(9600);         \
while (!Serial) {} \
Serial.print("{\"plan\":"); \
Serial.print(n);            \
Serial.println("}")

#define TEST_NOPLAN() TEST_PLAN(-1)

#define TEST_DONE() \
Serial.println("{\"done\":true}")

void _cino_check(bool result, char *quoted_expr, char *file, int line, bool fatal)
{
Serial.print("{\"result\":");
Serial.print(result ? "true" : "false");
Serial.print(",\"expr\":");
Serial.print(quoted_expr);
Serial.print(",\"file\":\"");
String f(file);
f.replace("\"", "");
Serial.print(f.substring(f.lastIndexOf('/')+1));
Serial.print("\",\"line\":");
Serial.print(line);
if (!result)
{
  Serial.print(",\"fatal\":");
  Serial.print(fatal ? "true" : "false");
}
Serial.println("}");
if (fatal && !result)
  while (1)
  {
  }
}

#define _quote(x) #x
#define REQUIRE(expr) _cino_check((expr), _quote(#expr), __FILE__, __LINE__, 1)
#define CHECK(expr) _cino_check((expr), _quote(#expr), __FILE__, __LINE__, 0)

#endif
	`)
	ioutil.WriteFile(filepath.Join(cinoLibDir, "src", "cino.h"), cinoh, 0644)

	return cinoLibDir, nil
}

func readln(reader *bufio.Reader, timeout time.Duration) ([]byte, error) {
	s := make(chan []byte)
	e := make(chan error)

	go func() {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			e <- err
		} else {
			s <- line
		}
		close(s)
		close(e)
	}()

	select {
	case line := <-s:
		return line, nil
	case err := <-e:
		return nil, err
	case <-time.After(timeout):
		return nil, io.EOF
	}
}
