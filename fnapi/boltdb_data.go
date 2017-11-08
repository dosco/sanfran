package main

import (
	"bytes"

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
	v, err = fn.Marshal()
	if err != nil {
		return err
	} else if err := b.Put(k, v); err != nil {
		return err
	}

	ck := bytes.Join([][]byte{FN_BKT, k}, nil)
	if _, err := d.cache.Get(ck); err == nil {
		d.cache.Set(ck, v, d.expiry)
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

func (d *datastore) ListFn() ([]data.Function, error) {
	list, err := d.list(FN_BKT)
	if err != nil {
		return nil, err
	}

	var fns []data.Function
	for i := range list {
		var fn data.Function
		if err := fn.Unmarshal(list[i]); err != nil {
			return nil, err
		}
		fns = append(fns, fn)
	}

	return fns, nil
}
