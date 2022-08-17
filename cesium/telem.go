package cesium

import "github.com/arya-analytics/x/telem"

type (
	TimeStamp = telem.TimeStamp
	TimeRange = telem.TimeRange
	TimeSpan  = telem.TimeSpan
	DataType  = telem.DataType
	DataRate  = telem.DataRate
	Density   = telem.Density
	Size      = telem.Size
)

func Now() TimeStamp { return telem.Now() }

const (
	TimeStampMin = telem.TimeStampMin
	TimeStampMax = telem.TimeStampMax
	TimeSpanMin  = telem.TimeSpanZero
	TimeSpanMax  = telem.TimeSpanMax
	Kilobytes    = telem.Kilobytes
	Float64      = telem.Float64
	Float32      = telem.Float32
	Int64        = telem.Int64
	Int32        = telem.Int32
	Int16        = telem.Int16
	Int8         = telem.Int8
	Uint64       = telem.Uint64
	Uint32       = telem.Uint32
	Uint16       = telem.Uint16
	Uint8        = telem.Uint8
	Microsecond  = telem.Microsecond
	Millisecond  = telem.Millisecond
	Second       = telem.Second
	Minute       = telem.Minute
	Hour         = telem.Hour
	Hz           = telem.Hz
	KHz          = telem.KHz
)

var (
	TimeRangeMax = telem.TimeRangeMax
)
