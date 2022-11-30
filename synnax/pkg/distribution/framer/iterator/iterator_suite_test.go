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
	channel channel.Service
	iter    *iterator.Service
}

func provision(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder(core.Config{Logger: logger})
		services   = make(map[core.NodeID]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		iterNet    = tmock.NewFramerIteratorNetwork()
	)
	for i := 0; i < n; i++ {
		var (
			c    = builder.New()
			cont serviceContainer
		)
		cont.channel = MustSucceed(channel.New(channel.Config{
			HostResolver: c.Cluster,
			ClusterDB:    c.Storage.Gorpify(),
			Transport:    channelNet.New(c.Config.AdvertiseAddress),
			TS:           c.Storage.TS,
		}))
		cont.iter = MustSucceed(iterator.NewService(iterator.ServiceConfig{
			TS:            c.Storage.TS,
			ChannelReader: cont.channel,
			HostResolver:  c.Cluster,
			Transport:     iterNet.New(c.Config.AdvertiseAddress),
			Logger:        logger,
		}))
		services[c.Cluster.HostID()] = cont
	}
	return builder, services
}
