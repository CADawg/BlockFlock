package main

import (
	"bytes"
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type GetBlockInfoParams struct {
	BlockNumber int64 `json:"blockNumber"`
}

var db *badger.DB
var config struct {
	Domain             string `json:"domain"`
	Node               string `json:"node"`
	BlockchainEndpoint string `json:"blockchainEndpoint"`
	ContractsEndpoint  string `json:"contractsEndpoint"`
}

func main() {
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

	db, err = badger.Open(badger.DefaultOptions("./data"))

	if err != nil {
		panic(err)
	}

	defer func(db *badger.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)

	app := gorouter.New()

	app.GET("/", func(res http.ResponseWriter, req *http.Request) {
		resp, err := JsonGet[EngineInfo](config.Node)

		if err != nil {
			err = json.NewEncoder(res).Encode(EngineInfoError{
				Error:   err.Error(),
				Success: false,
			})

			if err != nil {
				panic(err)
			}
		}

		resp.DisabledMethods.Message = strings.Replace(resp.DisabledMethods.Message, "h-e", "h-e and cadengine", 1) + ". Source code and licence available at https://github.com/CADawg/BlockFlock."
		resp.Success = true
		resp.Domain = "https://engine.rishipanthee.com/"

		res.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(res).Encode(resp)
	})

	app.POST("/blockchain", func(res http.ResponseWriter, req *http.Request) {
		var request JSONRPCRequest

		// read all bytes into var
		body, err := io.ReadAll(req.Body)

		if err != nil {
			json.NewEncoder(res).Encode(JSONRPCErrorResponse{
				Version: "2.0",
				ID:      request.ID,
				Error:   JSONRPCError{Code: -69, Message: err.Error()},
			})
		}

		err = json.Unmarshal(body, &request)

		if err != nil {
			// need to try reading it as an array
			var requests []JSONRPCRequest

			err = json.Unmarshal(body, &requests)

			if err == nil {
				// pass it straight on
				resp, err := FetchFromUpstreamMulti(requests)

				if err != nil {
					err = json.NewEncoder(res).Encode(JSONRPCErrorResponse{
						Version: "2.0",
						ID:      request.ID,
						Error:   JSONRPCError{Code: -69, Message: err.Error()},
					})

					if err != nil {
						panic(err)
					}
				}

				err = json.NewEncoder(res).Encode(resp)

				if err != nil {
					panic(err)
				}

				return
			}
		}

		if request.Method == "getBlockInfo" {
			// check our cache
			value, err := CheckCacheForBlock(request)

			if err == nil {
				// return cached value
				_, err = io.WriteString(res, string(value))

				if err == nil {
					return
				}
			}

			// fetch from upstream
			response, err := FetchFromUpstream(request)

			if err != nil {
				err = json.NewEncoder(res).Encode(JSONRPCErrorResponse{
					Version: "2.0",
					ID:      request.ID,
					Error:   JSONRPCError{Code: -69, Message: err.Error()},
				})

				if err != nil {
					panic(err)
				}
			}

			// marshal response
			responseBytes, err := json.Marshal(response)

			if err != nil {
				err = json.NewEncoder(res).Encode(JSONRPCErrorResponse{
					Version: "2.0",
					ID:      request.ID,
					Error:   JSONRPCError{Code: -69, Message: err.Error()},
				})

				if err != nil {
					panic(err)
				}
			}

			// write response to cache
			err = WriteCacheForBlock(request, responseBytes)

			res.Header().Set("Content-Type", "application/json")
			_, err = res.Write(responseBytes)

			if err != nil {
				fmt.Println("Error writing response to client: ", err)
			}
		} else {
			// forward onto full node
		}
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           app,
		ReadHeaderTimeout: 2 * time.Second, // Avoid Slowloris attacks
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	<-signals

	if err := server.Shutdown(nil); err != nil {
		panic(err)
	}

	fmt.Println("Bye!")
}

// HandleOneRequest handles everything a request needs to do
// this includes checking the cache, populating the cache and forwarding to upstream
func HandleOneRequest(request JSONRPCRequest, endpoint string) ([]byte, error) {
	if request.Method == "getBlockInfo" {
		// check our cache
		value, err := CheckCacheForBlock(request)

		if err == nil {
			// return cached value
			return value, nil
		}

		// fetch from upstream
		response, err := FetchFromUpstream(request, endpoint)

		if err != nil {
			return nil, err
		}

		// marshal response
		responseBytes, err := json.Marshal(response)

		if err != nil {
			return nil, err
		}

		// write response to cache
		err = WriteCacheForBlock(request, responseBytes)

		return responseBytes, nil
	} else {
		// forward onto full node
		return FetchFromUpstream(request, endpoint)
	}
}

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
}

type JSONRPCErrorResponse struct {
	Version string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Error   JSONRPCError `json:"error"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (j *JSONRPCResponse) IsNullResult() bool {
	return bytes.Equal(j.Result, []byte("null"))
}

func MakeErrorResponse(request JSONRPCRequest, message string, code int) ([]byte, error) {
	response := JSONRPCResponse{
		Version: request.Version,
		ID:      request.ID,
		Result:  nil,
	}

	return json.Marshal(response)
}

func FetchFromUpstream(request JSONRPCRequest, endpoint string) ([]byte, error) {

	// serialize request
	data, err := json.Marshal(request)

	if err != nil {
		return nil, err
	}

	var endpointUrl string
	if endpoint == "blockchain" {
		endpointUrl = config.BlockchainEndpoint
	} else {
		endpointUrl = config.ContractsEndpoint
	}

	path, err := url.JoinPath(config.Node, endpointUrl)

	if err != nil {
		return nil, err
	}

	// send request to upstream
	resp, err := http.Post(path, "application/json", bytes.NewBuffer(data))

	if err != nil {
		return nil, err
	}

	// read response
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	// close response body
	err = resp.Body.Close()

	if err != nil {
		return nil, err
	}

	return body, nil
}

func WriteCacheForBlock(request JSONRPCRequest, value []byte) error {
	tx := db.NewTransaction(true)
	defer tx.Discard()

	// decode block number
	var params GetBlockInfoParams

	err := json.Unmarshal(request.Params, &params)

	if err != nil {
		return err
	}

	// write to cache
	err = tx.Set([]byte(fmt.Sprintf("b_%d", params.BlockNumber)), value)

	if err != nil {
		return err
	}

	// commit transaction
	err = tx.Commit()

	if err != nil {
		return err
	}

	return nil
}

func CheckCacheForBlock(request JSONRPCRequest) ([]byte, error) {
	tx := db.NewTransaction(false)
	defer tx.Discard()

	// decode block number
	var params GetBlockInfoParams

	err := json.Unmarshal(request.Params, &params)

	if err != nil {
		return nil, err
	}

	// check if block exists in cache
	item, err := tx.Get([]byte(fmt.Sprintf("b_%d", params.BlockNumber)))

	if err != nil {
		return nil, err
	}

	// get value from cache
	value, err := item.ValueCopy(nil)

	if err != nil {
		return nil, err
	}

	return value, nil
}

// JsonGet gets json from a url
func JsonGet[T any](url string) (*T, error) {
	var decodeInto = new(T)

	resp, err := http.Get(url)

	if err != nil {
		return new(T), err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return new(T), err
	}

	err = json.Unmarshal(data, decodeInto)

	if err != nil {
		return new(T), err
	}

	return decodeInto, nil
}
