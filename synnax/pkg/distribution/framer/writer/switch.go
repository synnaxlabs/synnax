// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"go.uber.org/zap"
)

type peerSwitchSender struct {
	freightfluence.BatchSwitchSender[Request, Request]
	addresses proxy.AddressMap
	logger    *zap.Logger
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
	if r.Command == Data {
		for nodeKey, frame := range r.Frame.SplitByNodeKey() {
			addr, ok := rs.addresses[nodeKey]
			if !ok {
				rs.logger.DPanic("missing address for node", zap.Uint32("node", uint32(nodeKey)))
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

type peerGatewaySwitch struct {
	confluence.BatchSwitch[Request, Request]
	host core.NodeKey
}

func newPeerGatewaySwitch(host core.NodeKey) *peerGatewaySwitch {
	rl := &peerGatewaySwitch{host: host}
	rl.Switch = rl._switch
	return rl
}

func (rl *peerGatewaySwitch) _switch(ctx context.Context, r Request, oReqs map[address.Address]Request) error {
	local, remote := r.Frame.SplitByHost(rl.host)
	pr, gr := r, r
	pr.Frame = remote
	gr.Frame = local
	oReqs[gatewayWriterAddr] = gr
	oReqs[peerSenderAddr] = pr
	return nil
}
