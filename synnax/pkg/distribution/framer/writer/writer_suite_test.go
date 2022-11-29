package writer_test

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"go.uber.org/zap"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

func TestWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Write Suite")
}

type serviceContainer struct {
	channel   *channel.Service
	transport struct {
		channel channel.Transport
		writer  writer.Transport
	}
}

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	var (
		builder = mock.NewCoreBuilder(core.Config{
			Logger:  logger,
			Storage: storage.Config{MemBacked: config.BoolPointer(true)},
		})
		services   = make(map[core.NodeID]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		writerNet  = tmock.NewFramerWriterNetwork()
	)
	for i := 0; i < n; i++ {
		_core := builder.New()
		var container serviceContainer
		container.transport.channel = channelNet.New(_core.Config.AdvertiseAddress)
		container.transport.writer = writerNet.New(_core.Config.AdvertiseAddress /*buffer*/, 10)
		container.channel = MustSucceed(channel.New(channel.Config{
			HostResolver: _core.Cluster,
			ClusterDB:    _core.Storage.Gorpify(),
			TS:           _core.Storage.TS,
			Transport:    container.transport.channel,
		}))
		writer.NewServer(writer.Config{
			TS:             _core.Storage.TS,
			ChannelService: container.channel,
			HostResolver:   _core.Cluster,
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
			HostResolver:   builder.Cores[nodeID].Cluster,
			Transport:      services[nodeID].transport.writer,
			Keys:           keys,
			Logger:         log,
		},
	)
}
