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
	// sent counts the responses the delta has put in responses. Only the delta
	// goroutine reads or writes it.
	sent uint64
	// mu guards drops and orders the signal of a drop after its record.
	mu sync.Mutex
	// drops holds the drops since the last takeMissed, oldest first, with one entry
	// for each run of drops with no response put in responses between them.
	drops []drop
}

// drop is a run of dropped responses.
type drop struct {
	// due is the number of responses put in responses before the run.
	due uint64
	// ranges holds the time range of each channel's dropped samples.
	ranges map[channel.Key]telem.TimeRange
}

// addRange adds tr to the range of key in ranges. A zero range stays zero, since part
// of the samples have no known time.
func addRange(
	ranges map[channel.Key]telem.TimeRange,
	key channel.Key,
	tr telem.TimeRange,
) {
	held, ok := ranges[key]
	switch {
	case !ok:
		ranges[key] = tr
	case held.IsZero() || tr.IsZero():
		ranges[key] = telem.TimeRange{}
	default:
		ranges[key] = held.Union(tr)
	}
}

// recordDrop adds a dropped response to the drops and signals missed.
func (s *subscriber) recordDrop(res Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := len(s.drops); n == 0 || s.drops[n-1].due != s.sent {
		s.drops = append(s.drops, drop{
			due:    s.sent,
			ranges: make(map[channel.Key]telem.TimeRange),
		})
	}
	ranges := s.drops[len(s.drops)-1].ranges
	for key, series := range res.Frame.Entries() {
		if series.Len() > 0 {
			addRange(ranges, key, series.TimeRange)
		}
	}
	select {
	case s.missed <- struct{}{}:
	default:
	}
}

// takeMissed removes the drops whose buffered responses are among the first received
// responses and returns their merged ranges, with ok false if there are none. It takes
// a signal still pending on missed when it removes drops. pending reports whether
// drops remain. Only the receiver of responses may call it.
func (s *subscriber) takeMissed(
	received uint64,
) (ranges map[channel.Key]telem.TimeRange, ok, pending bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for n < len(s.drops) && s.drops[n].due <= received {
		n++
	}
	if n == 0 {
		return nil, false, len(s.drops) > 0
	}
	select {
	case <-s.missed:
	default:
	}
	ranges = s.drops[0].ranges
	for _, d := range s.drops[1:n] {
		for key, tr := range d.ranges {
			addRange(ranges, key, tr)
		}
	}
	s.drops = slices.Delete(s.drops, 0, n)
	return ranges, true, len(s.drops) > 0
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
			s.sent++
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
