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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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

// MatchAncestors returns a filter for the ranges nested at any depth in any of the
// given ranges.
func MatchAncestors(keys ...Key) Filter {
	return func(r Retrieve) gorp.Filter[Key, Range] {
		return gorp.MatchKeysFunc[Key, Range](func(ctx gorp.Context) ([]Key, error) {
			return r.descendants(ctx, keys)
		})
	}
}

func (r Retrieve) descendants(ctx gorp.Context, keys []Key) ([]Key, error) {
	found, err := walk(keys, func(parents []Key) ([]Key, error) {
		return r.indexes.parent.Get(ctx, ctx.Snapshot, parents...)
	})
	if !errors.Is(err, gorp.ErrIndexInvalid) {
		return found, err
	}
	// Scan when the index failed to populate, as an index Filter does.
	var ranges []Range
	if err = gorp.NewRetrieve[Key, Range]().
		Entries(&ranges).
		Exec(ctx, ctx.Snapshot); err != nil {
		return nil, err
	}
	children := make(map[Key][]Key)
	for _, rng := range ranges {
		if rng.Parent != nil {
			children[*rng.Parent] = append(children[*rng.Parent], rng.Key)
		}
	}
	return walk(keys, func(parents []Key) ([]Key, error) {
		return lo.FlatMap(parents, func(p Key, _ int) []Key { return children[p] }), nil
	})
}

// walk returns the keys below keys at any depth, where children returns the keys one
// level below parents.
func walk(keys []Key, children func(parents []Key) ([]Key, error)) ([]Key, error) {
	// A given range can be nested in another, so the walk can reach it twice.
	found := make(set.Set[Key])
	for frontier := keys; len(frontier) > 0; {
		next, err := children(frontier)
		if err != nil {
			return nil, err
		}
		frontier = nil
		for _, c := range next {
			if !found.Contains(c) {
				found.Add(c)
				frontier = append(frontier, c)
			}
		}
	}
	return found.Slice(), nil
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
