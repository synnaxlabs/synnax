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
	// writes holds this cycle's samples, except body writes to indexed channels.
	writes map[uint32]telem.Series
	// unstamped holds body writes to indexed channels until Flush stamps them.
	unstamped map[uint32]telem.Series
	// positions records where body writes fall among each channel's timestamped writes.
	positions       map[uint32][]position
	activeWriteKeys []uint32
	indexes         map[uint32]uint32
	// lastStamps holds the last timestamp flushed to each index.
	lastStamps map[uint32]telem.TimeStamp
	// writers counts the channels that wrote to each index this cycle. Flush reuses it.
	writers map[uint32]int
	// newest holds the newest stamp ingested on each index since the last Flush.
	newest map[uint32]telem.TimeStamp
	// offsets holds the clock offset learned for each index.
	offsets map[uint32]clockOffset
}

// offsetWindow is the width of the windows clock offsets are kept in. An observed
// offset counts until the end of the window after the one it was observed in.
const offsetWindow = 5 * telem.Second

// offsetBucket holds the minimum offset observed in one window.
type offsetBucket struct {
	window int64
	min    telem.TimeSpan
	set    bool
}

// clockOffset estimates how far the local clock runs ahead of the stamps on an index,
// delivery delay included: the minimum offset observed in the current and previous
// window. Older offsets age out, so clock drift and jumps do not persist.
type clockOffset [2]offsetBucket

func (o *clockOffset) observe(now telem.TimeStamp, offset telem.TimeSpan) {
	w := int64(now) / int64(offsetWindow)
	b := &o[w&1]
	if !b.set || b.window != w {
		*b = offsetBucket{window: w, min: offset, set: true}
		return
	}
	b.min = min(b.min, offset)
}

// at returns the estimate at local time now, and false when no offset is recent
// enough.
func (o clockOffset) at(now telem.TimeStamp) (offset telem.TimeSpan, ok bool) {
	w := int64(now) / int64(offsetWindow)
	for _, b := range o {
		if b.set && b.window >= w-1 && (!ok || b.min < offset) {
			offset, ok = b.min, true
		}
	}
	return offset, ok
}

// NewProgramState creates a new ProgramState from channel digests.
func NewProgramState(digests []Digest) *ProgramState {
	ps := &ProgramState{
		reads:      make(map[uint32]telem.MultiSeries),
		writes:     make(map[uint32]telem.Series),
		unstamped:  make(map[uint32]telem.Series),
		positions:  make(map[uint32][]position),
		indexes:    make(map[uint32]uint32),
		lastStamps: make(map[uint32]telem.TimeStamp),
		writers:    make(map[uint32]int),
		newest:     make(map[uint32]telem.TimeStamp),
		offsets:    make(map[uint32]clockOffset),
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
		s := fr.RawSeriesAt(rawI)
		ps.reads[key] = ps.reads[key].Append(s)
		if s.DataType == telem.TimestampT && s.Len() > 0 {
			ps.newest[key] = max(ps.newest[key], s.ValueAt[telem.TimeStamp](-1))
		}
	}
}

// learnOffsets folds the stamps ingested since the last call into each index's clock
// offset.
func (ps *ProgramState) learnOffsets(now telem.TimeStamp) {
	for idx, stamp := range ps.newest {
		o := ps.offsets[idx]
		o.observe(now, telem.TimeSpan(now-stamp))
		ps.offsets[idx] = o
	}
	clear(ps.newest)
}

// entryStamp returns the stamp on the index of key that matches local time now, or
// zero when the index has no recent clock offset.
func (ps *ProgramState) entryStamp(key uint32, now telem.TimeStamp) telem.TimeStamp {
	offset, ok := ps.offsets[ps.indexes[key]].at(now)
	if !ok {
		return 0
	}
	return now.Add(-offset)
}

// Flush extracts buffered channel writes into a frame and clears the write buffer. Only
// channels written in the current cycle are flushed. A channel alone on its index gets
// its samples in time order. Its body writes take the later of now and 1ns after the
// last timestamp of the index. Samples with equal timestamps keep the order the program
// wrote them in, and each moves 1ns after the one before it. Channels that share an
// index are stamped from now, and only when none of them supplied timestamps. Flush
// returns the highest stamp it synthesized, so the caller's clock can resume above it,
// and reports whether anything was flushed. The highest stamp is zero when nothing was
// synthesized. Flush also learns each index's clock offset from the stamps ingested
// since the last Flush, with now as their arrival time.
func (ps *ProgramState) Flush(
	fr telem.Frame[uint32],
	now telem.TimeStamp,
) (telem.Frame[uint32], telem.TimeStamp, bool) {
	ps.learnOffsets(now)
	if len(ps.activeWriteKeys) == 0 {
		return fr, 0, false
	}
	highest := ps.stampIndexes(now)
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
	return fr, highest, flushed
}

