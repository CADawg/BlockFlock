package main

import (
	"os"

	"github.com/goccy/go-json"
)

type ConfigStruct struct {
	Domain             string `json:"domain"`
	Node               string `json:"node"`
	BlockchainEndpoint string `json:"blockchainEndpoint"`
	ContractsEndpoint  string `json:"contractsEndpoint"`
}

func init() {
	// load config.json
	cfg, err := os.ReadFile("./config.json")

	if err != nil {
		panic("config.json not found")
	}

	// load config.json into config struct
	err = json.Unmarshal(cfg, &config)

	if err != nil {
		panic("config.json is invalid")
	}
}
