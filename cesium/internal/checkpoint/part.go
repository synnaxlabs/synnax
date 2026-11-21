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
	"encoding/binary"
	"fmt"
	"math"

	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// PartName returns the file name of the part id.
func PartName(id uint64) string { return fmt.Sprintf("%016x.part", id) }

// encodePart returns the bytes of the part id that the switch of generation writes in
// database db, which holds channels, and the record of it that a head holds.
func encodePart(db, id, generation uint64, channels []*state.Channel) ([]byte, Part) {
	b := partFormat.encode(db, body{file: id, second: generation, channels: channels})
	p := Part{
		ID:         id,
		Generation: generation,
		Length:     uint64(len(b)),
		First:      math.MaxInt64,
		CRC:        binary.LittleEndian.Uint32(b[sectorCRC:]),
	}
	for _, c := range channels {
		for _, seg := range c.Segments {
			for _, ptr := range seg.Pointers {
				p.First, p.Last = min(p.First, ptr.First), max(p.Last, ptr.Last)
			}
		}
	}
	if p.First == telem.TimeStamp(math.MaxInt64) {
		p.First = 0
	}
	return b, p
}

// writePart writes the part id of channels to the root fs, as Write does a head, and
// returns the record of it that a head holds.
func writePart(
	fs xfs.FS,
	db, id, generation uint64,
	channels []*state.Channel,
) (Part, error) {
	b, p := encodePart(db, id, generation, channels)
	return p, writeFile(fs, PartName(id), b)
}
