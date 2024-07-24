package connection

type ConnectionConfig struct {
	ConnString string `yaml:"conn_string"`
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Plugin     string `yaml:"plugin"`
}
