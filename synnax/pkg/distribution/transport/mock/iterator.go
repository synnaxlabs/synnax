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
