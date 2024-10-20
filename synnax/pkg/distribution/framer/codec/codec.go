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
	"encoding/binary"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

type Base struct {
	keys         channel.Keys
	keyDataTypes map[channel.Key]telem.DataType
}

var byteOrder = binary.LittleEndian

func NewCodec(dataTypes []telem.DataType, channels channel.Keys) Base {
	keyDataTypes := make(map[channel.Key]telem.DataType, len(channels))
	for i, key := range channels {
		keyDataTypes[key] = dataTypes[i]
	}
	return Base{keys: channels, keyDataTypes: keyDataTypes}
}

func (m Base) Encode(src framer.Frame) (dst []byte, err error) {
	var (
		curDataSize                                      = -1
		startTime, endTime               telem.TimeStamp = 0, 0
		byteArraySize                                    = 1
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
	byteArraySize = 0
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

func (m Base) Decode(src []byte) (dst framer.Frame, err error) {
	if len(src) < 1 {
		return
	}
	var (
		returnStruct                            = framer.Frame{}
		sizeFlag, alignFlag, channelFlag bool   = false, false, false
		index                            int    = 0
		sizeRepresentation, currSize     uint32 = 0, 0
		timeRangeStart, timeRangeEnd     uint64 = 0, 0
	)
	sizeFlag = types.NumericToTBool((src[index] >> 2) & 1)
	alignFlag = types.NumericToTBool((src[index] >> 1) & 1)
	channelFlag = types.NumericToTBool(src[index] & 1)
	index += 1
	if sizeFlag {
		sizeRepresentation = byteOrder.Uint32(src[index:])
		index += 4
	}
	if alignFlag {
		timeRangeStart = byteOrder.Uint64(src[index:])
		timeRangeEnd = byteOrder.Uint64(src[index+8:])
		index += 16
	}
	if channelFlag {
		returnStruct.Keys = m.keys
	}
	for _, k := range m.keys {
		if !channelFlag {
			if index >= len(src) || channel.Key(byteOrder.Uint32(src[index:])) != k {
				continue
			}
			returnStruct.Keys = append(returnStruct.Keys, channel.Key(byteOrder.Uint32(src[index:])))
			index += 4
		}
		currSize = 0
		if !sizeFlag {
			currSize = byteOrder.Uint32(src[index:])
			index += 4
		} else {
			currSize = sizeRepresentation
		}
		currSeries := telem.Series{}
		currSeries.DataType = m.keyDataTypes[k]
		byteArraySlice := int(currSize) * int(currSeries.DataType.Density())
		currSeries.Data = make([]byte, byteArraySlice)
		copy(currSeries.Data, src[index:index+byteArraySlice])
		index += byteArraySlice
		if !alignFlag {
			currSeries.TimeRange.Start = telem.TimeStamp(byteOrder.Uint64(src[index:]))
			currSeries.TimeRange.End = telem.TimeStamp(byteOrder.Uint64(src[index+8:]))
			index += 16
		} else {
			currSeries.TimeRange.Start = telem.TimeStamp(timeRangeStart)
			currSeries.TimeRange.End = telem.TimeStamp(timeRangeEnd)
		}
		returnStruct.Series = append(returnStruct.Series, currSeries)
	}
	return returnStruct, nil
}
