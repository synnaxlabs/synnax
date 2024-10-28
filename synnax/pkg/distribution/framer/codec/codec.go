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
	"encoding/binary"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	xbits "github.com/synnaxlabs/x/bits"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"io"
	"slices"
)

type Codec struct {
	keys         channel.Keys
	keyDataTypes map[channel.Key]telem.DataType
}

var byteOrder = binary.LittleEndian

func NewCodec(dataTypes []telem.DataType, channels channel.Keys) Codec {
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channels))
	for i, key := range channels {
		keyDataTypes[key] = dataTypes[i]
	}
	slices.Sort(channels)
	return Codec{keys: channels, keyDataTypes: keyDataTypes}
}

func NewCodecFromChannels(channels []channel.Channel) Codec {
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channels))
	keys := make([]channel.Key, len(channels))
	for i, ch := range channels {
		keyDataTypes[ch.Key()] = ch.DataType
		keys[i] = ch.Key()
	}
	slices.Sort(keys)
	return Codec{keys: keys, keyDataTypes: keyDataTypes}
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
	zeroAlignmentsFlagPos     xbits.Pos = 5
	equalAlignmentsFlagPos    xbits.Pos = 4
	equalLengthsFlagPos       xbits.Pos = 3
	equalTimeRangesFlagPos    xbits.Pos = 2
	timeRangesZeroFlagPos     xbits.Pos = 1
	allChannelsPresentFlagPos xbits.Pos = 0
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

func writeNaive(buf io.Writer, data any) {
	_ = binary.Write(buf, byteOrder, data)
}

func read(r io.Reader, data any) error {
	return binary.Read(r, byteOrder, data)
}

func (m Codec) Encode(src framer.Frame, startOffset int) (dst []byte, err error) {
	var (
		curDataSize  = -1
		refTr        = telem.TimeRangeZero
		refAlignment = telem.AlignmentPair(0)
		// include an extra byte for the flags
		byteArraySize = startOffset + 1
		fgs           = newFlags()
	)
	if len(src.Keys) != len(m.keys) {
		fgs.allChannelsPresent = false
		byteArraySize += len(src.Keys) * 4
	}
	src.Sort()
	for _, s := range src.Series {
		if curDataSize == -1 {
			curDataSize = int(s.Len())
			refTr = s.TimeRange
			refAlignment = s.Alignment
		}
		if int(s.Len()) != curDataSize {
			fgs.equalLens = false
		}
		if s.TimeRange != refTr {
			fgs.equalTimeRanges = false
		}
		if s.Alignment != refAlignment {
			fgs.equalAlignments = false
		}
		byteArraySize += len(s.Data)

	}
	fgs.timeRangesZero = fgs.equalTimeRanges && refTr.Start.IsZero() && refTr.End.IsZero()
	fgs.zeroAlignments = fgs.equalAlignments && refAlignment == 0
	if !fgs.equalLens {
		byteArraySize += len(src.Keys) * 4
	} else {
		byteArraySize += 4
	}
	if !fgs.timeRangesZero {
		if !fgs.equalTimeRanges {
			byteArraySize += len(src.Keys) * 16
		} else {
			byteArraySize += 16
		}
	}
	if !fgs.zeroAlignments {
		if !fgs.equalAlignments {
			byteArraySize += len(src.Keys) * 8
		} else {
			byteArraySize += 8
		}
	}
	buf := bytes.NewBuffer(make([]byte, startOffset, byteArraySize))
	buf.WriteByte(fgs.encode())
	// It's impossible for writing to the buffer to fail, so we just ignore all of the
	// errors.
	if fgs.equalLens {
		writeNaive(buf, uint32(curDataSize))
	}
	if fgs.equalTimeRanges && !fgs.timeRangesZero {
		writeTimeRange(buf, refTr)
	}
	if fgs.equalAlignments && !fgs.zeroAlignments {
		writeNaive(buf, refAlignment)
	}
	for i, s := range src.Series {
		seriesDataLength := uint32(len(s.Data))
		dataSize := uint32(s.DataType.Density())
		if !fgs.allChannelsPresent {
			writeNaive(buf, src.Keys[i])
		}
		if !fgs.equalLens {
			writeNaive(buf, seriesDataLength/dataSize)
		}
		_, _ = buf.Write(s.Data)
		if !fgs.equalTimeRanges {
			writeTimeRange(buf, s.TimeRange)
		}
		if !fgs.equalAlignments {
			writeNaive(buf, s.Alignment)
		}
	}
	return buf.Bytes(), nil
}

func (m Codec) Decode(src []byte) (dst framer.Frame, err error) {
	b := bytes.NewReader(src)
	return m.DecodeStream(b)
}

func readTimeRange(reader io.Reader) (tr telem.TimeRange, err error) {
	if err = binary.Read(reader, byteOrder, &tr.Start); err != nil {
		return
	}
	err = binary.Read(reader, byteOrder, &tr.End)
	return
}

func writeTimeRange(w io.Writer, tr telem.TimeRange) {
	writeNaive(w, tr.Start)
	writeNaive(w, tr.End)
}

func (m Codec) DecodeStream(reader io.Reader) (frame framer.Frame, err error) {
	var (
		dataLen      uint32
		refTr        telem.TimeRange
		refAlignment telem.AlignmentPair
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

		localDataLen := dataLen
		if !fgs.equalLens {
			if err = read(reader, &localDataLen); err != nil {
				return
			}
		}
		dataType, exists := m.keyDataTypes[key]
		if !exists {
			return errors.Newf("unknown channel key: %v", key)
		}
		s.DataType = dataType
		s.Data = make([]byte, dataType.Density().Size(int64(localDataLen)))
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
		frame.Keys = append(frame.Keys, key)
		frame.Series = append(frame.Series, s)
		return
	}

	if fgs.allChannelsPresent {
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
			err = errors.Ignore(err, io.EOF)
			return
		}
		if err = decodeSeries(k); err != nil {
			return
		}
	}
}
