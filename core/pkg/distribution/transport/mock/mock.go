// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distframer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer"
	"github.com/synnaxlabs/x/address"
)

// Network bundles the in-memory channel and framer transport networks for a cluster of
// nodes, so that a distribution.Transport can be provisioned per node. It mirrors the
// gRPC Transport bundle but retains shared in-memory network state across nodes.
type Network struct {
	channel *channel.Network
	framer  *framer.Network
}

// NewNetwork constructs a Network with freshly initialized channel and framer networks.
func NewNetwork() *Network {
	return &Network{channel: channel.NewNetwork(), framer: framer.NewNetwork()}
}

// New provisions an in-memory distribution.Transport for the node at addr. buffers sets
// the channel buffer sizes for the streaming framer transports.
func (n *Network) New(addr address.Address, buffers ...int) distribution.Transport {
	return transport{
		channel: n.channel.New(addr),
		framer:  n.framer.New(addr, buffers...),
	}
}

type transport struct {
	channel distchannel.Transport
	framer  distframer.Transport
}

var _ distribution.Transport = transport{}

// Channel implements the distribution.Transport interface.
func (t transport) Channel() distchannel.Transport { return t.channel }

// Framer implements the distribution.Transport interface.
func (t transport) Framer() distframer.Transport { return t.framer }
