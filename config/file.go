/*
Copyright © 2024 Zac Orndorff zac@orndorff.dev
*/
package config

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ConfigRootPath() string {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	return fmt.Sprintf("%s/.dt", home)
}

func WorkspacePath(workspace string) string {
	return fmt.Sprintf("%s/%s", ConfigRootPath(), workspace)
}

func EnsureConfig(vip *viper.Viper) {
	workspace := vip.GetString("workspace")
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	configPath := fmt.Sprintf("%s/.dt", home)
	_, err = SetWorkspaceDb(workspace, "dt.db", true)
	cobra.CheckErr(err)
	EnsureWorkspace(configPath, workspace)
	err = vip.SafeWriteConfig()
	cobra.CheckErr(err)
}

// setupConfig sets up the configuration for the application
// creating the default .dt/config.yaml config file if not present
// and setting up the workspace folder
func EnsureWorkspace(root_path string, workspace string) {
	log.Infof("Setting up the workspace folder")

	// set the workspace folder
	workspacePath := fmt.Sprintf("%s/%s", root_path, workspace)
	log.Infof("Workspace folder: %s", workspacePath)
	_, err := os.Stat(workspacePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(workspacePath, 0755)
		cobra.CheckErr(err)
		log.Infof("Workspace folder created at: %s", workspacePath)
	} else {
		log.Infof("Workspace folder already exists at: %s", workspacePath)
	}
}
