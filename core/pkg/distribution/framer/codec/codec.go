// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/bit"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const bitsPerByte = 8

// bitPackedByteCount returns the number of wire bytes needed to bit-pack nSamples
// boolean samples (one bit each, LSB-first).
func bitPackedByteCount[T int | int64 | uint32](nSamples T) T {
	return (nSamples + bitsPerByte - 1) / bitsPerByte
}

// wireSize returns the number of bytes a series occupies on the wire after per-type
// encoding. BooleanT samples are bit-packed at one bit each; variable-length types
// carry their length-prefixed in-memory representation directly; other fixed types use
// their in-memory byte count.
func wireSize(s telem.Series) int {
	if s.DataType == telem.BooleanT {
		return int(bitPackedByteCount(s.Len()))
	}
	return int(s.Size())
}

// packBoolBits packs byte-packed canonical bool data (one sample per byte) into
// bit-packed wire bytes (LSB-first within each byte).
func packBoolBits(src []byte) []byte {
	dst := make([]byte, bitPackedByteCount(len(src)))
	for i, b := range src {
		if b != 0 {
			dst[i/bitsPerByte] |= 1 << (i % bitsPerByte)
		}
	}
	return dst
}

// unpackBoolBits unpacks bit-packed wire bytes into byte-packed canonical bool
// data. n is the number of samples (not bytes).
func unpackBoolBits(src []byte, n int) []byte {
	dst := make([]byte, n)
	for i := range n {
		dst[i] = (src[i/bitsPerByte] >> (i % bitsPerByte)) & 1
	}
	return dst
}

func writeTimeRange(w *binary.Writer, tr telem.TimeRange) {
	w.Uint64(uint64(tr.Start))
	w.Uint64(uint64(tr.End))
}

type sorter struct {
	rawIndices []int
	keys       []channel.Key
	// positions holds the position of each key in the requested keys of the state, or
	// -1 for a declared clock.
	positions []int32
	offset    int
}

func (s *sorter) reset(size int) {
	s.offset = 0
	if cap(s.rawIndices) < size {
		s.rawIndices = make([]int, size)
		s.keys = make([]channel.Key, size)
		s.positions = make([]int32, size)
	}
}

func (s *sorter) insert(key channel.Key, rawIndex int, position int32) {
	s.keys[s.offset] = key
	s.rawIndices[s.offset] = rawIndex
	s.positions[s.offset] = position
	s.offset++
}

func (s *sorter) sort() { sort.Sort(s) }

// Len implements sort.Interface.
func (s *sorter) Len() int { return s.offset }

// Less implements sort.Interface. It sorts by key, then by raw index, so the series of
// one key keep the order the producer gave them. An alignment is never an order in
// time.
func (s *sorter) Less(i, j int) bool {
	if s.keys[i] != s.keys[j] {
		return s.keys[i] < s.keys[j]
	}
	return s.rawIndices[i] < s.rawIndices[j]
}

// Swap implements sort.Interface.
func (s *sorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.rawIndices[i], s.rawIndices[j] = s.rawIndices[j], s.rawIndices[i]
	s.positions[i], s.positions[j] = s.positions[j], s.positions[i]
}

// Codec is a high-performance encoder/decoder specifically designed for moving
// telemetry frames over the network. Codec is stateful, meaning that both the encoding
// and decoding sides must agree on the set of channels and their order before any
// encoding or decoding can occur.
type Codec struct {
	// buf is reused for each encode operation.
	buf *binary.Writer
	// reader is reused for each decode operation. Unlike the standard library
	// binary.Read, this avoids reflection overhead.
	reader *binary.Reader
	// tail receives the byte that readEnd reads past the last series.
	tail [1]byte
	// channelResolver is used in dynamic codecs to look up the data types of channels
	// when Update is called. It is nil for codecs created with NewStatic.
	channelResolver ChannelResolver
	// mergedSeriesResult is a reusable slice for storing merged series info, avoiding
	// allocations on each encode operation
	mergedSeriesResult []mergedSeriesInfo
	// mu is non-routine safe structures that must be used carefully.
	mu struct {
		// states is the current backlog of encoding states. We keep multiple states to
		// allow for temporary de-sync between the encoding and decoding sides. For
		// example, when updating the keys of a streamer, the receiving codec may get
		// the updated set of channel in its state before the sending codec may get
		// updated, which means that the receiving codec needs to decode according to
		// the previous state. seqNum and the states backlog are used to keep the two in
		// sync.
		states map[uint32]*state
		// oldest is the lowest sequence number that states can hold.
		oldest uint32
		// pending holds the states added by Update that Encode and Decode have not
		// applied yet, oldest first. pendingMu guards it.
		pending   []*state
		pendingMu sync.Mutex
		// seqNum corresponds to the most recent update in states. This is incremented
		// and communicated each time a state is added.
		seqNum uint32
		// updateAvailable is an atomic flag indicating that pending holds a state for
		// the encoding/decoding routine to apply. Checking it lets each encode and
		// decode skip pendingMu when no update is pending.
		updateAvailable atomic.Bool
	}
	// encodeSorter is used to sort source frames that are being encoded. Used instead
	// of sorting the frame directly in order to avoid excess heap allocations
	encodeSorter sorter
	// decodeLimit is the most bytes that one decoded frame can hold, or zero for no
	// limit.
	decodeLimit int64
	// codes, kinds, and clocks hold the type codes, the override kinds, and the agreed
	// clocks of the frame being encoded. See plan.
	codes  []byte
	kinds  []byte
	clocks []uint32
	// declared and all hold the declared clocks and the key list of the frame being
	// encoded.
	declared channel.Keys
	all      channel.Keys
}

