// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"io"
)

// demand represents a demand for streaming data from a specific entity.
// this entity should generate a unique address (preferably through address.Rand)
// and use it throughout its lifecycle. To update the requested keys, the entity
// should send a demand with variant Label, and to remove the demand, it should
// send a demand with variant DeleteChannel.
type demand = changex.Change[address.Address, Request]

// tap is a tap into a relay, whether another node's distribution relay or the hosts
// relay. It can receive updates for channels to stream, and sends frames it receives
// from the relay to an outlet.
type tap = confluence.Segment[Request, Response]

type tapController struct {
	confluence.Inlet[Request]
	closer io.Closer
}

// tapper tracks readers demands for channel's to stream. It uses these demands to tap
// into the relays of other nodes and the storage layer to receive frames. It then pipes
// these frames to an outlet, which, in this case is the relay's delta.
type tapper struct {
	Config
	// UnarySink is where we receive demands from, using them to update the set of
	// relay's we tap into.
	confluence.UnarySink[demand]
	// AbstractUnarySource is where we send our responses to, which are the frames
	// we receive from the tapController relays.
	confluence.AbstractUnarySource[Response]
	// demands track the current channels demanded by each entity.
	demands map[address.Address]channel.Keys
	// taps tracks the current taps we have open.
	taps map[core.NodeKey]tapController
	// freeWrites is where we receive writes from free channels in the distribution
	// write pipeline.
	freeWrites confluence.Outlet[Response]
}

func newTapper(config Config) confluence.Segment[demand, Response] {
	t := &tapper{
		Config:     config,
		demands:    make(map[address.Address]channel.Keys),
		taps:       make(map[core.NodeKey]tapController),
		freeWrites: config.FreeWrites,
	}
	t.Sink = t.sink
	return t
}

func (t *tapper) sink(ctx context.Context, d demand) error {
	// update our demands, so we know what channels we want from what nodes
	nodeDemands := t.updateDemands(d)
	// make sure we have open taps to all demanded nodes/channels
	t.updateTaps(ctx, nodeDemands)
	return nil
}

func (t *tapper) updateDemands(d demand) map[core.NodeKey]channel.Keys {
	if d.Variant == changex.Delete {
		delete(t.demands, d.Key)
	} else {
		t.demands[d.Key] = d.Value.Keys
	}
	nodeDemands := make(map[core.NodeKey]channel.Keys, len(t.taps))
	for _, d := range t.demands {
		for _, k := range d {
			nodeDemands[k.Lease()] = append(nodeDemands[k.Lease()], k)
		}
	}
	return nodeDemands
}

func (t *tapper) Flow(sCtx signal.Context, opts ...confluence.Option) {
	t.taps[core.Free], _ = t.tapInto(sCtx, core.Free, channel.Keys{})
	// HERE
	t.UnarySink.Flow(sCtx, append(opts,
		// Order is very important here, we need to make sure the tapper deferral
		// runs before we close the inlet to the delta.
		confluence.WithClosables(t.AbstractUnarySource.Out),
		confluence.Defer(t.close),
	)...)
}

func (t *tapper) close() {
	if len(t.taps) > 1 {
		panic("[relay] - tapper closed with open taps")
	}
	if err := t.taps[core.Free].closer.Close(); err != nil {
		t.L.Error("failed to close free write tap", zap.Error(err))
	}
}

func (t *tapper) updateTaps(
	ctx context.Context,
	nodeDemands map[core.NodeKey]channel.Keys,
) {
	// Open any new taps we may need
	for node, keys := range nodeDemands {
		if _, ok := t.taps[node]; !ok && !node.IsFree() {
			tc, err_ := t.tapInto(ctx, node, keys)
			if err_ != nil {
				t.L.Error("failed to open new tap", zap.Uint16("node", uint16(node)))
			} else {
				t.taps[node] = tc
			}
		}
	}

	// Update or close any taps we don't need
	for nk, tc := range t.taps {
		if keys, ok := nodeDemands[nk]; ok {
			// If we still need the tap, send the updated key set
			tc.Inlet.Inlet() <- Request{Keys: keys}
		} else if !nk.IsFree() {
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
	nodeKey core.NodeKey,
	keys channel.Keys,
) (tapController, error) {
	var (
		tp  tap
		err error
	)
	if nodeKey.IsFree() {
		tp, err = t.tapIntoFreeWrites()
	} else if nodeKey == t.HostResolver.HostKey() {
		tp, err = t.tapIntoGateway(ctx, keys)
	} else {
		tp, err = t.tapIntoPeer(ctx, nodeKey)
	}
	if err != nil {
		return tapController{}, err
	}
	requests := confluence.NewStream[Request](defaultBuffer)
	tp.InFrom(requests)
	tp.OutTo(t.AbstractUnarySource.Out)
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(t.Instrumentation.Child(fmt.Sprintf("tap-%v", nodeKey))),
	)
	tp.Flow(sCtx, confluence.RecoverWithErrOnPanic())
	return tapController{Inlet: requests, closer: signal.NewShutdown(sCtx, cancel)}, nil
}

// tapIntoGateway opens a new tap over the given storage layer streamer.
func (t *tapper) tapIntoGateway(ctx context.Context, keys channel.Keys) (tap, error) {
	sr, err := t.TS.NewStreamer(ctx, ts.StreamerConfig{Channels: keys.Storage()})
	return confluence.NewTranslator(sr, reqToStorage, resFromStorage), err
}

// tapIntoPeer opens a new tap that sends requests and receives responses
// over the given stream.
func (t *tapper) tapIntoPeer(ctx context.Context, nodeKey core.NodeKey) (tap, error) {
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

func (t *tapper) tapIntoFreeWrites() (tap, error) {
	return &freeWriteTap{freeWrites: t.freeWrites}, nil
}

type freeWriteTap struct {
	confluence.AbstractUnarySink[Request]
	confluence.AbstractUnarySource[Response]
	freeWrites confluence.Outlet[Response]
	keys       channel.Keys
}

func (f *freeWriteTap) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(f.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-f.In.Outlet():
				if !ok {
					return nil
				}
				f.keys = req.Keys
			case req := <-f.freeWrites.Outlet():
				req.Frame = req.Frame.FilterKeys(f.keys)
				if len(req.Frame.Keys) != 0 {
					f.Out.Inlet() <- req
				}
			}
		}
	}, o.Signal...)
}
