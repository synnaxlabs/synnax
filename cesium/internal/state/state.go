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

import "github.com/synnaxlabs/x/telem"

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
	// Length is the committed length in stream bytes, and TailCRC the running
	// checksum of the unfinished last block there.
	Length, TailCRC uint32
	// Pointers are the committed ranges, in stream order.
	Pointers []Pointer
}

// Channel is the state of one key. A Channel that the state publishes never changes.
type Channel struct {
	Key uint32
	// Segments are in rising order of identity.
	Segments []Segment
}

// Limits are the values below which the database gives out identities and numbers.
type Limits struct {
	// Segments is the segment identity limit.
	Segments uint64
	// Alignments is the alignment number limit.
	Alignments uint64
	// Parts is the part identity limit.
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
