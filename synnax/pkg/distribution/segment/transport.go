package segment

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/writer"
)

type Transport interface {
	IteratorClient() iterator.TransportClient
	IteratorServer() iterator.TransportServer
	WriterServer() writer.TransportServer
	WriterClient() writer.TransportClient
}
