// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	kvx "github.com/synnaxlabs/x/kv"
	"go/types"
)

var ErrLeaseNotTransferable = errors.New("[db] - cannot transfer leaseAlloc")

const DefaultLeaseholder node.ID = 0

type leaseAllocator struct{ Config }

func (la *leaseAllocator) allocate(ctx context.Context, op Operation) (Operation, error) {
	lh, err := la.getLease(ctx, op.Key)
	// If we get a nil error, that means this key has been set before.
	if err == nil {
		if op.Leaseholder == DefaultLeaseholder {
			op.Leaseholder = lh
		} else if lh != op.Leaseholder {
			// If the Leaseholder doesn't match the previous Leaseholder,
			// we return an error.
			return op, ErrLeaseNotTransferable
		}
	} else if err == kvx.NotFound && op.Variant == Set {
		if op.Leaseholder == DefaultLeaseholder {
			// If we can't find the Leaseholder, and the op doesn't have a Leaseholder assigned,
			// we assign the leaseAlloc to the cluster host.
			op.Leaseholder = la.Cluster.HostID()
		}
		// If we can't find the Leaseholder, and the op has a Leaseholder assigned,
		// that means it's a new key, so we let it choose its own leaseAlloc.
	} else {
		return op, err
	}
	return op, nil
}

func (la *leaseAllocator) getLease(ctx context.Context, key []byte) (node.ID, error) {
	digest, err := getDigestFromKV(ctx, la.Engine, key)
	return digest.Leaseholder, err
}

type leaseProxy struct {
	Config
	localTo  address.Address
	remoteTo address.Address
	confluence.Switch[WriteRequest]
}

func newLeaseProxy(cfg Config, localTo address.Address, remoteTo address.Address) segment {
	lp := &leaseProxy{Config: cfg, localTo: localTo, remoteTo: remoteTo}
	lp.Switch.ApplySwitch = lp._switch
	return lp
}

func (lp *leaseProxy) _switch(
	_ context.Context,
	b WriteRequest,
) (address.Address, bool, error) {
	if b.Leaseholder == lp.Cluster.HostID() {
		return lp.localTo, true, nil
	}
	return lp.remoteTo, true, nil
}

type (
	LeaseTransportClient = freighter.UnaryClient[WriteRequest, types.Nil]
	LeaseTransportServer = freighter.UnaryServer[WriteRequest, types.Nil]
)

type leaseSender struct {
	Config
	confluence.UnarySink[WriteRequest]
}

func newLeaseSender(cfg Config) sink {
	ls := &leaseSender{Config: cfg}
	ls.Sink = ls.send
	return ls
}

func (lf *leaseSender) send(ctx context.Context, br WriteRequest) (err error) {
	defer func() { br.done(err) }()
	lf.L.Debug("sending leased writer", br.logArgs()...)
	var addr address.Address
	addr, err = lf.Cluster.Resolve(br.Leaseholder)
	if err != nil {
		return err
	}
	_, err = lf.Config.LeaseTransportClient.Send(br.ctx, addr, br)
	return err
}

type leaseReceiver struct {
	Config
	confluence.AbstractUnarySource[WriteRequest]
	confluence.EmptyFlow
}

func newLeaseReceiver(cfg Config) source {
	lr := &leaseReceiver{Config: cfg}
	lr.LeaseTransportServer.BindHandler(lr.receive)
	return lr
}

func (lr *leaseReceiver) receive(ctx context.Context, br WriteRequest) (types.Nil, error) {
	lr.L.Debug("received leased writer", br.logArgs()...)
	bc := batchCoordinator{}
	bc.add(&br)
	lr.Out.Inlet() <- br
	return types.Nil{}, bc.wait()
}
