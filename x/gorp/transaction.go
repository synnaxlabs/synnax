package gorp

import (
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/query"
)

// Txn is a transaction that can be atomically executed against a DB. Retrieve queries
// will read from both the DB and any operations previously executed against the Txn.
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
