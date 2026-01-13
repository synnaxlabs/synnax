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
	"fmt"
	"io"
	"slices"
	"sort"
	"sync/atomic"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/bit"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type state struct {
	keys                 channel.Keys
	keyDataTypes         map[channel.Key]telem.DataType
	hasVariableDataTypes bool
}

func writeTimeRange(w *binary.Writer, tr telem.TimeRange) {
	w.Uint64(uint64(tr.Start))
	w.Uint64(uint64(tr.End))
}

type sorter struct {
	rawIndices []int
	keys       []channel.Key
	alignments []telem.Alignment
	offset     int
}

func (s *sorter) reset(size int) {
	s.offset = 0
	if cap(s.rawIndices) < size {
		s.rawIndices = make([]int, size)
		s.keys = make([]channel.Key, size)
		s.alignments = make([]telem.Alignment, size)
	}
}

func (s *sorter) insert(key channel.Key, rawIndex int, alignment telem.Alignment) {
	s.keys[s.offset] = key
	s.rawIndices[s.offset] = rawIndex
	s.alignments[s.offset] = alignment
	s.offset++
}

func (s *sorter) sort() { sort.Sort(s) }

// Len implements sort.Interface.
func (s *sorter) Len() int { return s.offset }

// Less implements sort.Interface. Sorts by (key, alignment, rawIndex).
func (s *sorter) Less(i, j int) bool {
	if s.keys[i] != s.keys[j] {
		return s.keys[i] < s.keys[j]
	}
	if s.alignments[i] != s.alignments[j] {
		return s.alignments[i] < s.alignments[j]
	}
	return s.rawIndices[i] < s.rawIndices[j]
}

// Swap implements sort.Interface.
func (s *sorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.rawIndices[i], s.rawIndices[j] = s.rawIndices[j], s.rawIndices[i]
	s.alignments[i], s.alignments[j] = s.alignments[j], s.alignments[i]
}

// Codec is a high-performance encoder/decoder specifically designed for moving
// telemetry frames over the network. Codec is stateful, meaning that both the
// encoding and decoding sides must agree on the set of channels and their order
// before any encoding or decoding can occur.
type Codec struct {
	// mu is non-routine safe structures that must be used carefully.
	mu struct {
		// states is the current backlog of encoding states. We keep multiple states
		// to allow for temporary de-sync between the encoding and decoding sides. For
		// example, when updating the keys of a streamer, the receiving codec may get
		// the updated set of channel in its state before the sending codec may get
		// updated, which means that the receiving codec needs to decode according
		// to the previous state. seqNum and the states backlog are used to keep the
		// two in sync.
		states map[uint32]state
		// seqNum corresponds to the most recent update in states. This is incremented
		// and communicated each time a state is added.
		seqNum uint32
		// updateAvailable is an atomic flag indicating that a new state update is
		// available for processing on the encoding/decoding routine. Checking this
		// boolean is more performant than using a non-blocking select on every
		// encode/decode operation.
		updateAvailable atomic.Bool
		// updates is a channel that the routine in Update pushes a new state down
		// for processing within Encode/Decode.
		updates chan state
	}
	// buf is reused for each encode operation.
	buf *binary.Writer
	// reader is reused for each decode operation. Unlike the standard library
	// binary.Read, this avoids reflection overhead.
	reader *binary.Reader
	// channels used in dynamic codecs to retrieve information about channels
	// when Update is called.
	channels *channel.Service
	// encodeSorter is used to sort source frames that are being encoded. Used instead
	// of sorting the frame directly in order to avoid excess heap allocations
	encodeSorter sorter
	// mergedSeriesResult is a reusable slice for storing merged series info, avoiding
	// allocations on each encode operation
	mergedSeriesResult []mergedSeriesInfo
	// opts holds configuration options for the codec.
	opts *options
}

