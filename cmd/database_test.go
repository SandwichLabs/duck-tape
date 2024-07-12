package cmd_test

import (
	"testing"

	"github.com/SandwichLabs/duck-tape/cmd"
	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	// Create a test client
	client := cmd.NewDatabaseClient(
		cmd.WithNumThreads(4),
		cmd.WithWorkspace("test_workspace"),
		cmd.WithDatabasePath("test.db"),
		cmd.InitDatabaseClient(),
	)

	// Open the database connection
	db, err := cmd.OpenConnection(*client)
	assert.NoError(t, err)
	defer db.Close()

	// Perform a simple query
	rows, err := cmd.Query(db, "SELECT 1+1 as answer")
	assert.NoError(t, err)
	defer rows.Close()

	// Assert the number of rows returned
	count := 0
	for rows.Next() {
		count++
	}
	assert.Equal(t, 1, count)
}

func TestPrepare(t *testing.T) {
	// Create a test client
	client := cmd.NewDatabaseClient(
		cmd.WithNumThreads(4),
		cmd.WithWorkspace("test_workspace"),
		cmd.WithDatabasePath("test.db"),
		cmd.InitDatabaseClient(),
	)

	// Open the database connection
	db, err := cmd.OpenConnection(*client)
	assert.NoError(t, err)
	defer db.Close()

	// Prepare a statement
	stmt, err := cmd.Prepare(db, "SELECT 1+? as answer")
	assert.NoError(t, err)
	defer stmt.Close()

	// Execute the prepared statement
	rows, err := stmt.Query(1)
	assert.NoError(t, err)
	defer rows.Close()

	// Assert the number of rows returned
	count := 0
	for rows.Next() {
		count++
	}
	assert.Equal(t, 1, count)
}
