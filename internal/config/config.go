package config

import (
	"encoding/json"
	"os"

	"github.com/kyrnas/gator/internal/database"
)

const configFileName = "/.gatorconfig.json"

type State struct {
	Conf *Config
	Queries *database.Queries
}

type Config struct {
	DbUrl string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homedir + configFileName, nil
}

func writeConfig(conf Config) error {
	rawData, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	err = os.WriteFile(configFilePath, rawData, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func Read() (Config, error) {
	var config Config
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return config, err
	}
	rawData, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}
	if err = json.Unmarshal(rawData, &config); err != nil {
		return config, err
	}
	return config, nil
}

func (conf *Config) SetUser(user string) error {
	conf.CurrentUserName = user
	err := writeConfig(*conf)
	return err
}