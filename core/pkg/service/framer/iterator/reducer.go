// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// groupSizer picks the group size of each channel that an iterator reads raw and
// reduces in the service layer. A channel's group size is the one storage picks for
// the busiest index the channel depends on, so reads of a channel reduce the same way
// with or without calculated channels in the request.
type groupSizer struct {
	framer *framer.Service
	// limit is the point limit and aggregation the reduction targets.
	limit telem.LimitReduction
	// indexes maps each output channel to the index channels its group size follows.
	indexes map[channel.Key][]channel.Key
}

// resolve returns the group size of every output channel for reads over bounds.
func (g *groupSizer) resolve(
	ctx context.Context,
	bounds telem.TimeRange,
) (map[channel.Key]uint32, error) {
	counts, err := g.count(ctx, bounds)
	if err != nil {
		return nil, err
	}
	sizes := make(map[channel.Key]uint32, len(g.indexes))
	for key, indexes := range g.indexes {
		var count int64
		for _, idx := range indexes {
			count = max(count, counts[idx])
		}
		sizes[key] = telem.GroupSize(count, g.limit.PointLimit, g.limit.Aggregation)
	}
	return sizes, nil
}

// count returns the number of samples each index channel holds in bounds, as storage
// counts them when it picks a group size. A decimated read with a limit of one point
// returns one point per domain, labeled with the count as its alignment multiple.
func (g *groupSizer) count(
	ctx context.Context,
	bounds telem.TimeRange,
) (counts map[channel.Key]int64, err error) {
	keys := make(set.Set[channel.Key])
	for _, indexes := range g.indexes {
		keys.Add(indexes...)
	}
	counts = make(map[channel.Key]int64, len(keys))
	if len(keys) == 0 {
		return counts, nil
	}
	iter, err := g.framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   keys.Slice(),
		Bounds: bounds,
		Reduction: telem.Reduction{Variant: telem.LimitReduction{
			Aggregation: telem.AggregationDecimate,
			PointLimit:  1,
		}},
	})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	if !iter.SeekFirst() || !iter.Next(telem.TimeSpanMax) {
		return counts, iter.Error()
	}
	for key, s := range iter.Value().Entries() {
		counts[key] = max(counts[key], int64(s.Multiple()), s.Len())
	}
	return counts, iter.Error()
}

// reducer reduces the series of each response with the group size of its channel.
type reducer struct {
	confluence.LinearTransform[Response, Response]
	aggregation telem.Aggregation
	// groupSizes maps each output channel to its group size. The request path replaces
	// it before forwarding a request with new bounds, so every response to that request
	// sees the new sizes.
	groupSizes atomic.Pointer[map[channel.Key]uint32]
}

func newReducer(agg telem.Aggregation, groupSizes map[channel.Key]uint32) *reducer {
	r := &reducer{aggregation: agg}
	r.groupSizes.Store(&groupSizes)
	r.Transform = r.transform
	return r
}

func (r *reducer) transform(
	_ context.Context,
	in Response,
) (Response, bool, error) {
	if in.Variant != ResponseVariantData {
		return in, true, nil
	}
	var (
		sizes = *r.groupSizes.Load()
		out   = frame.Alloc(in.Frame.Count())
		done  = make(set.Set[channel.Key], in.Frame.Count())
	)
	for _, key := range in.Frame.KeysSlice() {
		if done.Contains(key) {
			continue
		}
		done.Add(key)
		size, ok := sizes[key]
		if !ok {
			return in, false, errors.Newf("no group size for channel %v", key)
		}
		for _, s := range in.Frame.Get(key).Reduce(r.aggregation, size).Series {
			out = out.Append(key, s)
		}
	}
	in.Frame = out
	return in, true, nil
}

// boundsTracker re-resolves the reducer's group sizes when a request changes the
// iterator's bounds.
type boundsTracker struct {
	confluence.LinearTransform[Request, Request]
	sizer   *groupSizer
	reducer *reducer
}

func newBoundsTracker(sizer *groupSizer, r *reducer) *boundsTracker {
	t := &boundsTracker{sizer: sizer, reducer: r}
	t.Transform = t.transform
	return t
}

func (t *boundsTracker) transform(
	ctx context.Context,
	req Request,
) (Request, bool, error) {
	if req.Command != CommandSetBounds {
		return req, true, nil
	}
	sizes, err := t.sizer.resolve(ctx, req.Bounds)
	if err != nil {
		return req, false, err
	}
	t.reducer.groupSizes.Store(&sizes)
	return req, true, nil
}
