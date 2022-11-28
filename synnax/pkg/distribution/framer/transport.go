package framer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
)

type Transport interface {
	IteratorClient() iterator.TransportClient
	IteratorServer() iterator.TransportServer
	WriterServer() writer.TransportServer
	WriterClient() writer.TransportClient
}
