package config

import (
	"github.com/goccy/go-json"
	"os"
)

type Config struct {
	Domain             string `json:"domain"`
	Node               string `json:"node"`
	BlockchainEndpoint string `json:"blockchainEndpoint"`
	ContractsEndpoint  string `json:"contractsEndpoint"`
}

func LoadConfig(path string) (*Config, error) {
	var Config Config

	// read config file
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	// unmarshal config
	err = json.Unmarshal(data, &Config)

	if err != nil {
		return nil, err
	}

	return &Config, nil
}
