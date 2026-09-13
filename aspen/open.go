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
	"net"

	"github.com/cockroachdb/pebble/v2"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
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
	if o.transport.owned == nil && o.lis != nil {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"cannot serve a transport provided with WithTransport on a listener",
		)
	}
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
	wireTransport(o)
	var lis net.Listener
	if o.transport.owned != nil {
		// The listener is the first point at which an operating system assigned port is
		// known, so bind before the cluster advertises the host address.
		lis, o.cluster.HostAddress, err = openListener(o)
		// Release only a listener Open bound itself: one from WithListener stays the
		// caller's. Serve releases it when it stops, so tolerate an already closed
		// listener.
		lisCloser := io.NopCloser
		if o.lis == nil {
			lisCloser = io.CloserFunc(func() error {
				return errors.Skip(lis.Close(), net.ErrClosed)
			})
		}
		if !ok(err, lisCloser) {
			return nil, err
		}
		if err = o.transport.owned.Configure(
			o.Instrumentation,
		); !ok(err, o.transport.owned) {
			return nil, err
		}
	}
	if db.Cluster, err = cluster.Open(ctx, o.cluster); !ok(err, db.Cluster) {
		return nil, err
	}
	o.kv.Cluster = db.Cluster
	if db.DB, err = kv.Open(ctx, o.kv); !ok(err, db.DB) {
		return nil, err
	}
	if o.transport.owned != nil {
		if err = o.transport.owned.Serve(lis); !ok(err, nil) {
			return nil, err
		}
	}

	return db, err
}

// openListener opens the listener the owned transport serves on, and returns it with
// the address the host advertises. A listener from WithListener is used as is, and
// stays the caller's to close when it is not bound to a TCP address.
func openListener(o *options) (net.Listener, address.Address, error) {
	lis := o.lis
	if lis == nil {
		var err error
		if lis, err = net.Listen("tcp", o.addr.String()); err != nil {
			return nil, "", err
		}
	}
	tcp, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		return nil, "", errors.Wrapf(
			validate.ErrValidation,
			"listener address %q is not a TCP address",
			lis.Addr(),
		)
	}
	return lis, address.Newf("%s:%d", o.addr.Host(), tcp.Port), nil
}

func wireTransport(o *options) {
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
