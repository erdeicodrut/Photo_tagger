package main

import "github.com/dgraph-io/badger/v4"

type BadgerCache struct {
	db *badger.DB
}

func NewBadgerCache(path string) (*BadgerCache, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerCache{db: db}, nil
}

func (c *BadgerCache) Get(key string) (string, bool) {
	var value string
	var found bool

	c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		item.Value(func(val []byte) error {
			value = string(val)
			found = true
			return nil
		})
		return nil
	})

	return value, found
}

func (c *BadgerCache) Set(key, value string) error {
	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(value))
	})
}

func (c *BadgerCache) Close() error {
	return c.db.Close()
}