// Option configures a Codec.
type Option func(*Codec)

// WithDecodeLimit makes Decode and DecodeStream reject a frame whose series hold more
// than limit bytes once decoded. A boolean sample counts one byte, and each series also
// counts SeriesOverhead. The codec checks each series before it allocates the series.
func WithDecodeLimit(limit int64) Option {
	return func(c *Codec) { c.decodeLimit = limit }
}

// SeriesOverhead is the bytes that each decoded series counts against the decode limit
// in addition to its samples.
const SeriesOverhead = int64(
	unsafe.Sizeof(telem.Series{}) + unsafe.Sizeof(channel.Key(0)),
)

var byteOrder = telem.ByteOrder

// NewStatic creates a new codec that uses the given channel keys and data types as
// its encoding state. It is not safe to call Update on a codec instantiated using
// NewStatic.
func NewStatic(
	channelKeys channel.Keys,
	dataTypes []telem.DataType,
	opts ...Option,
) *Codec {
	if len(dataTypes) != len(channelKeys) {
		panic("data types and channel keys must be the same length")
	}
	c := newCodec(opts)
	c.update(newState(channelKeys, dataTypes))
	return c
}

// ChannelResolver resolves channel metadata (data types and names) by key. It is
// supplied to NewDynamic so the codec can look up channel information when Update is
// called. The channel service implementations satisfy this interface, allowing a
// dynamic codec to resolve channel metadata through them without depending on the
// service layer.
type ChannelResolver interface {
	// RetrieveDataTypes returns the data types of the channels with the given keys in
	// the same order as keys. The returned slice must contain exactly one entry per
	// key; Codec.Update returns an error if the lengths differ.
	RetrieveDataTypes(ctx context.Context, keys channel.Keys) ([]telem.DataType, error)
	// RetrieveName returns the name of the channel with the given key, or an empty
	// string if no channel with the key exists.
	RetrieveName(ctx context.Context, key channel.Key) string
}

// NewDynamic creates a new codec that can be dynamically updated by resolving channel
// data types through the provided ChannelResolver. Codec.Update must be called before
// the first call to Codec.Encode and Codec.Decode.
func NewDynamic(channelResolver ChannelResolver, opts ...Option) *Codec {
	c := newCodec(opts)
	c.channelResolver = channelResolver
	return c
}

func newCodec(opts []Option) *Codec {
	c := &Codec{
		buf:    binary.NewWriter(0, byteOrder),
		reader: binary.NewReader(nil, byteOrder),
	}
	for _, opt := range opts {
		opt(c)
	}
	c.mu.updateAvailable.Store(false)
	c.mu.states = make(map[uint32]*state)
	return c
}

// Update updates the codec to use the given keys in its state, resolving their data
// types through the codec's ChannelResolver. It returns an error if the channel
// resolver does not return exactly one data type per key.
func (c *Codec) Update(ctx context.Context, keys []channel.Key) error {
	dataTypes, err := c.channelResolver.RetrieveDataTypes(ctx, keys)
	if err != nil {
		return err
	}
	if len(dataTypes) != len(keys) {
		return errors.Wrapf(
			query.ErrNotFound,
			"channel resolver returned %d data types for %d channel keys",
			len(dataTypes),
			len(keys),
		)
	}
	c.update(newState(keys, dataTypes))
	return nil
}

// Initialized returns true if the codec was initialized using NewStatic or Update
// has been called at least once when using NewDynamic.
func (c *Codec) Initialized() bool {
	return c.mu.seqNum > 0 || c.mu.updateAvailable.Load()
}

func (c *Codec) update(s *state) {
	c.mu.pendingMu.Lock()
	c.mu.pending = append(c.mu.pending, s)
	c.mu.pendingMu.Unlock()
	// Set after the append, so the routine that clears the flag also finds s.
	c.mu.updateAvailable.Store(true)
}

