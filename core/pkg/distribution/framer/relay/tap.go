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
	"io"
	"sync/atomic"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// demand represents a demand for streaming data from a specific entity. this entity
// should generate a unique address (preferably through address.Rand) and use it
// throughout its lifecycle. To update the requested keys, the entity should send a
// demand with variant Label, and to remove the demand, it should send a demand with
// variant DeleteChannel.
type demand struct {
	change.Change[address.Address, Request]
	// ack, when non-nil, is closed by the tapper after it has fully applied this demand
	// to all local taps. It lets a streamer block until the relay is actually filtering
	// its channels in before acknowledging readiness, closing the window where a write
	// could be dropped between open and the demand propagating. nil for demands that
	// need no acknowledgment (deletes and mid-stream key reconfigurations).
	ack chan struct{}
}

// tap is a tap into a source of frames, whether another node's distribution relay or
// the host's local storage engine. A tap streams the frames it receives to the relay's
// delta until closed.
type tap struct {
	io.Closer
	requests confluence.Inlet[Request]
}

// setChannels replaces the set of channels the tap subscribes to by sending a request
// to the underlying segment. Gateway taps apply the new set to every frame written
// after setChannels returns (the cesium streamer drains pending requests before
// filtering each frame), which is what lets a demand acknowledged by the tapper
// guarantee its keys are already being filtered in (see demand.ack). Peer taps apply
// the new set asynchronously over the network and carry no such guarantee.
func (t tap) setChannels(keys channel.Keys) {
	t.requests.Inlet() <- Request{Keys: keys}
}

// tapper tracks readers demands for channel's to stream. It uses these demands to tap
// into the relays of other nodes and the storage layer to receive frames. It then pipes
// these frames to an outlet, which, in this case is the relay's delta.
type tapper struct {
	// UnarySink is where we receive demands from, using them to update the set of
	// relay's we tap into.
	confluence.UnarySink[demand]
	// AbstractUnarySource is where we send our responses to, which are the frames we
	// receive from the tapController relays.
	confluence.AbstractUnarySource[Response]
	// freeWrites is where we receive writes from free channels in the distribution
	// write pipeline.
	freeWrites confluence.Outlet[Response]
	// demands track the current channels demanded by each entity.
	demands map[address.Address]channel.Keys
	// taps tracks the current taps we have open.
	taps map[node.Key]tap
	// freeTap is the always-open tap for free (virtual) channel writes. It is held
	// directly, rather than only through taps, so updateTaps can apply free-channel
	// key updates synchronously and lock-free (see freeWriteTap.keys).
	freeTap *freeWriteTap
	Config
}

func newTapper(config Config) confluence.Segment[demand, Response] {
	t := &tapper{
		Config:     config,
		demands:    make(map[address.Address]channel.Keys),
		taps:       make(map[node.Key]tap),
		freeWrites: config.FreeWrites,
	}
	t.Sink = t.sink
	return t
}

func (t *tapper) sink(ctx context.Context, d demand) error {
	// update our demands, so we know what channels we want from what nodes
	nodeDemands := t.updateDemands(d)
	// open/close any taps we need to in order to meet the new demands
	t.updateTaps(ctx, nodeDemands)
	if d.ack != nil {
		close(d.ack)
	}
	return nil
}

// updateDemands modifies the current set of locations that the relay needs to stream
// channel data from.
func (t *tapper) updateDemands(d demand) map[node.Key]channel.Keys {
	if d.Variant == change.VariantDelete {
		delete(t.demands, d.Key)
	} else {
		t.demands[d.Key] = d.Value.Keys
	}
	nodeDemands := make(map[node.Key]channel.Keys, len(t.taps))
	for _, d := range t.demands {
		for _, k := range d {
			nodeDemands[k.Lease()] = append(nodeDemands[k.Lease()], k)
		}
	}
	return nodeDemands
}

// Flow starts the tapper goroutines, which listen for demands that update relevant taps
// into remote nodes, the host time-series db, or the free write pipeline.
func (t *tapper) Flow(sCtx signal.Context, opts ...confluence.Option) {
	t.taps[node.KeyFree], _ = t.tapInto(sCtx, node.KeyFree, channel.Keys{})
	t.UnarySink.Flow(sCtx, append(opts,
		// Order is very important here, we need to make sure the tapper deferral runs
		// before we close the inlet to the delta.
		confluence.WithClosables(t.Out),
		confluence.Defer(t.close),
	)...)
}

func (t *tapper) close() {
	if len(t.taps) > 1 {
		panic("[relay] - tapper closed with open taps")
	}
	if err := t.taps[node.KeyFree].Close(); err != nil {
		t.L.Error("failed to close free write tap", zap.Error(err))
	}
}

