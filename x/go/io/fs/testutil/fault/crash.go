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
	// DataSurvivalRate is the chance, from 0 to 1, that a block of a file written since
	// the last Sync reaches the disk. Each block has its own chance, so the blocks of
	// one write can part, and a later write can survive an earlier one.
	DataSurvivalRate float64
	// TornRate is the chance, from 0 to 1, that a surviving block keeps only a prefix
	// of its new bytes.
	TornRate float64
	// EntrySurvivalRate is the chance, from 0 to 1, that a create, a remove, or a
	// rename that no SyncDir made durable reaches the disk. A rename reaches it as a
	// whole or not at all, and a change that depends on a lost change is lost too.
	EntrySurvivalRate float64
	// Image sets the faults of the FS that Crash returns.
	Image Config
}

func (c CrashConfig) validate() error {
	v := validate.New("fault.CrashConfig")
	validateRate(v, "data_survival_rate", c.DataSurvivalRate)
	validateRate(v, "torn_rate", c.TornRate)
	validateRate(v, "entry_survival_rate", c.EntrySurvivalRate)
	v.Exec(c.Image.validate)
	return v.Error()
}

// Crash returns the disk as a power loss leaves it now. The new FS holds the bytes that
// a Sync made durable, the directory entries that a SyncDir made durable, and the part
// of the other state that cfg keeps. Crash does not change f, and files open on f stay
// on f. The error matches validate.ErrValidation when a rate of cfg is not from 0 to 1,
// or a capacity is under zero.
func (f *FS) Crash(cfg CrashConfig) (*FS, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	d := f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	c := crash{cfg: cfg, entries: map[*node]map[string]*node{}}
	for _, ch := range d.pending {
		if draw(cfg.RunKey, drawEntry, ch.key) < cfg.EntrySurvivalRate {
			c.apply(ch)
		}
	}
	return &FS{d: newDisk(cfg.Image, c.image(d.root, "/"), d.clock), root: f.root}, nil
}

type crash struct {
	// entries holds the entries that the crash leaves in each directory it changed.
	entries map[*node]map[string]*node
	cfg     CrashConfig
}

// dir returns the entries that the crash leaves in the directory n.
func (c *crash) dir(n *node) map[string]*node {
	entries, ok := c.entries[n]
	if !ok {
		entries = maps.Clone(n.durableEntries)
		c.entries[n] = entries
	}
	return entries
}

// apply makes the change ch in the image, but only when each entry that ch changes
// still holds what it held before ch. A change whose base was lost is lost too.
func (c *crash) apply(ch *change) {
	for _, h := range ch.halves {
		if c.dir(h.dir)[h.name] != h.from {
			return
		}
	}
	for _, h := range ch.halves {
		if h.to == nil {
			delete(c.dir(h.dir), h.name)
		} else {
			c.dir(h.dir)[h.name] = h.to
		}
	}
}

// image returns the node that the crash leaves in place of n, which is at the path p.
func (c *crash) image(n *node, p string) *node {
	out := &node{dir: n.dir, modTime: n.modTime}
	if !n.dir {
		out.data = c.data(n, hashPath(p))
		out.durable = slices.Clone(out.data)
		return out
	}
	entries := c.dir(n)
	out.entries = make(map[string]*node, len(entries))
	for _, name := range slices.Sorted(maps.Keys(entries)) {
		out.entries[name] = c.image(entries[name], path.Join(p, name))
	}
	out.durableEntries = maps.Clone(out.entries)
	return out
}

// keeps reports whether the draw of kind for the block index of the file with the path
// hash file falls under rate.
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
