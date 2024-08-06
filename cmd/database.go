package cmd

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/marcboeker/go-duckdb"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

type DatabaseClient struct {
	config Config
	*duckdb.Connector
}

type Config struct {
	NumThreads   int
	Plugins      []string
	Connections  []ConnectionConfig
	DatabasePath string
	Workspace    string
	BootQueries  []string
}

func NewDatabaseClient(options ...func(*DatabaseClient)) *DatabaseClient {
	svr := &DatabaseClient{}
	for _, o := range options {
		o(svr)
	}
	return svr
}

func WithNumThreads(num int) func(*DatabaseClient) {
	return func(c *DatabaseClient) {
		c.config.NumThreads = num
	}
}

func WithPlugins(plugins []string) func(*DatabaseClient) {
	return func(c *DatabaseClient) {
		c.config.Plugins = plugins
	}
}

// Setup the connectionConfig with the connection details, plugins.
func WithConnectionsByName(connectionNames []string) func(*DatabaseClient) {
	return func(c *DatabaseClient) {

		connectionConfigs := []ConnectionConfig{}
		pluginsList := []string{}

		for _, connection_name := range connectionNames {
			conn, err := WorkspaceConnection(c.config.Workspace, connection_name)

			cobra.CheckErr(err)

			connectionConfigs = append(connectionConfigs, conn)
			pluginsList = append(pluginsList, conn.Type)
		}

		c.config.Plugins = pluginsList
		c.config.Connections = connectionConfigs
	}
}

func WithWorkspace(workspace string) func(*DatabaseClient) {
	return func(c *DatabaseClient) {
		c.config.Workspace = workspace
	}
}

func WithDatabasePath(path string) func(*DatabaseClient) {
	return func(c *DatabaseClient) {
		c.config.DatabasePath = path
	}
}

func WithBootQueries(queries []string) func(*DatabaseClient) {
	return func(c *DatabaseClient) {
		c.config.BootQueries = queries
	}
}

func InitDatabaseClient() func(*DatabaseClient) {
	return func(c *DatabaseClient) {
		connector, err := duckdb.NewConnector(fmt.Sprintf("%s?threads=%d", c.config.DatabasePath, c.config.NumThreads), func(execer driver.ExecerContext) error {
			var bootQueries []string

			for _, plugin := range c.config.Plugins {
				bootQueries = append(bootQueries, fmt.Sprintf("INSTALL '%s'", plugin))
				bootQueries = append(bootQueries, fmt.Sprintf("LOAD '%s'", plugin))
			}

			for _, attachment := range c.config.Connections {
				bootQueries = append(bootQueries, fmt.Sprintf("ATTACH '%s' as %s (TYPE %s, READ_ONLY);", attachment.ConnString, attachment.Name, attachment.Type))
			}

			for _, query := range bootQueries {
				_, errs := execer.ExecContext(context.Background(), query, nil)
				if errs != nil {
					slog.Error("Error running initial boot setup", "Error", errs)
				}
				slog.Debug("Executed boot query", "query", query)
			}
			return nil
		})

		if err != nil {
			slog.Error("Error initializing database client", err)
		}

		c.Connector = connector
	}
}

func ConnectionPlugins([]string) []string {
	return plugins
}

func OpenConnection(conn DatabaseClient) (*sql.DB, error) {
	db := sql.OpenDB(conn.Connector)
	return db, nil
}

func Prepare(db *sql.DB, query string) (*sql.Stmt, error) {
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

func Query(db *sql.DB, query string) (*sql.Rows, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	return rows, nil
}
