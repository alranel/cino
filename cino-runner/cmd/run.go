package cmd

import (
	"fmt"
	"os"

	"github.com/alranel/cino/cino-runner/runner"
	. "github.com/alranel/cino/lib"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <test> [<test>...]",
	Short: "Run one or more tests",
	Long:  `This command runs all the available tests in the supplied directories`,
	Run:   runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func init() {
	runCmd.Flags().StringP("fqbn", "b", "", "Fully Qualified Board Name, e.g.: arduino:avr:uno")
	runCmd.Flags().StringP("port", "p", "", "Upload port, e.g.: COM10 or /dev/ttyACM0")
}

func runRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		os.Exit(0)
	}

	// Use configured devices by default, unless one was specified manually.
	{
		board, _ := cmd.Flags().GetString("fqbn")
		port, _ := cmd.Flags().GetString("port")
		if board != "" || port != "" {
			if board == "" || port == "" {
				fmt.Fprintln(os.Stderr, "Cannot specify --board without --port and viceversa")
				os.Exit(1)
			}

			runner.Config.Devices = make([]runner.Device, 1)
			runner.Config.Devices[0].FQBN = board
			runner.Config.Devices[0].Port = port
		}
	}

	success := true
	for _, path := range args {
		// Find tests to run
		tests, err := FindTests(path)
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
			os.Exit(1)
		}

		// Run tests
		if len(tests) == 0 {
			fmt.Printf("No tests found in %s\n", path)
			continue
		}
		fmt.Printf("Running %d tests(s) in %s\n", len(tests), path)

		for _, test := range tests {
			fmt.Printf("Running test in %s\n", test.RelPath())
			devices := runner.AssignDevices(test.GetRequirements())
			if err = runner.RunTest(&test, devices); err != nil {
				os.Stderr.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
				os.Exit(1)
			}

			if test.Status == "failure" {
				success = false
			}
		}
	}

	if success == false {
		os.Exit(1)
	}
}
