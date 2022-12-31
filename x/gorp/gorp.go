// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"github.com/synnaxlabs/x/kv"
)

// Wrap wraps the provided key-value database in a DB.
func Wrap(kv kv.DB, opts ...Option) *DB {
	return &DB{DB: kv, opts: newOptions(opts...)}
}

// DB is a wrapper around a kv.DB that queries can be executed against. DB implements
// the Txn interface, so it can be provided to Query.Write.
type DB struct {
	kv.DB
	opts options
}

var _ Txn = (*DB)(nil)

func (db *DB) options() options { return db.opts }

// BeginTxn begins a new Txn against the DB.
func (db *DB) BeginTxn() Txn { return txn{Batch: db.NewBatch(), db: db} }

// Commit does nothing, and is here to implement the Txn interface. If DB is used
// as a Txn, all queries will be executed directly against the DB.
func (db *DB) Commit(opts ...interface{}) error { return nil }
