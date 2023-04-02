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
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
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
) (DB, error) {
	o := newOptions(dirname, addr, peers, opts...)

	d := &db{}

	if err := openStorageEngine(o); err != nil {
		return nil, err
	}

	transportCtx, cancel := signal.WithCancel(ctx)
	d.transport.wg = transportCtx
	d.transport.shutdown = cancel

	if err := configureTransport(transportCtx, o); err != nil {
		return nil, err
	}

	c, err := cluster.Join(ctx, o.cluster)
	if err != nil {
		return nil, err
	}
	o.kv.Cluster = c

	d.DB, err = kv.Open(ctx, o.kv)
	if err != nil {
		return nil, err
	}

	return d, nil
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

func configureTransport(ctx signal.Context, o *options) error {
	if err := o.transport.Configure(ctx, o.addr, o.externalTransport); err != nil {
		return err
	}
	o.cluster.Gossip.TransportClient = o.transport.GossipClient()
	o.cluster.Gossip.TransportServer = o.transport.GossipServer()
	o.cluster.Pledge.TransportClient = o.transport.PledgeClient()
	o.cluster.Pledge.TransportServer = o.transport.PledgeServer()
	o.kv.BatchTransportServer = o.transport.BatchServer()
	o.kv.BatchTransportClient = o.transport.BatchClient()
	o.kv.LeaseTransportServer = o.transport.LeaseServer()
	o.kv.LeaseTransportClient = o.transport.LeaseClient()
	o.kv.FeedbackTransportServer = o.transport.FeedbackServer()
	o.kv.FeedbackTransportClient = o.transport.FeedbackClient()
	return nil
}
