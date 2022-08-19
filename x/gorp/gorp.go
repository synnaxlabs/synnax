package gorp

import (
	"github.com/arya-analytics/x/kv"
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
