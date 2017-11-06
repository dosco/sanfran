package main

import (
	"errors"

	"github.com/boltdb/bolt"
)

var (
	ErrKeyExists    = errors.New("key exists")
	ErrKeyNotExists = errors.New("key doesn't exists")
)

type Marshaler interface {
	Marshal() (data []byte, err error)
}

type Unmarshaler interface {
	Unmarshal(data []byte) error
}

type datastore struct {
	db *bolt.DB
}

func NewDatastore() (*datastore, error) {
	// Open new BoltDB database
	db, err := bolt.Open("sanfran.db", 0600, nil)
	if err != nil {
		return nil, err
	}

	// Start writable transaction.
	tx, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Initialize top-level buckets.
	if _, err := tx.CreateBucketIfNotExists(FN_BKT); err != nil {
		return nil, err
	}

	// Save transaction to disk.
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &datastore{db: db}, nil
}

func (d *datastore) Close() error {
	return d.db.Close()
}

func (d *datastore) create(bn []byte, k []byte, obj Marshaler) error {
	// Start read-write transaction.
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	b := tx.Bucket(bn)

	// Check if key exists.
	if v := b.Get(k); v != nil {
		return ErrKeyExists
	}

	// Marshal and insert record.
	if v, err := obj.Marshal(); err != nil {
		return err
	} else if err := b.Put(k, v); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *datastore) update(bn []byte, k []byte, obj Marshaler) error {
	// Start read-write transaction.
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	b := tx.Bucket(bn)

	// Check if key exists.
	if v := b.Get(k); v == nil {
		return ErrKeyNotExists
	}

	// Marshal and insert record.
	if v, err := obj.Marshal(); err != nil {
		return err
	} else if err := b.Put(k, v); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *datastore) delete(bn []byte, k []byte) error {
	// Start read-write transaction.
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	b := tx.Bucket(bn)

	// Check if key exists.
	if v := b.Get(k); v == nil {
		return ErrKeyNotExists
	}

	// Delete record.
	if err := b.Delete(k); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *datastore) get(bn []byte, k []byte) ([]byte, error) {
	// Start read-write transaction.
	tx, err := d.db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	b := tx.Bucket(bn)

	// Fetch record.
	v := b.Get(k)
	if v == nil {
		return nil, ErrKeyNotExists
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return v, nil
}

func (d *datastore) list(bn []byte) ([][]byte, error) {
	// Start read-write transaction.
	tx, err := d.db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	b := tx.Bucket(bn)
	var l [][]byte

	// Map though all the records
	err = b.ForEach(func(k, v []byte) error {
		l = append(l, v)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return l, nil
}
