// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"fmt"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve ranges from the cluster using a builder pattern.
type Retrieve struct {
	gorp    gorp.Retrieve[Key, Range]
	indexes indexes
	// now is the Core's clock when the query opened. Every filter of the query that
	// reads the time reads now.
	now telem.TimeStamp
}

type bound = gorp.Bound[telem.TimeStamp]

// MatchOverlaps returns a filter that matches ranges whose TimeRange overlaps tr.
func MatchOverlaps(tr telem.TimeRange) Filter {
	valid := tr.MakeValid()
	return func(r Retrieve) gorp.Filter[Key, Range] {
		return gorp.And(
			r.indexes.timeRangeEnd.FilterRange(&bound{Value: valid.Start}, nil),
			r.indexes.timeRangeStart.FilterRange(nil, &bound{Value: valid.End}),
			gorp.Match(func(_ gorp.Context, rng *Range) (bool, error) {
				return rng.TimeRange.OverlapsWith(tr), nil
			}),
		)
	}
}

// MatchStages returns a filter for ranges in any of the given stages when the query
// opened. It panics on an invalid stage.
func MatchStages(stages ...Stage) Filter {
	return func(r Retrieve) gorp.Filter[Key, Range] {
		fs := make([]gorp.Filter[Key, Range], len(stages))
		for i, s := range stages {
			fs[i] = r.matchStage(s)
		}
		return gorp.Or(fs...)
	}
}

// MatchRecent returns a filter for ranges that overlap the span of time that ends when
// the query opened. A negative span starts when the query opened and reaches forward.
func MatchRecent(span telem.TimeSpan) Filter {
	return func(r Retrieve) gorp.Filter[Key, Range] {
		if span >= 0 {
			return MatchOverlaps(r.now.Add(-span).Range(r.now))(r)
		}
		// Negating the minimum span overflows, so step one nanosecond out first.
		return MatchOverlaps(r.now.Range(r.now.Add(1).Add(-(span + 1))))(r)
	}
}

func (r Retrieve) matchStage(s Stage) gorp.Filter[Key, Range] {
	var (
		upToNow  = &bound{Value: r.now}
		afterNow = &bound{Value: r.now, Exclusive: true}
	)
	switch s {
	case StageToDo:
		return r.indexes.timeRangeStart.FilterRange(afterNow, nil)
	case StageInProgress:
		return gorp.And(
			r.indexes.timeRangeStart.FilterRange(nil, upToNow),
			r.indexes.timeRangeEnd.FilterRange(afterNow, nil),
		)
	case StageCompleted:
		return r.indexes.timeRangeEnd.FilterRange(nil, upToNow)
	}
	panic(fmt.Sprintf("ranger: invalid stage %q", s))
}
