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
	"maps"
	"slices"
)

// Crash returns the disk as a power loss leaves it now. The new FS holds only the
// bytes that a Sync made durable and the directory entries that a SyncDir made
// durable. Crash does not change f, and files open on f stay on f.
func (f *FS) Crash() *FS {
	d := f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	c := crash{copies: map[*node]*node{}}
	return &FS{d: &disk{root: c.image(d.root), clock: d.clock}, root: f.root}
}

type crash struct {
	// copies maps each node to its image, so two names of one node keep one node.
	copies map[*node]*node
}

// image returns the node that the crash leaves in place of n.
func (c *crash) image(n *node) *node {
	if out, ok := c.copies[n]; ok {
		return out
	}
	out := &node{dir: n.dir, modTime: n.modTime}
	c.copies[n] = out
	if !n.dir {
		out.data, out.durable = slices.Clone(n.durable), slices.Clone(n.durable)
		return out
	}
	out.entries = make(map[string]*node, len(n.durableEntries))
	for name, child := range n.durableEntries {
		out.entries[name] = c.image(child)
	}
	out.durableEntries = maps.Clone(out.entries)
	return out
}