func (t *tapper) updateTaps(
	ctx context.Context,
	nodeDemands map[node.Key]channel.Keys,
) {
	// Apply free-channel keys synchronously and lock-free. This is what lets a
	// streamer's readiness acknowledgment guarantee its demanded free channels are
	// already being filtered in by the time the ack fires.
	t.freeTap.setKeys(nodeDemands[node.KeyFree])

	// Open any new taps we may need
	for nk, keys := range nodeDemands {
		if nk.IsFree() {
			continue
		}
		if _, ok := t.taps[nk]; !ok {
			tc, err := t.tapInto(ctx, nk, keys)
			if err != nil {
				t.L.Error(
					"failed to open new tap",
					zap.Uint16("node", uint16(nk)),
					zap.Error(err),
				)
			} else {
				t.taps[nk] = tc
			}
		}
	}

	// Update or close any non-free taps. The free tap is driven by setKeys above and
	// is never opened or closed here.
	for nk, tc := range t.taps {
		if nk.IsFree() {
			continue
		}
		if keys, ok := nodeDemands[nk]; ok {
			// If we still need the tap, send the updated key set
			tc.setChannels(keys)
		} else {
			// This does a hard shutdown on the tap, cancelling its context and causing
			// it to immediately exit.
			if err := tc.Close(); err != nil {
				t.L.Error("tap failed to close", zap.Error(err))
			}
			// If we need this tap again, we'll just open it again.
			delete(t.taps, nk)
		}
	}
}

func (t *tapper) tapInto(
	ctx context.Context,
	nodeKey node.Key,
	keys channel.Keys,
) (tap, error) {
	if nodeKey.IsFree() {
		return t.tapIntoFreeWrites()
	}
	if nodeKey == t.HostResolver.HostKey() {
		return t.tapIntoGateway(keys)
	}
	return t.tapIntoPeer(ctx, nodeKey)
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
	seg.Flow(sCtx, confluence.RecoverWithErrOnPanic(), confluence.WithAddress(address.Address(tapKey)))
	return requests, signal.NewHardShutdown(sCtx, cancel)
}

// tapIntoGateway opens a new tap over the host's local storage engine, subscribed to
// the given channels.
func (t *tapper) tapIntoGateway(keys channel.Keys) (tap, error) {
	str, err := cesium.NewTranslatedStreamer(
		t.TS,
		ts.StreamerConfig{Channels: keys.Storage()},
		reqToStorage,
		resFromStorage,
	)
	if err != nil {
		return tap{}, err
	}
	requests, closer := t.startTap(str, "gateway_tap")
	return tap{Closer: closer, requests: requests}, nil
}

// tapIntoPeer opens a new tap that sends requests and receives responses
// over the given stream.
func (t *tapper) tapIntoPeer(ctx context.Context, nodeKey node.Key) (tap, error) {
	addr, err := t.HostResolver.Resolve(nodeKey)
	if err != nil {
		return tap{}, err
	}
	stream, err := t.Transport.Client().Stream(ctx, addr)
	if err != nil {
		return tap{}, err
	}
	receiver := &freightfluence.Receiver[Response]{Receiver: stream}
	sender := &freightfluence.Sender[Request]{Sender: stream}
	p := plumber.New()
	plumber.SetSink[Request](p, "sender", sender)
	plumber.SetSource[Response](p, "receiver", receiver)
	seg := &plumber.Segment[Request, Response]{Pipeline: p}
	lo.Must0(seg.RouteOutletFrom("receiver"))
	lo.Must0(seg.RouteInletTo("sender"))
	requests, closer := t.startTap(seg, fmt.Sprintf("peer_tap_%v", nodeKey))
	return tap{Closer: closer, requests: requests}, nil
}

// tapIntoFreeWrites opens the always-on tap that filters free-channel writes from the
// distribution write pipeline into the relay.
func (t *tapper) tapIntoFreeWrites() (tap, error) {
	f := &freeWriteTap{freeWrites: t.freeWrites}
	t.freeTap = f
	requests, closer := t.startTap(f, "free_write_tap")
	return tap{Closer: closer, requests: requests}, nil
}

// freeWriteTap filters writes to free channels from the distribution write pipeline
// into the relay's delta, keeping only the channels currently demanded by streamers.
type freeWriteTap struct {
	confluence.AbstractUnarySink[Request]
	confluence.AbstractUnarySource[Response]
	freeWrites confluence.Outlet[Response]
	// keys is the set of free channels currently being filtered in. It is updated
	// synchronously by the tapper (setKeys) and read on every free write, so it is
	// an atomic pointer: the read path takes a single lock-free atomic load and the
	// write path publishes a new slice without blocking the tapper. nil means no
	// free channels are demanded yet, in which case all free writes are filtered out.
	keys atomic.Pointer[channel.Keys]
}

// setKeys publishes the set of free channels the tap should filter in. It is safe
// to call from the tapper goroutine concurrently with the tap's own read loop.
func (f *freeWriteTap) setKeys(keys channel.Keys) { f.keys.Store(&keys) }

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
				if !req.Frame.Empty() {
					f.Out.Inlet() <- req
				}
			}
		}
	}, append(o.Signal, signal.WithKey("free-write-tap"))...)
}
