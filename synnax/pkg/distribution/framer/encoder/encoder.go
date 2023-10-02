package encoder

import (
	"encoding/binary"
	"errors"
	"fmt"
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
		curDataSize                      int             = len(src.Series[0].Data)
		startTime, endTime               telem.TimeStamp = src.Series[0].TimeRange.Start, src.Series[0].TimeRange.End
		returnArray                      []byte
		sizeFlag, alignFlag, channelFlag int = 1, 1, 0
		flagSwitch                       int = 0 // SameData = int[2]  strongAligned = int[1] allChannels = int[0]
	)
	channelFlag |= map[bool]int{false: 0, true: 1}[len(m.keys) == len(src.Keys)] | (1 << 1) | (1 << 2)
	for i := 0; i < len(src.Series); i++ {
		sizeFlag &= (map[bool]int{false: 0, true: 1}[len(src.Series[i].Data) == curDataSize]) << 2
		alignFlag &= (map[bool]int{false: 0, true: 1}[src.Series[i].TimeRange.Start == startTime && src.Series[i].TimeRange.End == endTime]) << 1
	}

	flagSwitch |= (sizeFlag << 2) | (alignFlag << 1) | channelFlag
	// First byte for any is
	// sameData_strongAligned_allChannels_curDataSize (5 bits)
	switch flagSwitch {
	case 0: // all data length included, timestamps, and keys
		returnArray = append(returnArray, byte((sizeFlag<<2)|(alignFlag<<1)|(channelFlag)))
		for i := 0; i < len(src.Series); i++ {
			lenSeriesData := len(src.Series[i].Data)
			bytesToAppend := make([]byte, 4)
			binary.LittleEndian.PutUint32(bytesToAppend, uint32(lenSeriesData))
			returnArray = append(returnArray, bytesToAppend...)
			for j := 0; j < lenSeriesData; j++ {
				returnArray = append(returnArray, src.Series[i].Data[i])
			}
		}
	case 1: // all data lengths, timestamps
		fmt.Println("It's one.")
	case 2: // all data lengths and keys
		fmt.Println("It's two.")
	case 3: // all data lengths
		fmt.Println("It's three.")
	case 4: // all timestamps and keys
		fmt.Println("It's four.")
	case 5: // all timestamps
		fmt.Println("It's five.")
	case 6: // all keys
		fmt.Println("It's six.")
	case 7: // none
		returnArray = append(returnArray, byte(lookup(curDataSize)))
		fmt.Println("It's seven.")
	default:
		return returnArray, errors.New("not a possible switch output")
	}

	return returnArray, nil

	if (dataSizeFlag == 1) && (stronglyAlignedFlag == 1) {
		// allChannels (1), stronglyAligned (1), Data Size (6)
		returnArray = append(returnArray, byte(curDataSize|(dataSizeFlag<<5)|(stronglyAlignedFlag<<6)|(allChannelKeys<<7)))
		for i := 0; i < len(src.Series); i++ {
			for j := 0; j < len(src.Series[i].Data); j++ {
				returnArray = append(returnArray, src.Series[i].Data[i])
			}
		}
	} else if dataSizeFlag == 1 {
		returnArray = append(returnArray, byte(curDataSize|(dataSizeFlag<<5)|(stronglyAlignedFlag<<6)|(allChannelKeys<<7)))
		for i := 0; i < len(src.Series); i++ {
			for j := 0; j < len(src.Series[i].Data); j++ {
				returnArray = append(returnArray, src.Series[i].Data[i])
			}
			returnArray = append(returnArray, src.Series[i].TimeRange.Start[63:32])
			returnArray = append(returnArray, src.Series[i].TimeRange.Start[31:0])
			returnArray = append(returnArray, src.Series[i].TimeRange.End[63:32])
			returnArray = append(returnArray, src.Series[i].TimeRange.End[31:0])
		}
	} else if stronglyAlignedFlag == 1 {
		returnArray = append(returnArray, byte(0x00|(dataSizeFlag<<5)|(stronglyAlignedFlag<<6)|(allChannelKeys<<7)))
		for i := 0; i < len(src.Series); i++ {
			size := len(src.Series[i].Data)
			returnArray = append(returnArray, byte(size))
			for j := 0; j < size; j++ {
				returnArray = append(returnArray, src.Series[i].Data[i])
			}
		}
	}
	return dst, nil

	// send all the keys
}

func (m EncoderDecoder) Decode(src []byte) (dst framer.Frame, err error) {
	return dst, nil
}
