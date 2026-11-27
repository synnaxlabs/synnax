// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package state holds what a Cesium database holds: the segments of each key, their
// committed pointers, the claims of writers, the delete intents, and the limits of the
// counters. One owner goroutine changes it, and applies or refuses each log record. It
// does no I/O.
package state

import (
	"iter"

	"github.com/synnaxlabs/x/telem"
)

// Pointer is a committed byte range of a segment.
type Pointer struct {
	// First and Last are the stamps of the first and last samples.
	First, Last telem.TimeStamp
	// Alignment is the alignment of the first sample.
	Alignment telem.Alignment
	// Offset and Length are the stream offset and the byte length of the range.
	Offset, Length uint32
	// Samples is the number of samples in the range.
	Samples uint32
}

// Segment is a file of samples of one key, with one data type and one clock.
type Segment struct {
	// ID is the identity of the segment, from 1.
	ID uint64
	// Clock is the key of the clock that times the samples.
	Clock uint32
	// Version is the format version of the segment.
	Version uint16
	// DataType is the data type code of the samples.
	DataType uint8
	// Sealed is true when the segment never grows again.
	Sealed bool
	// Length is the committed length in stream bytes, and TailCRC the running checksum
	// of the unfinished last block there.
	Length, TailCRC uint32
	// Pointers are the committed ranges, in stream order.
	Pointers []Pointer
}

// Channel is the state of one key. A Channel that the state publishes never changes.
// Read its segments with All.
type Channel struct {
	Key uint32
	// sealed holds the sealed segments and open the others, each in rising order of
	// identity. Published Channels share sealed, and the state appends to it only past
	// every published length.
	sealed, open []Segment
}

// NewChannel returns the Channel of key with segs, which are in rising order of
// identity.
func NewChannel(key uint32, segs ...Segment) *Channel {
	c := &Channel{Key: key}
	for _, seg := range segs {
		if seg.Sealed {
			c.sealed = append(c.sealed, seg)
		} else {
			c.open = append(c.open, seg)
		}
	}
	return c
}

// All returns the segments of c in rising order of identity. The caller must not change
// them.
func (c *Channel) All() iter.Seq[*Segment] {
	return func(yield func(*Segment) bool) {
		i, j := 0, 0
		for i < len(c.sealed) || j < len(c.open) {
			var seg *Segment
			if j == len(c.open) ||
				(i < len(c.sealed) && c.sealed[i].ID < c.open[j].ID) {
				seg, i = &c.sealed[i], i+1
			} else {
				seg, j = &c.open[j], j+1
			}
			if !yield(seg) {
				return
			}
		}
	}
}

// Len returns how many segments c holds.
func (c *Channel) Len() int { return len(c.sealed) + len(c.open) }

// Limits are the values below which the database gives out identities and numbers.
type Limits struct {
	// Segments is the segment identity limit.
	Segments uint64
	// Alignments is the alignment number limit.
	Alignments uint64
	// Parts is the identity limit of part records. This package gives out no part yet.
	Parts uint64
}

// Snapshot is the durable state after the log entry of Sequence.
type Snapshot struct {
	// Sequence is the sequence of the last log entry that the state covers.
	Sequence uint64
	Limits   Limits
	// Channels are in rising order of key.
	Channels []*Channel
	// Intents are the keys of the live delete intents, in rising order.
	Intents []uint32
}
