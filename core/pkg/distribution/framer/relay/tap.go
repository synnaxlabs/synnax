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
	"io"
	"sync/atomic"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// demand represents a demand for streaming data from a specific entity. This entity
// should generate a unique address (preferably through address.Rand) and use it
// throughout its lifecycle. To update the requested keys, the entity sends a demand
// with its keys. To remove the demand, it sends one with removed set.
type demand struct {
	// addr is the address of the entity.
	addr address.Address
	// keys are the channels that the entity demands.
	keys channel.Keys
	// removed is true when the entity no longer demands any channels.
	removed bool
	// ack, when non-nil, is closed by the tapper after it has fully applied this demand
	// to its taps. It lets a streamer acknowledge an open or a key update only once the
	// relay is actually filtering its channels in, closing the window where a write
	// could be dropped before the demand propagates. nil for demands that need no
	// acknowledgment.
	ack chan struct{}
}

// tapper tracks the channels that readers demand to stream, and filters the frames that
// the writers of this node write down to those channels for the relay's delta.
type tapper struct {
	// UnarySink is where we receive demands from.
	confluence.UnarySink[demand]
	// AbstractUnarySource is where we send the frames that readers demand.
	confluence.AbstractUnarySource[Response]
	// writes is where we receive the frames that the writers of this node write.
	writes confluence.Outlet[Response]
	// demands track the current channels demanded by each entity.
	demands map[address.Address]channel.Keys
	// tap is the always-open tap of writes. sink applies key updates to it
	// synchronously and lock-free (see writeTap.keys).
	tap *writeTap
	// closeTap hard-shuts down the tap.
	closeTap io.Closer
	Config
}

func newTapper(config Config) confluence.Segment[demand, Response] {
	t := &tapper{
		Config:  config,
		demands: make(map[address.Address]channel.Keys),
		writes:  config.Writes,
	}
	t.Sink = t.sink
	return t
}

// sink applies d. The tap filters in the demanded channels before d's ack closes, so a
// write issued after the ack reaches the reader.
func (t *tapper) sink(_ context.Context, d demand) error {
	t.tap.setKeys(t.updateDemands(d))
	if d.ack != nil {
		close(d.ack)
	}
	return nil
}

// updateDemands applies d to the current demands and returns every demanded channel.
func (t *tapper) updateDemands(d demand) channel.Keys {
	if d.removed {
		delete(t.demands, d.addr)
	} else {
		t.demands[d.addr] = d.keys
	}
	var keys channel.Keys
	for _, demanded := range t.demands {
		keys = append(keys, demanded...)
	}
	return keys
}

// Flow starts the tapper goroutines, which listen for demands that update the tap of
// writes.
func (t *tapper) Flow(sCtx signal.Context, opts ...confluence.Option) {
	t.tap = &writeTap{writes: t.writes}
	_, t.closeTap = t.startTap(t.tap, "write_tap")
	t.UnarySink.Flow(sCtx, append(opts,
		// Order is very important here, we need to make sure the tapper deferral runs
		// before we close the inlet to the delta.
		confluence.WithClosables(t.Out),
		confluence.Defer(t.close),
	)...)
}

func (t *tapper) close() {
	if err := t.closeTap.Close(); err != nil {
		t.L.Error("failed to close write tap", zap.Error(err))
	}
}

// startTap wires the given segment's frames into the tapper's outlet and starts it,
// returning the segment's request inlet and a closer that hard-shuts it down.
func (t *tapper) startTap(
	seg confluence.Segment[Request, Response],
	tapKey string,
) (confluence.Inlet[Request], io.Closer) {
	requests := confluence.NewStream[Request](1)
	seg.InFrom(requests)
	seg.OutTo(t.Out)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(t.Child(tapKey)))
	seg.Flow(
		sCtx,
		confluence.RecoverWithErrOnPanic(),
		confluence.WithAddress(address.Address(tapKey)),
	)
	return requests, signal.NewHardShutdown(sCtx, cancel)
}

// writeTap filters the frames that the writers of this node write into the relay's
// delta, keeping only the channels currently demanded by streamers.
type writeTap struct {
	confluence.AbstractUnarySink[Request]
	confluence.AbstractUnarySource[Response]
	writes confluence.Outlet[Response]
	// keys is the set of channels currently being filtered in. It is updated
	// synchronously by the tapper (setKeys) and read on every write, so it is an atomic
	// pointer: the read path takes a single lock-free atomic load and the write path
	// publishes a new set without blocking the tapper. nil means no channels are
	// demanded yet, in which case all writes are filtered out.
	keys atomic.Pointer[set.Set[channel.Key]]
}

// setKeys publishes the set of channels the tap should filter in. It is safe
// to call from the tapper goroutine concurrently with the tap's own read loop.
func (f *writeTap) setKeys(keys channel.Keys) {
	s := set.New(keys...)
	f.keys.Store(&s)
}

func (f *writeTap) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(f.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case _, ok := <-f.In.Outlet():
				// Keys are applied via setKeys, not through this inlet; we only watch
				// it for closure.
				if !ok {
					return nil
				}
			case req := <-f.writes.Outlet():
				keys := f.keys.Load()
				if keys == nil {
					continue
				}
				req.Frame = req.Frame.KeepKeys(*keys)
				if !req.Frame.Empty() {
					f.Out.Inlet() <- req
				}
			}
		}
	}, append(o.Signal, signal.WithKey("write-tap"))...)
}
