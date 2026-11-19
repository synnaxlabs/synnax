// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault

import (
	"iter"
	"maps"
	"math/bits"
	"slices"
	"syscall"

	"github.com/synnaxlabs/x/set"
)

// blockSize is the unit of file data that reaches the disk, or does not, as a whole.
const blockSize = 4096

// blocks is a set of block indexes.
type blocks []uint64

// mark adds each block that holds a byte of the range from, to.
func (b *blocks) mark(from, to int64) {
	if from >= to {
		return
	}
	last := (to - 1) / blockSize
	if grow := int(last/64) + 1 - len(*b); grow > 0 {
		*b = append(*b, make([]uint64, grow)...)
	}
	for i := from / blockSize; i <= last; i++ {
		(*b)[i/64] |= 1 << (i % 64)
	}
}

// all returns the blocks in rising order.
func (b *blocks) all() iter.Seq[int64] {
	return func(yield func(int64) bool) {
		for i, word := range *b {
			for ; word != 0; word &= word - 1 {
				if !yield(int64(i*64 + bits.TrailingZeros64(word))) {
					return
				}
			}
		}
	}
}

// resized returns data with the length size. New bytes are zeros.
func resized(data []byte, size int64) []byte {
	if grow := size - int64(len(data)); grow > 0 {
		return append(data, make([]byte, grow)...)
	}
	return data[:size]
}

// sync makes the bytes and the length of the file n at the path p durable. It copies
// only the dirty blocks. A sync that fails drops them.
func (d *disk) sync(n *node, p string) error {
	if len(n.dirty) > 0 {
		file, nth := hashPath(p), d.next(drawSync, p, 0)
		if d.draw(drawSync, file, nth) < d.cfg.SyncErrorRate {
			d.drop(n, file, nth)
			return pathError("sync", p, syscall.EIO)
		}
	}
	size := int64(len(n.data))
	n.durable = resized(n.durable, size)
	for b := range n.dirty.all() {
		if from := b * blockSize; from < size {
			to := min(from+blockSize, size)
			copy(n.durable[from:to], n.data[from:to])
		}
	}
	n.dirty = nil
	return nil
}

// drop loses the dirty blocks of the file n, as a failed Sync does. Each block keeps
// its new bytes for a read, or at the eviction rate goes back to its durable bytes.
func (d *disk) drop(n *node, file, nth uint64) {
	size := int64(len(n.data))
	for b := range n.dirty.all() {
		from := b * blockSize
		evicted := d.draw(drawEvict, file, nth, uint64(b)) < d.cfg.EvictionRate
		if from >= size || !evicted {
			continue
		}
		block := n.data[from:min(from+blockSize, size)]
		clear(block)
		if from < int64(len(n.durable)) {
			copy(block, n.durable[from:])
		}
	}
	n.dirty = nil
}

// entry names one directory entry.
type entry struct {
	dir  *node
	name string
}

// syncDir makes the entries of the directory n at the path p durable. A rename between
// n and another directory becomes durable in both, as one journal change does, and so
// does each earlier change to an entry that such a change reads.
func (d *disk) syncDir(n *node, p string) error {
	if !maps.Equal(n.entries, n.durableEntries) {
		nth := d.next(drawSync, p, 0)
		if d.draw(drawSync, hashPath(p), nth) < d.cfg.SyncErrorRate {
			return pathError("sync", p, syscall.EIO)
		}
	}
	// A change is durable now when it changes n, or when a later durable change changes
	// one of its entries. The walk goes from the newest change to the oldest.
	durable, needed := make([]bool, len(d.pending)), set.Set[entry]{}
	for i := len(d.pending) - 1; i >= 0; i-- {
		halves := d.pending[i].halves
		durable[i] = slices.ContainsFunc(halves, func(h half) bool {
			return h.dir == n || needed.Contains(entry{dir: h.dir, name: h.name})
		})
		for _, h := range halves {
			if durable[i] {
				needed.Add(entry{dir: h.dir, name: h.name})
			}
		}
	}
	kept := d.pending[:0]
	for i, c := range d.pending {
		if !durable[i] {
			kept = append(kept, c)
			continue
		}
		for _, h := range c.halves {
			if h.to == nil {
				delete(h.dir.durableEntries, h.name)
			} else {
				h.dir.durableEntries[h.name] = h.to
			}
		}
	}
	d.pending = kept
	return nil
}
