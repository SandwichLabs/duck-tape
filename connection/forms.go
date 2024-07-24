package connection

import (
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

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
