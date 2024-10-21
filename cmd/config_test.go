package cmd_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/SandwichLabs/duck-tape/cmd"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestEnsureConfig(t *testing.T) {
	viper.Set("workspace", "test_workspace")

	configDirPath := cmd.GetConfigPath()

	cmd.InitConfig()

	// Assert that the workspace folder is created
	workspacePath := fmt.Sprintf("%s/test_workspace", configDirPath)
	defer os.RemoveAll(workspacePath)

	_, err := os.Stat(workspacePath)
	assert.NoError(t, err)

	// Assert that the config file is written
	configFilePath := fmt.Sprintf("%s/config.yaml", configDirPath)
	_, err = os.Stat(configFilePath)
	assert.NoError(t, err)
}
