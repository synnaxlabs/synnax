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

// misdirect replaces p, the bytes that the read nth of the block at gets, at the
// misdirected read rate. The new bytes come from the same offset in another block that
// the disk holds: a block of the file n, or of another file. off is the offset of p in
// the file. It reports if it replaced p.
func (d *disk) misdirect(p []byte, n *node, at place, nth uint64, off int64) bool {
	file, block := uint64(at.file), uint64(at.block)
	if d.draw(drawMisdirect, file, block, nth) >= d.cfg.MisdirectedReadRate {
		return false
	}
	files := d.files(d.root, nil)
	others := int64(-1)
	for _, f := range files {
		others += blockCount(f)
	}
	source := int64(d.draw(drawMisdirectSource, file, block, nth) * float64(others))
	for _, f := range files {
		count := blockCount(f)
		if f == n && source >= at.block {
			source++
		}
		if source >= count {
			source -= count
			continue
		}
		clear(p)
		if start := (source-at.block)*blockSize + off; start < int64(len(f.data)) {
			copy(p, f.data[start:])
		}
		return true
	}
	return false
}

func blockCount(n *node) int64 {
	return (int64(len(n.data)) + blockSize - 1) / blockSize
}

// files adds each file under the directory n to out, in the order of the names.
func (d *disk) files(n *node, out []*node) []*node {
	if !n.dir {
		return append(out, n)
	}
	for _, name := range slices.Sorted(maps.Keys(n.entries)) {
		out = d.files(n.entries[name], out)
	}
	return out
}
