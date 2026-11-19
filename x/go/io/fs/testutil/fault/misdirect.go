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

// misdirect replaces p with the bytes at the same place of another block of the disk,
// at the misdirected read rate, and reports if it did. p holds the bytes of one block
// that a read of the file n at path returns from off.
func (d *disk) misdirect(p []byte, n *node, path string, off int64) bool {
	file, at, nth := hashPath(path), uint64(off), d.next(drawMisdirect, path, off)
	if d.draw(drawMisdirect, file, at, nth) >= d.cfg.MisdirectedReadRate {
		return false
	}
	victim := off / blockSize
	files := d.files(d.root, nil)
	others := int64(-1)
	for _, f := range files {
		others += blockCount(f)
	}
	source := int64(d.draw(drawMisdirectSource, file, at, nth) * float64(others))
	for _, f := range files {
		count := blockCount(f)
		if f == n && source >= victim {
			source++
		}
		if source >= count {
			source -= count
			continue
		}
		clear(p)
		if start := (source-victim)*blockSize + off; start < int64(len(f.data)) {
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