func (c *Codec) processUpdates() {
	if !c.mu.updateAvailable.CompareAndSwap(true, false) {
		return
	}
	c.mu.pendingMu.Lock()
	pending := c.mu.pending
	c.mu.pending = nil
	c.mu.pendingMu.Unlock()
	for _, s := range pending {
		c.mu.seqNum++
		c.mu.states[c.mu.seqNum] = s
	}
}

type flags struct {
	// equalsLens is true when all series in the frame have the same number of samples.
	// This lets us consolidate the length of all series into one number.
	equalLens bool
	// equalTimeRanges is true when all series in the frame have the same time range.
	// This lets use consolidate all time ranges into one, 16-byte section.
	equalTimeRanges bool
	// timeRangesZero is true when the start and end timestamps are all zero.
	// This lets us omit time ranges entirely.
	timeRangesZero bool
	// allChannelsPresent is true if all channels for the codec are also present in the
	// frame. This lets us omit the channel mapping from the frame.
	allChannelsPresent bool
	// equalAlignments is true if the alignments for all series are equal.
	equalAlignments bool
	// zeroAlignments is true if all alignments are zero
	zeroAlignments bool
	// multiplesPresent is true when each series carries its alignment multiple after
	// its alignment. It is set only when a series has a multiple above one.
	multiplesPresent bool
	// extended is true when the extension flags follow the sequence number.
	extended bool
}

const (
	extendedFlagPos           bit.FlagPos = 7
	multiplesPresentFlagPos   bit.FlagPos = 6
	zeroAlignmentsFlagPos     bit.FlagPos = 5
	equalAlignmentsFlagPos    bit.FlagPos = 4
	equalLengthsFlagPos       bit.FlagPos = 3
	equalTimeRangesFlagPos    bit.FlagPos = 2
	timeRangesZeroFlagPos     bit.FlagPos = 1
	allChannelsPresentFlagPos bit.FlagPos = 0
)

func (f flags) encode() byte {
	b := byte(0)
	b = equalLengthsFlagPos.Set(b, f.equalLens)
	b = equalTimeRangesFlagPos.Set(b, f.equalTimeRanges)
	b = timeRangesZeroFlagPos.Set(b, f.timeRangesZero)
	b = allChannelsPresentFlagPos.Set(b, f.allChannelsPresent)
	b = equalAlignmentsFlagPos.Set(b, f.equalAlignments)
	b = zeroAlignmentsFlagPos.Set(b, f.zeroAlignments)
	b = multiplesPresentFlagPos.Set(b, f.multiplesPresent)
	b = extendedFlagPos.Set(b, f.extended)
	return b
}

func decodeFlags(b byte) flags {
	f := newFlags()
	f.equalLens = equalLengthsFlagPos.Get(b)
	f.equalTimeRanges = equalTimeRangesFlagPos.Get(b)
	f.timeRangesZero = timeRangesZeroFlagPos.Get(b)
	f.allChannelsPresent = allChannelsPresentFlagPos.Get(b)
	f.equalAlignments = equalAlignmentsFlagPos.Get(b)
	f.zeroAlignments = zeroAlignmentsFlagPos.Get(b)
	f.multiplesPresent = multiplesPresentFlagPos.Get(b)
	f.extended = extendedFlagPos.Get(b)
	return f
}

func newFlags() flags {
	return flags{
		equalLens:          true,
		equalTimeRanges:    true,
		timeRangesZero:     true,
		allChannelsPresent: true,
		equalAlignments:    true,
		zeroAlignments:     true,
	}
}

// Encode encodes the given frame into bytes. The returned byte slice is a copy
// and safe to retain after subsequent Encode calls.
func (c *Codec) Encode(ctx context.Context, src framer.Frame) ([]byte, error) {
	err := c.encodeInternal(ctx, src)
	if err != nil {
		return nil, err
	}
	// Return a copy of the internal buffer to avoid aliasing issues
	result := make([]byte, len(c.buf.Bytes()))
	copy(result, c.buf.Bytes())
	return result, nil
}

// prune drops the states older than seqNum. A remote encoder never goes back to an
// older state, and an encoder only uses its newest one.
func (c *Codec) prune(seqNum uint32) {
	for ; c.mu.oldest < seqNum; c.mu.oldest++ {
		delete(c.mu.states, c.mu.oldest)
	}
}

func (c *Codec) panicIfNotUpdated(opName string) {
	if c.mu.seqNum < 1 {
		panic(
			"dynamic codec was not updated for first call to " + opName,
		)
	}
}

