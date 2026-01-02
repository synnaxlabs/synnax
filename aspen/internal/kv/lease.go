// Copyright 2025 Synnax Labs, Inc.
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
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
)

var ErrLeaseNotTransferable = errors.New("[cesium] - cannot transfer leaseAlloc")

const DefaultLeaseholder node.Key = 0

type leaseAllocator struct{ Config }

func (la *leaseAllocator) allocate(ctx context.Context, op Operation) (Operation, error) {
	lh, err := la.getLease(ctx, op.Key)
	// If we get a nil error, that means this key has been set before.
	if err == nil {
		if op.Leaseholder == DefaultLeaseholder {
			op.Leaseholder = lh
			if lh == DefaultLeaseholder {
				la.L.DPanic("Lease allocator returned unexpected node key 0 for leaseholder")
			}
		} else if lh != op.Leaseholder {
			// If the Leaseholder doesn't match the previous Leaseholder,
			// we return an error.
			return op, ErrLeaseNotTransferable
		}
	} else if errors.Is(err, xkv.NotFound) && op.Variant == change.Set {
		if op.Leaseholder == DefaultLeaseholder {
			// If we can't find the Leaseholder, and the op doesn't have a Leaseholder assigned,
			// we assign the leaseAlloc to the cluster host.
			op.Leaseholder = la.Cluster.HostKey()
		}
		// If we can't find the Leaseholder, and the op has a Leaseholder assigned,
		// that means it's a new key, so we let it choose its own leaseAlloc.
	} else {
		return op, err
	}
	return op, nil
}

func (la *leaseAllocator) getLease(ctx context.Context, key []byte) (node.Key, error) {
	digest, err := getDigestFromKV(ctx, la.Engine, key)
	return digest.Leaseholder, err
}

type leaseProxy struct {
	Config
	localTo  address.Address
	remoteTo address.Address
	confluence.Switch[TxRequest]
}

func newLeaseProxy(cfg Config, localTo address.Address, remoteTo address.Address) segment {
	lp := &leaseProxy{Config: cfg, localTo: localTo, remoteTo: remoteTo}
	lp.Switch.Switch = lp._switch
	return lp
}

func (lp *leaseProxy) _switch(
	_ context.Context,
	b TxRequest,
) (address.Address, bool, error) {
	route := lo.Ternary(b.Leaseholder == lp.Cluster.HostKey(), lp.localTo, lp.remoteTo)
	return route, true, nil
}

type (
	LeaseTransportClient = freighter.UnaryClient[TxRequest, types.Nil]
	LeaseTransportServer = freighter.UnaryServer[TxRequest, types.Nil]
)

type leaseSender struct {
	Config
	confluence.UnarySink[TxRequest]
}

func newLeaseSender(cfg Config) sink {
	ls := &leaseSender{Config: cfg}
	ls.Sink = ls.send
	return ls
}

func (lf *leaseSender) send(_ context.Context, br TxRequest) error {
	addr, err := lf.Cluster.Resolve(br.Leaseholder)
	defer func() { br.done(err) }()
	if err != nil {
		return nil
	}
	_, err = lf.LeaseTransportClient.Send(br.Context, addr, br)
	return nil
}

type leaseReceiver struct {
	Config
	confluence.AbstractUnarySource[TxRequest]
	confluence.NopFlow
}

func newLeaseReceiver(cfg Config) source {
	lr := &leaseReceiver{Config: cfg}
	lr.LeaseTransportServer.BindHandler(lr.receive)
	return lr
}

func (lr *leaseReceiver) receive(_ context.Context, br TxRequest) (types.Nil, error) {
	bc := txCoordinator{}
	bc.add(&br)
	lr.Out.Inlet() <- br
	return types.Nil{}, bc.wait()
}
