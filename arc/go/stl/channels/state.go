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
}

// NewProgramState creates a new ProgramState from channel digests.
func NewProgramState(digests []Digest) *ProgramState {
	cs := &ProgramState{
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
func (cs *ProgramState) Ingest(fr telem.Frame[uint32]) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		cs.reads[key] = cs.reads[key].Append(fr.RawSeriesAt(rawI))
	}
}

// Flush extracts buffered channel writes into a frame and clears the write
// buffer. Only channels written in the current cycle are flushed. An index
// whose channels wrote no timestamps of their own is stamped from now, one
// sample per data sample, spaced 1ns apart. Flush returns the highest stamp it
// synthesized, so the caller's clock can resume above it, and reports whether
// anything was flushed. The highest stamp is zero when nothing was synthesized.
func (cs *ProgramState) Flush(
	fr telem.Frame[uint32],
	now telem.TimeStamp,
) (telem.Frame[uint32], telem.TimeStamp, bool) {
	if len(cs.activeWriteKeys) == 0 {
		return fr, 0, false
	}
	highest := cs.stampIndexes(now)
	flushed := false
	for _, key := range cs.activeWriteKeys {
		data, ok := cs.writes[key]
		if !ok || len(data.Data) == 0 {
			continue
		}
		fr = fr.Append(key, data.DeepCopy())
		flushed = true
		data.Data = data.Data[:0]
		data.TimeRange = telem.TimeRangeZero
		data.Alignment = 0
		cs.writes[key] = data
	}
	cs.activeWriteKeys = cs.activeWriteKeys[:0]
	return fr, highest, flushed
}

// stampIndexes fills every empty index buffer whose channels wrote this cycle,
// and returns the highest timestamp it wrote. A non-empty buffer already holds
// the provenance a sink forwarded and is left alone. Every member of a group
// starts at now, so one cycle's writes stay aligned across channels.
func (cs *ProgramState) stampIndexes(now telem.TimeStamp) (highest telem.TimeStamp) {
	for _, key := range cs.activeWriteKeys {
		idx := cs.indexes[key]
		if idx == 0 || len(cs.writes[idx].Data) > 0 {
			continue
		}
		data := cs.writes[key]
		count := data.Len()
		if count == 0 {
			continue
		}
		stamps := telem.Arrange(now, int(count), 1*telem.NanosecondTS)
		stamps.Alignment = data.Alignment
		stamps.TimeRange = data.TimeRange
		cs.writes[idx] = stamps
		cs.activeWriteKeys = append(cs.activeWriteKeys, idx)
		if last := telem.ValueAt[telem.TimeStamp](stamps, -1); last > highest {
			highest = last
		}
	}
	return highest
}

// clearReadsReallocThreshold is the backing array capacity above which
// ClearReads allocates a fresh slice instead of re-slicing in place. Below
// this threshold, slices.Delete zeroes old references (allowing GC) without
// allocating.
const clearReadsReallocThreshold = 64

// ClearReads clears accumulated channel read buffers while preserving the
// latest series for each channel.
func (cs *ProgramState) ClearReads() {
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
func (cs *ProgramState) ReadValue(key uint32) (telem.Series, bool) {
	ms, ok := cs.reads[key]
	if !ok || len(ms.Series) == 0 {
		return telem.Series{}, false
	}
	return ms.Series[len(ms.Series)-1], ok
}

// writeValue writes a single value to a channel (for WASM runtime bindings).
func (cs *ProgramState) writeValue(key uint32, value telem.Series) {
	cs.appendWriteSeries(key, value)
}

func (cs *ProgramState) WriteChannelU8(key uint32, v uint8) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelU16(key uint32, v uint16) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelU32(key, v uint32) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelU64(key uint32, v uint64) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelI8(key uint32, v int8) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelI16(key uint32, v int16) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelI32(key uint32, v int32) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelI64(key uint32, v int64) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelF32(key uint32, v float32) {
	appendFixedWriteSample(cs, key, v)
}

func (cs *ProgramState) WriteChannelF64(key uint32, v float64) {
	appendFixedWriteSample(cs, key, v)
}

func appendFixedWriteSample[T telem.FixedSample](
	cs *ProgramState,
	key uint32,
	value T,
) {
	dt := telem.InferDataType[T]()
	acc, exists := cs.writes[key]
	if !exists {
		acc = telem.Series{DataType: dt}
	}
	if len(acc.Data) == 0 {
		cs.activeWriteKeys = append(cs.activeWriteKeys, key)
	}
	if acc.DataType == telem.UnknownT {
		acc.DataType = dt
	}
	if acc.DataType != dt && len(acc.Data) > 0 {
		cs.writes[key] = telem.NewSeriesV(value)
		return
	}
	den := int(dt.Density())
	sampleStart := len(acc.Data)
	acc.Data = slices.Grow(acc.Data, den)
	acc.Data = acc.Data[:sampleStart+den]
	unsafe.CastSlice[byte, T](acc.Data)[sampleStart/den] = value
	cs.writes[key] = acc
}

// readSeries reads buffered data and time series from a channel.
func (cs *ProgramState) readSeries(
	key uint32,
) (data, time telem.MultiSeries, ok bool) {
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

func (cs *ProgramState) writeChannel(key uint32, data, time telem.Series) {
	cs.appendWriteSeries(key, data)
	idx := cs.indexes[key]
	if idx != 0 {
		cs.appendWriteSeries(idx, time)
	}
}

func (cs *ProgramState) appendWriteSeries(key uint32, source telem.Series) {
	acc, exists := cs.writes[key]
	if !exists {
		acc = telem.Series{DataType: source.DataType}
	}
	if len(acc.Data) == 0 {
		cs.activeWriteKeys = append(cs.activeWriteKeys, key)
	}
	if acc.DataType == telem.UnknownT {
		acc.DataType = source.DataType
	}
	if len(source.Data) == 0 {
		cs.writes[key] = acc
		return
	}
	if acc.DataType != source.DataType && len(acc.Data) > 0 {
		acc = source.DeepCopy()
		cs.writes[key] = acc
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
	cs.writes[key] = acc
}
