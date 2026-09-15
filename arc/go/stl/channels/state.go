// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels

import (
	"slices"

	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
)

// Digest provides metadata about a channel for state initialization.
type Digest struct {
	DataType telem.DataType
	Key      uint32
	Index    uint32
}

// ProgramState manages channel I/O buffers and index mapping.
type ProgramState struct {
	reads           map[uint32]telem.MultiSeries
	writes          map[uint32]telem.Series
	activeWriteKeys []uint32
	indexes         map[uint32]uint32
	// clock provides monotonically increasing timestamps for indexed
	// channel writes, avoiding duplicate timestamps on platforms with
	// coarse clock resolution (e.g. Windows).
	clock telem.MonoClock
	// missingRead is the first channel a host read found without a buffered
	// value since the last TakeMissingRead; valid when hasMissingRead is set.
	missingRead    uint32
	hasMissingRead bool
}

// noteMissingRead records that a host read of key found no buffered value. Only
// the first miss since the last TakeMissingRead is kept.
func (ps *ProgramState) noteMissingRead(key uint32) {
	if ps.hasMissingRead {
		return
	}
	ps.missingRead, ps.hasMissingRead = key, true
}

// TakeMissingRead returns and clears the channel a host read found empty since
// the last call. ok is false when every read hit.
func (ps *ProgramState) TakeMissingRead() (key uint32, ok bool) {
	key, ok = ps.missingRead, ps.hasMissingRead
	ps.missingRead, ps.hasMissingRead = 0, false
	return key, ok
}

// NewProgramState creates a new ProgramState from channel digests.
func NewProgramState(digests []Digest) *ProgramState {
	ps := &ProgramState{
		reads:   make(map[uint32]telem.MultiSeries),
		writes:  make(map[uint32]telem.Series),
		indexes: make(map[uint32]uint32),
	}
	for _, d := range digests {
		ps.indexes[d.Key] = d.Index
	}
	return ps
}

// Ingest adds external channel data to the read buffer.
func (ps *ProgramState) Ingest(fr telem.Frame[uint32]) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		ps.reads[key] = ps.reads[key].Append(fr.RawSeriesAt(rawI))
	}
}

// Flush extracts buffered channel writes into a frame and clears the write
// buffer. Only channels written in the current cycle are flushed.
func (ps *ProgramState) Flush(
	fr telem.Frame[uint32],
) (telem.Frame[uint32], bool) {
	if len(ps.activeWriteKeys) == 0 {
		return fr, false
	}
	flushed := false
	for _, key := range ps.activeWriteKeys {
		data, ok := ps.writes[key]
		if !ok || len(data.Data) == 0 {
			continue
		}
		fr = fr.Append(key, data.DeepCopy())
		flushed = true
		data.Data = data.Data[:0]
		data.TimeRange = telem.TimeRangeZero
		data.Alignment = 0
		ps.writes[key] = data
	}
	ps.activeWriteKeys = ps.activeWriteKeys[:0]
	return fr, flushed
}

// clearReadsReallocThreshold is the backing array capacity above which
// ClearReads allocates a fresh slice instead of re-slicing in place. Below
// this threshold, slices.Delete zeroes old references (allowing GC) without
// allocating.
const clearReadsReallocThreshold = 64

// ClearReads clears accumulated channel read buffers while preserving the
// latest series for each channel.
func (ps *ProgramState) ClearReads() {
	for key, ser := range ps.reads {
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
		ps.reads[key] = ser
	}
}

// readValue returns the latest series buffered on key.
func (ps *ProgramState) readValue(key uint32) (telem.Series, bool) {
	ms, ok := ps.reads[key]
	if !ok || len(ms.Series) == 0 {
		return telem.Series{}, false
	}
	return ms.Series[len(ms.Series)-1], ok
}

// writeValue writes a single value to a channel (for WASM runtime bindings).
func (ps *ProgramState) writeValue(key uint32, value telem.Series) {
	ps.appendWriteSeries(key, value)
	ps.writeIndexedTimestamp(key)
}

// writeSample appends one fixed-size sample to key's write buffer and stamps its
// index channel when it has one.
func writeSample[T telem.FixedSample](ps *ProgramState, key uint32, v T) {
	appendFixedWriteSample(ps, key, v)
	ps.writeIndexedTimestamp(key)
}

func (ps *ProgramState) writeIndexedTimestamp(key uint32) {
	idx := ps.indexes[key]
	if idx != 0 {
		appendFixedWriteSample(ps, idx, ps.clock.Now())
	}
}

func appendFixedWriteSample[T telem.FixedSample](
	ps *ProgramState,
	key uint32,
	value T,
) {
	dt := telem.InferDataType[T]()
	acc, exists := ps.writes[key]
	if !exists {
		acc = telem.Series{DataType: dt}
	}
	if len(acc.Data) == 0 {
		ps.activeWriteKeys = append(ps.activeWriteKeys, key)
	}
	if acc.DataType == telem.UnknownT {
		acc.DataType = dt
	}
	if acc.DataType != dt && len(acc.Data) > 0 {
		ps.writes[key] = telem.NewSeriesV(value)
		return
	}
	den := int(dt.Density())
	sampleStart := len(acc.Data)
	acc.Data = slices.Grow(acc.Data, den)
	acc.Data = acc.Data[:sampleStart+den]
	unsafe.CastSlice[byte, T](acc.Data)[sampleStart/den] = value
	ps.writes[key] = acc
}

// readSeries reads buffered data and time series from a channel.
func (ps *ProgramState) readSeries(
	key uint32,
) (data, time telem.MultiSeries, ok bool) {
	data, ok = ps.reads[key]
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	indexKey := ps.indexes[key]
	if indexKey == 0 {
		return data, telem.MultiSeries{}, len(data.Series) > 0
	}
	time, ok = ps.reads[indexKey]
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	return data, time, len(time.Series) > 0 && len(data.Series) > 0
}

func (ps *ProgramState) writeChannel(key uint32, data, time telem.Series) {
	ps.appendWriteSeries(key, data)
	idx := ps.indexes[key]
	if idx != 0 {
		ps.appendWriteSeries(idx, time)
	}
}

func (ps *ProgramState) appendWriteSeries(key uint32, source telem.Series) {
	acc, exists := ps.writes[key]
	if !exists {
		acc = telem.Series{DataType: source.DataType}
	}
	if len(acc.Data) == 0 {
		ps.activeWriteKeys = append(ps.activeWriteKeys, key)
	}
	if acc.DataType == telem.UnknownT {
		acc.DataType = source.DataType
	}
	if len(source.Data) == 0 {
		ps.writes[key] = acc
		return
	}
	if acc.DataType != source.DataType && len(acc.Data) > 0 {
		acc = source.DeepCopy()
		ps.writes[key] = acc
		return
	}
	if len(acc.Data) == 0 {
		acc.TimeRange = source.TimeRange
		acc.Alignment = source.Alignment
	} else {
		if source.TimeRange.Start < acc.TimeRange.Start {
			acc.TimeRange.Start = source.TimeRange.Start
		}
		if source.TimeRange.End > acc.TimeRange.End {
			acc.TimeRange.End = source.TimeRange.End
		}
	}
	acc.Data = append(acc.Data, source.Data...)
	ps.writes[key] = acc
}
