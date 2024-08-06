/*
Copyright © 2024 Zac Orndorff zac@orndorff.dev
*/
package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ConnectionConfig struct {
	ConnString string `yaml:"conn_string"`
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
}

func (c ConnectionConfig) String() string {
	return fmt.Sprintf("ConnectionConfig{ConnString: %s, Name: %s, Type: %s}", c.ConnString, c.Name, c.Type)
}

func ConnectionFromViper(v *viper.Viper) ConnectionConfig {
	return ConnectionConfig{
		ConnString: v.GetString("ConnString"),
		Name:       v.GetString("Name"),
		Type:       v.GetString("Type"),
	}
}

func ConnectionConfigForm() ConnectionConfig {
	var (
		name     string
		conn     string
		connType string
	)

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
		))

	err := form.Run()
	cobra.CheckErr(err)

	return ConnectionConfig{
		Name:       name,
		ConnString: conn,
		Type:       connType,
	}
}