// mergeable returns true when s2 can join s1 in one series: both have the same
// alignment multiple, clock, and data type, and s2 starts at the alignment where s1
// ends.
func mergeable(s1, s2 telem.Series) bool {
	return s1.Multiple() == s2.Multiple() &&
		s1.AlignmentBounds().Upper == s2.AlignmentBounds().Lower &&
		s1.Clock == s2.Clock &&
		s1.DataType == s2.DataType
}

// mergedSeriesInfo holds information about a series that may be merged or original.
type mergedSeriesInfo struct {
	series telem.Series
	key    channel.Key
	// index is the position of key in the requested keys of the state, or -1 for a
	// declared clock.
	index int32
}

// mergeContiguousSeries processes sorted series and merges the adjacent series of each
// key that are mergeable. Returns the merged series info using the codec's reusable
// result slice. NOTE: The returned slice is reused across calls, so callers must not
// hold references to it beyond the current encode operation.
func (c *Codec) mergeContiguousSeries(
	sortedKeys []channel.Key,
	sortedIndices []int,
	src framer.Frame,
	count int,
) []mergedSeriesInfo {
	// Reuse the result slice, growing capacity only if needed
	if cap(c.mergedSeriesResult) < count {
		c.mergedSeriesResult = make([]mergedSeriesInfo, 0, count)
	}
	c.mergedSeriesResult = c.mergedSeriesResult[:0]

	if count == 0 {
		return c.mergedSeriesResult
	}

	// Slice arrays to actual size to avoid reading old data
	sortedKeys = sortedKeys[:count]
	sortedIndices = sortedIndices[:count]

	i := 0

	for i < count {
		// Find the run of series with the same key
		key := sortedKeys[i]
		groupEnd := i + 1

		// Find all series with the same key
		for groupEnd < count && sortedKeys[groupEnd] == key {
			groupEnd++
		}

		// Process contiguous sub-groups within this key group
		for runStart := i; runStart < groupEnd; {
			runEnd := contiguousRunEnd(sortedIndices, src, runStart, groupEnd)
			c.mergedSeriesResult = append(c.mergedSeriesResult, mergedSeriesInfo{
				series: mergeRun(sortedIndices[runStart:runEnd], src),
				key:    key,
				index:  c.encodeSorter.positions[i],
			})
			runStart = runEnd
		}

		i = groupEnd
	}

	return c.mergedSeriesResult
}

// contiguousRunEnd returns the exclusive end of the run that starts at runStart, where
// each series is mergeable with the one before it.
func contiguousRunEnd(
	sortedIndices []int,
	src framer.Frame,
	runStart, groupEnd int,
) int {
	runEnd := runStart + 1
	for runEnd < groupEnd {
		prevSeries := src.RawSeriesAt(sortedIndices[runEnd-1])
		currSeries := src.RawSeriesAt(sortedIndices[runEnd])
		if !mergeable(prevSeries, currSeries) {
			break
		}
		runEnd++
	}
	return runEnd
}

// mergeRun merges the series at the given raw indices into one series. A run of one
// series returns that series unchanged.
func mergeRun(rawIndices []int, src framer.Frame) telem.Series {
	if len(rawIndices) == 1 {
		return src.RawSeriesAt(rawIndices[0])
	}
	totalSize := 0
	for _, rawIdx := range rawIndices {
		totalSize += len(src.RawSeriesAt(rawIdx).Data)
	}
	var (
		mergedData   = make([]byte, totalSize)
		offset       = 0
		mergedSeries telem.Series
	)
	for k, rawIdx := range rawIndices {
		s := src.RawSeriesAt(rawIdx)
		if k == 0 {
			mergedSeries = telem.Series{
				DataType:          s.DataType,
				Alignment:         s.Alignment,
				AlignmentMultiple: s.AlignmentMultiple,
				TimeRange:         s.TimeRange,
				Clock:             s.Clock,
			}
		} else {
			if s.TimeRange.End > mergedSeries.TimeRange.End {
				mergedSeries.TimeRange.End = s.TimeRange.End
			}
			if s.TimeRange.Start < mergedSeries.TimeRange.Start {
				mergedSeries.TimeRange.Start = s.TimeRange.Start
			}
		}
		copy(mergedData[offset:], s.Data)
		offset += len(s.Data)
	}
	mergedSeries.Data = mergedData
	return mergedSeries
}

const (
	flagsSize  = 1
	seqNumSize = 4
)

// EncodeStream encodes the given frame into the provided io writer, returning any
// encoding errors encountered.
func (c *Codec) EncodeStream(ctx context.Context, w io.Writer, src framer.Frame) error {
	if err := c.encodeInternal(ctx, src); err != nil {
		return err
	}
	_, err := w.Write(c.buf.Bytes())
	return err
}

