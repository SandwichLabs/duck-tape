package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/SandwichLabs/dt/config"
	"github.com/charmbracelet/log"
	"github.com/marcboeker/go-duckdb"
	"github.com/spf13/cobra"
)

type Client struct {
	config Config
	*duckdb.Connector
}

type Config struct {
	NumThreads   int
	Plugins      []string
	Connections  []config.ConnectionConfig
	DatabasePath string
	Workspace    string
	BootQueries  []string
}

func New(options ...func(*Client)) *Client {
	svr := &Client{}
	for _, o := range options {
		o(svr)
	}
	return svr
}

func WithNumThreads(num int) func(*Client) {
	return func(c *Client) {
		c.config.NumThreads = num
	}
}

func WithPlugins(plugins []string) func(*Client) {
	return func(c *Client) {
		c.config.Plugins = plugins
	}
}

func WithConnectionStrings(connections []string) func(*Client) {
	return func(c *Client) {

		connectionConfigs := []config.ConnectionConfig{}

		for _, connection_name := range connections {
			conn, err := config.WorkspaceConnection(c.config.Workspace, connection_name)

			cobra.CheckErr(err)

			connectionConfigs = append(connectionConfigs, conn)
		}

		c.config.Connections = connectionConfigs
	}
}

func WithWorkspace(workspace string) func(*Client) {
	return func(c *Client) {
		c.config.Workspace = workspace
	}
}

func WithDatabasePath(path string) func(*Client) {
	return func(c *Client) {
		c.config.DatabasePath = path
	}
}

func WithBootQueries(queries []string) func(*Client) {
	return func(c *Client) {
		c.config.BootQueries = queries
	}
}

func Init() func(*Client) {
	return func(c *Client) {
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
					log.Fatal(errs)
				}
				log.Debug("executed: ", query)
			}
			return nil
		})

		if err != nil {
			log.Fatal(err)
		}

		c.Connector = connector
	}
}

func Open(conn Client) (*sql.DB, error) {
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
