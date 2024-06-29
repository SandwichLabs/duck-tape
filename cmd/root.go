/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/SandwichLabs/dt/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dt",
	Short: "dt, or 'duck tape' is a tool for building delightful data shell scripts.",
	Long: `
	dt, or 'duck tape' for building delightful shell scripts and data pipelines.
	Usage examples:
		# Read a csv file from a remote url.
		dt query "select * from 'https://example.com/some_csv.csv'"

		# Read a csv file from a local file.
		dt query "select * from './path/to/some_csv.csv'"

		# Read a csv file from a local file and transform it using jq.
		dt query "select * from './path/to/some_csv.csv'" | jq .some_field
	
	dt supports most operations that you would expect from DuckDB, with additional helper functions for managing connections and workspaces.
	
	See the DuckDB documentation for more information:
	https://duckdb.org/docs
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("workspace", "w", "dev", "workspace folder (default is $HOME/.dt/dev)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dt/.dt.yaml)")
	err := viper.BindPFlag("workspace", rootCmd.PersistentFlags().Lookup("workspace"))
	cobra.CheckErr(err)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
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
		viper.AddConfigPath(configPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")

		config.EnsureWorkspace(configPath, viper.GetString("workspace"))

		cobra.CheckErr(err)

		err = viper.WriteConfig()
		cobra.CheckErr(err)

		log.Debugf("configPath: %s", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}
}
