/*
Copyright © Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// createCmd represents the get command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Parent command for creating resources",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		cobra.CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(addConnectionCmd)
}
