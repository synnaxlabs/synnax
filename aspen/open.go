// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included pebble code is copyrighted by the cockroachdb team, and is licensed under
// the BSD 3-Clause License. See the repository file license/BSD-3-Clause.txt for more
// information.

package aspen

import (
	"context"
	"io"

	"github.com/cockroachdb/pebble/v2"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/x/address"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/service"
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
		o           = newOptions(dirname, addr, peers, opts...)
		db          = &DB{}
		err         error
		cleanup, ok = service.NewOpener(ctx, &db.closer)
	)
	defer func() {
		err = cleanup(err)
	}()
	if o.kv.Engine == nil {
		if o.kv.Engine, err = openKV(o); !ok(err, o.kv.Engine) {
			return nil, err
		}
	}
	o.cluster.Storage = o.kv.Engine
	var transportCloser io.Closer
	if transportCloser, err = configureTransport(ctx, o); !ok(err, transportCloser) {
		return nil, err
	}
	if db.Cluster, err = cluster.Open(ctx, o.cluster); !ok(err, db.Cluster) {
		return nil, err
	}
	o.kv.Cluster = db.Cluster
	if db.DB, err = kv.Open(ctx, o.kv); !ok(err, db.DB) {
		return nil, err
	}

	return db, err
}

func configureTransport(ctx context.Context, o *options) (io.Closer, error) {
	sCtx, cancel := signal.WithCancel(
		o.T.Transfer(ctx, context.Background()),
		signal.WithInstrumentation(o.Instrumentation),
	)
	transportShutdown := signal.NewHardShutdown(sCtx, cancel)
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
