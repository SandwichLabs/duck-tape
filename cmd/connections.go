/*
Copyright © 2024 Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"github.com/SandwichLabs/dt/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var addConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Add a new connection",
	Run: func(cmd *cobra.Command, args []string) {
		workspace := viper.GetString("workspace")
		connection := config.ConnectionConfigForm()

		log.Infof("saving connection: %v", connection)
		_, err := config.SetWorkspaceConnection(workspace, connection, true)
		cobra.CheckErr(err)
	},
}
