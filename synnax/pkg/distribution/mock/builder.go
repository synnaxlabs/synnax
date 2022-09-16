package mock

import (
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/writer"
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
		iterator: b.iterNet.RouteStream(core.Config.AdvertiseAddress, 0),
		writer:   b.writerNet.RouteStream(core.Config.AdvertiseAddress, 0),
	}

	var err error
	d.Ontology, err = ontology.Open(d.Storage.Gorpify())
	if err != nil {
		panic(err)
	}
	d.Channel = channel.New(
		d.Cluster,
		d.Storage.Gorpify(),
		d.Storage.TS,
		b.channelNet.RouteUnary(core.Config.AdvertiseAddress),
	)
	d.Segment = segment.New(d.Channel, d.Storage.TS, trans, d.Cluster, zap.NewNop())

	return d
}

type mockSegmentTransport struct {
	iterator iterator.Transport
	writer   writer.Transport
}

func (m mockSegmentTransport) Iterator() iterator.Transport {
	return m.iterator
}

func (m mockSegmentTransport) Writer() writer.Transport {
	return m.writer
}
