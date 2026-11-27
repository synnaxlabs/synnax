// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"encoding/binary"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/telem"
)

const (
	db         = 0xD8D8
	generation = 3
)

// sample returns a snapshot of a clock with two segments, two data keys on it, one data
// segment on another clock, and two intents. pointers sets how many pointers the open
// clock segment holds, so a large value spans more than one chunk.
func sample(pointers int) *state.Snapshot {
	open := state.Segment{
		ID: 12, Clock: 5, Version: 1, DataType: 7, Length: 0, TailCRC: 0xABCD,
	}
	for i := range pointers {
		open.Pointers = append(open.Pointers, state.Pointer{
			First:     telem.TimeStamp(1000 + 10*i),
			Last:      telem.TimeStamp(1000 + 10*i + 4),
			Alignment: telem.NewAlignment(uint32(10+i), 0),
			Offset:    uint32(40 * i),
			Length:    40,
			Samples:   5,
		})
		open.Length += 40
	}
	return &state.Snapshot{
		Sequence: 77,
		Limits:   state.Limits{Segments: 64, Alignments: 1 << 20, Parts: 3},
		Channels: []*state.Channel{
			{Key: 5, Segments: []state.Segment{
				{
					ID: 2, Clock: 5, Version: 1, DataType: 7, Sealed: true,
					Length: 24, TailCRC: 0x1234,
					Pointers: []state.Pointer{{
						First: 10, Last: 30, Alignment: telem.NewAlignment(1, 0),
						Offset: 0, Length: 24, Samples: 3,
					}},
				},
				open,
			}},
			{Key: 6, Segments: []state.Segment{{
				ID: 3, Clock: 5, Version: 1, DataType: 4, Length: 12, TailCRC: 9,
				Pointers: []state.Pointer{{
					First: 10, Last: 30, Alignment: telem.NewAlignment(1, 0),
					Offset: 0, Length: 12, Samples: 3,
				}},
			}}},
			{Key: 9, Segments: []state.Segment{
				{ID: 4, Clock: 5, Version: 1, DataType: 1, Sealed: true},
				{
					ID: 20, Clock: 7, Version: 1, DataType: 1, Length: 8,
					Pointers: []state.Pointer{{
						First: 50, Last: 50, Alignment: telem.NewAlignment(2, 7),
						Offset: 0, Length: 8, Samples: 1,
					}},
				},
			}},
		},
		Intents: []uint32{2, 6},
	}
}

// section returns the offset and record count of section kind of head b.
func section(b []byte, kind int) (off, count uint64) {
	table := binary.LittleEndian.Uint64(b[72:])
	e := b[table+uint64(kind-1)*24:]
	return binary.LittleEndian.Uint64(e[8:]), binary.LittleEndian.Uint64(e[16:])
}

// reseal sets every checksum of head b again, so a change to b reaches the checks
// after the checksums.
func reseal(b []byte) []byte {
	if len(b) < checkpoint.SectorSize {
		return b
	}
	dbID := binary.LittleEndian.Uint64(b[16:])
	gen := binary.LittleEndian.Uint64(b[24:])
	chunks := binary.LittleEndian.Uint64(b[88:])
	if chunks < checkpoint.BodyStart || chunks > uint64(len(b)) {
		return b
	}
	for i, off := 0, uint64(checkpoint.BodyStart); off < chunks; i, off = i+1,
		off+checkpoint.ChunkSize {
		at := chunks + 4*uint64(i)
		if at+4 > uint64(len(b)) {
			return b
		}
		end := min(off+checkpoint.ChunkSize, chunks)
		binary.LittleEndian.PutUint32(
			b[at:],
			place.Head(dbID, gen, off).Sum(b[off:end]),
		)
	}
	binary.LittleEndian.PutUint32(b[96:], place.Head(dbID, gen, chunks).Sum(b[chunks:]))
	binary.LittleEndian.PutUint32(b[508:], place.Head(dbID, gen, 0).Sum(b[:508]))
	return b
}
