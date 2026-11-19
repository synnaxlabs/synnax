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

	"github.com/synnaxlabs/x/set"
)

// misdirect replaces the bytes of one block in p with the bytes of another block of the
// disk, at the misdirected read rate. p holds the bytes that a read of the file n at
// path and off returns.
func (d *disk) misdirect(p []byte, n *node, path string, off int64) {
	if d.cfg.MisdirectedReadRate == 0 || len(p) == 0 {
		return
	}
	file, at, nth := hashPath(path), uint64(off), d.next(drawMisdirect, path, off)
	if draw(d.cfg.RunKey, drawMisdirect, file, at, nth) >= d.cfg.MisdirectedReadRate {
		return
	}
	first, last := off/blockSize, (off+int64(len(p))-1)/blockSize
	pick := draw(d.cfg.RunKey, drawMisdirectBlock, file, at, nth)
	victim := first + int64(pick*float64(last-first+1))
	files := d.files(d.root, nil, set.Set[*node]{})
	others := int64(-1)
	for _, f := range files {
		others += blockCount(f)
	}
	if others < 1 {
		return
	}
	source := int64(
		draw(d.cfg.RunKey, drawMisdirectSource, file, at, nth) * float64(others),
	)
	for _, f := range files {
		count := blockCount(f)
		if f == n && source >= victim {
			source++
		}
		if source < count {
			from, to := max(
				off,
				victim*blockSize,
			), min(
				off+int64(len(p)),
				(victim+1)*blockSize,
			)
			block := p[from-off : to-off]
			clear(block)
			if start := source*blockSize + from - victim*blockSize; start < int64(
				len(f.data),
			) {
				copy(block, f.data[start:])
			}
			return
		}
		source -= count
	}
}

func blockCount(
	n *node,
) int64 {
	return (int64(len(n.data)) + blockSize - 1) / blockSize
}

// files adds each file under the directory n that is not in seen to out, in the order
// of the names.
func (d *disk) files(n *node, out []*node, seen set.Set[*node]) []*node {
	if seen.Contains(n) {
		return out
	}
	seen.Add(n)
	if !n.dir {
		return append(out, n)
	}
	for _, name := range slices.Sorted(maps.Keys(n.entries)) {
		out = d.files(n.entries[name], out, seen)
	}
	return out
}
