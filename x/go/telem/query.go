// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"github.com/synnaxlabs/x/query"
)

const (
	rateOptKey   query.Parameter = "telem.rate"
	timeRangeKey query.Parameter = "telem.timeRange"
	densityKey   query.Parameter = "telem.density"
	dataTypeKey  query.Parameter = "telem.dataType"
)

func SetRate(q query.Parameters, dr Rate) { q.Set(rateOptKey, dr) }

func SetTimeRange(q query.Parameters, tr TimeRange) { q.Set(timeRangeKey, tr) }

func SetDensity(q query.Parameters, dt Density) { q.Set(densityKey, dt) }

func SetDataType(q query.Parameters, dt DataType) { q.Set(dataTypeKey, dt) }

func GetRate(q query.Parameters) (Rate, error) {
	if v, ok := q.Get(rateOptKey); ok {
		return v.(Rate), nil
	}
	return 0, InvalidRate
}

func GetTimeRange(q query.Parameters) (TimeRange, error) {
	if v, ok := q.Get(timeRangeKey); ok {
		return v.(TimeRange), nil
	}
	return TimeRangeZero, InvalidTimeRange
}

func GetDensity(q query.Parameters) (Density, error) {
	if v, ok := q.Get(densityKey); ok {
		return v.(Density), nil
	}
	return 0, InvalidDensity
}

func GetDataType(q query.Parameters) (DataType, error) {
	if v, ok := q.Get(dataTypeKey); ok {
		return v.(DataType), nil
	}
	return UnknownT, InvalidDataType
}
