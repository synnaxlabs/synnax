package mock

import (
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"go.uber.org/zap"
)

type Builder struct {
	mock.CoreBuilder
	Nodes      []distribution.Distribution
	writerNet  *fmock.Network[writer.Request, writer.Response]
	iterNet    *fmock.Network[iterator.Request, iterator.Response]
	channelNet *fmock.Network[channel.CreateMessage, channel.CreateMessage]
}

func NewBuilder(cfg ...distribution.Config) *Builder {
	coreBuilder := mock.NewCoreBuilder(cfg...)

	return &Builder{
		CoreBuilder: *coreBuilder,
		writerNet:   fmock.NewNetwork[writer.Request, writer.Response](),
		iterNet:     fmock.NewNetwork[iterator.Request, iterator.Response](),
		channelNet:  fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage](),
	}
}

func (b *Builder) New() distribution.Distribution {
	core := b.CoreBuilder.New()
	d := distribution.Distribution{Core: core}

	trans := mockSegmentTransport{
		iteratorServer: b.iterNet.StreamServer(core.Config.AdvertiseAddress, 1),
		writerServer:   b.writerNet.StreamServer(core.Config.AdvertiseAddress, 1),
		iteratorClient: b.iterNet.StreamClient(1),
		writerClient:   b.writerNet.StreamClient(1),
	}

	var err error
	d.Ontology, err = ontology.Open(d.Storage.Gorpify())
	if err != nil {
		panic(err)
	}

	nodeOntologySvc := &distribcore.NodeOntologyService{
		Logger:   d.Config.Logger.Sugar(),
		Cluster:  d.Cluster,
		Ontology: d.Ontology,
	}
	clusterOntologySvc := &distribcore.ClusterOntologyService{
		Cluster: d.Cluster,
	}
	d.Ontology.RegisterService(nodeOntologySvc)
	d.Ontology.RegisterService(clusterOntologySvc)
	nodeOntologySvc.ListenForChanges()

	d.Channel = channel.New(
		d.Cluster,
		d.Storage.Gorpify(),
		d.Storage.TS,
		b.channelNet.UnaryClient(),
		b.channelNet.UnaryServer(core.Config.AdvertiseAddress),
		d.Ontology,
	)
	d.Framer = framer.Open(d.Channel, d.Storage.TS, trans, d.Cluster, zap.NewNop())

	return d
}

type mockSegmentTransport struct {
	iteratorServer iterator.TransportServer
	iteratorClient iterator.TransportClient
	writerServer   writer.TransportServer
	writerClient   writer.TransportClient
}

func (m mockSegmentTransport) IteratorServer() iterator.TransportServer {
	return m.iteratorServer
}

func (m mockSegmentTransport) WriterServer() writer.TransportServer {
	return m.writerServer
}

func (m mockSegmentTransport) IteratorClient() iterator.TransportClient {
	return m.iteratorClient
}

func (m mockSegmentTransport) WriterClient() writer.TransportClient {
	return m.writerClient
}
