// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	xbinary "github.com/synnaxlabs/x/binary"
	xbits "github.com/synnaxlabs/x/bit"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type state struct {
	keys                 channel.Keys
	keyDataTypes         map[channel.Key]telem.DataType
	hasVariableDataTypes bool
}

func writeTimeRange(w *xbinary.Writer, tr telem.TimeRange) {
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
	buf *xbinary.Writer
	// reader is reused for each decode operation. Unlike the standard library
	// binary.Read, this avoids reflection overhead.
	reader *xbinary.Reader
	// channels used in dynamic codecs to retrieve information about channels
	// when Update is called.
	channels channel.Readable
	// encodeSorter is used to sort source frames that are being encoded. Used instead
	// of sorting the frame directly in order to avoid excess heap allocations
	encodeSorter sorter
	// mergeBuf is a reusable buffer for merging contiguous series data, avoiding
	// allocations on each encode operation
	mergeBuf []byte
	// enableAlignmentCompression controls whether to merge contiguous series with
	// the same channel key during encoding. When enabled, reduces bandwidth at the
	// cost of some CPU overhead during encoding.
	enableAlignmentCompression bool
}

// Config contains configuration options for the codec.
type Config struct {
	// EnableAlignmentCompression enables merging of contiguous series with the same
	// channel key during encoding. This can significantly reduce bandwidth usage
	// (30-70%) for frames with many small contiguous series at the cost of 5-15%
	// additional CPU overhead during encoding. Defaults to true.
	EnableAlignmentCompression bool
}

var byteOrder = telem.ByteOrder

// DefaultConfig returns the default codec configuration with alignment compression enabled.
func DefaultConfig() Config {
	return Config{
		EnableAlignmentCompression: true,
	}
}

// NewStatic creates a new codec that uses the given channel keys and data types as
// its encoding state with default configuration (alignment compression enabled).
// It is not safe to call Update on a codec instantiated using NewStatic.
func NewStatic(channelKeys channel.Keys, dataTypes []telem.DataType) *Codec {
	return NewStaticWithConfig(channelKeys, dataTypes, DefaultConfig())
}

// NewStaticWithConfig creates a new codec with custom configuration.
func NewStaticWithConfig(channelKeys channel.Keys, dataTypes []telem.DataType, cfg Config) *Codec {
	if len(dataTypes) != len(channelKeys) {
		panic("data types and channel keys must be the same length")
	}
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channelKeys))
	for i, key := range channelKeys {
		keyDataTypes[key] = dataTypes[i]
	}
	c := newCodec(cfg)
	c.update(channelKeys, keyDataTypes)
	return c
}

// NewDynamic creates a new codec that can be dynamically updated by retrieving channels
// from the provided channel store with default configuration (alignment compression enabled).
// Codec.Update must be called before the first call to Codec.Encode and Codec.Decode.
func NewDynamic(channels channel.Readable) *Codec {
	return NewDynamicWithConfig(channels, DefaultConfig())
}

// NewDynamicWithConfig creates a new dynamic codec with custom configuration.
func NewDynamicWithConfig(channels channel.Readable, cfg Config) *Codec {
	c := newCodec(cfg)
	c.channels = channels
	return c
}

func newCodec(cfg Config) *Codec {
	c := &Codec{
		buf:                        xbinary.NewWriter(0, byteOrder),
		reader:                     xbinary.NewReader(nil, byteOrder),
		enableAlignmentCompression: cfg.EnableAlignmentCompression,
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
	zeroAlignmentsFlagPos     xbits.FlagPos = 5
	equalAlignmentsFlagPos    xbits.FlagPos = 4
	equalLengthsFlagPos       xbits.FlagPos = 3
	equalTimeRangesFlagPos    xbits.FlagPos = 2
	timeRangesZeroFlagPos     xbits.FlagPos = 1
	allChannelsPresentFlagPos xbits.FlagPos = 0
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

// Encode encodes the given frame into bytes.
func (c *Codec) Encode(ctx context.Context, src framer.Frame) ([]byte, error) {
	w := bytes.NewBuffer([]byte{})
	err := c.EncodeStream(ctx, w, src)
	return w.Bytes(), err
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
// and contiguous alignments. Returns the merged series info and updates the codec's
// merge buffer.
func (c *Codec) mergeContiguousSeries(
	sortedKeys []channel.Key,
	sortedIndices []int,
	src framer.Frame,
	count int,
) []mergedSeriesInfo {
	if count == 0 {
		return nil
	}

	// Slice arrays to actual size to avoid reading old data
	sortedKeys = sortedKeys[:count]
	sortedIndices = sortedIndices[:count]

	result := make([]mergedSeriesInfo, 0, count)
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
				result = append(result, mergedSeriesInfo{
					series: src.RawSeriesAt(rawIdx),
					key:    key,
				})
			} else {
				// Multiple contiguous series - merge them
				totalSize := 0
				for k := runStart; k < runEnd; k++ {
					rawIdx := sortedIndices[k]
					s := src.RawSeriesAt(rawIdx)
					totalSize += len(s.Data)
				}

				// Ensure merge buffer is large enough
				if cap(c.mergeBuf) < totalSize {
					c.mergeBuf = make([]byte, totalSize)
				}
				c.mergeBuf = c.mergeBuf[:totalSize]

				// Copy all series data into merge buffer
				offset := 0
				var mergedSeries telem.Series
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

					copy(c.mergeBuf[offset:], s.Data)
					offset += len(s.Data)
				}

				// Create a copy of the merged data for the series
				// (we can't use mergeBuf directly as it will be reused)
				mergedData := make([]byte, totalSize)
				copy(mergedData, c.mergeBuf[:totalSize])
				mergedSeries.Data = mergedData

				result = append(result, mergedSeriesInfo{
					series: mergedSeries,
					key:    key,
				})
			}

			j = runEnd
		}

		i = groupEnd
	}

	return result
}