func (c *Codec) retrieveName(ctx context.Context, key channel.Key) string {
	if c.channelResolver != nil {
		if name := c.channelResolver.RetrieveName(ctx, key); name != "" {
			return name
		}
	}
	return key.String()
}

// encodeInternal encodes the frame into c.buf. After calling this method,
// c.buf.Bytes() contains the encoded data.
func (c *Codec) encodeInternal(ctx context.Context, src framer.Frame) error {
	c.encodeSorter.reset(src.Count())
	c.processUpdates()
	c.panicIfNotUpdated("Encode")
	c.prune(c.mu.seqNum)
	currState := c.mu.states[c.mu.seqNum]
	diff := c.sortSeries(src, currState)
	mergedSeries := c.mergeContiguousSeries(
		c.encodeSorter.keys,
		c.encodeSorter.rawIndices,
		src,
		c.encodeSorter.offset,
	)
	ext, err := c.plan(ctx, mergedSeries, currState.keys, &currState.enc, diff)
	if err != nil {
		return err
	}
	h := newEncodeHeader(
		mergedSeries,
		c.frameKeys(ext, currState.keys, &currState.enc),
		(currState.enc.variable || diff.types) && slices.ContainsFunc(
			mergedSeries,
			func(msi mergedSeriesInfo) bool { return msi.series.DataType.IsVariable() },
		),
	)
	h.fgs.extended = ext != 0
	c.buf.Resize(h.byteArraySize + c.planSize(ext))
	c.buf.Reset()
	c.buf.Uint8(h.fgs.encode())
	c.buf.Uint32(c.mu.seqNum)
	if h.fgs.extended {
		c.buf.Uint8(ext)
		c.buf.Write(c.codes)
	}
	if clocksDeclaredFlagPos.Get(ext) {
		c.buf.Uint32(uint32(len(c.declared)))
		for _, key := range c.declared {
			c.buf.Uint32(uint32(key))
		}
	}
	if h.fgs.equalLens {
		c.buf.Uint32(uint32(h.dataSize))
	}
	if h.fgs.equalTimeRanges && !h.fgs.timeRangesZero {
		writeTimeRange(c.buf, h.tr)
	}
	if h.fgs.equalAlignments && !h.fgs.zeroAlignments {
		c.buf.Uint64(uint64(h.alignment))
	}
	if overridesPresentFlagPos.Get(ext) {
		c.buf.Uint32(c.nextOverride(0))
		for i, msi := range mergedSeries {
			c.writeKey(h.fgs, msi.key)
			c.writeOverride(i, msi.series)
			c.writeSeries(h.fgs, msi.series)
		}
	} else {
		for _, msi := range mergedSeries {
			c.writeKey(h.fgs, msi.key)
			c.writeSeries(h.fgs, msi.series)
		}
	}
	c.commit(ext, &currState.enc)
	return nil
}

// sortSeries sorts the series in src that currState requests, and the time series under
// their own clocks, by key and raw index into c.encodeSorter, and returns what they
// change in the agreement. It drops the series of other keys.
func (c *Codec) sortSeries(src framer.Frame, currState *state) frameDiff {
	var (
		ag   = &currState.enc
		diff frameDiff
	)
	for rawI, s := range src.RawSeries() {
		if src.ShouldExcludeRaw(rawI) {
			continue
		}
		key := src.RawKeyAt(rawI)
		if position, ok := currState.index[key]; ok {
			c.encodeSorter.insert(key, rawI, int32(position))
			diff.clocks |= s.Clock ^ ag.clocks[position]
			diff.types = diff.types ||
				needsTypeOverride(ag.dataTypes[position], s.DataType)
		} else if isOwnClock(key, s) {
			c.encodeSorter.insert(key, rawI, -1)
			diff.declared = true
		}
	}
	c.encodeSorter.sort()
	return diff
}

// encodeHeader holds the flags, the shared reference values, and the total byte size
// of an encoded frame.
type encodeHeader struct {
	fgs           flags
	dataSize      int
	tr            telem.TimeRange
	alignment     telem.Alignment
	byteArraySize int
}

