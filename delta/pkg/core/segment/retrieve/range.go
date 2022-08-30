package retrieve

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/telem"
)

type Value interface {
	float64 |
		float32 |
		int64 |
		int32 |
		int16 |
		int8 |
		uint64 |
		uint32 |
		uint16 |
		uint8 |
		byte |
		bool |
		[]byte
}

type RangeHeader struct {
	ChannelKey channel.Key
	Start      telem.TimeStamp
	DataType   telem.DataType
	Rate       telem.Rate
}

type ValueRange[V Value] struct {
	RangeHeader
	Values []V
}

type Sample[V Value] struct {
	TimeStamp telem.TimeStamp
	Value     V
}

type SampleRange[V Value] struct {
	RangeHeader
	Values []Sample[V]
}

type StampValueRange[V Value] struct {
	RangeHeader
	Stamps []telem.TimeStamp
	Values []V
}

type CompoundRange[V Value, F Format[V]] map[channel.Key]F
