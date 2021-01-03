package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindTests looks for all the cino.yml files under the given path,
// detects the type of package contains them and returns a slice of
// Test objects.
func FindTests(path string) ([]Test, error) {
	var tests []Test

	// Does the supplied path(s) exist?
	if stat, err := os.Stat(path); os.IsNotExist(err) || !stat.IsDir() {
		return nil, fmt.Errorf("Not a directory: %s", path)
	}

	// Do we have a cino.yml file?
	if _, err := os.Stat(filepath.Join(path, "cino.yml")); !os.IsNotExist(err) {
		// If we have a cino.yml file, path is a single test or a sketch
		test, err := NewTest(path, path, Sketch)
		if err != nil {
			return nil, err
		}
		tests = append(tests, *test)
	} else {
		// No cino.yml file, look for tests in subdirectories
		var testsInSubdirectories []string
		err := filepath.Walk(path,
			func(subpath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					if _, err := os.Stat(filepath.Join(subpath, "cino.yml")); !os.IsNotExist(err) {
						testsInSubdirectories = append(testsInSubdirectories, subpath)
					}
				}
				return nil
			})
		if err != nil {
			fmt.Println(err)
		}

		if len(testsInSubdirectories) == 0 {
			return nil, fmt.Errorf("No tests were found in %s", path)
		}

		// let's check if this is a library or a core
		cType := Sketch
		if IsLibrary(path) {
			cType = Library
		} else if IsCore(path) {
			cType = Core
		}

		for _, subpath := range testsInSubdirectories {
			test, err := NewTest(subpath, path, cType)
			if err != nil {
				return nil, err
			}
			tests = append(tests, *test)
		}
	}

	return tests, nil
}

func IsLibrary(path string) bool {
	_, err := os.Stat(filepath.Join(path, "library.properties"))
	return !os.IsNotExist(err)
}

func IsCore(path string) bool {
	_, err := os.Stat(filepath.Join(path, "boards.txt"))
	return !os.IsNotExist(err)
}

func CloneRepo(cloneURL string, commitRef string) (string, error) {
	repoDir, err := ioutil.TempDir("/tmp", ".cino-server")
	if err != nil {
		return "", err
	}

	cmds := [][]string{
		{"init"},
		{"remote", "add", "origin", cloneURL},
		{"fetch", "--depth", "1", "origin", commitRef},
		{"checkout", "FETCH_HEAD"},
	}
	for _, c := range cmds {
		cmd := exec.Command("git", c...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			os.Stderr.WriteString(fmt.Sprintf("%s: %s", strings.Join(c, " "), out))
			return "", err
		}
	}

	return repoDir, nil
}
