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
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
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

// tap is a tap into the storage engine or the free write pipeline. A tap streams the
// frames it receives to the relay's delta until closed.
type tap struct {
	io.Closer
	requests confluence.Inlet[Request]
}

// setChannels replaces the set of channels the tap subscribes to by sending a request
// to the underlying segment. The storage tap applies the new set to every frame written
// after setChannels returns (the cesium streamer drains pending requests before
// filtering each frame), which is what lets a demand acknowledged by the tapper
// guarantee its keys are already being filtered in (see demand.ack).
func (t tap) setChannels(ctx context.Context, keys channel.Keys) error {
	return signal.SendUnderContext(ctx, t.requests.Inlet(), Request{Keys: keys})
}

// tapper tracks the channels that readers demand to stream. It taps into the storage
// engine and the free write pipeline to meet these demands, and pipes the frames it
// receives to the relay's delta.
type tapper struct {
	// UnarySink is where we receive demands from.
	confluence.UnarySink[demand]
	// AbstractUnarySource is where we send the frames received from open taps.
	confluence.AbstractUnarySource[Response]
	// freeWrites is where we receive writes from free channels in the distribution
	// write pipeline.
	freeWrites confluence.Outlet[Response]
	// demands track the current channels demanded by each entity.
	demands map[address.Address]channel.Keys
	// storage is the tap into the storage engine. It is open while any reader demands a
	// channel that is not free.
	storage *tap
	// free is the always-open tap for free (virtual) channel writes. updateTaps applies
	// free-channel key updates to it synchronously and lock-free (see
	// freeWriteTap.keys).
	free *freeWriteTap
	// closeFree hard-shuts down the free tap.
	closeFree io.Closer
	Config
}

func newTapper(config Config) confluence.Segment[demand, Response] {
	t := &tapper{
		Config:     config,
		demands:    make(map[address.Address]channel.Keys),
		freeWrites: config.FreeWrites,
	}
	t.Sink = t.sink
	return t
}

func (t *tapper) sink(ctx context.Context, d demand) error {
	stored, free := t.updateDemands(d)
	if err := t.updateTaps(ctx, stored, free); err != nil {
		return err
	}
	if d.ack != nil {
		close(d.ack)
	}
	return nil
}

// updateDemands applies d to the current demands and returns every demanded channel,
// split into the channels that have storage and the free channels.
func (t *tapper) updateDemands(d demand) (stored, free channel.Keys) {
	if d.removed {
		delete(t.demands, d.addr)
	} else {
		t.demands[d.addr] = d.keys
	}
	for _, keys := range t.demands {
		for _, k := range keys {
			if k.Free() {
				free = append(free, k)
			} else {
				stored = append(stored, k)
			}
		}
	}
	return stored, free
}

// Flow starts the tapper goroutines, which listen for demands that update the taps into
// the storage engine and the free write pipeline.
func (t *tapper) Flow(sCtx signal.Context, opts ...confluence.Option) {
	t.free = &freeWriteTap{freeWrites: t.freeWrites}
	_, t.closeFree = t.startTap(sCtx, t.free, "free_write_tap")
	t.UnarySink.Flow(sCtx, append(opts,
		// Order is very important here, we need to make sure the tapper deferral runs
		// before we close the inlet to the delta.
		confluence.WithClosables(t.Out),
		confluence.Defer(t.close),
	)...)
}

func (t *tapper) close() {
	t.closeStorage()
	if err := t.closeFree.Close(); err != nil {
		t.L.Error("failed to close free write tap", zap.Error(err))
	}
}

func (t *tapper) updateTaps(ctx context.Context, stored, free channel.Keys) error {
	// Apply free-channel keys synchronously and lock-free. This is what lets a
	// streamer's readiness acknowledgment guarantee its demanded free channels are
	// already being filtered in by the time the ack fires.
	t.free.setKeys(free)
	if len(stored) == 0 {
		t.closeStorage()
		return nil
	}
	if t.storage != nil {
		return t.storage.setChannels(ctx, stored)
	}
	tc, err := t.tapIntoStorage(ctx, stored)
	if err != nil {
		t.L.Error("failed to open storage tap", zap.Error(err))
		return nil
	}
	t.storage = &tc
	return nil
}

// closeStorage stops the storage tap, if one is open, and waits for it to exit.
func (t *tapper) closeStorage() {
	if t.storage == nil {
		return
	}
	if err := t.storage.Close(); err != nil {
		t.L.Error("tap failed to close", zap.Error(err))
	}
	t.storage = nil
}

// startTap wires the given segment's frames into the tapper's outlet and starts it
// under a child of ctx, returning the segment's request inlet and a closer that
// hard-shuts it down.
func (t *tapper) startTap(
	ctx context.Context,
	seg confluence.Segment[Request, Response],
	tapKey string,
) (confluence.Inlet[Request], io.Closer) {
	requests := confluence.NewStream[Request](1)
	seg.InFrom(requests)
	seg.OutTo(t.Out)
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithInstrumentation(t.Child(tapKey)),
	)
	seg.Flow(
		sCtx,
		confluence.RecoverWithErrOnPanic(),
		confluence.WithAddress(address.Address(tapKey)),
	)
	return requests, signal.NewHardShutdown(sCtx, cancel)
}

// tapIntoStorage opens a new tap over the storage engine, subscribed to the given
// channels.
func (t *tapper) tapIntoStorage(ctx context.Context, keys channel.Keys) (tap, error) {
	str, err := t.TS.NewTranslatedStreamer(
		ts.StreamerConfig{Channels: keys.Storage()},
		reqToStorage,
		resFromStorage,
	)
	if err != nil {
		return tap{}, err
	}
	requests, closer := t.startTap(ctx, str, "storage_tap")
	return tap{Closer: closer, requests: requests}, nil
}

// freeWriteTap filters writes to free channels from the distribution write pipeline
// into the relay's delta, keeping only the channels currently demanded by streamers.
type freeWriteTap struct {
	confluence.AbstractUnarySink[Request]
	confluence.AbstractUnarySource[Response]
	freeWrites confluence.Outlet[Response]
	// keys is the set of free channels currently being filtered in. It is updated
	// synchronously by the tapper (setKeys) and read on every free write, so it is an
	// atomic pointer: the read path takes a single lock-free atomic load and the write
	// path publishes a new set without blocking the tapper. nil means no free channels
	// are demanded yet, in which case all free writes are filtered out.
	keys atomic.Pointer[set.Set[channel.Key]]
}

// setKeys publishes the set of free channels the tap should filter in. It is safe
// to call from the tapper goroutine concurrently with the tap's own read loop.
func (f *freeWriteTap) setKeys(keys channel.Keys) {
	s := set.New(keys...)
	f.keys.Store(&s)
}

func (f *freeWriteTap) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(f.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case _, ok := <-f.In.Outlet():
				// Free-channel keys are applied via setKeys, not through this inlet;
				// we only watch it for closure.
				if !ok {
					return nil
				}
			case req := <-f.freeWrites.Outlet():
				keys := f.keys.Load()
				if keys == nil {
					continue
				}
				req.Frame = req.Frame.KeepKeys(*keys)
				if req.Frame.Empty() {
					continue
				}
				if err := signal.SendUnderContext(ctx, f.Out.Inlet(), req); err != nil {
					return err
				}
			}
		}
	}, append(o.Signal, signal.WithKey("free-write-tap"))...)
}
