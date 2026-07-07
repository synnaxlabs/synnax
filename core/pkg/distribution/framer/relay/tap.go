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
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// demand represents a demand for streaming data from a specific entity.
// this entity should generate a unique address (preferably through address.Rand)
// and use it throughout its lifecycle. To update the requested keys, the entity
// should send a demand with variant Label, and to remove the demand, it should
// send a demand with variant DeleteChannel.
type demand struct {
	change.Change[address.Address, Request]
	// ack, when non-nil, is closed by the tapper after it has fully applied this
	// demand to all local taps. It lets a streamer block until the relay is
	// actually filtering its channels in before acknowledging readiness, closing
	// the window where a write could be dropped between open and the demand
	// propagating. nil for demands that need no acknowledgment (deletes and
	// mid-stream key reconfigurations).
	ack chan struct{}
}

// tap is a tap into a relay, whether another node's distribution relay or the hosts
// relay. It can receive updates for channels to stream, and sends frames it receives
// from the relay to an outlet.
type tap = confluence.Segment[Request, Response]

type tapController struct {
	confluence.Inlet[Request]
	closer io.Closer
	// setChannels, when non-nil, synchronously replaces the tap's subscribed
	// channel set. It is set for gateway taps into local storage, where key
	// updates must be applied before a demand is acknowledged (see demand.ack).
	// Taps without setChannels receive key updates asynchronously through the
	// request inlet.
	setChannels func(channel.Keys)
}

// tapper tracks readers demands for channel's to stream. It uses these demands to tap
// into the relays of other nodes and the storage layer to receive frames. It then pipes
// these frames to an outlet, which, in this case is the relay's delta.
type tapper struct {
	// UnarySink is where we receive demands from, using them to update the set of
	// relay's we tap into.
	confluence.UnarySink[demand]
	// AbstractUnarySource is where we send our responses to, which are the frames
	// we receive from the tapController relays.
	confluence.AbstractUnarySource[Response]
	// demands track the current channels demanded by each entity.
	demands map[address.Address]channel.Keys
	// taps tracks the current taps we have open.
	taps map[node.Key]tapController
	Config
}

func newTapper(config Config) confluence.Segment[demand, Response] {
	t := &tapper{
		Config:  config,
		demands: make(map[address.Address]channel.Keys),
		taps:    make(map[node.Key]tapController),
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
			// Free channels are registered transiently in every node's local
			// storage, so their writes are served by the gateway tap.
			if nk.IsFree() {
				nk = host
			}
			nodeDemands[nk] = append(nodeDemands[nk], k)
		}
	}
	return nodeDemands
}

// Flow starts the tapper goroutines, which listen for demands that update relevant
// taps into remote nodes or the host time-series db.
func (t *tapper) Flow(sCtx signal.Context, opts ...confluence.Option) {
	t.UnarySink.Flow(sCtx, append(opts,
		// Order is very important here, we need to make sure the tapper deferral
		// runs before we close the inlet to the delta.
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
				t.L.Error("failed to open new tap", zap.Uint16("node", uint16(nk)))
			} else {
				t.taps[nk] = tc
			}
		}
	}

	for nk, tc := range t.taps {
		if keys, ok := nodeDemands[nk]; ok {
			// If we still need the tap, send the updated key set
			if tc.setChannels != nil {
				// Applied synchronously so that a demand acknowledged by the
				// tapper is guaranteed to already be filtering its keys in - the
				// happens-before barrier SendOpenAck provides for channels served
				// from local storage.
				tc.setChannels(keys)
			} else {
				tc.Inlet.Inlet() <- Request{Keys: keys}
			}
		} else {
			// This does a hard shutdown on the tap, cancelling its context and causing
			// it to immediately exit.
			if err := tc.closer.Close(); err != nil {
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
) (tapController, error) {
	var (
		tp          tap
		err         error
		tapKey      string
		setChannels func(channel.Keys)
	)
	if nodeKey == t.HostResolver.HostKey() {
		tp, setChannels, err = t.tapIntoGateway(keys)
		tapKey = "gateway_tap"
	} else {
		tp, err = t.tapIntoPeer(ctx, nodeKey)
		tapKey = fmt.Sprintf("peer_tap_%v", nodeKey)
	}
	if err != nil {
		return tapController{}, err
	}
	requests := confluence.NewStream[Request](1)
	tp.InFrom(requests)
	tp.OutTo(t.Out)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(t.Child(tapKey)))
	tp.Flow(sCtx, confluence.RecoverWithErrOnPanic(), confluence.WithAddress(address.Address(tapKey)))
	return tapController{
		Inlet:       requests,
		closer:      signal.NewHardShutdown(sCtx, cancel),
		setChannels: setChannels,
	}, nil
}

// tapIntoGateway opens a new tap over the given storage layer streamer. The returned
// function synchronously replaces the tap's subscribed channel set (see
// cesium.Streamer.SetChannels).
func (t *tapper) tapIntoGateway(keys channel.Keys) (tap, func(channel.Keys), error) {
	str, err := cesium.NewTranslatedStreamer(
		t.TS,
		ts.StreamerConfig{Channels: keys.Storage()},
		reqToStorage,
		resFromStorage,
	)
	if err != nil {
		return nil, nil, err
	}
	return str, func(keys channel.Keys) { str.SetChannels(keys.Storage()) }, nil
}

// tapIntoPeer opens a new tap that sends requests and receives responses
// over the given stream.
func (t *tapper) tapIntoPeer(ctx context.Context, nodeKey node.Key) (tap, error) {
	addr, err := t.HostResolver.Resolve(nodeKey)
	if err != nil {
		return nil, err
	}
	stream, err := t.Transport.Client().Stream(ctx, addr)
	if err != nil {
		return nil, err
	}
	receiver := &freightfluence.Receiver[Response]{Receiver: stream}
	sender := &freightfluence.Sender[Request]{Sender: stream}
	p := plumber.New()
	plumber.SetSink[Request](p, "sender", sender)
	plumber.SetSource[Response](p, "receiver", receiver)
	seg := &plumber.Segment[Request, Response]{Pipeline: p}
	lo.Must0(seg.RouteOutletFrom("receiver"))
	lo.Must0(seg.RouteInletTo("sender"))
	return seg, nil
}
