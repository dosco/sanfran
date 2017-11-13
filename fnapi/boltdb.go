package main

import (
	"bytes"
	"errors"
	"os"
	"runtime/debug"

	"github.com/boltdb/bolt"
	"github.com/coocood/freecache"
	"github.com/golang/glog"
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
	db     *bolt.DB
	cache  *freecache.Cache
	expiry int
}

func NewDatastore(cacheSize, cacheExpiry int) (*datastore, error) {
	var dbFile string

	if s, err := os.Stat("/data"); os.IsExist(err) && s.IsDir() {
		dbFile = "/data/sanfran.db"
	} else {
		dbFile = "sanfran.db"
	}

	// Open new BoltDB database
	db, err := bolt.Open(dbFile, 0600, nil)
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

	cache := freecache.NewCache(cacheSize)
	debug.SetGCPercent(20)

	return &datastore{db: db, cache: cache, expiry: cacheExpiry}, nil
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
	v, err := obj.Marshal()
	if err != nil {
		return err
	} else if err := b.Put(k, v); err != nil {
		return err
	}

	ck := bytes.Join([][]byte{bn, k}, nil)
	if _, err := d.cache.Get(ck); err == nil {
		d.cache.Set(ck, v, d.expiry)
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
	ck := bytes.Join([][]byte{bn, k}, nil)
	d.cache.Del(ck)

	return tx.Commit()
}

func (d *datastore) get(bn []byte, k []byte) ([]byte, error) {
	ck := bytes.Join([][]byte{bn, k}, nil)

	// Attempt to fetch from cache
	if v, err := d.cache.Get(ck); err != nil && err != freecache.ErrNotFound {
		glog.Errorln(err.Error())
	} else if v != nil && err == nil {
		return v, nil
	}

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
	d.cache.Set(ck, v, d.expiry)

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
