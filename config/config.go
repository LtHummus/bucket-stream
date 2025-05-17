package config

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var ReadConfiguration Configuration

func ReadConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("bucket-stream")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.WithError(err).Fatal("could not read config file")
	}

	err = viper.Unmarshal(&ReadConfiguration)
	if err != nil {
		log.WithError(err).Fatal("could not unmarshal config")
	}
}

func SaveConfig() error {
	yamlData, err := yaml.Marshal(ReadConfiguration.AsMap())
	if err != nil {
		return fmt.Errorf("config: SaveConfig: could not marshall config: %w", err)
	}

	f, err := os.OpenFile(viper.ConfigFileUsed(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("config: SaveConfig: could not create file for writing: %w", err)
	}
	defer f.Close()

	_, err = f.Write(yamlData)
	if err != nil {
		return fmt.Errorf("config: SaveConfig: could not write new config data: %w", err)
	}

	return nil
}
