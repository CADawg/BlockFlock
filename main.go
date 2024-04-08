package main

import (
	"fmt"
	"github.com/CADawg/BlockFlock/internal/cache"
	"github.com/CADawg/BlockFlock/internal/config"
	"github.com/CADawg/BlockFlock/internal/hive_engine"
	"github.com/CADawg/BlockFlock/internal/jsonrpc"
	"github.com/CADawg/BlockFlock/internal/normalise"
	"github.com/CADawg/BlockFlock/internal/upstream"
	"github.com/dgraph-io/badger"
	"github.com/goccy/go-json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var db *badger.DB
var configuration *config.Config
var err error
var upstreamProvider *upstream.Upstream

func main() {
	configuration, err = config.LoadConfig("./config.json")

	if err != nil {
		panic(err)
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

	// Create a new cache
	cacheProvider := cache.NewBadgerCache(db)

	// Create a new upstream
	upstreamProvider = upstream.NewUpstream(cacheProvider, configuration.Node)

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)

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

	if err := server.Shutdown(nil); err != nil {
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

	responses, err := upstreamProvider.HandleRequests(reqs)

	if err != nil {
		SendError(res, 0, err, -69)
		return
	}

	if len(responses) == 1 && responses[0].Single {
		err = json.NewEncoder(res).Encode(responses[0])
		return
	}

	err = json.NewEncoder(res).Encode(responses)
}

func SendError(res http.ResponseWriter, reqId int, err error, code int) {
	err = json.NewEncoder(res).Encode(jsonrpc.ErrorResponse{
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
