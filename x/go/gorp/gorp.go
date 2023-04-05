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
	"github.com/synnaxlabs/x/kv"
)

// Wrap wraps the provided key-value database in a DB.
func Wrap(kv kv.DB, opts ...Option) *DB {
	return &DB{DB: kv, options: newOptions(opts...)}
}

// DB is a wrapper around a kv.DB that queries can be executed against. DB implements
// the Writer interface, so it can be provided to Query.Write.
type DB struct {
	kv.DB
	options
}

// BeginWrite begins a new Writer against the DB.
func (db *DB) BeginWrite(ctx context.Context) WriteTxn {
	return writer{Writer: db.BeginWrite(ctx), opts: db.options}
}

// BeginRead begins a new ReadTxn against the DB.
func (db *DB) BeginRead(ctx context.Context) ReadTxn {
	return reader{Reader: db.BeginRead(ctx), opts: db.options}
}

func (db *DB) WithWriteTxn(ctx context.Context, f func(WriteTxn) error) (err error) {
	txn := db.BeginWrite(ctx)
	defer func() {
		if err_ := txn.Close(); err_ != nil {
			err = err_
		}
	}()
	if err = f(txn); err == nil {
		err = txn.Commit()
	}
	return
}

type ReadTxn interface {
	kv.Reader
	options() options
}

type WriteTxn interface {
	kv.Writer
	options() options
}

type writer struct {
	opts options
	kv.Writer
}

func (w writer) options() options { return w.opts }

type reader struct {
	opts options
	kv.Reader
}

func (r reader) options() options { return r.opts }
