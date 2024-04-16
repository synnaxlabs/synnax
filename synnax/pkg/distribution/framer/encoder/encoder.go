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

package encoder

import (
	"encoding/binary"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
)

type DecoderEncoder struct {
	dtypes []telem.DataType
	keys   channel.Keys
}

func New(DataTypes []telem.DataType, ChannelKeys channel.Keys) DecoderEncoder {
	return DecoderEncoder{dtypes: DataTypes, keys: ChannelKeys}
}

func (m DecoderEncoder) Encode(src framer.Frame) (dst []byte, err error) {
	var (
		curDataSize, index               int             = -1, 0
		startTime, endTime               telem.TimeStamp = 0, 0
		byteArraySize                    int             = 1 // Label to one to handle start array
		encoded                          []byte
		sizeFlag, alignFlag, channelFlag uint8 = 1, 1, 0
	)

	if len(m.keys) == len(src.Keys) {
		channelFlag = 1
	} else {
		channelFlag = 0
		byteArraySize += len(src.Keys) * 4
	}

	for i := 0; i < len(m.keys); i++ {
		if index >= len(src.Keys) || src.Keys[index] != m.keys[i] {
			continue
		}

		if curDataSize == -1 {
			curDataSize = len(src.Series[index].Data) / int(m.dtypes[index].Density())
			startTime = src.Series[index].TimeRange.Start
			endTime = src.Series[index].TimeRange.End
		}

		if len(src.Series[index].Data)/int(m.dtypes[index].Density()) == curDataSize {
			sizeFlag &= 1
		} else {
			sizeFlag &= 0
		}

		if src.Series[index].TimeRange.Start == startTime && src.Series[index].TimeRange.End == endTime {
			alignFlag &= 1
		} else {
			alignFlag &= 0
		}

		byteArraySize += len(src.Series[index].Data)
		index += 1
	}

	if sizeFlag == 0 {
		byteArraySize += len(src.Keys) * 4
	} else {
		byteArraySize += 4
	}

	if alignFlag == 0 {
		byteArraySize += len(src.Keys) * 16
	} else {
		byteArraySize += 16
	}

	encoded = make([]byte, byteArraySize)
	byteArraySize = 0

	encoded[byteArraySize] = byte((sizeFlag << 2) | (alignFlag << 1) | (channelFlag))
	byteArraySize += 1

	if sizeFlag == 1 {
		binary.LittleEndian.PutUint32(encoded[byteArraySize:], uint32(curDataSize))
		byteArraySize += 4
	}

	if alignFlag == 1 {
		binary.LittleEndian.PutUint64(encoded[byteArraySize:], uint64(src.Series[index-1].TimeRange.Start))
		byteArraySize += 8
		binary.LittleEndian.PutUint64(encoded[byteArraySize:], uint64(src.Series[index-1].TimeRange.End))
		byteArraySize += 8
	}

	index = 0

	for i := 0; i < len(m.keys); i++ {
		if index >= len(src.Keys) || src.Keys[index] != m.keys[i] {
			continue
		}
		lenSeriesData := uint32(len(src.Series[index].Data))
		dataSize := uint32(m.dtypes[i].Density())

		// Adding Task
		if channelFlag == 0 {
			binary.LittleEndian.PutUint32(encoded[byteArraySize:], uint32(src.Keys[index]))
			byteArraySize += 4
		}
		// Adding Data Length
		if sizeFlag == 0 {
			binary.LittleEndian.PutUint32(encoded[byteArraySize:], uint32(lenSeriesData/dataSize))
			byteArraySize += 4
		}

		// Adding Data
		copy(encoded[byteArraySize:], src.Series[index].Data)
		byteArraySize += int(lenSeriesData)

		// Adding Time range
		if alignFlag == 0 {
			binary.LittleEndian.PutUint64(encoded[byteArraySize:], uint64(src.Series[index].TimeRange.Start))
			byteArraySize += 8
			binary.LittleEndian.PutUint64(encoded[byteArraySize:], uint64(src.Series[index].TimeRange.End))
			byteArraySize += 8
		}

		index += 1
	}

	return encoded, nil
}

func (m DecoderEncoder) Decode(src []byte) (dst framer.Frame, err error) {
	var (
		returnStruct                                   = framer.Frame{}
		sizeFlag, alignFlag, channelFlag, index int    = 0, 0, 0, 0
		sizeRepresentation, curSize             uint32 = 0, 0
		timeRangeStart, timeRangeEnd            uint64 = 0, 0
	)

	sizeFlag = int((src[index] >> 2) & 1)
	alignFlag = int((src[index] >> 1) & 1)
	channelFlag = int(src[index] & 1)
	index += 1

	if sizeFlag == 1 {
		sizeRepresentation = binary.LittleEndian.Uint32(src[index:])
		index += 4
	}

	if alignFlag == 1 {
		timeRangeStart = binary.LittleEndian.Uint64(src[index:])
		timeRangeEnd = binary.LittleEndian.Uint64(src[index+8:])
		index += 16
	}

	if channelFlag == 1 {
		returnStruct.Keys = m.keys
	}

	for i := 0; i < len(m.keys); i++ {
		if channelFlag == 0 {
			if index >= len(src) || channel.Key(binary.LittleEndian.Uint32(src[index:])) != m.keys[i] {
				continue
			}
			returnStruct.Keys = append(returnStruct.Keys, channel.Key(binary.LittleEndian.Uint32(src[index:])))
			index += 4
		}

		// Obtain size if not given yet
		curSize = 0
		if sizeFlag == 0 {
			curSize = binary.LittleEndian.Uint32(src[index:])
			index += 4
		} else {
			curSize = sizeRepresentation
		}

		curSeries := telem.Series{}
		curSeries.DataType = m.dtypes[i]
		byteArraySlice := int(curSize) * int(curSeries.DataType.Density())
		curSeries.Data = make([]byte, byteArraySlice)
		copy(curSeries.Data, src[index:index+byteArraySlice])
		index += byteArraySlice
		if alignFlag == 0 {
			curSeries.TimeRange.Start = telem.TimeStamp(binary.LittleEndian.Uint64(src[index:]))
			curSeries.TimeRange.End = telem.TimeStamp(binary.LittleEndian.Uint64(src[index+8:]))
			index += 16
		} else {
			curSeries.TimeRange.Start = telem.TimeStamp(timeRangeStart)
			curSeries.TimeRange.End = telem.TimeStamp(timeRangeEnd)
		}

		returnStruct.Series = append(returnStruct.Series, curSeries)
	}

	return returnStruct, nil
}
