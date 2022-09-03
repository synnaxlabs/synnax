package writer_test

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/freighter/fmock"
	"go.uber.org/zap"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ctx = context.Background()

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Write Suite")
}

type serviceContainer struct {
	channel   *channel.Service
	transport struct {
		channel channel.CreateTransport
		writer  writer.Transport
	}
}

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	builder := mock.NewCoreBuilder(core.Config{Logger: logger, Storage: storage.Config{MemBacked: true}})
	services := make(map[core.NodeID]serviceContainer)
	channelNet := fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
	writerNet := fmock.NewNetwork[writer.Request, writer.Response]()
	for i := 0; i < n; i++ {
		_core := builder.New()
		var container serviceContainer
		container.transport.channel = channelNet.RouteUnary(_core.Config.AdvertiseAddress)
		container.transport.writer = writerNet.RouteStream(_core.Config.AdvertiseAddress, 0)
		container.channel = channel.New(
			_core.Cluster,
			_core.Storage.Gorpify(),
			_core.Storage.TS,
			container.transport.channel,
		)
		writer.NewServer(writer.Config{
			TS:             _core.Storage.TS,
			ChannelService: container.channel,
			Resolver:       _core.Cluster,
			Transport:      container.transport.writer,
		})
		services[_core.Cluster.HostID()] = container
	}
	return builder, services
}

func openWriter(
	nodeID core.NodeID,
	services map[core.NodeID]serviceContainer,
	builder *mock.CoreBuilder,
	keys channel.Keys,
	log *zap.Logger,
) (writer.Writer, error) {
	return writer.New(
		ctx,
		writer.Config{
			TS:             builder.Cores[nodeID].Storage.TS,
			ChannelService: services[nodeID].channel,
			Resolver:       builder.Cores[nodeID].Cluster,
			Transport:      services[nodeID].transport.writer,
			ChannelKeys:    keys,
			Logger:         log,
		},
	)
}