// newEncodeHeader returns the header of the merged series of a frame whose keys are
// keys. variable is true when one of the series has a variable data type.
func newEncodeHeader(
	mergedSeries []mergedSeriesInfo,
	keys channel.Keys,
	variable bool,
) encodeHeader {
	h := encodeHeader{
		dataSize:      -1,
		tr:            telem.TimeRangeZero,
		byteArraySize: flagsSize + seqNumSize,
		fgs:           newFlags(),
	}
	h.fgs.equalLens = !variable
	// allChannelsPresent is true only when the merged series form a 1-to-1
	// correspondence with keys (each key has exactly one series in the same order).
	// Multi-domain frames that produce multiple series for the same key
	// would otherwise be silently miscounted into a flag that tells the decoder to skip
	// per-series keys on the wire.
	h.fgs.allChannelsPresent = len(mergedSeries) == len(keys)
	if h.fgs.allChannelsPresent {
		for i, msi := range mergedSeries {
			if msi.key != keys[i] {
				h.fgs.allChannelsPresent = false
				break
			}
		}
	}
	if !h.fgs.allChannelsPresent {
		h.byteArraySize += len(mergedSeries) * 4
	}
	for _, msi := range mergedSeries {
		h.addSeries(msi.series)
	}
	h.fgs.multiplesPresent = slices.ContainsFunc(
		mergedSeries,
		func(msi mergedSeriesInfo) bool { return msi.series.Multiple() > 1 },
	)
	h.fgs.timeRangesZero = h.fgs.equalTimeRanges && h.tr.Start.IsZero() &&
		h.tr.End.IsZero()
	h.fgs.zeroAlignments = h.fgs.equalAlignments && h.alignment == 0
	h.byteArraySize += h.metadataSize(len(mergedSeries))
	return h
}

// addSeries adds the wire size of s and clears the flags that s breaks.
func (h *encodeHeader) addSeries(s telem.Series) {
	sLen := int(s.Len())
	h.byteArraySize += wireSize(s)
	if h.dataSize == -1 {
		h.dataSize = sLen
		h.tr = s.TimeRange
		h.alignment = s.Alignment
		return
	}
	if sLen != h.dataSize {
		h.fgs.equalLens = false
	}
	if s.TimeRange != h.tr {
		h.fgs.equalTimeRanges = false
	}
	if s.Alignment != h.alignment {
		h.fgs.equalAlignments = false
	}
}

// metadataSize returns the bytes of lengths, time ranges, alignments, and multiples
// that the flags require for count series.
func (h *encodeHeader) metadataSize(count int) int {
	size := 0
	if !h.fgs.equalLens {
		size += count * 4
	} else {
		size += 4
	}
	if !h.fgs.timeRangesZero {
		if !h.fgs.equalTimeRanges {
			size += count * 16
		} else {
			size += 16
		}
	}
	if !h.fgs.zeroAlignments {
		if !h.fgs.equalAlignments {
			size += count * 8
		} else {
			size += 8
		}
	}
	if h.fgs.multiplesPresent {
		size += count * 4
	}
	return size
}

// writeKey writes the key of a series when the frame does not hold every key.
func (c *Codec) writeKey(fgs flags, key channel.Key) {
	if !fgs.allChannelsPresent {
		c.buf.Uint32(uint32(key))
	}
}

// writeSeries writes the length, data, and metadata of s that the flags do not share.
func (c *Codec) writeSeries(fgs flags, s telem.Series) {
	if !fgs.equalLens {
		if s.DataType.IsVariable() {
			c.buf.Uint32(uint32(s.Size()))
		} else {
			c.buf.Uint32(uint32(s.Len()))
		}
	}
	if s.DataType == telem.BooleanT {
		c.buf.Write(packBoolBits(s.Data))
	} else {
		c.buf.Write(s.Data)
	}
	if !fgs.equalTimeRanges {
		writeTimeRange(c.buf, s.TimeRange)
	}
	if !fgs.equalAlignments {
		c.buf.Uint64(uint64(s.Alignment))
	}
	if fgs.multiplesPresent {
		c.buf.Uint32(s.Multiple())
	}
}

// Decode decodes a frame from the given src bytes.
func (c *Codec) Decode(src []byte) (dst framer.Frame, err error) {
	return c.DecodeStream(bytes.NewReader(src))
}

// decodeHeader holds the flags, the state, and the shared reference values that a
// frame header carries, and the position of the decoder in the frame.
type decodeHeader struct {
	fgs       flags
	ext       byte
	seqNum    uint32
	state     *state
	dataLen   uint32
	tr        telem.TimeRange
	alignment telem.Alignment
	// decoded is the running count of bytes the frame decodes to.
	decoded int64
	// next is the ordinal of the next series that has an override.
	next uint32
}

