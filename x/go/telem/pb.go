// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

func TranslateTimeRangeForward(tr TimeRange) *PBTimeRange {
	return &PBTimeRange{Start: int64(tr.Start), End: int64(tr.End)}
}

func TranslateTimeRangeBackward(tr *PBTimeRange) (otr TimeRange) {
	if tr == nil {
		return otr
	}
	otr.Start = TimeStamp(tr.Start)
	otr.End = TimeStamp(tr.End)
	return
}

func TranslateSeriesForward(s Series) *PBSeries {
	return &PBSeries{
		DataType:  string(s.DataType),
		TimeRange: TranslateTimeRangeForward(s.TimeRange),
		Data:      s.Data,
		Alignment: uint64(s.Alignment),
	}
}

func TranslateManySeriesForward(s []Series) []*PBSeries {
	series := make([]*PBSeries, len(s))
	for i := range s {
		series[i] = TranslateSeriesForward(s[i])
	}
	return series
}

func TranslateSeriesBackward(s *PBSeries) Series {
	return Series{
		DataType:  DataType(s.DataType),
		TimeRange: TranslateTimeRangeBackward(s.TimeRange),
		Data:      s.Data,
		Alignment: Alignment(s.Alignment),
	}
}

func TranslateManySeriesBackward(s []*PBSeries) []Series {
	series := make([]Series, len(s))
	for i := range s {
		series[i] = TranslateSeriesBackward(s[i])
	}
	return series
}
