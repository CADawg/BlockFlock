package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cfoxon/hiveenginego"
)

func GetSyncedToBlockNumber() int {
	tx := db.NewTransaction(true)
	defer tx.Discard()

	item, err := tx.Get([]byte("synced_to_block"))

	if err != nil {
		return 0
	}

	var syncedToBlock int

	err = item.Value(func(val []byte) error {
		syncedToBlock, err = strconv.Atoi(string(val))

		return err
	})

	if err != nil {
		return 0
	}

	return syncedToBlock
}

func SetSyncedToBlockNumber(blockNumber int) error {
	tx := db.NewTransaction(true)
	defer tx.Discard()

	err := tx.Set([]byte("synced_to_block"), []byte(strconv.Itoa(blockNumber)))

	if err != nil {
		return err
	}

	err = tx.Commit()

	if err != nil {
		return err
	}

	return nil
}

func SyncGetBlockInfo() {
	rpc := hiveenginego.NewHiveEngineRpc(config.Node)

	lbi, err := rpc.GetLatestBlockInfo()

	if err != nil {
		panic(err) // if we err here likely the node is down
	}

	syncedTo := GetSyncedToBlockNumber()

	for {
		if syncedTo >= lbi.BlockNumber {
			time.Sleep(3 * time.Second)

			lbi, _ = rpc.GetLatestBlockInfo()

			syncedTo = GetSyncedToBlockNumber()

			continue
		}

		// don't hurt poor little node
		time.Sleep(100 * time.Millisecond)

		var id int = 0

		jrcReq := JSONRPCRequest{
			Version: "2.0",
			ID:      &id,
			Method:  "getBlockInfo",
			Params:  []byte(`{"blockNumber":` + strconv.Itoa(syncedTo+1) + `}`),
		}

		// get block from node
		blockResponse, err := FetchFromUpstream(jrcReq, "blockchain")

		if err != nil {
			continue
		}

		// write block to cache
		err = WriteCacheForBlock(jrcReq, blockResponse)

		if err != nil {
			continue
		}

		fmt.Println("Synced to block", syncedTo+1)

		// update synced to block
		_ = SetSyncedToBlockNumber(syncedTo + 1)

		syncedTo += 1
	}
}
