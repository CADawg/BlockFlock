package main

import (
	"context"
	"fmt"
	"github.com/CADawg/BlockFlock/internal/cache"
	"github.com/CADawg/BlockFlock/internal/config"
	"github.com/CADawg/BlockFlock/internal/hive_engine"
	"github.com/CADawg/BlockFlock/internal/jsonrpc"
	"github.com/CADawg/BlockFlock/internal/normalise"
	"github.com/CADawg/BlockFlock/internal/upstream"
	"github.com/dgraph-io/badger"
	"github.com/goccy/go-json"
	_ "github.com/joho/godotenv/autoload"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var db *badger.DB
var configuration *config.Config
var err error
var upstreamProvider *upstream.Upstream
var latestSafeBlock int64

func main() {
	configuration, err = config.LoadConfig("./config.json")

	if err != nil {
		panic(err)
	}

	db, err = badger.Open(badger.DefaultOptions("./data"))

	if err != nil {
		panic(err)
	}

	// get args
	args := os.Args[1:]

	if len(args) > 0 {
		if args[0] == "auto" {
			go CacheOverTime()
		} else {
			go GetLatestSafeBlock()
		}
	}

	// Create a new cache
	cacheProvider := cache.NewBadgerCache(db)

	// Create a new upstream
	upstreamProvider = upstream.NewUpstream(cacheProvider, configuration.Node)

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	app := http.NewServeMux()

	app.HandleFunc("GET /", func(res http.ResponseWriter, req *http.Request) {
		resp, err := jsonrpc.JsonGet[hive_engine.NodeInfo](configuration.Node)

		if err != nil {
			err = json.NewEncoder(res).Encode(hive_engine.ErrorResponse{
				Error:   err.Error(),
				Success: false,
			})

			if err != nil {
				panic(err)
			}
		}

		resp.DisabledMethods.Message = strings.Replace(resp.DisabledMethods.Message, "dw-he for witness", "dw-he for hive_engine witness and cadawg for hive witness", 1) + ". Source code and licence available at https://github.com/CADawg/BlockFlock."
		resp.Success = true
		resp.Domain = "https://engine.rishipanthee.com/"

		res.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(res).Encode(resp)

		if err != nil {
			SendError(res, 0, err, -69)
		}
	})

	app.HandleFunc("OPTIONS /", func(res http.ResponseWriter, req *http.Request) {
		SendCorsHeaders(res)
		res.WriteHeader(http.StatusOK)
	})

	app.HandleFunc("POST /blockchain", func(res http.ResponseWriter, req *http.Request) {
		HandleRequests(res, req, "blockchain")
	})

	app.HandleFunc("POST /contracts", func(res http.ResponseWriter, req *http.Request) {
		HandleRequests(res, req, "contracts")
	})

	app.HandleFunc("POST /", func(res http.ResponseWriter, req *http.Request) {
		HandleRequests(res, req, "")
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           app,
		ReadHeaderTimeout: 2 * time.Second, // Avoid Slow Loris attacks
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	<-signals

	if err := db.Close(); err != nil {
		panic(err)
	}

	if err := server.Shutdown(context.TODO()); err != nil {
		panic(err)
	}

	fmt.Println("Bye!")
}

func HandleRequests(res http.ResponseWriter, req *http.Request, endpoint string) {
	SendCorsHeaders(res)
	res.Header().Set("Content-Type", "application/json")

	reqs, err := normalise.Normalise(req.Body, endpoint)

	if err != nil {
		SendError(res, 0, err, -69)
		return
	}

	responses, err := upstreamProvider.HandleRequests(reqs, latestSafeBlock)

	if err != nil {
		SendError(res, 0, err, -69)
		return
	}

	if len(responses) == 1 && responses[0].Single {
		err = json.NewEncoder(res).Encode(responses[0])

		if err != nil {
			SendError(res, 0, err, -69)
		}

		return
	}

	err = json.NewEncoder(res).Encode(responses)

	if err != nil {
		SendError(res, 0, err, -69)
	}
}

func SendError(res http.ResponseWriter, reqId int, err error, code int) {
	// We're sending an error, so there's no point capturing the error here as it means we'd have no way to
	// display it to the client anyway (as this is the way we display errors to the client)
	_ = json.NewEncoder(res).Encode(jsonrpc.ErrorResponse{
		Error: jsonrpc.Error{
			Code:    code,
			Message: err.Error(),
		},
		ID:      reqId,
		Version: "2.0",
	})
}

func SendCorsHeaders(res http.ResponseWriter) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	res.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("Access-Control-Max-Age", "86400")
}

func GetLatestSafeBlock() {
	for {
		time.Sleep(time.Second * 3)

		resp, err := jsonrpc.JsonGet[hive_engine.NodeInfo](configuration.Node)

		if err == nil {
			latestSafeBlock = resp.LastVerifiedBlockNumber
		}
	}
}

func CacheOverTime() {
	// get min block from db
	var minBlock int64 = 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if strings.HasPrefix(string(k), "b_") {
				var blockNumber, err = strconv.ParseInt(strings.TrimPrefix(string(k), "b_"), 10, 64)

				if err != nil {
					return err
				}

				if blockNumber > minBlock && minBlock == blockNumber-1 {
					minBlock = blockNumber
				}
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(time.Millisecond * 500)

		resp, err := jsonrpc.JsonGet[hive_engine.NodeInfo](configuration.Node)

		if err != nil {
			panic(err)
		}

		latestSafeBlock = resp.LastVerifiedBlockNumber

		var i int64

		var latestSafe100 = (latestSafeBlock/100)*100 - 100

		for i = minBlock + 1; i <= latestSafe100; i += 100 {
			if !(os.Getenv("NO_RATE_LIMIT") == "true") {
				time.Sleep(time.Millisecond * 100)
			}

			var reqs []jsonrpc.Request

			for j := i; j < i+100; j++ {
				reqs = append(reqs, jsonrpc.Request{
					Version: "2.0",
					Method:  "blockchain.getBlockInfo",
					Params:  json.RawMessage(fmt.Sprintf(`{"blockNumber":%d}`, j)),
					Single:  false,
				})
			}

			_, err := upstreamProvider.HandleRequests(reqs, latestSafeBlock)

			if err != nil {
				i -= 100
				continue
			}
		}

	}
}
