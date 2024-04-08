package main

import (
	"net/http"
	"os"
	"time"

	"github.com/xujiajun/gorouter"
)

func MustServeApp(app *gorouter.Router) *http.Server {
	server := &http.Server{
		Addr:              ":8080",
		Handler:           app,
		ReadHeaderTimeout: 2 * time.Second, // Avoid Slowloris attacks
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			signals <- os.Interrupt
		}
	}()

	return server
}
