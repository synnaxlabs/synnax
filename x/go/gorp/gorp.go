// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/binary"

	"github.com/synnaxlabs/x/kv"
)

// Wrap wraps the provided key-value database in a DB.
func Wrap(kv kv.DB, opts ...Option) *DB {
	return &DB{DB: kv, options: newOptions(opts...)}
}

// DB is a wrapper around a kv.DB that queries can be executed against. DB implements
// the Writer interface, so it can be provided to Params.set.
type DB struct {
	kv.DB
	options
}

var _ Tx = (*DB)(nil)

// OpenTx begins a new Tx against the DB.
func (db *DB) OpenTx() Tx { return tx{Tx: db.DB.OpenTx(), options: db.options} }

func (db *DB) WithTx(ctx context.Context, f func(tx Tx) error) (err error) {
	txn := db.OpenTx()
	defer func() {
		if err_ := txn.Close(); err_ != nil {
			err = err_
		}
	}()
	if err = f(txn); err == nil {
		err = txn.Commit(ctx)
	}
	return
}

func (db *DB) OverrideTx(override Tx) Tx { return OverrideTx(override, db) }

func OverrideTx(base Tx, override Tx) Tx {
	return lo.Ternary(override != nil, override, base)
}

type Tx interface {
	kv.Tx
	Options
}

type Options interface {
	binary.EncoderDecoder
	noPrefix() bool
}

type BaseReader interface {
	kv.Reader
	Options
}

type BaseWriter interface {
	kv.Writer
	Options
}

type BaseObservable interface {
	kv.Observable
	Options
}

type tx struct {
	kv.Tx
	options
}

var _ Tx = (*tx)(nil)
