package telem

import (
	"github.com/arya-analytics/x/query"
)

const (
	dataRateOptKey query.OptionKey = "telem.dataRate"
	timeRangeKey   query.OptionKey = "telem.timeRange"
	dataTypeKey    query.OptionKey = "telem.dataType"
)

func SetDataRate(q query.Query, dr DataRate) { q.Set(dataRateOptKey, dr) }

func SetTimeRange(q query.Query, tr TimeRange) { q.Set(timeRangeKey, tr) }

func SetDataType(q query.Query, dt DataType) { q.Set(dataTypeKey, dt) }

func GetDataRate(q query.Query) (DataRate, error) {
	if v, ok := q.Get(dataRateOptKey); ok {
		return v.(DataRate), nil
	}
	return 0, InvalidDataRate
}

func GetTimeRange(q query.Query) (TimeRange, error) {
	if v, ok := q.Get(timeRangeKey); ok {
		return v.(TimeRange), nil
	}
	return TimeRangeZero, InvalidTimeRange
}

func GetDataType(q query.Query) (DataType, error) {
	if v, ok := q.Get(dataTypeKey); ok {
		return v.(DataType), nil
	}
	return 0, InvalidDataType
}
