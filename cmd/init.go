/*
Copyright © 2024 Zac Orndorff <zac@orndorff.dev>
*/
package cmd

import (
	"github.com/SandwichLabs/dt/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// initCmd represents the get command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your ducktape installation",
	Long: `Initialize your ducktape installation.
	Usage: dt init
	`,
	Run: func(cmd *cobra.Command, args []string) {
		homeDir, err := os.UserHomeDir() // get the user's home directory, default location for config
		cobra.CheckErr(err)
		log.Printf("Initializing config file in: %s", homeDir)
		config.EnsureConfig(viper.GetViper(), homeDir)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
