// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"sync"

	"github.com/synnaxlabs/cesium/internal/lru"
)

// Cache holds full blocks of the segments of one database that passed their check, up
// to a byte budget, for the readers of the database to share. It drops the least
// recently used block that no reader holds first. A committed full block never changes,
// so a cached block needs no check and no invalidation. A Cache is safe for concurrent
// use.
type Cache struct {
	mu struct {
		sync.Mutex
		blocks *lru.Cache[blockKey, *block]
		// spare is the block that the last eviction took out, which the next put uses.
		spare *block
	}
}

// blockKey names the block at stream offset start of segment id.
type blockKey struct {
	id    uint64
	start int64
}

// block is a checked full block, with its trailer, and the number of readers that hold
// it. A held block stays in the cache, and its bytes never change.
type block struct {
	data []byte
	refs int
}

func held(b *block) bool { return b.refs > 0 }

// NewCache returns an empty cache that holds at most budget bytes of blocks.
func NewCache(budget int) *Cache {
	c := &Cache{}
	c.mu.blocks = lru.New[blockKey, *block](budget)
	return c
}

// get holds the block at k for a reader, and releases prev, which the reader held.
func (c *Cache) get(k blockKey, prev *block) (*block, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	release(prev)
	b, ok := c.mu.blocks.Get(k)
	if ok {
		b.refs++
	}
	return b, ok
}

// put adds data, a block that passed its check, at k, and holds it for the reader. A
// block that another reader added at k first stays. put returns the held block and a
// buffer that no reader holds for the next read of the reader, or nil.
func (c *Cache) put(k blockKey, data []byte) (*block, []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := c.mu.spare
	if b == nil {
		b = &block{}
	}
	b.data, b.refs = data, 1
	if !c.mu.blocks.Put(k, b, len(data)) {
		c.mu.spare = b
		b, _ = c.mu.blocks.Get(k)
		b.refs++
		return b, data
	}
	c.mu.spare = nil
	var spare []byte
	for {
		old, ok := c.mu.blocks.Evict(held)
		if !ok {
			return b, spare
		}
		if spare == nil {
			spare, c.mu.spare = old.data, old
		}
	}
}

// release drops the hold of a reader on b.
func (c *Cache) release(b *block) {
	c.mu.Lock()
	defer c.mu.Unlock()
	release(b)
}

// release drops the hold of a reader on b, or nothing for a nil b. The caller holds the
// lock of the cache of b.
func release(b *block) {
	if b != nil {
		b.refs--
	}
}
