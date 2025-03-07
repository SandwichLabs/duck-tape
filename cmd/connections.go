/*
Copyright © 2024 Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"github.com/SandwichLabs/duck-tape/connection"
	"github.com/SandwichLabs/duck-tape/workspace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
)

var setConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Set a connection",
	Run: func(cmd *cobra.Command, args []string) {
		workspaceStr := viper.GetString("workspace")
		connection := connection.ConnectionConfigForm()
		slog.Info("saving connection", "connection", connection)
		_, err := workspace.SetWorkspaceConnection(workspaceStr, connection, true)
		cobra.CheckErr(err)
	},
}
