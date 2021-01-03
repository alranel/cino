package cmd

import (
	"fmt"
	"os"

	"github.com/alranel/cino/cino-server/server"
	. "github.com/alranel/cino/lib"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cino-server",
	Short: "cino-server orchestrates CI jobs for Arduino boards.",
	Long:  `cino-server is the orchestrator component of the cino Continuous Integration framework.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		configFilePath, _ := cmd.Flags().GetString("config")
		return server.LoadConfig(configFilePath)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Test database connection
		ConnectDB(server.Config.DB)

		go server.StartScanner()
		go server.StartResultsHandler()
		server.StartWebService()
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
