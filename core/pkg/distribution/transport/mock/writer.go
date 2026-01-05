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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/address"
)

type FramerWriterNetwork struct {
	Internal *fmock.Network[writer.Request, writer.Response]
}

func (c *FramerWriterNetwork) New(addr address.Address, buffers ...int) writer.Transport {
	return &FramerWriterTransport{
		client: c.Internal.StreamClient(buffers...),
		server: c.Internal.StreamServer(addr, buffers...),
	}
}

func NewWriterNetwork() *FramerWriterNetwork {
	return &FramerWriterNetwork{Internal: fmock.NewNetwork[writer.Request, writer.Response]()}
}

type FramerWriterTransport struct {
	client writer.TransportClient
	server writer.TransportServer
}

var _ writer.Transport = (*FramerWriterTransport)(nil)

func (c FramerWriterTransport) Client() writer.TransportClient { return c.client }

func (c FramerWriterTransport) Server() writer.TransportServer { return c.server }
