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
		prefix  = typePrefix[K, E](opts)
	)
	for _, entry := range entries.All() {
		data, err := opts.encoder.Encode(entry)
		if err != nil {
			return err
		}
		key, err := opts.encoder.Encode(entry.GorpKey())
		if err != nil {
			return err
		}
		// NOTE: We need to be careful with this operation in the future.
		// Because we aren't copying prefix, we're modifying the underlying slice.
		if err = c.Txn.Set(append(prefix, key...), data, entry.SetOptions()...); err != nil {
			return err
		}
	}
	return nil
}
