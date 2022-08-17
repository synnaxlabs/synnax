package kv

import (
	"github.com/arya-analytics/x/binary"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"io"
)

// PersistedCounter implements a simple counter that writes its value to a DB store. To create a new PersistedCounter,
// call NewPersistedCounter.
type PersistedCounter struct {
	kve   DB
	key   []byte
	value int64
}

// NewPersistedCounter opens or creates a persisted counter at the given key. If the counter value is found in storage,
// sets it in internal state. If the counter value is not found in storage, sets the value to 0.
func NewPersistedCounter(kv DB, key []byte) (*PersistedCounter, error) {
	c := &PersistedCounter{kve: kv, key: key}
	err := Load(kv, c.key, c)
	if errors.Is(err, pebble.ErrNotFound) {
		err = nil
		c.value = 0
	}
	return c, err
}

// Load implements the Loader interface.
func (c *PersistedCounter) Load(r io.Reader) error { return binary.Read(r, &c.value) }

// Flush implements the Flusher interface.
func (c *PersistedCounter) Flush(w io.Writer) error { return binary.Write(w, c.value) }

// Increment increments the counter by the sum of hte given values. Returns the new value as well as any
// errors encountered while flushing the counter to storage.
func (c *PersistedCounter) Increment(values ...int64) (int64, error) {
	if len(values) == 0 {
		c.value++
	}
	for _, v := range values {
		c.value += v
	}
	return c.value, c.flushShelf()
}

// Decrement decrements the counter by the sum of the given values. Returns the new value as well as any
// errors encountered while flushing the counter to storage.
func (c *PersistedCounter) Decrement(values ...int64) (int64, error) {
	if len(values) == 0 {
		c.value--
	}
	for _, v := range values {
		c.value -= v
	}
	return c.value, c.flushShelf()
}

// Value returns the current value of the counter.
func (c *PersistedCounter) Value() int64 { return c.value }

func (c *PersistedCounter) flushShelf() error { return Flush(c.kve, c.key, c, pebble.NoSync) }
