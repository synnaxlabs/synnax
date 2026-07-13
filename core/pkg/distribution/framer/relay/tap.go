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
	"github.com/synnaxlabs/x/errors"
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
	// demands track the current channels demanded by each entity.
	demands map[address.Address]channel.Keys
	// taps tracks the current taps we have open.
	taps map[node.Key]tap
	Config
}

func newTapper(config Config) confluence.Segment[demand, Response] {
	t := &tapper{
		Config:  config,
		demands: make(map[address.Address]channel.Keys),
		taps:    make(map[node.Key]tap),
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
	host := t.HostResolver.HostKey()
	nodeDemands := make(map[node.Key]channel.Keys, len(t.taps))
	for _, d := range t.demands {
		for _, k := range d {
			nk := k.Lease()
			// Free channels are registered in every node's local storage, so their
			// writes are served by the gateway tap.
			if nk.IsFree() {
				nk = host
			}
			nodeDemands[nk] = append(nodeDemands[nk], k)
		}
	}
	return nodeDemands
}

// Flow starts the tapper goroutines, which listen for demands that update relevant taps
// into remote nodes or the host time-series db.
func (t *tapper) Flow(sCtx signal.Context, opts ...confluence.Option) {
	t.UnarySink.Flow(sCtx, append(opts,
		// Order is very important here, we need to make sure the tapper deferral runs
		// before we close the inlet to the delta.
		confluence.WithClosables(t.Out),
		confluence.Defer(t.close),
	)...)
}

func (t *tapper) close() {
	if len(t.taps) > 0 {
		panic("[relay] - tapper closed with open taps")
	}
}

func (t *tapper) updateTaps(
	ctx context.Context,
	nodeDemands map[node.Key]channel.Keys,
) {
	// Open any new taps we may need
	for nk, keys := range nodeDemands {
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

	for nk, tc := range t.taps {
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
	if nodeKey == t.HostResolver.HostKey() {
		return t.tapIntoGateway(ctx, keys)
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
// the given channels. Keys that do not resolve in local storage (e.g. a channel that
// was deleted after a streamer demanded it) are dropped with a warning rather than
// failing the tap, so a single stale key cannot starve every streamer demanding host
// channels.
func (t *tapper) tapIntoGateway(ctx context.Context, keys channel.Keys) (tap, error) {
	for {
		valid, err := t.filterToStorage(ctx, keys)
		if err != nil {
			return tap{}, err
		}
		str, err := cesium.NewTranslatedStreamer(
			t.TS,
			ts.StreamerConfig{Channels: valid.Storage()},
			reqToStorage,
			resFromStorage,
		)
		if err == nil {
			requests, closer := t.startTap(str, "gateway_tap")
			return tap{Closer: closer, requests: requests}, nil
		}
		if !errors.Is(err, ts.ErrChannelNotFound) {
			return tap{}, err
		}
		// A channel was deleted between the existence check and opening the streamer.
		// Retry against the latest storage state; the valid set strictly shrinks on
		// each pass, so this terminates.
		keys = valid
	}
}

// filterToStorage returns the subset of keys that resolve in the host's storage
// engine, logging a warning naming any dropped keys.
func (t *tapper) filterToStorage(
	ctx context.Context,
	keys channel.Keys,
) (channel.Keys, error) {
	valid := make(channel.Keys, 0, len(keys))
	var dropped channel.Keys
	for _, key := range keys {
		if _, err := t.TS.RetrieveChannel(ctx, key.StorageKey()); err != nil {
			if !errors.Is(err, ts.ErrChannelNotFound) {
				return nil, err
			}
			dropped = append(dropped, key)
		} else {
			valid = append(valid, key)
		}
	}
	if len(dropped) > 0 {
		t.L.Warn(
			"dropping channels not found in local storage from gateway tap",
			zap.Uint32s("keys", dropped.Uint32()),
		)
	}
	return valid, nil
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
