package kv

import (
	"encoding/binary"
	atomicx "github.com/arya-analytics/x/atomic"
	"github.com/cockroachdb/errors"
)

// PersistedCounter implements a simple in64 counter that writes its value to a
// key-value store. PersistedCounter is safe for concurrent use. To create a new
// PersistedCounter, call NewPersistedCounter.
type PersistedCounter struct {
	atomicx.Int64Counter
	kve    DB
	key    []byte
	buffer []byte
}

// NewPersistedCounter opens or creates a persisted counter at the given key. If
// the counter value is found in storage, sets its internal state. If the counter
// value is not found in storage, sets the value to 0.
func NewPersistedCounter(kv DB, key []byte) (*PersistedCounter, error) {
	c := &PersistedCounter{kve: kv, key: key, buffer: make([]byte, 8)}
	b, err := kv.Get(key)
	if err == nil {
		c.Int64Counter.Add(int64(binary.LittleEndian.Uint64(b)))
	} else if errors.Is(err, NotFound) {
		err = nil
	}
	return c, err
}

// Add increments the counter by the sum of the given values. If no values are
// provided, increments the counter by 1.
// as well as any errors encountered while flushing the counter to storage.
func (c *PersistedCounter) Add(delta ...int64) (int64, error) {
	next := c.Int64Counter.Add(delta...)
	binary.LittleEndian.PutUint64(c.buffer, uint64(next))
	return next, c.kve.Set(c.key, c.buffer)
}
