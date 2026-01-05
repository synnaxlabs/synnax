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
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
	"go/types"
)

type ChannelNetwork struct {
	CreateNet *fmock.Network[channel.CreateMessage, channel.CreateMessage]
	DeleteNet *fmock.Network[channel.DeleteRequest, types.Nil]
	RenameNet *fmock.Network[channel.RenameRequest, types.Nil]
}

func (c *ChannelNetwork) New(add address.Address) channel.Transport {
	return &ChannelTransport{
		createClient: c.CreateNet.UnaryClient(),
		createServer: c.CreateNet.UnaryServer(add),
		deleteClient: c.DeleteNet.UnaryClient(),
		deleteServer: c.DeleteNet.UnaryServer(add),
		renameClient: c.RenameNet.UnaryClient(),
		renameServer: c.RenameNet.UnaryServer(add),
	}
}

func NewChannelNetwork() *ChannelNetwork {
	return &ChannelNetwork{
		CreateNet: fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage](),
		DeleteNet: fmock.NewNetwork[channel.DeleteRequest, types.Nil](),
		RenameNet: fmock.NewNetwork[channel.RenameRequest, types.Nil](),
	}
}

type ChannelTransport struct {
	createClient channel.CreateTransportClient
	createServer channel.CreateTransportServer
	deleteClient channel.DeleteTransportClient
	deleteServer channel.DeleteTransportServer
	renameClient channel.RenameTransportClient
	renameServer channel.RenameTransportServer
}

var _ channel.Transport = (*ChannelTransport)(nil)

func (c ChannelTransport) CreateClient() channel.CreateTransportClient { return c.createClient }

func (c ChannelTransport) CreateServer() channel.CreateTransportServer { return c.createServer }

func (c ChannelTransport) DeleteClient() channel.DeleteTransportClient { return c.deleteClient }

func (c ChannelTransport) DeleteServer() channel.DeleteTransportServer { return c.deleteServer }

func (c ChannelTransport) RenameClient() channel.RenameTransportClient { return c.renameClient }

func (c ChannelTransport) RenameServer() channel.RenameTransportServer { return c.renameServer }
