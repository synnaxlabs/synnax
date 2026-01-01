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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/address"
)

type FramerIteratorNetwork struct {
	Internal *fmock.Network[iterator.Request, iterator.Response]
}

func (c *FramerIteratorNetwork) New(addr address.Address, buffers ...int) iterator.Transport {
	return &FramerIteratorTransport{
		client: c.Internal.StreamClient(buffers...),
		server: c.Internal.StreamServer(addr, buffers...),
	}
}

func NewIteratorNetwork() *FramerIteratorNetwork {
	return &FramerIteratorNetwork{Internal: fmock.NewNetwork[iterator.Request, iterator.Response]()}
}

type FramerIteratorTransport struct {
	client iterator.TransportClient
	server iterator.TransportServer
}

var _ iterator.Transport = (*FramerIteratorTransport)(nil)

func (c FramerIteratorTransport) Client() iterator.TransportClient { return c.client }

func (c FramerIteratorTransport) Server() iterator.TransportServer { return c.server }
