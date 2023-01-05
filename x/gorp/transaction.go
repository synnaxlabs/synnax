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
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Txn represents a generalized key-value transaction that executes atomically against
// an underlying database. DB implements the Txn interface, which will execute
// queries directly against the DB. To open an isolated transaction against the DB, use
// DB.BeginTxn.
type Txn interface {
	kv.Batch
	options() options
}

type txn struct {
	// db is the underlying gorp DB the txn is operating on.
	db *DB
	kv.Batch
}

func (t txn) options() options { return t.db.opts }

const txnOpt query.OptionKey = "txn"

// SetTxn sets the provided Txn on the query.
func SetTxn(q query.Query, txn Txn) {
	q.Set(txnOpt, txn)
}

// GetTxn returns the Txn bound to the query, if it exists. Otherwise, returns def
// as the Txn.
func GetTxn(q query.Query, def Txn) Txn {
	if tx, ok := q.Get(txnOpt); ok {
		return tx.(Txn)
	}
	return def
}
