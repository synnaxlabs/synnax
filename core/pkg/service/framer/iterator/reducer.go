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
	"slices"
	"sync/atomic"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
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

// reducer reduces the series of each response with the group size of its channel. A
// group cut by the end of a response waits for the next response to complete it, so a
// read split into pieces reduces the same as one read. The reducer sends the waiting
// groups before any response that is not data.
type reducer struct {
	confluence.UnarySink[Response]
	confluence.AbstractUnarySource[Response]
	aggregation telem.Aggregation
	// groupSizes maps each output channel to its group size. The request path replaces
	// it before forwarding a request with new bounds, so every response to that request
	// sees the new sizes.
	groupSizes atomic.Pointer[map[channel.Key]uint32]
	// tails holds the samples of each channel's last group that are not yet reduced.
	tails map[channel.Key]telem.MultiSeries
}

func newReducer(agg telem.Aggregation, groupSizes map[channel.Key]uint32) *reducer {
	r := &reducer{aggregation: agg, tails: make(map[channel.Key]telem.MultiSeries)}
	r.groupSizes.Store(&groupSizes)
	return r
}

func (r *reducer) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-r.In.Outlet():
				if !ok {
					return nil
				}
				if err := r.process(res); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

func (r *reducer) process(res Response) error {
	if res.Variant == ResponseVariantData {
		fr, err := r.reduce(res.Frame)
		if err != nil {
			return err
		}
		if fr.Count() > 0 {
			res.Frame = fr
			r.Out.Inlet() <- res
		}
		return nil
	}
	if len(r.tails) > 0 {
		r.Out.Inlet() <- Response{
			Variant: ResponseVariantData,
			Command: res.Command,
			SeqNum:  res.SeqNum,
			Frame:   r.flush(),
		}
	}
	r.Out.Inlet() <- res
	return nil
}

// reduce reduces every complete group in fr, joined to the waiting tail of its channel,
// and keeps the last group of each channel as its new tail.
func (r *reducer) reduce(fr framer.Frame) (framer.Frame, error) {
	var (
		sizes = *r.groupSizes.Load()
		out   = frame.Alloc(fr.Count())
		done  = make(set.Set[channel.Key], fr.Count())
	)
	for _, key := range fr.KeysSlice() {
		if done.Contains(key) {
			continue
		}
		done.Add(key)
		size, ok := sizes[key]
		if !ok {
			return out, errors.Newf("no group size for channel %v", key)
		}
		series := fr.Get(key)
		if size < 2 || !telem.Reducible(series.DataType()) {
			for _, s := range series.Series {
				out = out.Append(key, s)
			}
			continue
		}
		series.Series = slices.Concat(r.tails[key].Series, series.Series)
		head, tail := splitLastGroup(series, size)
		if len(tail.Series) > 0 {
			r.tails[key] = tail
		} else {
			delete(r.tails, key)
		}
		for _, s := range head.Reduce(r.aggregation, size).Series {
			out = out.Append(key, s)
		}
	}
	return out, nil
}

// flush reduces and clears every waiting tail.
func (r *reducer) flush() framer.Frame {
	sizes := *r.groupSizes.Load()
	out := frame.Alloc(len(r.tails))
	for key, tail := range r.tails {
		for _, s := range tail.Reduce(r.aggregation, sizes[key]).Series {
			out = out.Append(key, s)
		}
	}
	clear(r.tails)
	return out
}

// splitLastGroup splits m before the group of size groupSize that holds its last
// sample, unless that group is complete. A split series keeps its time range in the
// head, and the tail's time range starts and ends at the series' end.
func splitLastGroup(
	m telem.MultiSeries,
	groupSize uint32,
) (head, tail telem.MultiSeries) {
	upper := m.AlignmentBounds().Upper
	if upper.SampleIndex()%groupSize == 0 {
		return m, telem.MultiSeries{}
	}
	cut := telem.GroupStart(
		telem.NewAlignment(upper.DomainIndex(), upper.SampleIndex()-1),
		groupSize,
	)
	for _, s := range m.Series {
		switch bounds := s.AlignmentBounds(); {
		case bounds.Upper <= cut:
			head.Series = append(head.Series, s)
		case bounds.Lower >= cut:
			tail.Series = append(tail.Series, s)
		default:
			at := int64(cut.SampleIndex()-s.Alignment.SampleIndex()) *
				int64(s.DataType.Density())
			h, t := s, s
			h.Data = s.Data[:at]
			t.Data = s.Data[at:]
			t.Alignment = cut
			t.TimeRange = telem.TimeRange{Start: s.TimeRange.End, End: s.TimeRange.End}
			head.Series = append(head.Series, h)
			tail.Series = append(tail.Series, t)
		}
	}
	return head, tail
}

// requestPlanner adapts the requests of a reduced read that runs calculations. It
// re-resolves the reducer's group sizes when a request changes the iterator's bounds,
// and turns Next with AutoSpan into a read of every remaining sample, so the reply
// holds exact groups.
type requestPlanner struct {
	confluence.LinearTransform[Request, Request]
	sizer   *groupSizer
	reducer *reducer
}

func newRequestPlanner(sizer *groupSizer, r *reducer) *requestPlanner {
	p := &requestPlanner{sizer: sizer, reducer: r}
	p.Transform = p.transform
	return p
}

func (p *requestPlanner) transform(
	ctx context.Context,
	req Request,
) (Request, bool, error) {
	switch req.Command {
	case CommandNext:
		if req.Span == AutoSpan {
			req.Span = telem.TimeSpanMax
		}
	case CommandSetBounds:
		sizes, err := p.sizer.resolve(ctx, req.Bounds)
		if err != nil {
			return req, false, err
		}
		p.reducer.groupSizes.Store(&sizes)
	}
	return req, true, nil
}