const (
	flagsSize  = 1
	seqNumSize = 4
)

// EncodeStream encodes the given frame into the provided io writer, returning any
// encoding errors encountered.
func (c *Codec) EncodeStream(ctx context.Context, w io.Writer, src framer.Frame) (err error) {
	c.encodeSorter.reset(src.Count())
	c.processUpdates()
	c.panicIfNotUpdated("Encode")
	var (
		currState = c.mu.states[c.mu.seqNum]
	)
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
				validate.Error,
				"encoder was provided a key %s not present in current state",
				channel.TryToRetrieveStringer(ctx, c.channels, key),
			)
		}
		if dt != s.DataType {
			return errors.Wrapf(
				validate.Error, "data type %s for channel %s does not match series data type %s",
				dt, channel.TryToRetrieveStringer(ctx, c.channels, key), s.DataType,
			)
		}
	}

	// Sort by (key, alignment, rawIndex) using pre-extracted data for cache efficiency
	c.encodeSorter.sort()

	// Merge contiguous series with the same key if enabled
	var mergedSeries []mergedSeriesInfo
	if c.enableAlignmentCompression {
		mergedSeries = c.mergeContiguousSeries(
			c.encodeSorter.keys,
			c.encodeSorter.rawIndices,
			src,
			c.encodeSorter.offset,
		)
	} else {
		// No merging - create series info directly from sorted data
		mergedSeries = make([]mergedSeriesInfo, 0, c.encodeSorter.offset)
		for i := 0; i < c.encodeSorter.offset; i++ {
			rawIdx := c.encodeSorter.rawIndices[i]
			mergedSeries = append(mergedSeries, mergedSeriesInfo{
				series: src.RawSeriesAt(rawIdx),
				key:    c.encodeSorter.keys[i],
			})
		}
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

	_, err = w.Write(c.buf.Bytes())
	return err
}

// Decode decodes a frame from the given src bytes.
func (c *Codec) Decode(src []byte) (dst framer.Frame, err error) {
	return c.DecodeStream(bytes.NewReader(src))
}

// DecodeStream decodes a frame from the given io reader.
func (c *Codec) DecodeStream(reader io.Reader) (frame framer.Frame, err error) {
	c.processUpdates()
	c.panicIfNotUpdated("Decode")
	c.reader.Reset(reader)

	var (
		dataLen      uint32
		refTr        telem.TimeRange
		refAlignment telem.Alignment
	)

	flagB, err := c.reader.Uint8()
	if err != nil {
		return
	}
	seqNum, err := c.reader.Uint32()
	if err != nil {
		return
	}
	cState, ok := c.mu.states[seqNum]
	if !ok {
		states := lo.Keys(c.mu.states)
		err = errors.Wrapf(validate.Error, "[framer.codec] - remote sent invalid sequence number %d. Valid rawIndices are %v", seqNum, states)
		return
	}
	fgs := decodeFlags(flagB)
	if fgs.equalLens {
		if dataLen, err = c.reader.Uint32(); err != nil {
			return
		}
	}
	if fgs.equalTimeRanges && !fgs.timeRangesZero {
		if refTr, err = c.readTimeRange(); err != nil {
			return
		}
	}
	if fgs.equalAlignments && !fgs.zeroAlignments {
		v, readErr := c.reader.Uint64()
		if readErr != nil {
			err = readErr
			return
		}
		refAlignment = telem.Alignment(v)
	}

	decodeSeries := func(key channel.Key) (err error) {
		s := telem.Series{TimeRange: refTr, Alignment: refAlignment}
		dataLenOrSize := dataLen
		if !fgs.equalLens {
			if dataLenOrSize, err = c.reader.Uint32(); err != nil {
				return
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
				return
			}
		}
		if !fgs.equalAlignments {
			v, readErr := c.reader.Uint64()
			if readErr != nil {
				return readErr
			}
			s.Alignment = telem.Alignment(v)
		}
		frame = frame.Append(key, s)
		return
	}

	if fgs.allChannelsPresent {
		frame = core.AllocFrame(len(cState.keys))
		for _, k := range cState.keys {
			if err = decodeSeries(k); err != nil {
				return
			}
		}
		return
	}

	for {
		k, readErr := c.reader.Uint32()
		if readErr != nil {
			err = errors.Skip(readErr, io.EOF)
			return
		}
		if err = decodeSeries(channel.Key(k)); err != nil {
			return
		}
	}
}

// readTimeRange reads a time range using the codec's reader.
func (c *Codec) readTimeRange() (tr telem.TimeRange, err error) {
	start, err := c.reader.Uint64()
	if err != nil {
		return
	}
	end, err := c.reader.Uint64()
	if err != nil {
		return
	}
	return telem.TimeRange{Start: telem.TimeStamp(start), End: telem.TimeStamp(end)}, nil
}
