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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/address"
)

type FramerNetwork struct {
	Iterator *FramerIteratorNetwork
	Writer   *FramerWriterNetwork
}

func NewFramerNetwork() *FramerNetwork {
	return &FramerNetwork{
		Iterator: NewFramerIteratorNetwork(),
		Writer:   NewFramerWriterNetwork(),
	}
}

func (f *FramerNetwork) New(add address.Address) framer.Transport {
	return &FramerTransport{
		iterator: f.Iterator.New(add),
		writer:   f.Writer.New(add),
	}
}

type FramerTransport struct {
	iterator iterator.Transport
	writer   writer.Transport
}

var (
	_ framer.Transport = (*FramerTransport)(nil)
)

func (c FramerTransport) Iterator() iterator.Transport { return c.iterator }

func (c FramerTransport) Writer() writer.Transport { return c.writer }

type FramerIteratorNetwork struct {
	Internal *fmock.Network[iterator.Request, iterator.Response]
}

func (c *FramerIteratorNetwork) New(addr address.Address, buffers ...int) iterator.Transport {
	return &FramerIteratorTransport{
		client: c.Internal.StreamClient(buffers...),
		server: c.Internal.StreamServer(addr, buffers...),
	}
}

func NewFramerIteratorNetwork() *FramerIteratorNetwork {
	return &FramerIteratorNetwork{Internal: fmock.NewNetwork[iterator.Request, iterator.Response]()}
}

type FramerIteratorTransport struct {
	client iterator.TransportClient
	server iterator.TransportServer
}

var _ iterator.Transport = (*FramerIteratorTransport)(nil)

func (c FramerIteratorTransport) Client() iterator.TransportClient { return c.client }

func (c FramerIteratorTransport) Server() iterator.TransportServer { return c.server }

type FramerWriterNetwork struct {
	Internal *fmock.Network[writer.Request, writer.Response]
}

func (c *FramerWriterNetwork) New(addr address.Address, buffers ...int) writer.Transport {
	return &FramerWriterTransport{
		client: c.Internal.StreamClient(buffers...),
		server: c.Internal.StreamServer(addr, buffers...),
	}
}

func NewFramerWriterNetwork() *FramerWriterNetwork {
	return &FramerWriterNetwork{Internal: fmock.NewNetwork[writer.Request, writer.Response]()}
}

type FramerWriterTransport struct {
	client writer.TransportClient
	server writer.TransportServer
}

var _ writer.Transport = (*FramerWriterTransport)(nil)

func (c FramerWriterTransport) Client() writer.TransportClient { return c.client }

func (c FramerWriterTransport) Server() writer.TransportServer { return c.server }
