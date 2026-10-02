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
	"cmp"
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
	reads map[uint32]telem.MultiSeries
	// writes holds each channel's samples for this cycle. For an indexed channel it
	// holds only the samples that came with a timestamp, in the order of the timestamps
	// in its index's buffer.
	writes map[uint32]telem.Series
	// unstamped holds the samples body writes added to indexed channels this cycle.
	// Flush gives them timestamps.
	unstamped       map[uint32]telem.Series
	activeWriteKeys []uint32
	indexes         map[uint32]uint32
	// lastStamps holds the last timestamp flushed to each index.
	lastStamps map[uint32]telem.TimeStamp
	// writers counts the channels that wrote to each index this cycle. Flush reuses it.
	writers map[uint32]int
}

// NewProgramState creates a new ProgramState from channel digests.
func NewProgramState(digests []Digest) *ProgramState {
	cs := &ProgramState{
		reads:      make(map[uint32]telem.MultiSeries),
		writes:     make(map[uint32]telem.Series),
		unstamped:  make(map[uint32]telem.Series),
		indexes:    make(map[uint32]uint32),
		lastStamps: make(map[uint32]telem.TimeStamp),
		writers:    make(map[uint32]int),
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

// Flush extracts buffered channel writes into a frame and clears the write buffer. Only
// channels written in the current cycle are flushed. A channel alone on its index keeps
// its supplied timestamps, sorted by time. Its body writes follow, 1ns apart, starting
// after the later of now and the last timestamp of the index. Channels that share an
// index are stamped from now, and only when none of them supplied timestamps. Flush
// returns the highest stamp it synthesized, so the caller's clock can resume above it,
// and reports whether anything was flushed. The highest stamp is zero when nothing was
// synthesized.
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

// stampIndexes gives every sample written this cycle a timestamp and returns the
// highest timestamp it synthesized.
func (cs *ProgramState) stampIndexes(now telem.TimeStamp) (highest telem.TimeStamp) {
	clear(cs.writers)
	for _, key := range cs.activeWriteKeys {
		if idx := cs.indexes[key]; idx != 0 && idx != key {
			cs.writers[idx]++
		}
	}
	for _, key := range cs.activeWriteKeys {
		idx := cs.indexes[key]
		if idx == 0 || idx == key {
			continue
		}
		var last telem.TimeStamp
		if cs.writers[idx] == 1 {
			last = cs.stampAlone(key, idx, now)
		} else {
			last = cs.stampGroup(key, idx, now)
		}
		highest = max(highest, last)
	}
	return highest
}

// stampAlone sorts the samples key wrote with timestamps into time order, then appends
// its body writes with timestamps after them. It returns the last timestamp it
// synthesized, or zero when key made no body writes.
func (cs *ProgramState) stampAlone(
	key, idx uint32,
	now telem.TimeStamp,
) telem.TimeStamp {
	data, times := cs.writes[key], cs.writes[idx]
	sortByTime(&data, &times)
	body := cs.unstamped[key]
	var last telem.TimeStamp
	if n := int(body.Len()); n > 0 {
		start := max(now, cs.lastStamps[idx]+1)
		if times.Len() > 0 {
			start = max(start, times.ValueAt[telem.TimeStamp](-1)+1)
		} else {
			times = telem.Series{
				DataType:  telem.TimestampT,
				Data:      times.Data[:0],
				Alignment: body.Alignment,
				TimeRange: body.TimeRange,
			}
			cs.activeWriteKeys = append(cs.activeWriteKeys, idx)
		}
		times.Data = slices.Grow(times.Data, n*int(telem.TimestampT.Density()))
		for i := range n {
			times.Data = telem.ByteOrder.AppendUint64(
				times.Data,
				uint64(start)+uint64(i),
			)
		}
		last = start + telem.TimeStamp(n-1)
		data = cs.appendBody(data, key)
	}
	if times.Len() > 0 {
		cs.lastStamps[idx] = times.ValueAt[telem.TimeStamp](-1)
	}
	cs.writes[key], cs.writes[idx] = data, times
	return last
}

// stampGroup appends key's body writes to its timestamped samples. When the shared
// index has no timestamps yet, it stamps one per sample of key, starting at now, and
// returns the last one. Otherwise it returns zero.
func (cs *ProgramState) stampGroup(
	key, idx uint32,
	now telem.TimeStamp,
) telem.TimeStamp {
	data := cs.appendBody(cs.writes[key], key)
	cs.writes[key] = data
	if len(cs.writes[idx].Data) > 0 {
		return 0
	}
	count := data.Len()
	if count == 0 {
		return 0
	}
	stamps := telem.Arrange(now, int(count), 1*telem.NanosecondTS)
	stamps.Alignment = data.Alignment
	stamps.TimeRange = data.TimeRange
	cs.writes[idx] = stamps
	cs.activeWriteKeys = append(cs.activeWriteKeys, idx)
	last := stamps.ValueAt[telem.TimeStamp](-1)
	cs.lastStamps[idx] = last
	return last
}

// appendBody moves key's body writes to the end of data and returns the result.
func (cs *ProgramState) appendBody(data telem.Series, key uint32) telem.Series {
	body := cs.unstamped[key]
	if len(body.Data) == 0 {
		return data
	}
	if data.DataType == telem.UnknownT {
		data.DataType = body.DataType
		data.Alignment = body.Alignment
		data.TimeRange = body.TimeRange
	}
	data.Data = append(data.Data, body.Data...)
	body.Data = body.Data[:0]
	cs.unstamped[key] = body
	return data
}

// sortByTime reorders the samples of data and times together so that times does not
// decrease. It keeps the order of samples with equal timestamps.
func sortByTime(data, times *telem.Series) {
	stamps := unsafe.CastSlice[byte, telem.TimeStamp](times.Data)
	if slices.IsSorted(stamps) {
		return
	}
	order := make([]int, len(stamps))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(stamps[a], stamps[b])
	})
	sortedStamps := make([]byte, 0, len(times.Data))
	sortedData := make([]byte, 0, len(data.Data))
	for _, i := range order {
		sortedStamps = telem.ByteOrder.AppendUint64(sortedStamps, uint64(stamps[i]))
		if data.DataType.IsVariable() {
			sortedData = append(sortedData, telem.MarshalVariableSample(data.At(i))...)
		} else {
			sortedData = append(sortedData, data.At(i)...)
		}
	}
	times.Data, data.Data = sortedStamps, sortedData
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
	cs.appendWriteSeries(cs.bodyWrites(key), key, value)
}

