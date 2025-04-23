// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// For detailed information about the specifications,
// please refer to the official RFC 0016 document.
// Document here: docs/tech/rfc/0016-231001-frame-flight-protocol.md

package codec

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	xbinary "github.com/synnaxlabs/x/binary"
	xbits "github.com/synnaxlabs/x/bit"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// LazyCodec is an extension of Codec that can lazily load the channels it needs to
// decode a frame. This is useful for cases where the channels are not known when
// initializing the codec, or the list of channels needs ot be updated through the
// codec's lifetime.
//
// Update must be called at least once before any decoding/encoding methods are called.
type LazyCodec struct {
	core *Codec
	// Channels is used to retrieve channel information to update the codec.
	Channels channel.Readable
}

func (l *LazyCodec) panicIfNotInitialized() {
	if l.core == nil {
		panic("[framer] - LazyCodec was not initialized. Call Update() before using the codec.")
	}
}

func (l *LazyCodec) Encode(src framer.Frame) ([]byte, error) {
	return l.core.Encode(src)
}

func (l *LazyCodec) EncodeStream(w io.Writer, src framer.Frame) error {
	return l.core.EncodeStream(w, src)
}

func (l *LazyCodec) Decode(src []byte) (framer.Frame, error) {
	l.panicIfNotInitialized()
	return l.core.Decode(src)
}

func (l *LazyCodec) DecodeStream(reader io.Reader) (framer.Frame, error) {
	l.panicIfNotInitialized()
	return l.core.DecodeStream(reader)
}

func NewLazyCodec(channels channel.Readable) *LazyCodec {
	return &LazyCodec{Channels: channels}
}

func WrapWithLazy(codec *Codec) *LazyCodec {
	return &LazyCodec{core: codec}
}

// Update updates the codec to use the channels with the given keys. If channels with
// the given keys are not found, Update returns a query.NotFound error.
func (l *LazyCodec) Update(ctx context.Context, keys channel.Keys) error {
	channels := make([]channel.Channel, 0, len(keys))
	err := l.Channels.NewRetrieve().WhereKeys(keys...).Entries(&channels).Exec(ctx, nil)
	l.core = NewCodecFromChannels(channels)
	return err
}

// Codec is a high-performance encoder/decoder specifically designed for moving
// telemetry frames over the network. Codec is stateful, meaning that both the
// encoding and decoding side must agree on the set of channels and their order
// before any encoding or decoding can occur.
type Codec struct {
	// keys is the numerically sorted list of keys that the codec will encode/decode.
	keys                channel.Keys
	keyDataTypes        map[channel.Key]telem.DataType
	hasVariableDataType bool
	buf                 *xbinary.Writer
}

var byteOrder = binary.LittleEndian

func NewCodec(dataTypes []telem.DataType, channelKeys channel.Keys) *Codec {
	if len(dataTypes) != len(channelKeys) {
		panic("data types and channel keys must be the same length")
	}
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channelKeys))
	for i, key := range channelKeys {
		keyDataTypes[key] = dataTypes[i]
	}
	return newCodec(channelKeys, keyDataTypes)
}

func NewCodecFromChannels(channels []channel.Channel) *Codec {
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channels))
	keys := make([]channel.Key, len(channels))
	for i, ch := range channels {
		keyDataTypes[ch.Key()] = ch.DataType
		keys[i] = ch.Key()
	}
	return newCodec(keys, keyDataTypes)
}

func newCodec(keys channel.Keys, keyDataTypes map[channel.Key]telem.DataType) *Codec {
	hasVariableDataType := false
	for _, dt := range keyDataTypes {
		if dt.IsVariable() {
			hasVariableDataType = true
			break
		}
	}
	slices.Sort(keys)
	return &Codec{
		keys:                keys,
		keyDataTypes:        keyDataTypes,
		hasVariableDataType: hasVariableDataType,
		buf:                 xbinary.NewWriter(0, 0, byteOrder),
	}
}

type flags struct {
	// equalsLens is true when all series in the frame have the same number of samples.
	// This lets us consolidate the length of all series into one number.
	equalLens bool
	// equalTimeRanges is true when all series in the frame have the same time range.
	// This lets use consolidate all time ranges into one, 16 byte section.
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

func (m *Codec) Encode(src framer.Frame) ([]byte, error) {
	w := bytes.NewBuffer([]byte{})
	err := m.EncodeStream(w, src)
	return w.Bytes(), err
}

func (m *Codec) EncodeStream(w io.Writer, src framer.Frame) (err error) {
	var (
		curDataSize                   = -1
		refTr                         = telem.TimeRangeZero
		refAlignment  telem.Alignment = 0
		byteArraySize                 = 1
		fgs                           = newFlags()
	)
	if m.hasVariableDataType {
		fgs.equalLens = false
	}
	if src.Count() != len(m.keys) {
		fgs.allChannelsPresent = false
		byteArraySize += src.Count() * 4
	}
	src.Sort()
	for rawI, s := range src.RawSeries() {
		if src.ShouldExcludeRaw(rawI) {
			continue
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
	m.buf.Resize(byteArraySize)
	m.buf.Reset()
	m.buf.Uint8(fgs.encode())
	if fgs.equalLens {
		m.buf.Uint32(uint32(curDataSize))
	}
	if fgs.equalTimeRanges && !fgs.timeRangesZero {
		writeTimeRange(m.buf, refTr)
	}
	if fgs.equalAlignments && !fgs.zeroAlignments {
		m.buf.Uint64(uint64(refAlignment))
	}
	for rawI, s := range src.RawSeries() {
		if src.ShouldExcludeRaw(rawI) {
			continue
		}
		if !fgs.allChannelsPresent {
			m.buf.Uint32(uint32(src.KeysAtRaw(rawI)))
		}
		if !fgs.equalLens {
			if s.DataType.IsVariable() {
				m.buf.Uint32(uint32(s.Size()))
			} else {
				m.buf.Uint32(uint32(s.Len()))
			}
		}
		m.buf.Write(s.Data)
		if !fgs.equalTimeRanges {
			writeTimeRange(m.buf, s.TimeRange)
		}
		if !fgs.equalAlignments {
			m.buf.Uint64(uint64(s.Alignment))
		}
	}
	_, err = w.Write(m.buf.Bytes())
	return err
}

func (m *Codec) Decode(src []byte) (dst framer.Frame, err error) {
	return m.DecodeStream(bytes.NewReader(src))
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

func (m *Codec) DecodeStream(reader io.Reader) (frame framer.Frame, err error) {
	var (
		dataLen      uint32
		refTr        telem.TimeRange
		refAlignment telem.Alignment
		flagB        byte
	)
	if err = read(reader, &flagB); err != nil {
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
		dataType, exists := m.keyDataTypes[key]
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
		frame = core.AllocFrame(len(m.keys))
		for _, k := range m.keys {
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
