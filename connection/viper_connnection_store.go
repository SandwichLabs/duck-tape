package connection

import "github.com/spf13/viper"

type ViperConnectionStore struct {
	viper *viper.Viper
}

func NewViperConnectionStore(configStore *viper.Viper) *ViperConnectionStore {

	return &ViperConnectionStore{
		viper: configStore,
	}
}

// Implement Get method
func (vcs *ViperConnectionStore) Get(name string) (*ConnectionConfig, error) {
	configConn := vcs.viper.Sub("connections." + name)
	if configConn == nil {
		return nil, nil
	}
	conn := &ConnectionConfig{}
	err := configConn.Unmarshal(conn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