type options struct {
	// enableAlignmentCompression controls whether to merge contiguous series with
	// the same channel key during encoding. When enabled, reduces bandwidth at the
	// cost of some CPU overhead during encoding.
	enableAlignmentCompression bool
}

type Option = func(*options)

// DisableAlignmentCompression disables merging of contiguous series with the same
// channel key during encoding. This can significantly increase bandwidth usage
// (30-70%) for frames with many small contiguous series at the cost of 5-15%
// additional CPU overhead during encoding. Defaults to true.
func DisableAlignmentCompression() Option {
	return func(o *options) { o.enableAlignmentCompression = false }
}

func newOptions(opts []Option) *options {
	o := &options{enableAlignmentCompression: true}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

var byteOrder = telem.ByteOrder

// NewStatic creates a new codec that uses the given channel keys and data types as
// its encoding state with default configuration (alignment compression enabled).
// It is not safe to call Update on a codec instantiated using NewStatic.
func NewStatic(channelKeys channel.Keys, dataTypes []telem.DataType, opts ...Option) *Codec {
	if len(dataTypes) != len(channelKeys) {
		panic("data types and channel keys must be the same length")
	}
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channelKeys))
	for i, key := range channelKeys {
		keyDataTypes[key] = dataTypes[i]
	}
	c := newCodec(opts...)
	c.update(channelKeys, keyDataTypes)
	return c
}

// NewDynamic creates a new codec that can be dynamically updated by retrieving channels
// from the provided channel store with default configuration (alignment compression enabled).
// Codec.Update must be called before the first call to Codec.Encode and Codec.Decode.
func NewDynamic(channels *channel.Service, opts ...Option) *Codec {
	c := newCodec(opts...)
	c.channels = channels
	return c
}

func newCodec(opts ...Option) *Codec {
	o := newOptions(opts)
	c := &Codec{
		buf:    binary.NewWriter(0, byteOrder),
		reader: binary.NewReader(nil, byteOrder),
		opts:   o,
	}
	c.mu.updates = make(chan state, 50)
	c.mu.updateAvailable.Store(false)
	c.mu.states = make(map[uint32]state)
	return c
}

// Update updates the codec to use the given keys in its state.
func (c *Codec) Update(ctx context.Context, keys []channel.Key) error {
	channels := make([]channel.Channel, 0, len(keys))
	if err := c.channels.NewRetrieve().
		WhereKeys(keys...).
		Entries(&channels).Exec(ctx, nil); err != nil {
		return err
	}
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channels))
	for _, ch := range channels {
		keyDataTypes[ch.Key()] = ch.DataType
	}
	c.update(keys, keyDataTypes)
	return nil
}

// Initialized returns true if the codec was initialized using NewStatic or Update
// has been called at least once when using NewDynamic.
func (c *Codec) Initialized() bool {
	return c.mu.seqNum > 0 || c.mu.updateAvailable.Load()
}

func (c *Codec) update(keys channel.Keys, keyDataTypes map[channel.Key]telem.DataType) {
	s := state{
		keys:                 keys,
		keyDataTypes:         keyDataTypes,
		hasVariableDataTypes: false,
	}
	for _, dt := range keyDataTypes {
		if dt.IsVariable() {
			s.hasVariableDataTypes = true
			break
		}
	}
	slices.Sort(s.keys)
	c.mu.updateAvailable.Store(true)
	c.mu.updates <- s
}

