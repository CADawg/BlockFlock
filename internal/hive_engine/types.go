package hive_engine

type DisabledMethods struct {
	Blockchain []string `json:"blockchain,omitempty"`
	Contracts  []string `json:"contracts,omitempty"`
	Message    string   `json:"message,omitempty"`
}

type NodeInfo struct {
	Success                     bool            `json:"success"`
	LastBlockNumber             int64           `json:"lastBlockNumber"`
	LastBlockRefHiveBlockNumber int64           `json:"lastBlockRefHiveBlockNumber"`
	LastHash                    string          `json:"lastHash"`
	LastParsedHiveBlockNumber   int64           `json:"lastParsedHiveBlockNumber"`
	SSCNodeVersion              string          `json:"SSCnodeVersion"`
	Domain                      string          `json:"domain"`
	ChainID                     string          `json:"chainId"`
	DisabledMethods             DisabledMethods `json:"disabledMethods,omitempty"`
	LightNode                   bool            `json:"lightNode"`
	LastVerifiedBlockNumber     int64           `json:"lastVerifiedBlockNumber"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

type GetBlockInfoParams struct {
	BlockNumber int64 `json:"blockNumber"`
}
