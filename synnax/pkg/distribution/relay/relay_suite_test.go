package relay_test

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/relay"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/x/config"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx context.Context
	ins alamos.Instrumentation
)

func TestRelay(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Relay Suite")
}

type serviceContainer struct {
	channel   channel.Service
	relay     *relay.Relay
	transport struct {
		relay relay.Transport
	}
}

var _ = BeforeSuite(func() {
	ins = Instrumentation("relay", InstrumentationConfig{
		Trace: config.True(),
	})
	ctx = context.Background()
})

func provision(n int) (*mock.CoreBuilder, map[core.NodeKey]serviceContainer) {
	var (
		builder    = mock.NewCoreBuilder(distribution.Config{Instrumentation: ins})
		services   = make(map[core.NodeKey]serviceContainer)
		channelNet = tmock.NewChannelNetwork()
		relayNet   = tmock.NewRelayNetwork()
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
		container.relay = MustSucceed(relay.Open(relay.Config{
			Instrumentation: ins,
			HostResolver:    c.Cluster,
			Transport:       relayNet.New(c.Config.AdvertiseAddress),
		}))
		services[c.Cluster.HostKey()] = container
	}

	builder.WaitForTopologyToStabilize()
	return builder, services
}
