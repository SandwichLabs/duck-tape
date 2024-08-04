package connection

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestViperConnectionStore_Get(t *testing.T) {
	// Create a new Viper instance
	v := viper.New()

	// Set up the test data
	testData := map[string]interface{}{
		"connections": map[string]interface{}{
			"test": map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"username": "testuser",
				"password": "testpass",
			},
		},
	}
	v.Set("connections", testData["connections"])

	// Create a new ViperConnectionStore instance
	vcs := NewViperConnectionStore(v)

	// Test getting an existing connection
	conn, err := vcs.Get("test")
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, "localhost", conn.Host)
	assert.Equal(t, 5432, conn.Port)
	assert.Equal(t, "testuser", conn.Username)
	assert.Equal(t, "testpass", conn.Password)

	// Test getting a non-existing connection
	conn, err = vcs.Get("nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, conn)
}
