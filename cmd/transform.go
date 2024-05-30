/*
Copyright © 2024 NAME HERE <zac@orndorff.dev>
*/
package cmd

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

// transformCmd represents the transform command
var transformCmd = &cobra.Command{
	Use:   "transform",
	Short: "do stuff with stdin",
	Long:  `read standard input, do stuff with it.`,
	Run: func(cmd *cobra.Command, args []string) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Printf("This test worked %s \n", scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(transformCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// transformCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// transformCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
