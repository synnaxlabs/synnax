// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included Pebble code is copyrighted by the CockroachDB team, and is licensed
// under the BSD 3-Clause License. See the repository file licenses/BSD-3-Clause.txt for
// more information.

package aspen

import (
	"context"

	"github.com/cockroachdb/pebble/v2"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/x/address"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/service"
)

func Open(
	ctx context.Context,
	dirname string,
	addr address.Address,
	peers []address.Address,
	opts ...Option,
) (db *DB, err error) {
	db = &DB{}
	var (
		o           = newOptions(dirname, addr, peers, opts...)
		cleanup, ok = service.NewOpener(ctx, &db.closer)
	)
	defer func() { err = cleanup(err) }()
	// Register the owned gRPC client pool first so it closes LAST. The transport (added
	// below) and any cluster goroutines that hold it must stop using the pool before
	// pool.Close runs.
	if o.transport.ownedPool != nil {
		if !ok(nil, o.transport.ownedPool) {
			return nil, ctx.Err()
		}
	}
	if o.kv.Engine == nil {
		if o.kv.Engine, err = openKV(o); !ok(err, o.kv.Engine) {
			return nil, err
		}
	}
	o.cluster.Storage = o.kv.Engine
	// configureTransport binds the address, so the transport must be registered as a
	// closer here to release it on any later failures.
	if err = configureTransport(o); !ok(err, o.transport) {
		return nil, err
	}
	// The transport binds in configureTransport, so this is the first point at which a
	// operating system-assigned port is known.
	o.cluster.HostAddress = o.transport.Address()
	if db.Cluster, err = cluster.Open(ctx, o.cluster); !ok(err, db.Cluster) {
		return nil, err
	}
	o.kv.Cluster = db.Cluster
	if db.DB, err = kv.Open(ctx, o.kv); !ok(err, db.DB) {
		return nil, err
	}
	if err = o.transport.Serve(); !ok(err, nil) {
		return nil, err
	}

	return db, err
}

func configureTransport(o *options) error {
	if err := o.transport.Configure(
		o.addr, o.Instrumentation, o.transport.external, o.lis,
	); err != nil {
		return err
	}
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
	return nil
}

func openKV(o *options) (xkv.DB, error) {
	var (
		log           = pebblekv.NewLogger(o.Child("kv"))
		ev            = pebble.MakeLoggingEventListener(log)
		opts          = &pebble.Options{FS: o.fs, Logger: log, EventListener: &ev}
		pebbleDB, err = pebble.Open(o.dirname, opts)
	)
	if err != nil {
		return nil, err
	}
	return pebblekv.Wrap(pebbleDB, pebblekv.DisableObservation()), nil
}