// stampIndexes gives every sample written this cycle a timestamp and returns the
// highest timestamp it synthesized.
func (ps *ProgramState) stampIndexes(now telem.TimeStamp) (highest telem.TimeStamp) {
	clear(ps.writers)
	for _, key := range ps.activeWriteKeys {
		if idx := ps.indexes[key]; idx != 0 && idx != key {
			ps.writers[idx]++
		}
	}
	for _, key := range ps.activeWriteKeys {
		idx := ps.indexes[key]
		if idx == 0 || idx == key {
			continue
		}
		var last telem.TimeStamp
		if ps.writers[idx] == 1 {
			last = ps.stampAlone(key, idx, now)
		} else {
			last = ps.stampGroup(key, idx, now)
		}
		highest = max(highest, last)
	}
	return highest
}

// stampAlone gives the body writes of key a timestamp and puts all of its samples in
// time order. It returns the last timestamp it synthesized, or zero when it synthesized
// none.
func (ps *ProgramState) stampAlone(
	key, idx uint32,
	now telem.TimeStamp,
) telem.TimeStamp {
	data, times := ps.writes[key], ps.writes[idx]
	var (
		order   []int
		highest telem.TimeStamp
	)
	if body := ps.unstamped[key]; len(body.Data) > 0 {
		if len(times.Data) == 0 {
			times = telem.Series{
				DataType:  telem.TimestampT,
				Data:      times.Data[:0],
				Alignment: body.Alignment,
				TimeRange: body.TimeRange,
			}
			ps.activeWriteKeys = append(ps.activeWriteKeys, idx)
		}
		stamped, n := int(times.Len()), int(body.Len())
		highest = max(now, ps.lastStamps[idx]+1)
		times.Data = slices.Grow(times.Data, n*int(telem.TimestampT.Density()))
		for range n {
			times.Data = telem.ByteOrder.AppendUint64(times.Data, uint64(highest))
		}
		data = ps.appendBody(data, key)
		if pos := ps.positions[key]; len(pos) > 0 {
			order = writeOrder(stamped, n, pos)
			ps.positions[key] = pos[:0]
		}
	}
	sortByTime(&data, &times, order)
	highest = max(highest, breakTies(times))
	if times.Len() > 0 {
		ps.lastStamps[idx] = times.ValueAt[telem.TimeStamp](-1)
	}
	ps.writes[key], ps.writes[idx] = data, times
	return highest
}

// stampGroup appends key's body writes to its timestamped samples. When the shared
// index has no timestamps yet, it stamps one per sample of key, starting at now, and
// returns the last one. Otherwise it returns zero.
func (ps *ProgramState) stampGroup(
	key, idx uint32,
	now telem.TimeStamp,
) telem.TimeStamp {
	data := ps.appendBody(ps.writes[key], key)
	ps.writes[key] = data
	ps.positions[key] = ps.positions[key][:0]
	if len(ps.writes[idx].Data) > 0 {
		return 0
	}
	count := data.Len()
	if count == 0 {
		return 0
	}
	stamps := telem.Arrange(now, int(count), 1*telem.NanosecondTS)
	stamps.Alignment = data.Alignment
	stamps.TimeRange = data.TimeRange
	ps.writes[idx] = stamps
	ps.activeWriteKeys = append(ps.activeWriteKeys, idx)
	last := stamps.ValueAt[telem.TimeStamp](-1)
	ps.lastStamps[idx] = last
	return last
}

// appendBody moves key's body writes to the end of data and returns the result.
func (ps *ProgramState) appendBody(data telem.Series, key uint32) telem.Series {
	body := ps.unstamped[key]
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
	ps.unstamped[key] = body
	return data
}

// position holds the sample counts of a channel's body writes and timestamped writes
// when a timestamped write arrives after a body write.
type position struct{ body, stamped int }

// writeOrder returns the order the program wrote a channel's samples in. Indexes below
// stamped are timestamped samples, and the rest are body writes.
func writeOrder(stamped, body int, positions []position) []int {
	order := make([]int, 0, stamped+body)
	var s, b int
	take := func(p position) {
		for ; s < p.stamped; s++ {
			order = append(order, s)
		}
		for ; b < p.body; b++ {
			order = append(order, stamped+b)
		}
	}
	for _, p := range positions {
		take(p)
	}
	take(position{body: body, stamped: stamped})
	return order
}

