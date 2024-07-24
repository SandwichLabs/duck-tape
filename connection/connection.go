package connection

import (
	"fmt"

	"github.com/spf13/viper"
)

type ConnectionConfig struct {
	ConnString string `yaml:"conn_string"`
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
}

type Factory() *ConnectionConfig {

	
}

func (c *ConnectionConfig) String() string {
	return fmt.Sprintf("ConnectionConfig{ConnString: %s, Name: %s, Type: %s}", c.ConnString, c.Name, c.Type)
}

func FromViper(v *viper.Viper) *ConnectionConfig {
	return &ConnectionConfig{
		ConnString: v.GetString("ConnString"),
		Name:       v.GetString("Name"),
		Type:       v.GetString("Type"),
	}
}
