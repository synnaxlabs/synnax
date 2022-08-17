package gorp

import (
	"github.com/arya-analytics/x/kv"
)

// KVIterator provides a simple wrapper around a kv.Write that decodes a byte-value
// before returning it to the caller. It provides no abstracted utilities for the
// iteration itself, and is focused only on maintaining a nearly identical interface to
// the underlying iterator. To create a new KVIterator, call WrapKVIter.
type KVIterator[E any] struct {
	kv.Iterator
	error error
	*options
}

// WrapKVIter wraps the provided iterator. All valid calls to iter.Value are
// decoded into the entry type E.
func WrapKVIter[E any](iter kv.Iterator, opts ...Option) *KVIterator[E] {
	return &KVIterator[E]{Iterator: iter, options: newOptions(opts...)}
}

// Value returns the decoded value from the iterator. Iter.Alive must be true
// for calls to return a valid value.
func (k *KVIterator[E]) Value() (entry E) {
	if err := k.decoder.Decode(k.Iterator.Value(), &entry); err != nil {
		k.error = err
	}
	return entry
}

func (k *KVIterator[E]) Error() error {
	if k.error != nil {
		return k.error
	}
	return k.Iterator.Error()
}
