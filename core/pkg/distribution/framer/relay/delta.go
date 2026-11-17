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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// subscriber is a streamer's connection to the delta.
type subscriber struct {
	responses chan Response
	// accepts reports whether the streamer wants a response. It runs on the delta's
	// goroutine, so it must be safe to call concurrently with the streamer.
	accepts func(Response) bool
	addr    address.Address
	// dropping is true from the first dropped response until the buffer empties.
	dropping bool
}

// delta copies each relay response into the buffer of every subscriber that accepts
// it. It never waits for a subscriber: a full buffer drops the response for that
// subscriber alone, so a slow streamer does not delay the relay or other streamers.
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
		close(s.responses)
	}
	d.subscribers = nil
}
