package connection

type Repository interface {
	// Get all connections
	GetAll() ([]*ConnectionConfig, error)
	// Get a connection by name
	Get(name string) (*ConnectionConfig, error)
	// Create a new connection
	CreateOrUpdate(c *ConnectionConfig) error
	// Delete a connection by name
	Delete(name string) error
}
