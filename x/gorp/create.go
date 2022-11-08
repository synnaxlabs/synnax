package gorp

import (
	"github.com/synnaxlabs/x/query"
)

// Create is a query that creates Entries in the DB.
type Create[K Key, E Entry[K]] struct{ query.Query }

// NewCreate opens a new Create query.
func NewCreate[K Key, E Entry[K]]() Create[K, E] { return Create[K, E]{query.New()} }

// Entries sets the Entries to write to the DB.
func (c Create[K, E]) Entries(entries *[]E) Create[K, E] { SetEntries[K, E](c, entries); return c }

// Entry sets the entry to write to the DB.
func (c Create[K, E]) Entry(entry *E) Create[K, E] { SetEntry[K, E](c, entry); return c }

// Exec executes the Query against the provided DB. It returns any errors encountered during execution.
func (c Create[K, E]) Exec(txn Txn) error { return (&createExecutor[K, E]{Txn: txn}).exec(c) }

// |||||| EXECUTOR ||||||

type createExecutor[K Key, E Entry[K]] struct{ Txn }

func (c *createExecutor[K, E]) exec(q query.Query) error {
	var (
		opts    = c.options()
		entries = GetEntries[K, E](q)
	)
	w := &KVBatch[K, E]{Batch: c.Txn, opts: opts}
	for _, entry := range entries.All() {
		if err := w.Write(entry); err != nil {
			return err
		}
	}
	return nil
}
