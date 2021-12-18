package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func ReadConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("bucket-stream")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.WithError(err).Fatal("could not read config file")
	}
}
