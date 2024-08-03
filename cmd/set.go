/*
Copyright © Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// setCmd represents the get command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Parent command for setting connections, workspaces, etc.",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		cobra.CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
	setCmd.AddCommand(setConnectionCmd)
}
