package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestEnsureConfig(t *testing.T) {
	vip := viper.New()
	vip.Set("workspace", "test_workspace")

	configDirPath := "./tmp/test_config_dir"

	// Assert that the workspace database is set
	dbPath := fmt.Sprintf("%s/.dt/dt.db", configDirPath)
	_, err := os.Stat(dbPath)
	assert.NoError(t, err)

	// Assert that the workspace folder is created
	workspacePath := fmt.Sprintf("%s/test_workspace", configDirPath)
	_, err = os.Stat(workspacePath)
	assert.NoError(t, err)

	// Assert that the config file is written
	configFilePath := fmt.Sprintf("%s/.dt/config.yaml", configDirPath)
	_, err = os.Stat(configFilePath)
	assert.NoError(t, err)
}
