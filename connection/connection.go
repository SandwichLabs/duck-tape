/*
Copyright © 2024 Zac Orndorff zac@orndorff.dev
*/
package connection

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ConnectionConfig struct {
	ConnString  string `yaml:"conn_string"`
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	EnableWrite bool   `yaml:"enable_write"` // Optional field to enable write operations
}

func (c ConnectionConfig) String() string {
	return fmt.Sprintf("ConnectionConfig{ConnString: %s, Name: %s, Type: %s}", c.ConnString, c.Name, c.Type)
}

func (c ConnectionConfig) ReadWriteMode() string {
	if c.EnableWrite {
		return ""
	}
	return ", READ_ONLY"
}

func ConnectionFromViper(v *viper.Viper) ConnectionConfig {
	return ConnectionConfig{
		ConnString:  v.GetString("conn_string"),
		Name:        v.GetString("name"),
		Type:        v.GetString("type"),
		EnableWrite: v.GetBool("enable_write"), // Read the optional field from viper
	}
}

func ConnectionConfigForm() ConnectionConfig {
	var (
		name        string
		conn        string
		connType    string
		enableWrite bool
	)
	// Set default values for the form fields
	enableWrite = false

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Connection Name").
				Value(&name),
			huh.NewSelect[string]().
				Title("Connection Type.").
				Options(
					huh.NewOption("PostgreSql", "POSTGRES"),
					huh.NewOption("Http(s3, http)", "HTTPSFS"),
					huh.NewOption("MySql", "MYSQL"),
					huh.NewOption("SQLite", "SQLITE"),
				).
				Value(&connType),
			huh.NewInput().
				Title("Connection String").
				Value(&conn),
			huh.NewConfirm().
				Title("Enable writes?").
				Value(&enableWrite),
		))

	err := form.Run()
	cobra.CheckErr(err)

	return ConnectionConfig{
		Name:       name,
		ConnString: conn,
		Type:       connType,
	}
}
