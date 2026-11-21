// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint

import (
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// Create writes head 1 of database db, which holds s, to the root fs. When the sealed
// segment records of s pass partSize bytes, they go to parts of about partSize bytes
// each, which Create writes and syncs before the head, and the head holds the rest. A
// partSize of zero is the default of Config.PartSize. Create first removes each part
// file that a cut-short create left. It returns the index of the parts. The sync of the
// header sector of the head is the commit point.
func Create(fs xfs.FS, db uint64, s *state.Snapshot, partSize int) (*Index, error) {
	if partSize == 0 {
		partSize = defaultPartSize
	}
	if err := removeParts(fs); err != nil {
		return nil, err
	}
	var (
		x     = &Index{limit: max(s.Limits.Parts, 1)}
		f     = flatten(s)
		parts []Part
	)
	if f.sealedSize() > partSize {
		for _, channels := range split(f.keep(true), partSize) {
			p, err := writePart(fs, db, x.limit, 1, channels)
			if err != nil {
				return nil, err
			}
			parts, x.parts = append(parts, p), append(x.parts, index(p, channels))
			x.limit++
		}
		f.part(f.sealedAt())
	}
	head := *s
	head.Channels, head.Limits.Parts = f.keep(false), x.limit
	return x, Write(fs, 1, encodeHead(db, 1, &head, parts))
}

// removeParts removes each part file of the root fs, and syncs the root when it removed
// one.
func removeParts(fs xfs.FS) error {
	entries, err := fs.List("")
	if err != nil {
		return err
	}
	removed := false
	for _, e := range entries {
		if id, ok := generationOf(e.Name(), ".part"); ok {
			if err = fs.Remove(partName(id)); err != nil {
				return err
			}
			removed = true
		}
	}
	if removed {
		return fs.SyncDir("")
	}
	return nil
}

// split cuts channels into groups whose segment records fill about size bytes each, at
// segment boundaries, keeping their order.
func split(channels []*state.Channel, size int) [][]*state.Channel {
	var (
		groups [][]*state.Channel
		group  []*state.Channel
		filled int
	)
	for _, ch := range channels {
		start := 0
		for k, seg := range ch.Segments {
			filled += int(headFormat.sizes[sectionSegs]) +
				len(seg.Pointers)*int(headFormat.sizes[sectionPtrs])
			if filled < size {
				continue
			}
			group = append(
				group,
				&state.Channel{Key: ch.Key, Segments: ch.Segments[start : k+1]},
			)
			groups, group, filled, start = append(groups, group), nil, 0, k+1
		}
		if start < len(ch.Segments) {
			group = append(
				group,
				&state.Channel{Key: ch.Key, Segments: ch.Segments[start:]},
			)
		}
	}
	if len(group) > 0 {
		groups = append(groups, group)
	}
	return groups
}
