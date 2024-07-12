/*
Copyright © 2024 Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var addConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Add a new connection",
	Run: func(cmd *cobra.Command, args []string) {
		workspace := viper.GetString("workspace")
		connection := ConnectionConfigForm()
		slog.Info("saving connection", "connection", connection)
		_, err := SetWorkspaceConnection(workspace, connection, true)
		cobra.CheckErr(err)
	},
}
