package mock

import (
	"context"
	"github.com/arya-analytics/aspen"
	aspentransmock "github.com/arya-analytics/aspen/transport/mock"
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/storage"
	mockstorage "github.com/arya-analytics/delta/pkg/storage/mock"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/config"
)

type CoreBuilder struct {
	mockstorage.Builder
	Config      core.Config
	net         *aspentransmock.Network
	Cores       map[core.NodeID]core.Core
	addrFactory *address.Factory
}

func NewCoreBuilder(cfg ...distribution.Config) *CoreBuilder {
	_cfg, err := config.OverrideAndValidate(distribution.DefaultConfig, append([]distribution.Config{{
		Storage: storage.Config{
			MemBacked: config.BoolPointer(true),
		},
	}}, cfg...)...)
	if err != nil {
		panic(err)
	}
	storeBuilder := mockstorage.NewBuilder(_cfg.Storage)
	net := aspentransmock.NewNetwork()
	addrFactory := &address.Factory{Host: "localhost", PortStart: 0}
	return &CoreBuilder{
		Config:      _cfg,
		Builder:     *storeBuilder,
		Cores:       make(map[core.NodeID]core.Core),
		net:         net,
		addrFactory: addrFactory,
	}
}

func (c *CoreBuilder) New() core.Core {
	store := c.Builder.New()
	trans := c.net.NewTransport()
	addr := c.addrFactory.Next()

	clusterKV, err := aspen.Open(
		context.TODO(),
		/* dirname */ "",
		addr,
		c.peerAddresses(),
		aspen.WithEngine(store.KV),
		aspen.WithExperiment(c.Config.Experiment),
		aspen.WithLogger(c.Config.Logger.Named("aspen").Sugar()),
		aspen.WithTransport(trans),
		aspen.WithPropagationConfig(aspen.FastPropagationConfig),
	)
	if err != nil {
		panic(err)
	}

	cfg := c.Config
	cfg.AdvertiseAddress = addr

	store.KV = clusterKV

	_core := distribution.Core{Config: cfg, Cluster: clusterKV, Storage: store}

	c.Cores[_core.Cluster.HostID()] = _core
	return _core
}

func (c *CoreBuilder) Close() error {
	return c.Builder.Close()
}

func (c *CoreBuilder) peerAddresses() (peerAddresses []address.Address) {
	for _, _core := range c.Cores {
		peerAddresses = append(peerAddresses, _core.Config.AdvertiseAddress)
	}
	return peerAddresses
}
