package relay_test

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx = context.Background()
	ins alamos.Instrumentation
)

func TestRelay(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Relay Suite")
}

type serviceContainer struct {
	channel   channel.Service
	writer    *writer.Service
	relay     *relay.Relay
	transport struct {
		channel channel.Transport
		writer  writer.Transport
		relay   relay.Transport
	}
}

func provision(n int) (*mock.CoreBuilder, map[core.NodeKey]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder()
		service    = make(map[core.NodeKey]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		writerNet  = tmock.NewFramerWriterNetwork()
		relayNet   = tmock.NewRelayNetwork()
	)
	for i := 0; i < n; i++ {
		var (
			c         = builder.New()
			container serviceContainer
		)
		container.channel = MustSucceed(channel.New(ctx, channel.ServiceConfig{
			HostResolver: c.Cluster,
			ClusterDB:    c.Storage.Gorpify(),
			TSChannel:    c.Storage.TS,
			Transport:    channelNet.New(c.Config.AdvertiseAddress),
		}))
		container.relay = MustSucceed(relay.Open(relay.Config{
			Instrumentation: ins,
			TS:              c.Storage.TS,
			Transport:       relayNet.New(c.Config.AdvertiseAddress),
			HostResolver:    c.Cluster,
		}))
		container.writer = MustSucceed(writer.OpenService(writer.ServiceConfig{
			Instrumentation: ins,
			TS:              c.Storage.TS,
			ChannelReader:   container.channel,
			Transport:       writerNet.New(c.Config.AdvertiseAddress),
			HostResolver:    c.Cluster,
			FreeWrites:      container.relay.Writes,
		}))
		service[c.Cluster.HostKey()] = container
	}
	builder.WaitForTopologyToStabilize()
	return builder, service
}
