package cmd

import (
	"fmt"
	"os"

	"github.com/alranel/cino/cino-runner/runner"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cino-runner",
	Short: "cino-runner runs tests on Arduino boards.",
	Long:  `cino-runner runs tests on physical Arduino boards.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		configFilePath, _ := cmd.Flags().GetString("config")
		return runner.LoadConfig(configFilePath)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "Configuration file")
}

// Execute starts the cobra command parsing chain.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
