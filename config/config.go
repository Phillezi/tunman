package config

import (
	"errors"
	"os"
	"path"

	"github.com/Phillezi/tunman/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// InitConfig initializes configuration
func InitConfig(filenames ...string) {
	if len(filenames) > 0 {
		filenames = append(filenames, "config")
	} else {
		filenames = []string{"config"}
	}
	viper.SetConfigName(utils.Or(filenames...)) // Name of the config file (without extension)
	viper.SetConfigType("yaml")                 // File format (yaml)
	viper.AddConfigPath(GetConfigPath())
	viper.AutomaticEnv() // Read environment variables

	// Load config file
	if err := viper.ReadInConfig(); err != nil {
		zap.L().Debug("Config file not found, using defaults or environment variables.")
	} else {
		zap.L().Debug("Using config file", zap.String("configFile", viper.ConfigFileUsed()))
	}

}

func getConfigPath() (string, error) {
	basePath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configPath := path.Join(basePath, ".tunman")
	fileDescr, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			return "", err
		}
		fileDescr, err = os.Stat(configPath)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	if !fileDescr.IsDir() {
		return "", errors.New("default config dir is file")
	}
	return configPath, nil
}

func GetConfigPath() string {
	configPath, err := getConfigPath()
	if err != nil {
		zap.L().Warn("error getting config path", zap.Error(err))
		zap.L().Info("defaulting to", zap.String("cwd", func() string {
			wd, err := os.Getwd()
			if err != nil {
				return "."
			}
			return wd
		}()))
		configPath = "."
	}
	return configPath
}
