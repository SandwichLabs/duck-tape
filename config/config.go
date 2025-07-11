/*
Copyright © 2024 Zac Orndorff zac@orndorff.dev
*/
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SandwichLabs/duck-tape/workspace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetConfigPath() string {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	return fmt.Sprintf("%s/.dt", home)
}

func WorkspacePath(workspace string) string {
	return fmt.Sprintf("%s/%s", GetConfigPath(), workspace)
}

// setupConfig sets up the configuration for the application
// creating the default .dt/config.yaml config file if not present
// and setting up the workspace folder
func EnsureWorkspace(root_path string, workspaceName string) {
	slog.Debug("Setting up the workspace folder")

	// set the workspace folder
	workspacePath := fmt.Sprintf("%s/%s", root_path, workspaceName)
	slog.Debug("Workspace folder", "workspacePath", workspacePath)
	_, err := os.Stat(workspacePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(workspacePath, 0755)
		cobra.CheckErr(err)
		slog.Debug("Workspace folder created", "workspacePath", workspacePath)
	} else {
		slog.Debug("Workspace folder already exists", "workspacePath", workspacePath)
	}
	currentDbPath := viper.GetString(fmt.Sprintf("%s.dbLocation", workspaceName))

	if currentDbPath == "" {
		_, err = workspace.SetWorkspaceDb(workspaceName, fmt.Sprintf("%s/%s", workspacePath, "dt.db"), false)
		cobra.CheckErr(err)
		slog.Debug("Default database path set", "dbPath", "dt.db")
	}
	cobra.CheckErr(err)
}
