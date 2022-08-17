package gorp

import (
	"github.com/arya-analytics/x/query"
	"reflect"
)

type Key any

// Entry is a go type that can be queried against a DB.
// All go types must implement the Entry interface so that they can be
// stored. Entry must be serializable by the Encodings and Decoder provided to the DB.
type Entry[K Key] interface {
	// GorpKey returns a unique key for the entry. gorp.DB will not raise
	// an error if the key is a duplicate. Key must be serializable by Encodings and Decoder.
	GorpKey() K
	// SetOptions returns a slice of options passed to kv.db.Set.
	SetOptions() []interface{}
}

const entriesOptKey query.OptionKey = "entries"

// Entries is a query option used to bind entities from a retrieve query or
// write values to a create query.
type Entries[K Key, E Entry[K]] struct {
	// multiple is a boolean flag indicating whether the query expects one or multiple
	// entities.
	multiple bool
	// entry is used when the client expects/passes a single entry.
	entry *E
	// entries are used when the client expects/passes a slice of entries.
	entries *[]E
}

// Add adds the provided entry to the query. If the client expects a single result,
// sets the entry to the provided value. If the client expects a slice of entries,
// appends the provided value to the slice.
func (e *Entries[K, E]) Add(entry E) {
	if e.multiple {
		*e.entries = append(*e.entries, entry)
	} else if e.entry != nil {
		*e.entry = entry
	}
}

func (e *Entries[K, E]) Set(i int, entry E) {
	if !e.multiple {
		if i > 1 {
			panic("gorp: cannot set multiple entries on a single entry query")
		}
		*e.entry = entry
	} else {
		(*e.entries)[i] = entry
	}
}

// All returns a slice of all entries currently bound to the query.
func (e Entries[K, E]) All() []E {
	if e.multiple {
		return *e.entries
	}
	return []E{*e.entry}
}

// SetEntry sets the entry that the query will fill results into or write results to.
// 	Calls to SetEntry will override All previous calls to SetEntry or SetEntries.
func SetEntry[K Key, E Entry[K]](q query.Query, entry *E) {
	q.Set(entriesOptKey, &Entries[K, E]{entry: entry, multiple: false})
}

// SetEntries sets the entries that the query will fill results into or write results to.
// Calls to SetEntries will override All previous calls to SetEntry or SetEntries.
func SetEntries[K Key, E Entry[K]](q query.Query, e *[]E) {
	q.Set(entriesOptKey, &Entries[K, E]{entries: e, multiple: true})
}

// GetEntries returns the entries that the query will fill results into or write
// results from.
func GetEntries[K Key, E Entry[K]](q query.Query) *Entries[K, E] {
	re, ok := q.Get(entriesOptKey)
	if !ok {
		SetEntries[K, E](q, new([]E))
		return GetEntries[K, E](q)
	}
	return re.(*Entries[K, E])
}

func typePrefix[K Key, E Entry[K]](opts *options) []byte {
	if opts.noTypePrefix {
		return []byte{}
	}
	mName := reflect.TypeOf(*new(E)).Name()
	b, err := opts.encoder.Encode(mName)
	if err != nil {
		panic(err)
	}
	return b
}
