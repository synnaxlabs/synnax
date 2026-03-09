// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"slices"

	"github.com/synnaxlabs/x/telem"
)

// Digest provides metadata about a channel for state initialization.
type Digest struct {
	DataType telem.DataType
	Key      uint32
	Index    uint32
}

// State manages channel I/O buffers and index mapping.
type State struct {
	reads   map[uint32]telem.MultiSeries
	writes  map[uint32]telem.Series
	indexes map[uint32]uint32
}

// NewState creates a new State from channel digests.
func NewState(digests []Digest) *State {
	cs := &State{
		reads:   make(map[uint32]telem.MultiSeries),
		writes:  make(map[uint32]telem.Series),
		indexes: make(map[uint32]uint32),
	}
	for _, d := range digests {
		cs.indexes[d.Key] = d.Index
	}
	return cs
}

// Ingest adds external channel data to the read buffer.
func (cs *State) Ingest(fr telem.Frame[uint32]) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		cs.reads[key] = cs.reads[key].Append(fr.RawSeriesAt(rawI))
	}
}

// Flush extracts buffered channel writes into a frame and clears the write
// buffer.
func (cs *State) Flush(
	fr telem.Frame[uint32],
) (telem.Frame[uint32], bool) {
	if len(cs.writes) == 0 {
		return fr, false
	}
	for key, data := range cs.writes {
		fr = fr.Append(key, data.DeepCopy())
	}
	clear(cs.writes)
	return fr, true
}

// clearReadsReallocThreshold is the backing array capacity above which
// ClearReads allocates a fresh slice instead of re-slicing in place. Below
// this threshold, slices.Delete zeroes old references (allowing GC) without
// allocating.
const clearReadsReallocThreshold = 64

// ClearReads clears accumulated channel read buffers while preserving the
// latest series for each channel.
func (cs *State) ClearReads() {
	for key, ser := range cs.reads {
		if len(ser.Series) <= 1 {
			continue
		}
		if cap(ser.Series) > clearReadsReallocThreshold {
			ser.Series = []telem.Series{ser.Series[len(ser.Series)-1]}
		} else {
			ser.Series = slices.Delete(
				ser.Series, 0, len(ser.Series)-1,
			)
		}
		cs.reads[key] = ser
	}
}

// ReadValue reads a single value from a channel (for WASM runtime bindings).
func (cs *State) ReadValue(key uint32) (telem.Series, bool) {
	ms, ok := cs.reads[key]
	if !ok || len(ms.Series) == 0 {
		return telem.Series{}, false
	}
	return ms.Series[len(ms.Series)-1], ok
}

// writeValue writes a single value to a channel (for WASM runtime bindings).
// For channels with an index, it auto-generates a timestamp using telem.Now()
// and writes to both the data channel and its index channel.
func (cs *State) writeValue(key uint32, value telem.Series) {
	cs.writeChannel(key, value, telem.NewSeriesV(telem.Now()))
}

// readSeries reads buffered data and time series from a channel.
func (cs *State) readSeries(
	key uint32,
) (data telem.MultiSeries, time telem.MultiSeries, ok bool) {
	data, ok = cs.reads[key]
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	indexKey := cs.indexes[key]
	if indexKey == 0 {
		return data, telem.MultiSeries{}, len(data.Series) > 0
	}
	time, ok = cs.reads[indexKey]
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	return data, time, len(time.Series) > 0 && len(data.Series) > 0
}

func (cs *State) writeChannel(key uint32, data, time telem.Series) {
	cs.writes[key] = data
	idx := cs.indexes[key]
	if idx != 0 {
		cs.writes[idx] = time
	}
}
