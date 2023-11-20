// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telempb

import (
	"github.com/synnaxlabs/x/telem"
)

func TranslateTimeRangeForward(tr telem.TimeRange) *TimeRange {
	return &TimeRange{Start: int64(tr.Start), End: int64(tr.End)}
}

func TranslateTimeRangeBackward(tr *TimeRange) (otr telem.TimeRange) {
	if tr == nil {
		return otr
	}
	otr.Start = telem.TimeStamp(tr.Start)
	otr.End = telem.TimeStamp(tr.End)
	return
}

func TranslateSeriesForward(s telem.Series) *Series {
	return &Series{
		DataType:  string(s.DataType),
		TimeRange: TranslateTimeRangeForward(s.TimeRange),
		Data:      s.Data,
	}
}

func TranslateManySeriesForward(s []telem.Series) []*Series {
	series := make([]*Series, len(s))
	for i := range s {
		series[i] = TranslateSeriesForward(s[i])
	}
	return series
}

func TranslateSeriesBackward(s *Series) telem.Series {
	return telem.Series{
		DataType:  telem.DataType(s.DataType),
		TimeRange: TranslateTimeRangeBackward(s.TimeRange),
		Data:      s.Data,
	}
}

func TranslateManySeriesBackward(s []*Series) []telem.Series {
	series := make([]telem.Series, len(s))
	for i := range s {
		series[i] = TranslateSeriesBackward(s[i])
	}
	return series
}
