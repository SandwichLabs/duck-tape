/*
Copyright © Zac Orndorff zac@orndorff.dev
*/
package workspace

import (
	"errors"
	"fmt"
	"github.com/SandwichLabs/duck-tape/connection"
	"github.com/spf13/viper"
	"log/slog"
)

func getWorkspaceConnectionKey(workspace string, connectionName string) string {
	return fmt.Sprintf("%s.connections.%s", workspace, connectionName)
}

func SetWorkspaceDb(workspace string, name string, save bool) (ok bool, err error) {
	viper.Set(fmt.Sprintf("%s.dbLocation", workspace), name)
	if save {
		err := viper.WriteConfig()
		if err != nil {
			slog.Error("error writing config", "error", err)
			return false, errors.New("error saving workspace db location")
		}
	}
	return true, nil
}

func SetWorkspaceConnection(workspace string, connection connection.ConnectionConfig, save bool) (ok bool, err error) {
	viper.Set(getWorkspaceConnectionKey(workspace, connection.Name), connection)
	if save {
		err := viper.WriteConfig()
		if err != nil {
			slog.Error("SetWorkSpaceConnection Error", "Error", err)
			return false, errors.New("error setting workspace connection")
		}
	}
	return true, nil
}

func WorkspaceConnection(workspace string, connectionName string) (connection.ConnectionConfig, error) {
	configConn := viper.Sub(getWorkspaceConnectionKey(workspace, connectionName))
	if configConn == nil {
		return connection.ConnectionConfig{}, errors.New("connection not found in workspace")
	}
	return connection.ConnectionFromViper(configConn), nil
}

func ListWorkspaceConnections(workspace string) ([]string, error) {
	workspaceConnections := viper.Sub(fmt.Sprintf("%s.connections", workspace))

	if workspaceConnections == nil {
		return nil, errors.New("no connections found in workspace")
	}
	// Map of all keys in the workspaceConnections and return a string array container the 'name' of each connection

	return workspaceConnections.AllKeys(), nil
}