// sortByTime reorders the samples of data and times together so that times does not
// decrease. Samples with equal timestamps keep their place in order, or their place in
// times when order is nil.
func sortByTime(data, times *telem.Series, order []int) {
	stamps := unsafe.CastSlice[byte, telem.TimeStamp](times.Data)
	if order == nil {
		if slices.IsSorted(stamps) {
			return
		}
		order = make([]int, len(stamps))
		for i := range order {
			order[i] = i
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(stamps[a], stamps[b])
	})
	if slices.IsSorted(order) {
		return
	}
	sortedStamps := make([]byte, 0, len(times.Data))
	for _, i := range order {
		sortedStamps = telem.ByteOrder.AppendUint64(sortedStamps, uint64(stamps[i]))
	}
	sortedData := make([]byte, 0, len(data.Data))
	if data.DataType.IsVariable() {
		samples := slices.AppendSeq(make([][]byte, 0, len(order)), data.Samples())
		for _, i := range order {
			sortedData = telem.ByteOrder.AppendUint32(
				sortedData,
				uint32(len(samples[i])),
			)
			sortedData = append(sortedData, samples[i]...)
		}
	} else {
		den := int(data.DataType.Density())
		for _, i := range order {
			sortedData = append(sortedData, data.Data[i*den:(i+1)*den]...)
		}
	}
	times.Data, data.Data = sortedStamps, sortedData
}

// breakTies moves each timestamp that does not exceed the one before it to 1ns after
// that one. It returns the last timestamp it moved, or zero when it moved none.
func breakTies(times telem.Series) (last telem.TimeStamp) {
	stamps := unsafe.CastSlice[byte, telem.TimeStamp](times.Data)
	for i := 1; i < len(stamps); i++ {
		if stamps[i] <= stamps[i-1] {
			stamps[i] = stamps[i-1] + 1
			last = stamps[i]
		}
	}
	return last
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

// ReadValue reads a single value from a channel (for WASM runtime bindings).
func (ps *ProgramState) ReadValue(key uint32) (telem.Series, bool) {
	ms, ok := ps.reads[key]
	if !ok || len(ms.Series) == 0 {
		return telem.Series{}, false
	}
	return ms.Series[len(ms.Series)-1], ok
}

// writeValue writes a single value to a channel (for WASM runtime bindings).
func (ps *ProgramState) writeValue(key uint32, value telem.Series) {
	ps.appendWriteSeries(ps.bodyWrites(key), key, value)
}

// bodyWrites returns the buffer that body writes to key go into. Writes to an indexed
// channel wait in unstamped until Flush gives them timestamps.
func (ps *ProgramState) bodyWrites(key uint32) map[uint32]telem.Series {
	if idx := ps.indexes[key]; idx != 0 && idx != key {
		return ps.unstamped
	}
	return ps.writes
}

// activate records key as written this cycle, unless it already is.
func (ps *ProgramState) activate(key uint32) {
	if len(ps.writes[key].Data) == 0 && len(ps.unstamped[key].Data) == 0 {
		ps.activeWriteKeys = append(ps.activeWriteKeys, key)
	}
}

func appendFixedWriteSample[T telem.FixedSample](
	ps *ProgramState,
	key uint32,
	value T,
) {
	dt := telem.InferDataType[T]()
	buf := ps.bodyWrites(key)
	acc, exists := buf[key]
	if len(acc.Data) == 0 {
		ps.activate(key)
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
	idx := ps.indexes[key]
	if idx != 0 {
		ps.markPosition(key, idx)
	}
	ps.appendWriteSeries(ps.writes, key, data)
	if idx != 0 {
		ps.appendWriteSeries(ps.writes, idx, time)
	}
}

// markPosition records how many body writes to key came before a timestamped write.
func (ps *ProgramState) markPosition(key, idx uint32) {
	body := int(ps.unstamped[key].Len())
	if body == 0 {
		return
	}
	pos := ps.positions[key]
	if len(pos) > 0 && pos[len(pos)-1].body == body {
		return
	}
	ps.positions[key] = append(pos, position{
		body:    body,
		stamped: int(ps.writes[idx].Len()),
	})
}

func (ps *ProgramState) appendWriteSeries(
	buf map[uint32]telem.Series,
	key uint32,
	source telem.Series,
) {
	acc, exists := buf[key]
	if len(acc.Data) == 0 {
		ps.activate(key)
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
