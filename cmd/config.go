/*
Copyright © 2024 Zac Orndorff zac@orndorff.dev
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"

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
func EnsureWorkspace(root_path string, workspace string) {
	slog.Debug("Setting up the workspace folder")

	// set the workspace folder
	workspacePath := fmt.Sprintf("%s/%s", root_path, workspace)
	slog.Debug("Workspace folder", "workspacePath", workspacePath)
	_, err := os.Stat(workspacePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(workspacePath, 0755)
		cobra.CheckErr(err)
		slog.Debug("Workspace folder created", "workspacePath", workspacePath)
	} else {
		slog.Debug("Workspace folder already exists", "workspacePath", workspacePath)
	}
	_, err = SetWorkspaceDb(workspace, "dt.db", false)
	cobra.CheckErr(err)
}

// initConfig reads in config file and ENV variables if set.
func InitConfig() {
	viper.AutomaticEnv() // read in environment variables that match
	// Set the log level using the config file and/or environment variables

	SetLogLevel(viper.GetString("LOG_LEVEL"), &logLevel)

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		configPath := fmt.Sprintf("%s/.dt", home)

		// Search config in home directory with name ".testing" (without extension).
		// create a .dt folder in the home directory if it doesn't exist
		_, err = os.Stat(configPath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(configPath, 0755)
			cobra.CheckErr(err)
		}

		fullConfigPath := fmt.Sprintf("%s/config.yaml", configPath)
		_, err = os.Stat(fullConfigPath)
		if os.IsNotExist(err) {
			_, err = os.Create(fullConfigPath)
			cobra.CheckErr(err)
		}

		viper.AddConfigPath(configPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")

		EnsureWorkspace(configPath, viper.GetString("workspace"))

		cobra.CheckErr(err)

		slog.Debug("Configuration file", "configFile", viper.ConfigFileUsed())
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		slog.Debug("Using config file", "configFile", viper.ConfigFileUsed())
	}
}
