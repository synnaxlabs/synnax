package mock

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	aspentransmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	mockstorage "github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
)

// CoreBuilder is a utility for provisioning mock distribution cores that
// form a cluster. To create a new CoreBuilder, call NewCoreBuilder
type CoreBuilder struct {
	// Builder is the underlying storage builder.
	mockstorage.Builder
	// Config is the configuration used to provision new cores.
	Config core.Config
	// Cores contains a map of all cores paired with their respective host ID.
	Cores map[core.NodeID]core.Core
	// net is the network for transporting key-value operations.
	net *aspentransmock.Network
	// addrFactory generates unique addresses for nodes.
	addrFactory *address.Factory
}

// NewCoreBuilder opens a new CoreBuilder that provisions cores using the given
// configuration.
func NewCoreBuilder(configs ...distribution.Config) *CoreBuilder {
	cfg, err := config.OverrideAndValidate(distribution.DefaultConfig, append([]distribution.Config{{
		Storage: storage.Config{MemBacked: config.BoolPointer(true)},
	}}, configs...)...)
	if err != nil {
		panic(err)
	}
	storeBuilder := mockstorage.NewBuilder(cfg.Storage)
	net := aspentransmock.NewNetwork()
	addrFactory := &address.Factory{Host: "localhost", PortStart: 0}
	return &CoreBuilder{
		Config:      cfg,
		Builder:     *storeBuilder,
		Cores:       make(map[core.NodeID]core.Core),
		net:         net,
		addrFactory: addrFactory,
	}
}

// New provisions a new core connected to the rest of the nodes in the builder's cluster.
// Panics if the core cannot be opened.
func (c *CoreBuilder) New() core.Core {
	store := c.Builder.New()
	trans := c.net.NewTransport()
	addr := c.addrFactory.Next()

	clusterKV := lo.Must(aspen.Open(
		context.TODO(),
		/* dirname */ "",
		addr,
		c.peerAddresses(),
		aspen.WithEngine(store.KV),
		aspen.WithExperiment(c.Config.Experiment),
		aspen.WithLogger(c.Config.Logger.Named("aspen").Sugar()),
		aspen.WithTransport(trans),
		aspen.WithPropagationConfig(aspen.FastPropagationConfig),
	))

	cfg := c.Config
	cfg.AdvertiseAddress = addr

	store.KV = clusterKV

	_core := distribution.Core{Config: cfg, Cluster: clusterKV, Storage: store}

	c.Cores[_core.Cluster.HostID()] = _core
	return _core
}

// Close shuts down all other nodes in the cluster. It is not safe to call this method
// while the nodes are still in use.
func (c *CoreBuilder) Close() error { return c.Builder.Close() }

func (c *CoreBuilder) peerAddresses() (addrs []address.Address) {
	for _, _core := range c.Cores {
		addrs = append(addrs, _core.Config.AdvertiseAddress)
	}
	return
}
