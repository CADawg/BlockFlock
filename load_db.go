package main

import "github.com/dgraph-io/badger"

func init() {
	var err error

	db, err = badger.Open(badger.DefaultOptions("./data"))

	if err != nil {
		panic(err)
	}
}