func (c *Codec) processUpdates() {
	if !c.mu.updateAvailable.CompareAndSwap(true, false) {
		return
	}
	for {
		select {
		case s := <-c.mu.updates:
			c.mu.seqNum++
			c.mu.states[c.mu.seqNum] = s
		default:
			return
		}
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
}

const (
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

func (c *Codec) panicIfNotUpdated(opName string) {
	if c.mu.seqNum < 1 {
		panic(fmt.Sprintf("[framer.codec] - dynamic codec was not updated for first call to %s", opName))
	}
}

// isAlignmentContiguous checks if two series have contiguous alignments where
// the first series' upper bound equals the second series' lower bound.
func isAlignmentContiguous(s1, s2 telem.Series) bool {
	bounds1 := s1.AlignmentBounds()
	bounds2 := s2.AlignmentBounds()
	return bounds1.Upper == bounds2.Lower
}

// mergedSeriesInfo holds information about a series that may be merged or original.
type mergedSeriesInfo struct {
	series telem.Series
	key    channel.Key
}

// mergeContiguousSeries processes sorted series and merges those with the same key
// and contiguous alignments. Returns the merged series info using the codec's reusable
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
		groupStart := i
		groupEnd := i + 1

		// Find all series with the same key
		for groupEnd < count && sortedKeys[groupEnd] == key {
			groupEnd++
		}

		// Process contiguous sub-groups within this key group
		j := groupStart
		for j < groupEnd {
			// Start a new contiguous run
			runStart := j
			runEnd := j + 1

			// Extend the run while series are contiguous
			for runEnd < groupEnd {
				prevRawIdx := sortedIndices[runEnd-1]
				currRawIdx := sortedIndices[runEnd]
				prevSeries := src.RawSeriesAt(prevRawIdx)
				currSeries := src.RawSeriesAt(currRawIdx)

				if !isAlignmentContiguous(prevSeries, currSeries) {
					break
				}
				runEnd++
			}

			// If only one series in this run, append it directly
			if runEnd-runStart == 1 {
				rawIdx := sortedIndices[runStart]
				c.mergedSeriesResult = append(c.mergedSeriesResult, mergedSeriesInfo{
					series: src.RawSeriesAt(rawIdx),
					key:    key,
				})
			} else {
				// Multiple contiguous series - merge them
				// First pass: calculate total size
				totalSize := 0
				for k := runStart; k < runEnd; k++ {
					rawIdx := sortedIndices[k]
					s := src.RawSeriesAt(rawIdx)
					totalSize += len(s.Data)
				}

				// Allocate merged data buffer directly (no intermediate buffer)
				var (
					mergedData = make([]byte, totalSize)
					// Copy all series data directly into final buffer
					offset       = 0
					mergedSeries telem.Series
				)
				for k := runStart; k < runEnd; k++ {
					rawIdx := sortedIndices[k]
					s := src.RawSeriesAt(rawIdx)

					if k == runStart {
						// Initialize merged series with first series properties
						mergedSeries = telem.Series{
							DataType:  s.DataType,
							Alignment: s.Alignment,
							TimeRange: s.TimeRange,
						}
					} else {
						// Extend time range to encompass all series
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

				c.mergedSeriesResult = append(c.mergedSeriesResult, mergedSeriesInfo{
					series: mergedSeries,
					key:    key,
				})
			}

			j = runEnd
		}

		i = groupEnd
	}

	return c.mergedSeriesResult
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

// encodeInternal encodes the frame into c.buf. After calling this method,
// c.buf.Bytes() contains the encoded data.
func (c *Codec) encodeInternal(ctx context.Context, src framer.Frame) error {
	c.encodeSorter.reset(src.Count())
	c.processUpdates()
	c.panicIfNotUpdated("Encode")
	currState := c.mu.states[c.mu.seqNum]
	src = src.KeepKeys(currState.keys)

	// First pass: validate and insert into sorter with pre-extracted data
	for rawI, s := range src.RawSeries() {
		if src.ShouldExcludeRaw(rawI) {
			continue
		}
		key := src.RawKeyAt(rawI)
		c.encodeSorter.insert(key, rawI, s.Alignment)
		dt, ok := currState.keyDataTypes[key]
		if !ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"encoder was provided a key %s not present in current state",
				channel.TryToRetrieveStringer(ctx, c.channels, key),
			)
		}
		if dt != s.DataType {
			return errors.Wrapf(
				validate.ErrValidation, "data type %s for channel %s does not match series data type %s",
				dt, channel.TryToRetrieveStringer(ctx, c.channels, key), s.DataType,
			)
		}
	}

	// Sort by (key, alignment, rawIndex) using pre-extracted data for cache efficiency
	c.encodeSorter.sort()

	// Merge contiguous series with the same key if enabled, otherwise just
	// create series info directly. In both cases, we reuse c.mergedSeriesResult.
	var mergedSeries []mergedSeriesInfo
	if c.opts.enableAlignmentCompression {
		mergedSeries = c.mergeContiguousSeries(
			c.encodeSorter.keys,
			c.encodeSorter.rawIndices,
			src,
			c.encodeSorter.offset,
		)
	} else {
		// No merging - create series info directly from sorted data using reusable slice
		count := c.encodeSorter.offset
		if cap(c.mergedSeriesResult) < count {
			c.mergedSeriesResult = make([]mergedSeriesInfo, 0, count)
		}
		c.mergedSeriesResult = c.mergedSeriesResult[:0]
		for i := range count {
			rawIdx := c.encodeSorter.rawIndices[i]
			c.mergedSeriesResult = append(c.mergedSeriesResult, mergedSeriesInfo{
				series: src.RawSeriesAt(rawIdx),
				key:    c.encodeSorter.keys[i],
			})
		}
		mergedSeries = c.mergedSeriesResult
	}

	// Calculate flags and byte size based on merged series
	var (
		curDataSize                   = -1
		refTr                         = telem.TimeRangeZero
		refAlignment  telem.Alignment = 0
		byteArraySize                 = flagsSize + seqNumSize
		fgs                           = newFlags()
	)

	if currState.hasVariableDataTypes {
		fgs.equalLens = false
	}

	// Check if all original keys are present in merged series
	if len(mergedSeries) != len(currState.keys) {
		fgs.allChannelsPresent = false
		byteArraySize += len(mergedSeries) * 4
	}

	// Calculate flags and accumulate data size from merged series
	for _, msi := range mergedSeries {
		s := msi.series
		sLen := int(s.Len())
		byteArraySize += int(s.Size())

		if curDataSize == -1 {
			curDataSize = sLen
			refTr = s.TimeRange
			refAlignment = s.Alignment
			continue
		}
		if sLen != curDataSize {
			fgs.equalLens = false
		}
		if s.TimeRange != refTr {
			fgs.equalTimeRanges = false
		}
		if s.Alignment != refAlignment {
			fgs.equalAlignments = false
		}
	}

	fgs.timeRangesZero = fgs.equalTimeRanges && refTr.Start.IsZero() && refTr.End.IsZero()
	fgs.zeroAlignments = fgs.equalAlignments && refAlignment == 0

	// Calculate metadata size based on merged series count
	if !fgs.equalLens {
		byteArraySize += len(mergedSeries) * 4
	} else {
		byteArraySize += 4
	}
	if !fgs.timeRangesZero {
		if !fgs.equalTimeRanges {
			byteArraySize += len(mergedSeries) * 16
		} else {
			byteArraySize += 16
		}
	}
	if !fgs.zeroAlignments {
		if !fgs.equalAlignments {
			byteArraySize += len(mergedSeries) * 8
		} else {
			byteArraySize += 8
		}
	}

	// Allocate buffer and write headers
	c.buf.Resize(byteArraySize)
	c.buf.Reset()
	c.buf.Uint8(fgs.encode())
	c.buf.Uint32(c.mu.seqNum)
	if fgs.equalLens {
		c.buf.Uint32(uint32(curDataSize))
	}
	if fgs.equalTimeRanges && !fgs.timeRangesZero {
		writeTimeRange(c.buf, refTr)
	}
	if fgs.equalAlignments && !fgs.zeroAlignments {
		c.buf.Uint64(uint64(refAlignment))
	}

	// Write merged series data
	for _, msi := range mergedSeries {
		s := msi.series
		if !fgs.allChannelsPresent {
			c.buf.Uint32(uint32(msi.key))
		}
		if !fgs.equalLens {
			if s.DataType.IsVariable() {
				c.buf.Uint32(uint32(s.Size()))
			} else {
				c.buf.Uint32(uint32(s.Len()))
			}
		}
		c.buf.Write(s.Data)
		if !fgs.equalTimeRanges {
			writeTimeRange(c.buf, s.TimeRange)
		}
		if !fgs.equalAlignments {
			c.buf.Uint64(uint64(s.Alignment))
		}
	}

	return nil
}

// Decode decodes a frame from the given src bytes.
func (c *Codec) Decode(src []byte) (dst framer.Frame, err error) {
	return c.DecodeStream(bytes.NewReader(src))
}

// DecodeStream decodes a frame from the given io reader.
func (c *Codec) DecodeStream(reader io.Reader) (framer.Frame, error) {
	c.processUpdates()
	c.panicIfNotUpdated("Decode")
	c.reader.Reset(reader)

	var (
		dataLen      uint32
		refTr        telem.TimeRange
		refAlignment telem.Alignment
		fr           framer.Frame
		err          error
	)

	flagB, err := c.reader.Uint8()
	if err != nil {
		return framer.Frame{}, err
	}
	seqNum, err := c.reader.Uint32()
	if err != nil {
		return framer.Frame{}, err
	}
	cState, ok := c.mu.states[seqNum]
	if !ok {
		states := lo.Keys(c.mu.states)
		err = errors.Wrapf(validate.ErrValidation, "[framer.codec] - remote sent invalid sequence number %d. Valid rawIndices are %v", seqNum, states)
		return framer.Frame{}, err
	}
	fgs := decodeFlags(flagB)
	if fgs.equalLens {
		if dataLen, err = c.reader.Uint32(); err != nil {
			return framer.Frame{}, err
		}
	}
	if fgs.equalTimeRanges && !fgs.timeRangesZero {
		if refTr, err = c.readTimeRange(); err != nil {
			return framer.Frame{}, err
		}
	}
	if fgs.equalAlignments && !fgs.zeroAlignments {
		v, readErr := c.reader.Uint64()
		if readErr != nil {
			return framer.Frame{}, readErr
		}
		refAlignment = telem.Alignment(v)
	}

	decodeSeries := func(key channel.Key) error {
		s := telem.Series{TimeRange: refTr, Alignment: refAlignment}
		dataLenOrSize := dataLen
		if !fgs.equalLens {
			if dataLenOrSize, err = c.reader.Uint32(); err != nil {
				return err
			}
		}
		dataType, exists := cState.keyDataTypes[key]
		if !exists {
			return errors.Newf("unknown channel key: %v", key)
		}
		s.DataType = dataType
		if dataType.IsVariable() {
			s.Data = make([]byte, dataLenOrSize)
		} else {
			s.Data = make([]byte, dataType.Density().Size(int64(dataLenOrSize)))
		}
		if _, err = c.reader.Read(s.Data); err != nil {
			return err
		}
		if !fgs.equalTimeRanges {
			if s.TimeRange, err = c.readTimeRange(); err != nil {
				return err
			}
		}
		if !fgs.equalAlignments {
			v, readErr := c.reader.Uint64()
			if readErr != nil {
				return readErr
			}
			s.Alignment = telem.Alignment(v)
		}
		fr = fr.Append(key, s)
		return err
	}

	if fgs.allChannelsPresent {
		fr = frame.Alloc(len(cState.keys))
		for _, k := range cState.keys {
			if err = decodeSeries(k); err != nil {
				return framer.Frame{}, err
			}
		}
		return fr, nil
	}

	for {
		k, readErr := c.reader.Uint32()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return fr, nil
			}
			return framer.Frame{}, readErr
		}
		if err = decodeSeries(channel.Key(k)); err != nil {
			return framer.Frame{}, err
		}
	}
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
	return telem.TimeRange{Start: telem.TimeStamp(start), End: telem.TimeStamp(end)}, nil
}
