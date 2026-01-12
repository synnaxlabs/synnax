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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/x/address"
	"go/types"
)

type FramerDeleterNetwork struct {
	Internal *fmock.Network[deleter.Request, types.Nil]
}

func (c *FramerDeleterNetwork) New(addr address.Address) deleter.Transport {
	return &FramerDeleterTransport{
		client: c.Internal.UnaryClient(),
		server: c.Internal.UnaryServer(addr),
	}
}

func NewDeleterNetwork() *FramerDeleterNetwork {
	return &FramerDeleterNetwork{Internal: fmock.NewNetwork[deleter.Request, types.Nil]()}
}

type FramerDeleterTransport struct {
	client deleter.TransportClient
	server deleter.TransportServer
}

var _ deleter.Transport = (*FramerDeleterTransport)(nil)

func (c FramerDeleterTransport) Client() deleter.TransportClient { return c.client }

func (c FramerDeleterTransport) Server() deleter.TransportServer { return c.server }
