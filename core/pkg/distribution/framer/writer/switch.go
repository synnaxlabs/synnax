// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

type peerSwitchSender struct {
	freightfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
}

func newRequestSwitchSender(
	addresses proxy.AddressMap,
	senders map[address.Address]freighter.StreamSenderCloser[Request],
) confluence.Sink[Request] {
	rs := &peerSwitchSender{addresses: addresses}
	rs.Senders = freightfluence.MapTargetedSender[Request](senders)
	rs.Switch = rs._switch
	return rs
}

func (rs *peerSwitchSender) _switch(
	_ context.Context,
	r Request,
	oReqs map[address.Address]Request,
) error {
	if r.Command == CommandWrite {
		for nodeKey, frame := range r.Frame.SplitByLeaseholder() {
			addr, ok := rs.addresses[nodeKey]
			if !ok {
				// The frame targets channels leased to a node the writer was not opened
				// against, so no storage writer downstream could reject it.
				return errors.Wrapf(
					validate.ErrValidation,
					"invalid key(s) %v: writer is not open on leaseholder node %d",
					frame.KeysSlice(),
					nodeKey,
				)
			}
			r.Frame = frame
			oReqs[addr] = r
		}
	} else {
		for _, addr := range rs.addresses {
			oReqs[addr] = r
		}
	}

	return nil
}

type peerGatewayFreeSwitch struct {
	confluence.BatchSwitch[Request, Request]
	host node.Key
	has  struct {
		peer    bool
		gateway bool
		free    bool
	}
}

func newPeerGatewayFreeSwitch(
	host node.Key,
	hasPeer bool,
	hasGateway bool,
	hasFree bool,
) *peerGatewayFreeSwitch {
	rl := &peerGatewayFreeSwitch{host: host}
	rl.Switch = rl._switch
	rl.has.peer = hasPeer
	rl.has.gateway = hasGateway
	rl.has.free = hasFree
	return rl
}

func (rl *peerGatewayFreeSwitch) _switch(ctx context.Context, r Request, oReqs map[address.Address]Request) error {
	local, remote, free := r.Frame.SplitByHost(rl.host)
	if rl.has.peer {
		pr := r
		pr.Frame = remote
		oReqs[peerSenderAddr] = pr
	}
	if rl.has.gateway {
		gr := r
		gr.Frame = local
		oReqs[gatewayWriterAddr] = gr
	}
	if rl.has.free {
		fr := r
		fr.Frame = free
		oReqs[freeWriterAddr] = fr
	}
	return nil
}
