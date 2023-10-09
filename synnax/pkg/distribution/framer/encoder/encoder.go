package encoder

import (
	"encoding/binary"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
)

type EncoderDecoder struct {
	dtypes []telem.DataType
	keys   channel.Keys
}

func (m EncoderDecoder) JSONtoByte(src framer.Frame) ([]byte, error) {
	return nil, nil
}

func (m EncoderDecoder) BytetoJSON(src []byte) (framer.Frame, error) {
	return framer.Frame{}, nil
}

func NewEncoderDecoder(DataTypes []telem.DataType, ChannelKeys channel.Keys) EncoderDecoder {
	return EncoderDecoder{dtypes: DataTypes, keys: ChannelKeys}
}

// struct - CONFIG - ANOTHER STRUCT - Given the channels (keys of the channels)
// List of all channels
// All datatypes send at beginning
// Not guarenteed are channels exist in the Frame, if bit = 1 don't send keys, or bit = 0 send keys
//
//
//

// Ignoring Partial Channels and Strongly Aligned For Name
func (m EncoderDecoder) Encode(src framer.Frame) (dst []byte, err error) {
	var (
		curDataSize, index               int             = -1, 0
		startTime, endTime               telem.TimeStamp = 0, 0
		returnArray                      []byte
		sizeFlag, alignFlag, channelFlag int = 1, 1, 0
	)

	channelFlag = map[bool]int{false: 0, true: 1}[len(m.keys) == len(src.Keys)]

	for i := 0; i < len(m.keys); i++ {
		if index >= len(src.Keys) || src.Keys[index] != m.keys[i] {
			continue
		}

		if curDataSize == -1 {
			curDataSize = len(src.Series[index].Data) / int(m.dtypes[index].Density())
			startTime = src.Series[index].TimeRange.Start
			endTime = src.Series[index].TimeRange.End
		}

		sizeFlag &= map[bool]int{false: 0, true: 1}[len(src.Series[index].Data)/int(m.dtypes[index].Density()) == curDataSize]
		alignFlag &= map[bool]int{false: 0, true: 1}[src.Series[index].TimeRange.Start == startTime && src.Series[index].TimeRange.End == endTime]
		index += 1
	}

	// First byte for any is
	// sameData_strongAligned_allChannels_curDataSize (5 bits)
	returnArray = append(returnArray, byte((sizeFlag<<2)|(alignFlag<<1)|(channelFlag)))

	if sizeFlag == 1 {
		bytesToAppend := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytesToAppend, uint32(curDataSize))
		returnArray = append(returnArray, bytesToAppend...)
	}
	if alignFlag == 1 {
		timeRangeAppend := make([]byte, 8)
		binary.LittleEndian.PutUint64(timeRangeAppend, uint64(src.Series[index-1].TimeRange.Start))
		returnArray = append(returnArray, timeRangeAppend...)
		binary.LittleEndian.PutUint64(timeRangeAppend, uint64(src.Series[index-1].TimeRange.End))
		returnArray = append(returnArray, timeRangeAppend...)
	}

	index = 0

	for i := 0; i < len(m.keys); i++ {
		if index >= len(src.Keys) || src.Keys[index] != m.keys[i] {
			continue
		}
		lenSeriesData := uint32(len(src.Series[index].Data))
		dataSize := uint32(m.dtypes[i].Density())
		// Adding Key
		if channelFlag == 0 {
			keyInformation := make([]byte, 4)
			binary.LittleEndian.PutUint32(keyInformation, uint32(src.Keys[index]))
			returnArray = append(returnArray, keyInformation...)
		}
		// Adding Data Length
		if sizeFlag == 0 {
			bytesToAppend := make([]byte, 4)
			binary.LittleEndian.PutUint32(bytesToAppend, uint32(lenSeriesData/dataSize))
			returnArray = append(returnArray, bytesToAppend...)
		}
		// Adding Data
		for j := 0; j < int(lenSeriesData); j++ {
			returnArray = append(returnArray, src.Series[index].Data[j])
		}
		// Adding Time range
		if alignFlag == 0 {
			timeRangeAppend := make([]byte, 8)
			binary.LittleEndian.PutUint64(timeRangeAppend, uint64(src.Series[index].TimeRange.Start))
			returnArray = append(returnArray, timeRangeAppend...)
			binary.LittleEndian.PutUint64(timeRangeAppend, uint64(src.Series[index].TimeRange.End))
			returnArray = append(returnArray, timeRangeAppend...)
		}

		index += 1
	}
	return returnArray, nil
}

func (m EncoderDecoder) Decode(src []byte) (dst framer.Frame, err error) {
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
			if index < len(src) && channel.Key(binary.LittleEndian.Uint32(src[index:])) == m.keys[i] {
				returnStruct.Keys = append(returnStruct.Keys, channel.Key(binary.LittleEndian.Uint32(src[index:])))
				index += 4
			} else {
				continue
			}
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