// DecodeStream decodes a frame from the given io reader. A frame that fails to decode
// drops its state, so every later frame of the state fails.
func (c *Codec) DecodeStream(reader io.Reader) (framer.Frame, error) {
	c.processUpdates()
	c.panicIfNotUpdated("Decode")
	c.reader.Reset(reader)
	var h decodeHeader
	if err := c.readHeader(&h); err != nil {
		return framer.Frame{}, drop(&h, err)
	}
	if !h.fgs.allChannelsPresent {
		return c.decodeKeyedSeries(&h)
	}
	// The frame is built here, not in a helper, since each return copies it.
	var (
		ag = &h.state.dec
		n  = uint32(len(ag.all))
		fr = frame.Alloc(len(ag.all))
	)
	for i := uint32(0); ; i++ {
		for end := min(h.next, n); i < end; i++ {
			s, err := c.decodeSeries(&h, &ag.series[i])
			if err != nil {
				return framer.Frame{}, drop(&h, err)
			}
			fr = fr.Append(ag.all[i], s)
		}
		if i == n {
			break
		}
		s, err := c.decodeOverridden(&h, i, int(i))
		if err != nil {
			return framer.Frame{}, drop(&h, err)
		}
		fr = fr.Append(ag.all[i], s)
	}
	if err := c.readEnd(reader); err != nil {
		return framer.Frame{}, drop(&h, err)
	}
	if h.next != noOverride {
		return framer.Frame{}, drop(&h, pendingError(h.next))
	}
	return fr, nil
}

// drop marks the state of a frame that failed to decode with err, and returns the error
// to report.
func drop(h *decodeHeader, err error) error {
	if h.state != nil {
		h.state.dec.dropped = true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return invalidf("remote sent a frame that ends early")
	}
	return err
}

func pendingError(next uint32) error {
	return invalidf(
		"remote ended the frame with the override for series %d pending",
		next,
	)
}

