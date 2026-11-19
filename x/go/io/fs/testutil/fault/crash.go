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
	"path"
	"slices"

	"github.com/synnaxlabs/x/validate"
)

// CrashConfig sets how much of the state that is not durable a crash keeps. The zero
// value keeps none of it.
type CrashConfig struct {
	// RunKey fixes every random choice of the crash.
	RunKey uint64
	// DataSurvivalRate is the chance, from 0 to 1, that a block of a file written
	// since the last Sync reaches the disk. Each block has its own chance, so the
	// blocks of one write can part, and a later write can survive an earlier one.
	DataSurvivalRate float64
	// TornRate is the chance, from 0 to 1, that a surviving block keeps only a prefix
	// of its new bytes.
	TornRate float64
}

func (c CrashConfig) validate() error {
	v := validate.New("fault.CrashConfig")
	validateRate(v, "data_survival_rate", c.DataSurvivalRate)
	validateRate(v, "torn_rate", c.TornRate)
	return v.Error()
}

// Crash returns the disk as a power loss leaves it now. The new FS holds the bytes
// that a Sync made durable, the directory entries that a SyncDir made durable, and the
// part of the other state that cfg keeps. Crash does not change f, and files open on f
// stay on f. The error matches validate.ErrValidation when a rate of cfg is not from 0
// to 1.
func (f *FS) Crash(cfg CrashConfig) (*FS, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	d := f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	c := crash{cfg: cfg, copies: map[*node]*node{}}
	return &FS{d: &disk{root: c.image(d.root, "/"), clock: d.clock}, root: f.root}, nil
}

type crash struct {
	// copies maps each node to its image, so two names of one node keep one node.
	copies map[*node]*node
	cfg    CrashConfig
}

// image returns the node that the crash leaves in place of n, which is at the path p.
func (c *crash) image(n *node, p string) *node {
	if out, ok := c.copies[n]; ok {
		return out
	}
	out := &node{dir: n.dir, modTime: n.modTime}
	c.copies[n] = out
	if !n.dir {
		out.data = c.data(n, hashPath(p))
		out.durable = slices.Clone(out.data)
		return out
	}
	out.entries = make(map[string]*node, len(n.durableEntries))
	for _, name := range slices.Sorted(maps.Keys(n.durableEntries)) {
		out.entries[name] = c.image(n.durableEntries[name], path.Join(p, name))
	}
	out.durableEntries = maps.Clone(out.entries)
	return out
}

// keeps reports whether the draw of kind for the block index of the file with the
// path hash file falls under rate.
func (c *crash) keeps(rate float64, kind, file uint64, index int64) bool {
	return draw(c.cfg.RunKey, kind, file, uint64(index)) < rate
}

// data returns the bytes that the crash leaves in the file n. A block that does not
// survive keeps its durable bytes, and reads as zeros past the durable length.
func (c *crash) data(n *node, file uint64) []byte {
	size, rate := int64(len(n.data)), c.cfg.DataSurvivalRate
	out := slices.Clone(n.durable)
	if size < int64(len(out)) && c.keeps(rate, drawCut, file, 0) {
		out = out[:size]
	}
	for b := range n.dirty.all() {
		from := b * blockSize
		if from >= size || !c.keeps(rate, drawBlock, file, b) {
			continue
		}
		to := min(from+blockSize, size)
		if c.keeps(c.cfg.TornRate, drawTorn, file, b) {
			to = from + int64(
				draw(c.cfg.RunKey, drawTornSize, file, uint64(b))*float64(to-from),
			)
		}
		if to > int64(len(out)) {
			out = resized(out, to)
		}
		copy(out[from:to], n.data[from:to])
	}
	return out
}
