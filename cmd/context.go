package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"os"
	"strings"
)

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Evaluate connected databases and format a 'context snippet' for llm tasks.",
	Long: `Evaluate connected databases and format a 'context snippet' for llm tasks.
Example:
  dt context -c my_postgres_db
  dt context --fragment evidence.dev
`,
	Run: func(cmd *cobra.Command, args []string) {
		connectionNames, _ := cmd.Flags().GetStringArray("connections")
		workspace := viper.GetString("workspace")
		fragments, err := cmd.Flags().GetStringArray("fragments")

		cobra.CheckErr(err)

		runSummary, err := cmd.Flags().GetBool("summary")

		cobra.CheckErr(err)

		dbPath := viper.GetString(fmt.Sprintf("%s.dbLocation", workspace))
		slog.Debug("Database path:", "dbPath", dbPath)

		slog.Debug("Using database", "path", dbPath)

		client := NewDatabaseClient(
			WithNumThreads(4), // Or get from config/flags
			WithWorkspace(workspace),
			WithDatabasePath(dbPath),
			WithConnectionsByName(connectionNames), // Reuse connection logic
			InitDatabaseClient(),
		)

		db, err := OpenConnection(*client)
		cobra.CheckErr(err)
		defer db.Close()
		slog.Debug("Database connection established", "workspace", workspace, "db", dbPath)

		ioOutputStream := os.Stdout
		// --- Gather Context ---
		slog.Debug("Gathering database context...")
		// Add database info
		ioOutputStream.WriteString("<database_info>\n")
		ioOutputStream.WriteString("<schema>\n")
		err = getSchemaMarkdown(db, ioOutputStream)
		cobra.CheckErr(err)
		slog.Debug("Schema gathered")

		ioOutputStream.WriteString("\n</schema>\n")

		if runSummary {
			ioOutputStream.WriteString("\n<summary>\n")
			err = getDataSummaryMarkdown(db, ioOutputStream)
			cobra.CheckErr(err)
			slog.Debug("Data summaries gathered")
			ioOutputStream.WriteString("\n</summary>\n")
		}

		ioOutputStream.WriteString("</database_info>\n")
		// Add fragments if requested
		if fragments != nil {
			slog.Debug("Including fragments in the prompt", "fragments", fragments)
			for _, fragment := range fragments {
				fragmentContent, err := os.ReadFile(fragment)
				if err != nil {
					slog.Warn("Could not read fragment file, proceeding without it.", "path", fragment, "error", err)
				} else {
					ioOutputStream.WriteString(fmt.Sprintf("<document path=%s>\n", fragment))
					ioOutputStream.Write(fragmentContent)
					ioOutputStream.WriteString("\n</document>")
				}
			}
		}

		slog.Debug("Enjoy!")
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.Flags().StringArrayP("connections", "c", []string{}, "One or more connection configurations to attach (same as query)")
	contextCmd.Flags().StringArrayP("fragments", "f", []string{}, "Include additional context from a file (can be used multiple times)")
	contextCmd.Flags().Bool("summary", false, "Include the results of running SUMMARIZE on the tables in the database")

}

// Fetches schema and formats as markdown
func getSchemaMarkdown(db *sql.DB, ioOut *os.File) error {
	rows, err := db.QueryContext(context.Background(), "SHOW ALL TABLES;")
	if err != nil {
		return fmt.Errorf("failed to show tables: %w", err)
	}
	defer rows.Close()

	// Simulate markdown table output
	ioOut.WriteString("| database | schema | name | column_names | column_types | temporary |\n")
	ioOut.WriteString("|---|---|---|---|---|---|\n")

	cols, _ := rows.Columns() // Get column names for scanning
	pointers := make([]interface{}, len(cols))
	container := make([]interface{}, len(cols)) // Use interface{} for flexibility
	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return fmt.Errorf("failed to scan table row: %w", err)
		}
		ioOut.WriteString("| ")
		for i := range cols {
			if container[i] == nil {
				ioOut.WriteString("NULL")
			} else {
				ioOut.WriteString(fmt.Sprintf("%v", container[i]))
			}
			ioOut.WriteString(" | ")
		}
		ioOut.WriteString("\n")
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating table rows: %w", err)
	}

	return nil
}

// Fetches table names and their summaries, formats as markdown
func getDataSummaryMarkdown(db *sql.DB, ioOut *os.File) error {
	tableNames, err := getTableNames(db)
	if err != nil {
		return err
	}

	ioOut.WriteString("Summary of the table contents:\n\n")

	for _, tableName := range tableNames {
		slog.Debug("Getting summary for table", "table", tableName)
		// Sanitize table name just in case, although SHOW TABLES should be safe
		// A more robust approach would use parameterized queries if table names came from user input
		summaryQuery := fmt.Sprintf("SUMMARIZE TABLE %s;", strings.ReplaceAll(tableName, "\"", "\"\"")) // Basic quoting for safety

		summaryRows, err := db.QueryContext(context.Background(), summaryQuery)
		if err != nil {
			slog.Warn("Failed to summarize table", "table", tableName, "error", err)
			ioOut.WriteString(fmt.Sprintf("## %s:\n\n_Error fetching summary: %v_\n\n", tableName, err))
			continue // Skip to next table on error
		}

		defer summaryRows.Close() // Close rows for each summary query

		ioOut.WriteString(fmt.Sprintf("## %s:\n\n", tableName))
		summaryMd, err := formatRowsToMarkdown(summaryRows)
		if err != nil {
			slog.Warn("Failed to format summary to markdown", "table", tableName, "error", err)
			ioOut.WriteString(fmt.Sprintf("_Error formatting summary: %v_\n\n", err))
		} else {
			ioOut.WriteString(summaryMd)
			ioOut.WriteString("\n")
		}
	}

	return nil
}

// Helper to get table names
func getTableNames(db *sql.DB) ([]string, error) {
	// Slightly refined query to potentially exclude duckdb system tables if desired
	rows, err := db.QueryContext(context.Background(), "SELECT ( database || '.' || schema || '.' || name) as name FROM (SHOW ALL TABLES) WHERE name NOT LIKE 'sqlite_%' ORDER BY name;")
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		names = append(names, name)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table names: %w", err)
	}
	return names, nil
}

// Helper to format sql.Rows to a simple markdown table
func formatRowsToMarkdown(rows *sql.Rows) (string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	var md strings.Builder
	md.WriteString("| " + strings.Join(cols, " | ") + " |\n")
	md.WriteString("|" + strings.Repeat("---|", len(cols)) + "\n")

	// Use interface{} to handle different types, including duckdb.Decimal
	pointers := make([]interface{}, len(cols))
	container := make([]interface{}, len(cols)) // Use interface{} for flexibility
	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		md.WriteString("| ")
		for i := range cols {
			// Handle potential NULLs and unsupported types
			val := container[i]
			if val == nil {
				md.WriteString("NULL")
			} else {
				switch v := val.(type) {
				case []byte:
					md.WriteString(string(v)) // Handle RawBytes
				case string:
					md.WriteString(v)
				case float64, int64, int, float32:
					md.WriteString(fmt.Sprintf("%v", v))
				default:
					md.WriteString(fmt.Sprintf("%v", v)) // Fallback for other types
				}
			}
			md.WriteString(" | ")
		}
		md.WriteString("\n")
	}
	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating rows: %w", err)
	}

	return md.String(), nil
}

// Helper to limit the number of lines shown in confirmation
func limitStringLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > maxLines {
		return strings.Join(lines[:maxLines], "\n") + "\n... (output truncated)"
	}
	return s
}
