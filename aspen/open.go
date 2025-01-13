// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen

import (
	"context"
	"io"

	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/signal"
)

func Open(
	ctx context.Context,
	dirname string,
	addr address.Address,
	peers []address.Address,
	opts ...Option,
) (*DB, error) {
	var (
		o   = newOptions(dirname, addr, peers, opts...)
		d   = &DB{}
		err error
	)
	if err = openStorageEngine(o); err != nil {
		return nil, err
	}
	d.transportCloser, err = configureTransport(ctx, o)
	if err != nil {
		return nil, err
	}
	c, err := cluster.Open(ctx, o.cluster)
	d.Cluster = c
	if err != nil {
		return nil, err
	}
	o.kv.Cluster = c
	db, err := kv.Open(ctx, o.kv)
	d.DB = db
	return d, err
}

func openStorageEngine(opts *options) error {
	if opts.kv.Engine == nil {
		pebbleDB, err := pebble.Open(opts.dirname, &pebble.Options{FS: opts.fs})
		if err != nil {
			return err
		}
		opts.kv.Engine = pebblekv.Wrap(pebbleDB)
	}
	opts.cluster.Storage = opts.kv.Engine
	return nil
}

func configureTransport(ctx context.Context, o *options) (io.Closer, error) {
	sCtx, cancel := signal.WithCancel(
		o.T.Transfer(ctx, context.Background()),
		signal.WithInstrumentation(o.Instrumentation),
	)
	transportShutdown := signal.NewShutdown(sCtx, cancel)
	if err := o.transport.Configure(sCtx, o.addr, o.transport.external); err != nil {
		return transportShutdown, err
	}
	mw, err := falamos.Middleware(falamos.Config{Instrumentation: o.Instrumentation})
	if err != nil {
		return transportShutdown, err
	}
	o.transport.Use(mw)
	o.cluster.Gossip.TransportClient = o.transport.GossipClient()
	o.cluster.Gossip.TransportServer = o.transport.GossipServer()
	o.cluster.Pledge.TransportClient = o.transport.PledgeClient()
	o.cluster.Pledge.TransportServer = o.transport.PledgeServer()
	o.kv.BatchTransportServer = o.transport.TxServer()
	o.kv.BatchTransportClient = o.transport.TxClient()
	o.kv.LeaseTransportServer = o.transport.LeaseServer()
	o.kv.LeaseTransportClient = o.transport.LeaseClient()
	o.kv.FeedbackTransportServer = o.transport.FeedbackServer()
	o.kv.FeedbackTransportClient = o.transport.FeedbackClient()
	o.kv.RecoveryTransportServer = o.transport.RecoveryServer()
	o.kv.RecoveryTransportClient = o.transport.RecoveryClient()
	return transportShutdown, nil
}
