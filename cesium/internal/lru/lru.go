// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package lru holds values with sizes in bytes up to a budget, and drops the least
// recently used first.
package lru

// Cache maps keys to values with sizes in bytes. Past its budget, Evict drops the least
// recently used values. A Cache is not safe for concurrent use.
type Cache[K comparable, V any] struct {
	budget, size   int
	entries        map[K]*entry[K, V]
	newest, oldest *entry[K, V]
	// spare is the entry that the last Evict took out, which the next Put uses.
	spare *entry[K, V]
}

type entry[K comparable, V any] struct {
	key          K
	value        V
	size         int
	newer, older *entry[K, V]
}

// New returns an empty cache of budget bytes.
func New[K comparable, V any](budget int) *Cache[K, V] {
	return &Cache[K, V]{budget: budget, entries: make(map[K]*entry[K, V])}
}

// Get returns the value of k, and makes it the most recently used.
func (c *Cache[K, V]) Get(k K) (V, bool) {
	e, ok := c.entries[k]
	if !ok {
		var zero V
		return zero, false
	}
	c.unlink(e)
	c.push(e)
	return e.value, true
}

// Put adds v of size bytes under k as the most recently used value, and reports whether
// it did. A key that c holds keeps its value.
func (c *Cache[K, V]) Put(k K, v V, size int) bool {
	if _, ok := c.entries[k]; ok {
		return false
	}
	e := c.spare
	if e == nil {
		e = &entry[K, V]{}
	}
	c.spare = nil
	e.key, e.value, e.size = k, v, size
	c.entries[k] = e
	c.push(e)
	c.size += size
	return true
}

// Evict takes out and returns the least recently used value for which held is false,
// while c holds more than its budget. It never takes out the most recently used value.
// A nil held holds no value.
func (c *Cache[K, V]) Evict(held func(V) bool) (V, bool) {
	var zero V
	if c.size <= c.budget {
		return zero, false
	}
	for e := c.oldest; e != c.newest; e = e.newer {
		if held != nil && held(e.value) {
			continue
		}
		v := e.value
		c.unlink(e)
		delete(c.entries, e.key)
		c.size -= e.size
		e.value = zero
		c.spare = e
		return v, true
	}
	return zero, false
}

// push makes e the most recently used entry.
func (c *Cache[K, V]) push(e *entry[K, V]) {
	e.newer, e.older = nil, c.newest
	if c.newest != nil {
		c.newest.newer = e
	}
	c.newest = e
	if c.oldest == nil {
		c.oldest = e
	}
}

// unlink takes e out of the list of entries.
func (c *Cache[K, V]) unlink(e *entry[K, V]) {
	if e.newer != nil {
		e.newer.older = e.older
	} else {
		c.newest = e.older
	}
	if e.older != nil {
		e.older.newer = e.newer
	} else {
		c.oldest = e.newer
	}
	e.newer, e.older = nil, nil
}
