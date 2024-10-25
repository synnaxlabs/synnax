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
	"fmt"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
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

func (m Codec) Encode(src framer.Frame, startOffset int) (dst []byte, err error) {
	var (
		curDataSize                                      = -1
		startTime, endTime               telem.TimeStamp = 0, 0
		byteArraySize                                    = startOffset + 1
		sizeFlag, alignFlag, channelFlag                 = true, true, true
	)
	if len(src.Keys) != len(m.keys) {
		channelFlag = false
		byteArraySize += len(src.Keys) * 4
	}
	src.Sort()
	for _, s := range src.Series {
		if curDataSize == -1 {
			curDataSize = int(s.Len())
			startTime = s.TimeRange.Start
			endTime = s.TimeRange.End
		}
		if int(s.Len()) != curDataSize {
			sizeFlag = false
		}
		if s.TimeRange.Start != startTime || s.TimeRange.End != endTime {
			alignFlag = false
		}
		byteArraySize += len(s.Data)
	}
	if !sizeFlag {
		byteArraySize += len(src.Keys) * 4
	} else {
		byteArraySize += 4
	}
	if !alignFlag {
		byteArraySize += len(src.Keys) * 16
	} else {
		byteArraySize += 16
	}
	encoded := make([]byte, byteArraySize)
	byteArraySize = startOffset
	encoded[byteArraySize] = (types.BoolToUint8(sizeFlag) << 2) | (types.BoolToUint8(alignFlag) << 1) | types.BoolToUint8(channelFlag)
	byteArraySize += 1
	if sizeFlag {
		byteOrder.PutUint32(encoded[byteArraySize:], uint32(curDataSize))
		byteArraySize += 4
	}
	if alignFlag {
		byteOrder.PutUint64(encoded[byteArraySize:], uint64(src.Series[0].TimeRange.Start))
		byteArraySize += 8
		byteOrder.PutUint64(encoded[byteArraySize:], uint64(src.Series[0].TimeRange.End))
		byteArraySize += 8
	}
	for i, s := range src.Series {
		seriesDataLength := uint32(len(s.Data))
		dataSize := uint32(s.DataType.Density())
		if !channelFlag {
			key := src.Keys[i]
			byteOrder.PutUint32(encoded[byteArraySize:], uint32(key))
			byteArraySize += 4
		}
		if !sizeFlag {
			byteOrder.PutUint32(encoded[byteArraySize:], seriesDataLength/dataSize)
			byteArraySize += 4
		}
		copy(encoded[byteArraySize:], s.Data)
		byteArraySize += int(seriesDataLength)
		if !alignFlag {
			byteOrder.PutUint64(encoded[byteArraySize:], uint64(s.TimeRange.Start))
			byteArraySize += 8
			byteOrder.PutUint64(encoded[byteArraySize:], uint64(s.TimeRange.End))
			byteArraySize += 8
		}
	}
	return encoded, nil
}

func (m Codec) Decode(src []byte) (dst framer.Frame, err error) {
	b := bytes.NewReader(src)
	return m.DecodeStream(b)
}

func (m Codec) DecodeStream(reader io.Reader) (framer.Frame, error) {
	var (
		sizeFlag, alignFlag, channelFlag bool
		sizeRepresentation               uint32
		timeRangeStart, timeRangeEnd     uint64
		frame                            framer.Frame
	)

	// Read the flag byte
	var flagByte byte
	if err := binary.Read(reader, byteOrder, &flagByte); err != nil {
		return frame, err
	}

	sizeFlag = ((flagByte >> 2) & 1) == 1
	alignFlag = ((flagByte >> 1) & 1) == 1
	channelFlag = (flagByte & 1) == 1

	// Read size representation if sizeFlag is true
	if sizeFlag {
		if err := binary.Read(reader, byteOrder, &sizeRepresentation); err != nil {
			return frame, err
		}
	}

	// Read time range if alignFlag is true
	if alignFlag {
		if err := binary.Read(reader, byteOrder, &timeRangeStart); err != nil {
			return frame, err
		}
		if err := binary.Read(reader, byteOrder, &timeRangeEnd); err != nil {
			return frame, err
		}
	}

	// Prepare to read series data
	keys := m.keys
	if !channelFlag {
		keys = nil
	}

	for {
		var key channel.Key
		if !channelFlag {
			var keyVal uint32
			if err := binary.Read(reader, byteOrder, &keyVal); err != nil {
				if err == io.EOF {
					break
				}
				return frame, err
			}
			key = channel.Key(keyVal)
		} else {
			if len(keys) == 0 {
				break
			}
			key = keys[0]
			keys = keys[1:]
		}
		frame.Keys = append(frame.Keys, key)

		// Determine data size
		var currSize uint32
		if !sizeFlag {
			if err := binary.Read(reader, byteOrder, &currSize); err != nil {
				return frame, err
			}
		} else {
			currSize = sizeRepresentation
		}

		// Read series data
		dataType, exists := m.keyDataTypes[key]
		if !exists {
			return frame, fmt.Errorf("unknown channel key: %v", key)
		}

		dataLength := int(currSize) * int(dataType.Density())
		data := make([]byte, dataLength)
		if _, err := io.ReadFull(reader, data); err != nil {
			return frame, err
		}

		// Read time range if alignFlag is false
		startTime, endTime := timeRangeStart, timeRangeEnd
		if !alignFlag {
			if err := binary.Read(reader, byteOrder, &startTime); err != nil {
				return frame, err
			}
			if err := binary.Read(reader, byteOrder, &endTime); err != nil {
				return frame, err
			}
		}

		// Append series to frame
		frame.Series = append(frame.Series, telem.Series{
			DataType: dataType,
			Data:     data,
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(startTime),
				End:   telem.TimeStamp(endTime),
			},
		})
	}

	return frame, nil
}
