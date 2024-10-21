/*
Copyright © 2024 Zac Orndorff zac@orndorff.dev
*/
package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var logLevel slog.LevelVar

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
	SetJsonHandler(&logLevel)
	SetLogLevel("info", &logLevel)

	cobra.OnInitialize(InitConfig)

	rootCmd.PersistentFlags().StringP("workspace", "w", "dev", "workspace folder (default is $HOME/.dt/dev)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dt/.dt.yaml)")
	err := viper.BindPFlag("workspace", rootCmd.PersistentFlags().Lookup("workspace"))
	cobra.CheckErr(err)
}
