package writer_test

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
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
	channel   channel.Service
	writer    *writer.Service
	transport struct {
		channel channel.Transport
		writer  writer.Transport
	}
}

func provision(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder(core.Config{Logger: logger})
		services   = make(map[core.NodeID]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		writerNet  = tmock.NewFramerWriterNetwork()
	)
	for i := 0; i < n; i++ {
		var (
			c         = builder.New()
			container serviceContainer
		)
		container.channel = MustSucceed(channel.New(channel.ServiceConfig{
			HostResolver: c.Cluster,
			ClusterDB:    c.Storage.Gorpify(),
			TSChannel:    c.Storage.TS,
			Transport:    channelNet.New(c.Config.AdvertiseAddress),
		}))
		container.writer = MustSucceed(writer.NewService(writer.ServiceConfig{
			TS:            c.Storage.TS,
			ChannelReader: container.channel,
			HostResolver:  c.Cluster,
			Transport:     writerNet.New(c.Config.AdvertiseAddress /*buffer*/, 10),
			Logger:        logger,
		}))
		services[c.Cluster.HostID()] = container
	}
	builder.WaitForTopologyToStabilize()
	return builder, services
}
