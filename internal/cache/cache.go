package cache

import (
	"errors"
	"github.com/dgraph-io/badger/v4"
)

var ErrorKeyNotFound = errors.New("key not found")

type Cache interface {
	Set(typ rune, key string, value []byte) error
	Get(typ rune, key string) ([]byte, error)
	Has(typ rune, key string) (bool, error)
}

type BadgerCache struct {
	Db *badger.DB
}

func NewBadgerCache(db *badger.DB) *BadgerCache {
	return &BadgerCache{
		Db: db,
	}
}

func (c *BadgerCache) Set(typ rune, key string, value []byte) error {
	return c.Db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(string(typ)+"_"+key), value)
	})
}

func (c *BadgerCache) Get(typ rune, key string) ([]byte, error) {
	var value []byte
	err := c.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(string(typ) + "_" + key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, ErrorKeyNotFound
	}
	return value, err
}

func (c *BadgerCache) Has(typ rune, key string) (bool, error) {
	err := c.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(string(typ) + "_" + key))
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	return true, err
}
