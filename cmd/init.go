/*
Copyright © 2024 Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"github.com/SandwichLabs/dt/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the get command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your ducktape installation",
	Long: `Initialize your ducktape installation.
	Usage: dt init
	`,
	Run: func(cmd *cobra.Command, args []string) {
		config.EnsureConfig(viper.GetViper())
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
