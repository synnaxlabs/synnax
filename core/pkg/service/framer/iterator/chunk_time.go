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

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// chunkTime adds to each data response of a chunked read the time of each timed series,
// which it reads from the series' index over the series' time range. A chunked read
// counts the samples of each channel on its own, so a chunk of an index that other
// channels share holds other samples than a chunk of one of its data channels.
type chunkTime struct {
	confluence.AbstractLinear[Response, Response]
	// indexes reads the stamps of the indexes of the timed keys.
	indexes *framer.Iterator
	keys    set.Set[channel.Key]
	untimed set.Set[channel.Key]
	clocks  set.Set[channel.Key]
}

func newChunkTime(
	ctx context.Context,
	dist *framer.Service,
	cfg Config,
	indexes channel.Keys,
) (*chunkTime, error) {
	it, err := dist.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   indexes,
		Bounds: cfg.Bounds,
	})
	if err != nil {
		return nil, err
	}
	return &chunkTime{
		indexes: it,
		keys:    set.New(cfg.Keys...),
		untimed: set.New(cfg.UntimedKeys...),
		clocks:  set.New(indexes...),
	}, nil
}

// Flow implements confluence.Flow.
func (t *chunkTime) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(t.Out)
	ctx.Go(func(ctx context.Context) (err error) {
		defer func() { err = errors.Combine(err, t.indexes.Close()) }()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-t.In.Outlet():
				if !ok {
					return nil
				}
				if res.Variant == ResponseVariantData {
					if res.Frame, err = t.time(res.Frame); err != nil {
						return err
					}
				}
				if err = signal.SendUnderContext(ctx, t.Out.Inlet(), res); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

// time returns fr with the time of each of its timed series, once per clock and range
// of alignments.
func (t *chunkTime) time(fr frame.Frame) (frame.Frame, error) {
	out := fr
	var added []telem.AlignmentBounds
	for key, s := range fr.Entries() {
		clock := channel.Key(s.Clock)
		if !t.keys.Contains(key) || t.untimed.Contains(key) ||
			!t.clocks.Contains(clock) || s.AlignmentMultiple > 1 {
			continue
		}
		bounds := s.AlignmentBounds()
		if containsBounds(added, bounds) {
			continue
		}
		stamps, ok, err := t.stampsOf(clock, s)
		if err != nil {
			return fr, err
		}
		if ok {
			out = out.Append(clock, stamps)
			added = append(added, bounds)
		}
	}
	return out, nil
}

// stampsOf returns the stamps of clock at the alignments of s, and false when the index
// holds none there.
func (t *chunkTime) stampsOf(
	clock channel.Key,
	s telem.Series,
) (telem.Series, bool, error) {
	t.indexes.SetBounds(s.TimeRange)
	if !t.indexes.SeekFirst() || !t.indexes.Next(iterator.AutoSpan) {
		return telem.Series{}, false, t.indexes.Error()
	}
	bounds := s.AlignmentBounds()
	for _, stamps := range t.indexes.Value().Get(clock).Series {
		held := stamps.AlignmentBounds()
		if stamps.AlignmentMultiple > 1 ||
			held.Lower > bounds.Lower ||
			held.Upper < bounds.Upper {
			continue
		}
		density := int(stamps.DataType.Density())
		start := int(bounds.Lower-held.Lower) * density
		stamps.Data = stamps.Data[start : start+int(s.Len())*density]
		stamps.Alignment, stamps.TimeRange = s.Alignment, s.TimeRange
		return stamps, true, nil
	}
	return telem.Series{}, false, nil
}

func containsBounds(added []telem.AlignmentBounds, b telem.AlignmentBounds) bool {
	for _, a := range added {
		if a == b {
			return true
		}
	}
	return false
}
