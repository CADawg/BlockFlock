package main

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/tidwall/sjson"
)

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

func ReadCacheForBlock(request JSONRPCRequest) ([]byte, error) {
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

	val, err := sjson.Set(string(value), "id", request.ID)

	if err != nil {
		return nil, err
	}

	// set new id (this is cheaper than unmarshalling and marshalling again) - around 10x faster + 1/2 the memory vs goccy/go-json
	/*
			=== RUN   BenchmarkUnmarshalMarshal
		BenchmarkUnmarshalMarshal
		BenchmarkUnmarshalMarshal-2
		    1130            962243 ns/op         1391857 B/op          7 allocs/op
		=== RUN   BenchmarkGjsonSjson
		BenchmarkGjsonSjson
		BenchmarkGjsonSjson-2               9601            120036 ns/op          557161 B/op          5 allocs/op
	*/
	val, err = sjson.Set(val, "id", request.ID)

	if err != nil {
		return nil, err
	}

	return []byte(val), nil
}
