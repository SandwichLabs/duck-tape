/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// workspaceCmd represents the get command
var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Working with workspaces",
	Long: `Workspaces are a way to organize your data and queries within ODT. 
		A duckdb database is created for each workspace and all connections, data, queries are executed within a given 'workspace db'.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("get called")
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
}
