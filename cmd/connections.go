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

var connectionListCmd = &cobra.Command{
	Use:   "connections",
	Short: "list connections",
	Long:  `list connections`,
	Run: func(cmd *cobra.Command, args []string) {
		connections, err := config.ListWorkspaceConnections(viper.GetString("workspace"))
		cobra.CheckErr(err)
		log.Infof("connections: %v", connections)
	},
}

var addConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Add a new connection",
	Run: func(cmd *cobra.Command, args []string) {
		workspace := viper.GetString("workspace")
		connection := config.ConnectionConfigForm()

		log.Infof("saving connection: %v", connection)
		config.SetWorkspaceConnection(workspace, connection, true)
	},
}

var getConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Get a connection",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workspace := viper.GetString("workspace")
		connectionName := args[0]
		connection, err := config.WorkspaceConnection(workspace, connectionName)
		cobra.CheckErr(err)

		log.Infof("connection: %v", connection)
	},
}