// bodyWrites returns the buffer that body writes to key go into. Writes to an indexed
// channel wait in unstamped until Flush gives them timestamps.
func (cs *ProgramState) bodyWrites(key uint32) map[uint32]telem.Series {
	if idx := cs.indexes[key]; idx != 0 && idx != key {
		return cs.unstamped
	}
	return cs.writes
}

// activate records key as written this cycle, unless it already is.
func (cs *ProgramState) activate(key uint32) {
	if len(cs.writes[key].Data) == 0 && len(cs.unstamped[key].Data) == 0 {
		cs.activeWriteKeys = append(cs.activeWriteKeys, key)
	}
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
	buf := cs.bodyWrites(key)
	acc, exists := buf[key]
	if len(acc.Data) == 0 {
		cs.activate(key)
	}
	if !exists {
		acc = telem.Series{DataType: dt}
	}
	if acc.DataType == telem.UnknownT {
		acc.DataType = dt
	}
	if acc.DataType != dt && len(acc.Data) > 0 {
		buf[key] = telem.NewSeriesV(value)
		return
	}
	den := int(dt.Density())
	sampleStart := len(acc.Data)
	acc.Data = slices.Grow(acc.Data, den)
	acc.Data = acc.Data[:sampleStart+den]
	unsafe.CastSlice[byte, T](acc.Data)[sampleStart/den] = value
	buf[key] = acc
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
	cs.appendWriteSeries(cs.writes, key, data)
	idx := cs.indexes[key]
	if idx != 0 {
		cs.appendWriteSeries(cs.writes, idx, time)
	}
}

func (cs *ProgramState) appendWriteSeries(
	buf map[uint32]telem.Series,
	key uint32,
	source telem.Series,
) {
	acc, exists := buf[key]
	if len(acc.Data) == 0 {
		cs.activate(key)
	}
	if !exists {
		acc = telem.Series{DataType: source.DataType}
	}
	if acc.DataType == telem.UnknownT {
		acc.DataType = source.DataType
	}
	if len(source.Data) == 0 {
		buf[key] = acc
		return
	}
	if acc.DataType != source.DataType && len(acc.Data) > 0 {
		acc = source.DeepCopy()
		buf[key] = acc
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
	buf[key] = acc
}
