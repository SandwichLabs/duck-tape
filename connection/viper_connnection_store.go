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
