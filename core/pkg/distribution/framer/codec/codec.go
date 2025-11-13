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
	"encoding/binary"
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

func readTimeRange(reader io.Reader) (tr telem.TimeRange, err error) {
	if err = read(reader, &tr.Start); err != nil {
		return
	}
	err = read(reader, &tr.End)
	return
}

func writeTimeRange(w *xbinary.Writer, tr telem.TimeRange) {
	w.Uint64(uint64(tr.Start))
	w.Uint64(uint64(tr.End))
}

type sorter struct {
	rawIndices []int
	keys       []channel.Key
	offset     int
}

func (s *sorter) reset(size int) {
	s.offset = 0
	if cap(s.rawIndices) < size {
		s.rawIndices = make([]int, size)
		s.keys = make([]channel.Key, size)
	}
}

func (s *sorter) insert(key channel.Key, rawIndex int) {
	s.keys[s.offset] = key
	s.rawIndices[s.offset] = rawIndex
	s.offset++
}

func (s *sorter) sort() { sort.Sort(s) }

// Len implements sort.Interface.
func (s *sorter) Len() int { return s.offset }

// Less implements sort.Interface.
func (s *sorter) Less(i, j int) bool { return s.keys[i] < s.keys[j] }

// Swap implements sort.Interface.
func (s *sorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.rawIndices[i], s.rawIndices[j] = s.rawIndices[j], s.rawIndices[i]
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
	// channels used in dynamic codecs to retrieve information about channels
	// when Update is called.
	channels channel.Readable
	// encodeSorter is used to sort source frames that are being encoded. Used instead
	// of sorting the frame directly in order to avoid excess heap allocations
	encodeSorter sorter
}

var byteOrder = telem.ByteOrder

// NewStatic creates a new codec that uses the given channel keys and data types as
// its encoding state. It is not safe to call Update on a codec instantiated using
// NewStatic.
func NewStatic(channelKeys channel.Keys, dataTypes []telem.DataType) *Codec {
	if len(dataTypes) != len(channelKeys) {
		panic("data types and channel keys must be the same length")
	}
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channelKeys))
	for i, key := range channelKeys {
		keyDataTypes[key] = dataTypes[i]
	}
	c := newCodec()
	c.update(channelKeys, keyDataTypes)
	return c
}

// NewDynamic creates a new codec that can be dynamically updated by retrieving channels
// from the provided channel store. Codec.Update must be called before the first call
// to Codec.Encode and Codec.Decode.
func NewDynamic(channels channel.Readable) *Codec {
	c := newCodec()
	c.channels = channels
	return c
}

func newCodec() *Codec {
	c := &Codec{buf: xbinary.NewWriter(0, byteOrder)}
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

func read(r io.Reader, data any) error {
	return binary.Read(r, byteOrder, data)
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
		curDataSize                   = -1
		refTr                         = telem.TimeRangeZero
		refAlignment  telem.Alignment = 0
		byteArraySize                 = flagsSize + seqNumSize
		fgs                           = newFlags()
		currState                     = c.mu.states[c.mu.seqNum]
	)
	if currState.hasVariableDataTypes {
		fgs.equalLens = false
	}
	src = src.KeepKeys(currState.keys)
	if src.Count() != len(currState.keys) {
		fgs.allChannelsPresent = false
		byteArraySize += src.Count() * 4
	}
	for rawI, s := range src.RawSeries() {
		if src.ShouldExcludeRaw(rawI) {
			continue
		}
		key := src.RawKeyAt(rawI)
		c.encodeSorter.insert(key, rawI)
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
	if !fgs.equalLens {
		byteArraySize += src.Count() * 4
	} else {
		byteArraySize += 4
	}
	if !fgs.timeRangesZero {
		if !fgs.equalTimeRanges {
			byteArraySize += src.Count() * 16
		} else {
			byteArraySize += 16
		}
	}
	if !fgs.zeroAlignments {
		if !fgs.equalAlignments {
			byteArraySize += src.Count() * 8
		} else {
			byteArraySize += 8
		}
	}
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
	c.encodeSorter.sort()
	for offset := range c.encodeSorter.offset {
		rawI := c.encodeSorter.rawIndices[offset]
		if src.ShouldExcludeRaw(rawI) {
			continue
		}
		s := src.RawSeriesAt(rawI)
		if !fgs.allChannelsPresent {
			c.buf.Uint32(uint32(src.RawKeyAt(rawI)))
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
	var (
		dataLen      uint32
		refTr        telem.TimeRange
		refAlignment telem.Alignment
		seqNum       uint32
		flagB        byte
	)
	if err = read(reader, &flagB); err != nil {
		return
	}
	if err = read(reader, &seqNum); err != nil {
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
		if err = read(reader, &dataLen); err != nil {
			return
		}
	}
	if fgs.equalTimeRanges && !fgs.timeRangesZero {
		if refTr, err = readTimeRange(reader); err != nil {
			return
		}
	}
	if fgs.equalAlignments && !fgs.zeroAlignments {
		if err = read(reader, &refAlignment); err != nil {
			return
		}
	}

	decodeSeries := func(key channel.Key) (err error) {
		s := telem.Series{TimeRange: refTr, Alignment: refAlignment}
		dataLenOrSize := dataLen
		if !fgs.equalLens {
			if err = read(reader, &dataLenOrSize); err != nil {
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
		if _, err = io.ReadFull(reader, s.Data); err != nil {
			return err
		}
		if !fgs.equalTimeRanges {
			if s.TimeRange, err = readTimeRange(reader); err != nil {
				return
			}
		}
		if !fgs.equalAlignments {
			if err = read(reader, &s.Alignment); err != nil {
				return
			}
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

	var k channel.Key
	for {
		if err = read(reader, &k); err != nil {
			err = errors.Skip(err, io.EOF)
			return
		}
		if err = decodeSeries(k); err != nil {
			return
		}
	}
}
