package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/dgraph-io/badger"
	"github.com/goccy/go-json"

	"github.com/xujiajun/gorouter"
)

type DisabledMethods struct {
	Blockchain []string `json:"blockchain,omitempty"`
	Contracts  []string `json:"contracts,omitempty"`
	Message    string   `json:"message,omitempty"`
}

type EngineInfo struct {
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

type EngineInfoError struct {
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

type JSONRPCRequest struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type GetBlockInfoParams struct {
	BlockNumber int64 `json:"blockNumber"`
}

func ValidateRequest(j JSONRPCRequest) (bool, error) {
	if j.Version != "2.0" {
		return false, fmt.Errorf("invalid version")
	}

	if j.ID == nil {
		return false, fmt.Errorf("invalid id")
	}

	if j.Method == "" {
		return false, fmt.Errorf("invalid method")
	}

	return true, nil
}

var db *badger.DB

var config ConfigStruct

var signals chan os.Signal

func main() {
	app := gorouter.New()

	// todo: when a request is send in it should be checked for errors and then all the rest forwarded on in the same batch to save resources, time and requests
	// todo: batch get block info requests

	app.GET("/", func(res http.ResponseWriter, req *http.Request) {
		err := SendIndex(res)

		if err != nil {
			json.NewEncoder(res).Encode(EngineInfoError{
				Error:   err.Error(),
				Success: false,
			})
		}
	})

	app.POST("/blockchain", func(res http.ResponseWriter, req *http.Request) {
		requests, err, noBrackets := ReadJRPCBody(req)

		if err != nil {
			SendUnparsableErrorResponse(res, "Error processing request - "+err.Error())
			return
		}

		responses, err := HandleRequests(requests, "blockchain", noBrackets)

		if err != nil {
			SendErrorResponse(res, requests, err)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(responses)
	})

	app.POST("/contracts", func(res http.ResponseWriter, req *http.Request) {
		requests, err, noBrackets := ReadJRPCBody(req)

		if err != nil {
			SendUnparsableErrorResponse(res, "Error processing request - "+err.Error())
			return
		}

		responses, err := HandleRequests(requests, "contracts", noBrackets)

		if err != nil {
			SendErrorResponse(res, requests, err)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(responses)
	})

	app.POST("/", func(res http.ResponseWriter, req *http.Request) {
		requests, err, noBrackets := ReadJRPCBody(req)

		if err != nil {
			SendUnparsableErrorResponse(res, "Error processing request - "+err.Error())
			return
		}

		responses, err := HandleRequests(requests, "", noBrackets)

		if err != nil {
			SendErrorResponse(res, requests, err)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(responses)
	})

	var server *http.Server

	// sync unsynced getblockinfo in the background
	go SyncGetBlockInfo()

	go func() {
		server = MustServeApp(app)
	}()

	<-signals

	_ = server.Shutdown(context.TODO())

	err := db.Close()

	if err != nil {
		panic(err)
	}

	fmt.Println("Bye!")
}
