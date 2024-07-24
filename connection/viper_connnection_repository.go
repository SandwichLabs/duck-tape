package connection

type ViperConnectionRepository struct {
	GetAllFunc         func() ([]*ConnectionConfig, error)
	GetFunc            func(name string) (*ConnectionConfig, error)
	CreateOrUpdateFunc func(c *ConnectionConfig) error
	DeleteFunc         func(name string) error
}

func NewViperConnectionRepository(hourFactory Factory) *ViperConnectionRepository {

	return &ViperConnectionRepository{}
}
