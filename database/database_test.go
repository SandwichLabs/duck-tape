package database_test

import (
	"testing"

	"github.com/SandwichLabs/duck-tape/database"
	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	// Create a test client
	client := database.New(
		database.WithNumThreads(4),
		database.WithWorkspace("test_workspace"),
		database.WithDatabasePath("test.db"),
		database.Init(),
	)

	// Open the database connection
	db, err := database.Open(*client)
	assert.NoError(t, err)
	defer db.Close()

	// Perform a simple query
	rows, err := database.Query(db, "SELECT 1+1 as answer")
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
	client := database.New(
		database.WithNumThreads(4),
		database.WithWorkspace("test_workspace"),
		database.WithDatabasePath("test.db"),
		database.Init(),
	)

	// Open the database connection
	db, err := database.Open(*client)
	assert.NoError(t, err)
	defer db.Close()

	// Prepare a statement
	stmt, err := database.Prepare(db, "SELECT 1+? as answer")
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
