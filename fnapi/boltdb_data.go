package main

import (
	"github.com/dosco/sanfran/fnapi/data"
)

var (
	FN_BKT = []byte("functions")
)

func (d *datastore) CreateFn(fn *data.Function) error {
	return d.create(FN_BKT, []byte(fn.GetName()), fn)
}

func (d *datastore) UpdateFn(fn *data.Function) error {
	// Start read-write transaction.
	tx, err := d.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	b := tx.Bucket(FN_BKT)
	k := []byte(fn.GetName())

	v := b.Get(k)
	if v == nil {
		return ErrKeyNotExists
	}

	var oldFn data.Function
	if err := oldFn.Unmarshal(v); err != nil {
		return err
	}
	fn.Version = oldFn.Version + 1

	// Marshal and insert record.
	if v, err := fn.Marshal(); err != nil {
		return err
	} else if err := b.Put(k, v); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *datastore) GetFn(key string) (*data.Function, error) {
	b, err := d.get(FN_BKT, []byte(key))
	if err != nil {
		return nil, err
	}

	var fn data.Function
	if err := fn.Unmarshal(b); err != nil {
		return nil, err
	}

	return &fn, nil
}

func (d *datastore) DeleteFn(key string) error {
	return d.delete(FN_BKT, []byte(key))
}
