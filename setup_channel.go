package main

import (
	"os"
	"os/signal"
	"syscall"
)

func init() {
	signals = make(chan os.Signal, 10)

	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}
