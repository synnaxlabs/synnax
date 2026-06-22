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

// Network backs the in-memory channel transports for a cluster of nodes. It holds the
// shared per-operation freighter networks (create, delete, rename) so that a Transport
// provisioned for one node can reach the servers bound by another.
type Network struct {
	create *mock.Network[channel.CreateMessage, channel.CreateMessage]
	delete *mock.Network[channel.DeleteRequest, types.Nil]
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

// New provisions an in-memory channel.Transport whose servers are hosted at addr.
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
	createClient channel.CreateTransportClient
	createServer channel.CreateTransportServer
	deleteClient channel.DeleteTransportClient
	deleteServer channel.DeleteTransportServer
	renameClient channel.RenameTransportClient
	renameServer channel.RenameTransportServer
}

var _ channel.Transport = transport{}

func (t transport) CreateClient() channel.CreateTransportClient {
	return t.createClient
}

func (t transport) CreateServer() channel.CreateTransportServer {
	return t.createServer
}

func (t transport) DeleteClient() channel.DeleteTransportClient {
	return t.deleteClient
}

func (t transport) DeleteServer() channel.DeleteTransportServer {
	return t.deleteServer
}

func (t transport) RenameClient() channel.RenameTransportClient {
	return t.renameClient
}

func (t transport) RenameServer() channel.RenameTransportServer {
	return t.renameServer
}
