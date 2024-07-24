package connection

import "github.com/spf13/viper"

type ViperConnectionRepository struct {
	viper *viper.Viper
}

func NewViperConnectionRepository(configStore *viper.Viper) *ViperConnectionRepository {

	return &ViperConnectionRepository{
		viper: configStore,
	}
}
