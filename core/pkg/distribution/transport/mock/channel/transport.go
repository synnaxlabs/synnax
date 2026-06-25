// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"go/types"

	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory channel transports for a cluster of nodes. It aggregates
// the per-operation freighter networks (create, delete, rename) so that a single
// Transport can be provisioned per node, mirroring the gRPC channel.Transport that
// bundles the same operations.
type Network struct {
	// create backs the unary channel create transport.
	create *mock.Network[channel.CreateMessage, channel.CreateMessage]
	// delete backs the unary channel delete transport.
	delete *mock.Network[channel.DeleteRequest, types.Nil]
	// rename backs the unary channel rename transport.
	rename *mock.Network[channel.RenameRequest, types.Nil]
}

// NewNetwork constructs a Network with freshly initialized per-operation networks.
func NewNetwork() *Network {
	return &Network{
		create: mock.NewNetwork[channel.CreateMessage, channel.CreateMessage](),
		delete: mock.NewNetwork[channel.DeleteRequest, types.Nil](),
		rename: mock.NewNetwork[channel.RenameRequest, types.Nil](),
	}
}

// New provisions an in-memory channel.Transport for the node at addr.
func (n *Network) New(addr address.Address) channel.Transport {
	return transport{
		createClient: n.create.UnaryClient(),
		createServer: n.create.UnaryServer(addr),
		deleteClient: n.delete.UnaryClient(),
		deleteServer: n.delete.UnaryServer(addr),
		renameClient: n.rename.UnaryClient(),
		renameServer: n.rename.UnaryServer(addr),
	}
}

type transport struct {
	createClient channel.CreateClient
	createServer channel.CreateServer
	deleteClient channel.DeleteClient
	deleteServer channel.DeleteServer
	renameClient channel.RenameClient
	renameServer channel.RenameServer
}

var _ channel.Transport = transport{}

// CreateClient implements the channel.Transport interface.
func (t transport) CreateClient() channel.CreateClient { return t.createClient }

// CreateServer implements the channel.Transport interface.
func (t transport) CreateServer() channel.CreateServer { return t.createServer }

// DeleteClient implements the channel.Transport interface.
func (t transport) DeleteClient() channel.DeleteClient { return t.deleteClient }

// DeleteServer implements the channel.Transport interface.
func (t transport) DeleteServer() channel.DeleteServer { return t.deleteServer }

// RenameClient implements the channel.Transport interface.
func (t transport) RenameClient() channel.RenameClient { return t.renameClient }

// RenameServer implements the channel.Transport interface.
func (t transport) RenameServer() channel.RenameServer { return t.renameServer }
