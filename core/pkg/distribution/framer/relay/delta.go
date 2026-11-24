// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// subscriber is a streamer's connection to the delta.
type subscriber struct {
	responses chan Response
	// accepts reports whether the streamer wants a response. It runs on the delta's
	// goroutine, so it must be safe to call concurrently with the streamer.
	accepts func(Response) bool
	// missed receives a value for each dropped response when missedRequested is true.
	// It holds one value, so drops that the streamer has not seen yet share one value.
	missed          chan struct{}
	missedRequested bool
	addr            address.Address
	// dropping is true from the first dropped response until the buffer empties.
	dropping bool
	// dropped is true once disconnectAll closes responses. Only the delta goroutine
	// reads or writes it, and a delta restarted after a panic runs on the same one.
	dropped bool
	// mu guards missedRanges and orders the signal of a drop after its ranges.
	mu sync.Mutex
	// missedRanges holds the time range of each channel's dropped samples since the
	// last takeMissed.
	missedRanges map[channel.Key]telem.TimeRange
}

// recordDrop adds the time ranges of the series of a dropped response to the missed
// ranges and signals missed. A zero range stays zero, since part of the dropped
// samples have no known time.
func (s *subscriber) recordDrop(res Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.missedRanges == nil {
		s.missedRanges = make(map[channel.Key]telem.TimeRange)
	}
	for key, series := range res.Frame.Entries() {
		if series.Len() == 0 {
			continue
		}
		held, ok := s.missedRanges[key]
		switch {
		case !ok:
			s.missedRanges[key] = series.TimeRange
		case held.IsZero() || series.TimeRange.IsZero():
			s.missedRanges[key] = telem.TimeRange{}
		default:
			s.missedRanges[key] = held.Union(series.TimeRange)
		}
	}
	select {
	case s.missed <- struct{}{}:
	default:
	}
}

// takeMissed returns the missed ranges of every drop signaled so far and clears them.
// It takes a signal still pending on missed, since the returned ranges cover it. It
// takes nothing and returns false while responses holds frames, since they can come
// before a recorded drop. Only the receiver of responses may call it.
func (s *subscriber) takeMissed() (map[channel.Key]telem.TimeRange, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.responses) > 0 {
		return nil, false
	}
	select {
	case <-s.missed:
	default:
	}
	ranges := s.missedRanges
	s.missedRanges = nil
	return ranges, true
}

// delta copies each relay response into the buffer of every subscriber that accepts
// it. It never waits for a subscriber: a full buffer drops the response for that
// subscriber alone, so a slow streamer does not delay the relay or other streamers.
// Each drop notifies the subscriber's missed channel.
type delta struct {
	confluence.UnarySink[Response]
	ins            alamos.Instrumentation
	connections    chan *subscriber
	disconnections chan *subscriber
	subscribers    []*subscriber
}

func newDelta(ins alamos.Instrumentation) *delta {
	return &delta{
		ins:            ins,
		connections:    make(chan *subscriber),
		disconnections: make(chan *subscriber),
	}
}

func (d *delta) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	ctx.Go(func(ctx context.Context) error {
		defer d.disconnectAll()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case s := <-d.connections:
				d.subscribers = append(d.subscribers, s)
			case s := <-d.disconnections:
				d.disconnect(s)
			case res, ok := <-d.In.Outlet():
				if !ok {
					return nil
				}
				d.send(res)
			}
		}
	}, o.Signal...)
}

func (d *delta) send(res Response) {
	for _, s := range d.subscribers {
		if !s.accepts(res) {
			continue
		}
		if len(s.responses) == 0 {
			s.dropping = false
		}
		select {
		case s.responses <- res:
		default:
			if s.missedRequested {
				s.recordDrop(res)
			}
			if !s.dropping {
				s.dropping = true
				d.ins.L.Warn(fmt.Sprintf(
					"relay: dropping frames for slow streamer %s",
					s.addr,
				))
			}
		}
	}
}

func (d *delta) disconnect(s *subscriber) {
	i := slices.Index(d.subscribers, s)
	if i == -1 {
		if s.dropped {
			return
		}
		d.ins.L.DPanic(fmt.Sprintf(
			"relay: disconnected streamer %s that was never connected",
			s.addr,
		))
		return
	}
	d.subscribers = slices.Delete(d.subscribers, i, i+1)
	close(s.responses)
}

func (d *delta) disconnectAll() {
	for _, s := range d.subscribers {
		s.dropped = true
		close(s.responses)
	}
	d.subscribers = nil
}
