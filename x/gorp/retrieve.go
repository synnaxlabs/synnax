// Package gorp exposes a simple, type-safe ORM that wraps a key-value store.
package gorp

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct{ query.Query }

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{query.New()}
}

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (r Retrieve[K, E]) Where(filter func(*E) bool) Retrieve[K, E] {
	addFilter[K, E](r, filter)
	return r
}

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query. If called in
// conjunction with Where, the WhereKeys filter will be applied first. Subsequent calls
// to WhereKeys will append the keys to the existing filter.
func (r Retrieve[K, E]) WhereKeys(keys ...K) Retrieve[K, E] {
	setWhereKeys(r, keys...)
	return r
}

// Entries binds a slice that the Query will fill results into. Calls to Entry will
// override All previous calls to Entries or Entry.
func (r Retrieve[K, E]) Entries(entries *[]E) Retrieve[K, E] {
	SetEntries[K, E](r, entries)
	return r
}

// Entry binds the entry that the Query will fill results into. Calls to Entry will
// override All previous calls to Entries or Entry. If  multiple results are returned
// by the query, entry will be set to the last result.
func (r Retrieve[K, E]) Entry(entry *E) Retrieve[K, E] {
	SetEntry[K, E](r, entry)
	return r
}

// Exec executes the Query against the provided Txn. If the WhereKeys method is set on
// the query, Retrieve will return a query.NotFound  error if ANY of the keys do not
// exist in the database. If Where is set on the query, Retrieve will return a query.NotFound
// if NO keys pass the Where filter.
func (r Retrieve[K, E]) Exec(txn Txn) error {
	_, ok := getWhereKeys[K](r)
	if ok {
		return keysRetrieve[K, E](r, txn, txn.options())
	}
	return filterRetrieve[K, E](r, txn, txn.options())
}

// Exists checks whether records matching the query exist in the DB. If the WhereKeys method is
// set on the query, Exists will return true if ANY of the keys exist in the database. If
// Where is set on the query, Exists will return true if ANY keys pass the Where filter.
func (r Retrieve[K, E]) Exists(txn Txn) (bool, error) {
	return checkExists[K, E](r, txn, txn.options())
}

// |||||| FILTERS ||||||

const filtersKey query.OptionKey = "filters"

type filters[K Key, E Entry[K]] []func(*E) bool

func (f filters[K, E]) exec(entry *E) bool {
	if len(f) == 0 {
		return true
	}
	for _, filter := range f {
		if filter(entry) {
			return true
		}
	}
	return false
}

func addFilter[K Key, E Entry[K]](q query.Query, filter func(*E) bool) {
	var f filters[K, E]
	rf, ok := q.Get(filtersKey)
	if !ok {
		f = filters[K, E]{}
	} else {
		f = rf.(filters[K, E])
	}
	f = append(f, filter)
	q.Set(filtersKey, f)
}

func getFilters[K Key, E Entry[K]](q query.Query) filters[K, E] {
	rf, ok := q.Get(filtersKey)
	if !ok {
		return filters[K, E]{}
	}
	return rf.(filters[K, E])
}

// |||||| WHERE KEYS ||||||

const whereKeysKey query.OptionKey = "retrieveByKeys"

type whereKeys[K Key] []K

func (w whereKeys[K]) bytes(encoder binary.Encoder) ([][]byte, error) {
	byteWhereKeys := make([][]byte, len(w))
	for i, key := range w {
		var err error
		byteWhereKeys[i], err = encoder.Encode(key)
		if err != nil {
			return nil, err
		}
	}
	return byteWhereKeys, nil
}

func setWhereKeys[K Key](q query.Query, keys ...K) {
	var (
		keysToSet whereKeys[K]
		ok        bool
	)
	if keysToSet, ok = getWhereKeys[K](q); ok {
		keysToSet = append(keysToSet, keys...)
	} else {
		keysToSet = keys
	}
	q.Set(whereKeysKey, keysToSet)
}

func getWhereKeys[K Key](q query.Query) (whereKeys[K], bool) {
	keys, ok := q.Get(whereKeysKey)
	if !ok {
		return nil, false
	}
	return keys.(whereKeys[K]), true
}

func checkExists[K Key, E Entry[K]](q query.Query, reader kv.Reader, opts options) (bool, error) {
	if keys, ok := getWhereKeys[K](q); ok {
		entries := make([]E, 0, len(keys))
		SetEntries[K, E](q, &entries)
		if err := keysRetrieve[K, E](q, reader, opts); err != nil && !errors.Is(err, query.NotFound) {
			return false, err
		}
		return len(entries) == len(keys), nil
	}
	entries := make([]E, 0, 1)
	SetEntries[K, E](q, &entries)
	if err := filterRetrieve[K, E](q, reader, opts); err != nil && !errors.Is(err, query.NotFound) {
		return false, err
	}
	return len(entries) > 0, nil
}

func keysRetrieve[K Key, E Entry[K]](q query.Query, reader kv.Reader, opts options) error {
	var (
		entry   *E
		keys, _ = getWhereKeys[K](q)
		f       = getFilters[K, E](q)
		entries = GetEntries[K, E](q)
		prefix  = typePrefix[K, E](opts)
	)
	byteKeys, err := keys.bytes(opts.encoder)
	if err != nil {
		return err
	}
	for _, key := range byteKeys {
		prefixedKey := append(prefix, key...)
		b, _err := reader.Get(prefixedKey)
		if _err != nil {
			if _err == kv.NotFound {
				err = query.NotFound
			} else {
				err = _err
			}
			continue
		}
		if _err = opts.decoder.Decode(b, &entry); err != nil {
			return _err
		}
		if f.exec(entry) {
			entries.Add(*entry)
		}
	}
	return err
}

func filterRetrieve[K Key, E Entry[K]](q query.Query, reader kv.Reader, opts options) error {
	var (
		v       = new(E)
		f       = getFilters[K, E](q)
		entries = GetEntries[K, E](q)
		iter    = WrapKVIter[E](reader.NewIterator(kv.PrefixIter(typePrefix[K, E](opts))))
		found   = false
	)
	for iter.First(); iter.Valid(); iter.Next() {
		iter.BindValue(v)
		if f.exec(v) {
			found = true
			entries.Add(*v)
		}
	}
	if !entries.multiple && !found {
		return query.NotFound
	}
	return iter.Close()
}
