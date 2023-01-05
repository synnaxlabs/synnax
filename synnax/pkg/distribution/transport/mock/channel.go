// Copyright 2023 Synnax Labs, Inc.
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
)

type ChannelNetwork struct {
	Internal *fmock.Network[channel.CreateMessage, channel.CreateMessage]
}

func (c *ChannelNetwork) New(add address.Address) channel.Transport {
	return &ChannelTransport{
		client: c.Internal.UnaryClient(),
		server: c.Internal.UnaryServer(add),
	}
}

func NewChannelNetwork() *ChannelNetwork {
	return &ChannelNetwork{Internal: fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()}
}

type ChannelTransport struct {
	client channel.CreateTransportClient
	server channel.CreateTransportServer
}

var _ channel.Transport = (*ChannelTransport)(nil)

func (c ChannelTransport) CreateClient() channel.CreateTransportClient {
	return c.client
}

func (c ChannelTransport) CreateServer() channel.CreateTransportServer {
	return c.server
}
