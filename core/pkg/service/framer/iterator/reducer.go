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
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// groupSizer picks the group size of each channel that an iterator reads raw and
// reduces in the service layer. A channel's group size is the one storage picks for the
// busiest index the channel depends on, so reads of a channel reduce the same way with
// or without calculated channels in the request.
type groupSizer struct {
	framer *framer.Service
	limit  telem.LimitReduction
	// indexes maps each output channel to the index channels its group size follows.
	indexes map[channel.Key][]channel.Key
}

// resolve returns the group size of every output channel for reads over bounds.
func (g *groupSizer) resolve(
	ctx context.Context,
	bounds telem.TimeRange,
) (map[channel.Key]uint32, error) {
	decimated, err := g.decimate(ctx, bounds)
	if err != nil {
		return nil, err
	}
	sizes := make(map[channel.Key]uint32, len(g.indexes))
	for key, indexes := range g.indexes {
		var size uint32
		for _, idx := range indexes {
			size = max(size, decimated[idx])
		}
		sizes[key] = g.limit.Aggregation.GroupSize(size)
	}
	return sizes, nil
}

// decimate returns the group size storage picks for each index channel when it
// decimates the channel over bounds under the point limit. Every series of the read
// carries that size as its alignment multiple.
func (g *groupSizer) decimate(
	ctx context.Context,
	bounds telem.TimeRange,
) (sizes map[channel.Key]uint32, err error) {
	keys := make(set.Set[channel.Key])
	for _, indexes := range g.indexes {
		keys.Add(indexes...)
	}
	sizes = make(map[channel.Key]uint32, len(keys))
	if len(keys) == 0 {
		return sizes, nil
	}
	iter, err := g.framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   keys.Slice(),
		Bounds: bounds,
		Reduction: telem.Reduction{Variant: telem.LimitReduction{
			Aggregation: telem.AggregationDecimate,
			PointLimit:  g.limit.PointLimit,
		}},
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = errors.Combine(err, iter.Close()); err != nil {
			sizes = nil
		}
	}()
	if !iter.SeekFirst() || !iter.Next(telem.TimeSpanMax) {
		return sizes, iter.Error()
	}
	for key, s := range iter.Value().Entries() {
		sizes[key] = max(sizes[key], s.Multiple())
	}
	return sizes, iter.Error()
}

// reducer reduces the series of each response with the group size of its channel. The
// last reduced series of each channel waits for the next response, and takes the
// points of a group cut between the two responses. A read split into chunks therefore
// reduces the same as one read. The reducer sends the waiting series before any
// response that is not data.
type reducer struct {
	confluence.UnarySink[Response]
	confluence.AbstractUnarySource[Response]
	aggregation telem.Aggregation
	// groupSizes maps each output channel to its group size. The request path replaces
	// it before forwarding a request with new bounds, so every response to that request
	// sees the new sizes.
	groupSizes atomic.Pointer[map[channel.Key]uint32]
	// waiting holds the last reduced series of each channel.
	waiting map[channel.Key]waitingSeries
}

// waitingSeries is a reduced series that waits for the series after it.
type waitingSeries struct {
	series telem.Series
	// tail summarizes the last group of the series when the series holds only part of
	// it.
	tail *telem.Group
}

func newReducer(agg telem.Aggregation, groupSizes map[channel.Key]uint32) *reducer {
	r := &reducer{aggregation: agg, waiting: make(map[channel.Key]waitingSeries)}
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
	if len(r.waiting) > 0 {
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

// reduce reduces the series of fr and returns the series that no longer wait.
func (r *reducer) reduce(fr framer.Frame) (framer.Frame, error) {
	var (
		sizes = *r.groupSizes.Load()
		out   = frame.Alloc(fr.Count())
	)
	for key, s := range fr.Entries() {
		size, ok := sizes[key]
		if !ok {
			return framer.Frame{}, errors.Newf("no group size for channel %v", key)
		}
		if size < 2 || !telem.Reducible(s.DataType) {
			out = out.Append(key, s)
			continue
		}
		// An empty series would end the wait of the series before it.
		if s.Len() == 0 {
			continue
		}
		next, head, tail := s.ReduceCut(r.aggregation, size)
		prev, ok := r.waiting[key]
		if ok && prev.tail != nil && head != nil && prev.tail.Start() == head.Start() {
			var merged telem.Group
			prev.series, next, merged = prev.series.AbsorbNext(next, *prev.tail, *head)
			if next.Len() == 0 {
				if tail != nil {
					tail = &merged
				}
				r.waiting[key] = waitingSeries{series: prev.series, tail: tail}
				continue
			}
		}
		if ok {
			out = out.Append(key, prev.series)
		}
		r.waiting[key] = waitingSeries{series: next, tail: tail}
	}
	return out, nil
}

// flush returns and clears every waiting series.
func (r *reducer) flush() framer.Frame {
	out := frame.Alloc(len(r.waiting))
	for key, w := range r.waiting {
		out = out.Append(key, w.series)
	}
	clear(r.waiting)
	return out
}

// requestPlanner adapts the requests of a reduced read that runs calculations. It
// re-resolves the reducer's group sizes when a request changes the iterator's bounds,
// and turns Next or Prev with AutoSpan into a read of every remaining sample in its
// direction, so the reply holds exact groups.
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
	case CommandNext, CommandPrev:
		if req.Span == AutoSpan {
			req.Span = telem.TimeSpanMax
		}
	case CommandSetBounds:
		sizes, err := p.sizer.resolve(ctx, req.Bounds)
		if err != nil {
			return Request{}, false, err
		}
		p.reducer.groupSizes.Store(&sizes)
	}
	return req, true, nil
}
