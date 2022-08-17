package kv

type IteratorOptions struct {
	LowerBound []byte
	UpperBound []byte
}

// Iterator iterates over key-value pairs in order. It is not necessary to exhaust the
// Iterator, but it is necessary to close it after use.
type Iterator interface {
	// First moves the iterator to the first key-value pair. Returns true if the Iterator
	// contains at least one key-value pair.
	First() bool
	// Last moves the iterator to the last key-value pair. Returns true if the Iterator
	// contains at least one key-value pair.
	Last() bool
	// Next advances to the next key-value pair. Returns true if the Iterator is pointing
	// to a valid key-value pair (i.e. an exhausted Iterator will return false).
	Next() bool
	// Prev returns the previous key-value pair. Returns true if the Iterator is pointing
	// to a valid key-value pair (i.e. a reverse-exhausted Iterator will return false).
	Prev() bool
	// Valid returns true if the iterator is currently positioned at a valid
	// key-value pair.
	Valid() bool
	// Key returns the key of the current key-value pair. Returns true if the iterator
	// is currently positioned at a valid key-value pair.
	Key() []byte
	// Value returns the value of the current key-value pair, or nil if the Iterator
	// is not pointing at a valid key. The caller must not modify the contents of the
	// returned slice, as it may change on subsequent movements.
	Value() []byte
	// SeekLT moves the iterator to the last key-value pair whose key is less than
	// the given key. Returns true if the Iterator is pointing at a valid entry and
	// false otherwise.
	SeekLT(key []byte) bool
	// SeekGE moves the Iterator to the first key-value pair whose key is greater than
	// or equal to the given key. Returns true if such a pair is found and false if
	// otherwise.
	SeekGE(key []byte) bool
	// Error returns any accumulated error.
	Error() error
	// Close closes the Iterator and returns any accumulated error.
	Close() error
}

// PrefixIter returns IteratorOptions, that when passed to db.NewIterator, will
// return an Iterator that only iterates over keys with the given prefix.
func PrefixIter(prefix []byte) IteratorOptions {
	upper := func(b []byte) []byte {
		end := make([]byte, len(b))
		copy(end, b)
		for i := len(end) - 1; i >= 0; i-- {
			end[i] = end[i] + 1
			if end[i] != 0 {
				return end[:i+1]
			}
		}
		return nil
	}
	return IteratorOptions{LowerBound: prefix, UpperBound: upper(prefix)}
}

// RangeIter returns IteratorOptions, that when passed to db.NewIterator, will
// iterator through the range of keys between start and end.
func RangeIter(start, end []byte) IteratorOptions {
	return IteratorOptions{LowerBound: start, UpperBound: end}
}