// readEnd fails when src holds bytes after the last series.
func (c *Codec) readEnd(src io.Reader) error {
	n, err := src.Read(c.tail[:])
	if n == 0 && err == nil {
		n, err = io.ReadFull(src, c.tail[:])
	}
	if n > 0 {
		return trailingError()
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func trailingError() error { return invalidf("remote sent bytes after the last series") }

// decodeKeyedSeries decodes series that each carry their key until the frame ends.
// Once no override is pending, a series needs no ordinal.
func (c *Codec) decodeKeyedSeries(h *decodeHeader) (framer.Frame, error) {
	var (
		ag = &h.state.dec
		fr framer.Frame
	)
	for ordinal := uint32(0); h.next != noOverride; ordinal++ {
		pos, err := c.readKey(h)
		if err != nil {
			return endKeyed(h, fr, err)
		}
		var s telem.Series
		if ordinal == h.next {
			s, err = c.decodeOverridden(h, ordinal, pos)
		} else {
			s, err = c.decodeSeries(h, &ag.series[pos])
		}
		if err != nil {
			return framer.Frame{}, drop(h, err)
		}
		fr = fr.Append(ag.all[pos], s)
	}
	for {
		pos, err := c.readKey(h)
		if err != nil {
			return endKeyed(h, fr, err)
		}
		s, err := c.decodeSeries(h, &ag.series[pos])
		if err != nil {
			return framer.Frame{}, drop(h, err)
		}
		fr = fr.Append(ag.all[pos], s)
	}
}

// endKeyed returns fr when err is the end of a frame with no override pending, and
// drops the state otherwise.
func endKeyed(h *decodeHeader, fr framer.Frame, err error) (framer.Frame, error) {
	if !errors.Is(err, io.EOF) {
		return framer.Frame{}, drop(h, err)
	}
	if h.next != noOverride {
		return framer.Frame{}, drop(h, pendingError(h.next))
	}
	return fr, nil
}

// readKey reads the key of the next series and returns its position in the keys of
// the state.
func (c *Codec) readKey(h *decodeHeader) (int, error) {
	k, err := c.reader.Uint32()
	if err != nil {
		return 0, err
	}
	pos, ok := h.state.dec.positions[channel.Key(k)]
	if !ok {
		return 0, unknownKeyError(channel.Key(k))
	}
	return pos, nil
}

func unknownKeyError(key channel.Key) error {
	return errors.Newf("unknown channel key: %v", key)
}

func (c *Codec) readHeader(h *decodeHeader) error {
	flagB, err := c.reader.Uint8()
	if err != nil {
		return err
	}
	if h.seqNum, err = c.reader.Uint32(); err != nil {
		return err
	}
	cState, ok := c.mu.states[h.seqNum]
	if !ok {
		return errors.Wrapf(
			validate.ErrValidation,
			"remote sent invalid sequence number %d. Valid rawIndices are %v",
			h.seqNum,
			lo.Keys(c.mu.states),
		)
	}
	c.prune(h.seqNum)
	if cState.dec.dropped {
		return invalidf(
			"remote sent a frame for state %d after a frame of it failed to decode",
			h.seqNum,
		)
	}
	h.state = cState
	h.fgs = decodeFlags(flagB)
	if h.fgs.extended {
		err = c.readExtension(h)
	} else {
		err = checkTypesDeclared(h)
	}
	if err != nil {
		return err
	}
	if h.fgs.equalLens {
		if h.dataLen, err = c.reader.Uint32(); err != nil {
			return err
		}
	}
	if h.fgs.equalTimeRanges && !h.fgs.timeRangesZero {
		if h.tr, err = c.readTimeRange(); err != nil {
			return err
		}
	}
	if h.fgs.equalAlignments && !h.fgs.zeroAlignments {
		v, err := c.reader.Uint64()
		if err != nil {
			return err
		}
		h.alignment = telem.Alignment(v)
	}
	h.next = noOverride
	if overridesPresentFlagPos.Get(h.ext) {
		if h.next, err = c.reader.Uint32(); err != nil {
			return err
		}
	}
	return nil
}

// decodeOverridden reads the override of the series at ordinal, whose key is at pos in
// the keys of the state, and then the series.
func (c *Codec) decodeOverridden(
	h *decodeHeader,
	ordinal uint32,
	pos int,
) (telem.Series, error) {
	a, err := c.readOverride(h, ordinal, pos)
	if err != nil {
		return telem.Series{}, err
	}
	return c.decodeSeries(h, &a)
}

// decodeSeries reads one series that decodes as a, and adds its size to h.decoded.
func (c *Codec) decodeSeries(h *decodeHeader, a *agreed) (telem.Series, error) {
	// Each field is set alone: a literal that reads them builds in a temporary, whose
	// copy stalls on amd64 store forwarding.
	var s telem.Series
	s.TimeRange = h.tr
	s.Alignment = h.alignment
	s.DataType = a.dataType
	s.Clock = a.clock
	dataLenOrSize := h.dataLen
	var err error
	if !h.fgs.equalLens {
		if dataLenOrSize, err = c.reader.Uint32(); err != nil {
			return telem.Series{}, err
		}
	}
	size := int64(dataLenOrSize)
	switch a.layout {
	case fixedLayout:
		size = int64(a.density.Size(size))
	case variableLayout:
		if h.fgs.equalLens {
			return telem.Series{}, invalidf(
				"remote sent variable data type %s under a shared length",
				a.dataType,
			)
		}
	case bitLayout:
	}
	h.decoded += size + SeriesOverhead
	if err = c.checkDecodeLimit(h); err != nil {
		return telem.Series{}, err
	}
	if a.layout == bitLayout {
		packed, err := c.readBytes(bitPackedByteCount(int64(dataLenOrSize)))
		if err != nil {
			return telem.Series{}, err
		}
		s.Data = unpackBoolBits(packed, int(dataLenOrSize))
	} else if s.Data, err = c.readBytes(size); err != nil {
		return telem.Series{}, err
	}
	if err = c.readSeriesMetadata(h.fgs, &s); err != nil {
		return telem.Series{}, err
	}
	return s, nil
}

// readSeriesMetadata reads the time range, alignment, and multiple of s that the
// flags do not share across the frame.
func (c *Codec) readSeriesMetadata(fgs flags, s *telem.Series) error {
	var err error
	if !fgs.equalTimeRanges {
		if s.TimeRange, err = c.readTimeRange(); err != nil {
			return err
		}
	}
	if !fgs.equalAlignments {
		v, err := c.reader.Uint64()
		if err != nil {
			return err
		}
		s.Alignment = telem.Alignment(v)
	}
	if fgs.multiplesPresent {
		multiple, err := c.reader.Uint32()
		if err != nil {
			return err
		}
		if multiple > 1 {
			s.AlignmentMultiple = multiple
		}
	}
	return nil
}

// readTimeRange reads a time range using the codec's reader.
func (c *Codec) readTimeRange() (telem.TimeRange, error) {
	start, err := c.reader.Uint64()
	if err != nil {
		return telem.TimeRange{}, err
	}
	end, err := c.reader.Uint64()
	if err != nil {
		return telem.TimeRange{}, err
	}
	return telem.TimeRange{
		Start: telem.TimeStamp(start),
		End:   telem.TimeStamp(end),
	}, nil
}

// readChunk is the most bytes that readBytes allocates before the bytes arrive.
const readChunk = 1 << 20

// readBytes reads n bytes. It grows its buffer only as bytes arrive, so a length that
// the source does not hold fails on the read instead of allocating the whole length.
func (c *Codec) readBytes(n int64) ([]byte, error) {
	buf := make([]byte, 0, min(n, readChunk))
	for int64(len(buf)) < n {
		if len(buf) == cap(buf) {
			buf = slices.Grow(buf, int(min(n-int64(len(buf)), int64(len(buf)))))
		}
		chunk := buf[len(buf):min(int64(cap(buf)), n)]
		read, err := c.reader.Read(chunk)
		buf = buf[:len(buf)+read]
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}
