/*
Copyright © Zac Orndorff zac@orndorff.dev
*/
package cmd

import (
	"github.com/spf13/cobra"
	"log/slog"
)

// workspaceCmd represents the get command
var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Working with workspaces",
	Long: `Workspaces are a way to organize your data and queries within DuckTape. 
		A duckdb database is created for each workspace and all connections, data, queries are executed within a given 'workspace db'.`,
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("get called")
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
}
