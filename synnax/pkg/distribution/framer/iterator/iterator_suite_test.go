package iterator_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"testing"
)

var (
	ctx = context.Background()
)

func TestIterator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IteratorServer Suite")
}

type serviceContainer struct {
	channel   *channel.Service
	transport struct {
		channel channel.Transport
		iter    iterator.Transport
	}
}

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder(core.Config{Logger: logger, Storage: storage.Config{MemBacked: config.BoolPointer(true)}})
		services   = make(map[core.NodeID]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		iterNet    = tmock.NewFramerIteratorNetwork()
	)
	for i := 0; i < n; i++ {
		_core := builder.New()
		var container serviceContainer
		container.transport.channel = channelNet.New(_core.Config.AdvertiseAddress)
		container.transport.iter = iterNet.New(_core.Config.AdvertiseAddress)
		container.channel = MustSucceed(channel.New(channel.Config{
			HostResolver: _core.Cluster,
			ClusterDB:    _core.Storage.Gorpify(),
			Transport:    container.transport.channel,
		}))
		iterator.StartServer(iterator.Config{
			TS:           _core.Storage.TS,
			HostResolver: _core.Cluster,
			Transport:    container.transport.iter,
			Logger:       zap.NewNop(),
		})
		services[_core.Cluster.HostID()] = container
	}
	return builder, services
}

func openIter(
	nodeID core.NodeID,
	services map[core.NodeID]serviceContainer,
	builder *mock.CoreBuilder,
	keys channel.Keys,
) iterator.Iterator {
	iter, err := iterator.New(
		ctx,
		iterator.Config{
			Logger:         zap.NewNop(),
			TS:             builder.Cores[nodeID].Storage.TS,
			HostResolver:   builder.Cores[nodeID].Cluster,
			Transport:      services[nodeID].transport.iter,
			TimeRange:      telem.TimeRangeMax,
			ChannelKeys:    keys,
			ChannelService: services[nodeID].channel,
		},
	)
	Expect(err).ToNot(HaveOccurred())
	return iter
}
