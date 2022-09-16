package gorp

import (
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Delete is a query that deletes Entries from the DB.
type Delete[K Key, E Entry[K]] struct{ query.Query }

// NewDelete opens a new Delete query.
func NewDelete[K Key, E Entry[K]]() Delete[K, E] { return Delete[K, E]{Query: query.New()} }

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (d Delete[K, E]) Where(filter func(*E) bool) Delete[K, E] {
	addFilter[K, E](d.Query, filter)
	return d
}

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query.
// If called in conjunction with Where, the WhereKeys filter will be applied first.
// Subsequent calls to WhereKeys will append the keys to the existing filter.
func (d Delete[K, E]) WhereKeys(keys ...K) Delete[K, E] {
	setWhereKeys[K](d.Query, keys...)
	return d
}

// Exec executes the Query against the provided Txn. If any entries matching WhereKeys
// do not exist in the database, Delete will assume that the keys do not exist and
// do nothing.
func (d Delete[K, E]) Exec(txn Txn) error {
	return (&del[K, E]{Txn: txn}).exec(d)
}

type del[K Key, E Entry[K]] struct{ Txn }

func (d *del[K, E]) exec(q query.Query) error {
	opts := d.Txn.options()
	var entries []E
	err := (Retrieve[K, E]{Query: q}).Entries(&entries).Exec(d)
	if err != nil && err != query.NotFound {
		return err
	}
	prefix := typePrefix[K, E](opts)
	var keys whereKeys[K]
	for _, entry := range entries {
		keys = append(keys, entry.GorpKey())
	}
	byteKeys, err := keys.bytes(opts.encoder)
	if err != nil {
		return err
	}
	for _, key := range byteKeys {
		if err := d.Delete(append(prefix, key...)); err != nil && err != kv.NotFound {
			return err
		}
	}
	return nil
}
