/*
Copyright © 2024 NAME HERE <zac@orndorff.dev>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/SandwichLabs/dt/config"
	"github.com/SandwichLabs/dt/database"
	"github.com/SandwichLabs/dt/format"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// transformCmd represents the transform command
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "query datasources",
	Long: `Runs the query against the datasource.
	Basic Usage: 
	dt query "create table test (id int, name text);"
	dt query "insert into test (id, name) values (1, 'test');"
	dt query "select * from test;"

	With parameters:
	dt query "select * from test where id = ?;" -p 1

	With connections:
	dt create connection
	dt query "select * from connectionName.test;" -c connectionName 
	
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workspace := viper.GetString("workspace")
		workspaceRoot := config.WorkspacePath(workspace)

		dbPath := fmt.Sprintf("%s/%s", workspaceRoot, viper.GetString(fmt.Sprintf("%s.dbLocation", workspace)))
		log.Infof("Database path: %s", dbPath)

		query := args[0]

		connections, _ := cmd.Flags().GetStringArray("connections")

		client := database.New(
			database.WithNumThreads(4),
			database.WithPlugins([]string{"json", "httpfs", "postgres"}),
			database.WithWorkspace(workspace),
			database.WithConnectionStrings(connections),
			database.WithDatabasePath(dbPath),
			database.Init(),
		)

		db, err := database.Open(*client)

		cobra.CheckErr(err)

		defer db.Close()

		stmt, err := db.PrepareContext(context.Background(), query)

		cobra.CheckErr(err)
		queryParams, err := cmd.Flags().GetStringArray("param")

		cobra.CheckErr(err)

		interfaceParams := make([]interface{}, len(queryParams))

		for i, v := range queryParams {
			interfaceParams[i] = v
		}

		rows, err := stmt.QueryContext(context.Background(), interfaceParams...)

		cobra.CheckErr(err)

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			panic(err.Error())
		}

		// Make a slice for the values
		values := make([]interface{}, len(columns))

		// rows.Scan wants '[]interface{}' as an argument, so we must copy the
		// references into such a slice
		// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
		scanArgs := make([]interface{}, len(values))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		defer rows.Close()
		// Fetch rows
		for rows.Next() {
			err = rows.Scan(scanArgs...)

			cobra.CheckErr(err)
			// Make an interface slice to hold the values of each row that can be marshalled to JSON
			valueMap := make(map[string]interface{})
			for i, col := range values {
				valueMap[columns[i]] = col
			}

			valueString, err := format.ToJsonString(valueMap)
			cobra.CheckErr(err)

			fmt.Fprintln(cmd.OutOrStdout(), valueString)
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringArrayP("connections", "c", []string{}, "One or more connection configurations to attach")
	queryCmd.Flags().StringArrayP("param", "p", []string{}, "One or more parameters to pass to the query")
	// Optionally save the query to the ducktape folder for later use
	queryCmd.Flags().BoolP("save", "s", false, "Save the query to the ducktape folder")
}
