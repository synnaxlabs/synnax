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
	// Channel is the in-memory channel transport network.
	Channel *channel.Network
	// Framer is the in-memory framer transport network.
	Framer *framer.Network
}

// NewNetwork constructs a Network with freshly initialized channel and framer networks.
func NewNetwork() *Network {
	return &Network{Channel: channel.NewNetwork(), Framer: framer.NewNetwork()}
}

// New provisions an in-memory distribution Transport for the node at addr. buffers sets
// the channel buffer sizes for the streaming framer transports.
func (n *Network) New(addr address.Address, buffers ...int) Transport {
	return Transport{
		ChannelTransport: n.Channel.New(addr),
		FramerTransport:  n.Framer.New(addr, buffers...),
	}
}

// Transport bundles the in-memory channel and framer transports for a single node. It
// implements the distribution.Transport interface.
type Transport struct {
	// ChannelTransport is the in-memory channel transport for the node.
	ChannelTransport distchannel.Transport
	// FramerTransport is the in-memory framer transport for the node.
	FramerTransport distframer.Transport
}

var _ distribution.Transport = Transport{}

// Channel implements distribution.Transport.
func (t Transport) Channel() distchannel.Transport { return t.ChannelTransport }

// Framer implements distribution.Transport.
func (t Transport) Framer() distframer.Transport { return t.FramerTransport }
