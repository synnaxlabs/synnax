package cesium

import "github.com/arya-analytics/x/telem"

type (
	TimeStamp = telem.TimeStamp
	TimeRange = telem.TimeRange
	TimeSpan  = telem.TimeSpan
	Rate      = telem.Rate
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
	Bit64        = telem.Bit64
	Bit32        = telem.Bit32
	Bit16        = telem.Bit16
	Bit8         = telem.Bit8
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
