// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	aspentransmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	mockstorage "github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"io"
)

// CoreBuilder is a utility for provisioning mock distribution cores that
// form a cluster. To create a new CoreBuilder, call NewCoreBuilder
type CoreBuilder struct {
	// Builder is the underlying storage builder.
	mockstorage.Builder
	// Config is the configuration used to provision new cores.
	Config core.Config
	// Cores contains a map of all cores paired with their respective host Name.
	Cores map[core.NodeKey]core.Core
	// net is the network for transporting key-value operations.
	net *aspentransmock.Network
	// addrFactory generates unique addresses for nodes.
	addrFactory *address.Factory
	closers     []io.Closer
}

// NewCoreBuilder opens a new CoreBuilder that provisions cores using the given
// configuration.
func NewCoreBuilder(configs ...core.Config) *CoreBuilder {
	cfg, err := config.New(core.DefaultConfig, append([]core.Config{{
		Storage: storage.Config{MemBacked: config.Bool(true)},
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
		Cores:       make(map[core.NodeKey]core.Core),
		net:         net,
		addrFactory: addrFactory,
	}
}

// New provisions a new core connected to the rest of the nodes in the builder's cluster.
// Panics if the core cannot be opened.
func (cb *CoreBuilder) New() core.Core {
	store := cb.Builder.New()
	trans := cb.net.NewTransport()
	addr := cb.addrFactory.Next()

	clusterKV := lo.Must(aspen.Open(
		context.TODO(),
		/* dirname */ "",
		addr,
		cb.peerAddresses(),
		aspen.WithEngine(store.KV),
		aspen.WithInstrumentation(cb.Config.Instrumentation),
		aspen.WithTransport(trans),
		aspen.WithPropagationConfig(aspen.FastPropagationConfig),
	))

	cfg := cb.Config
	cfg.AdvertiseAddress = addr

	store.KV = clusterKV

	_core := distribution.Core{
		Config:  cfg,
		Cluster: clusterKV.Cluster,
		Storage: store,
	}

	cb.Cores[_core.Cluster.HostKey()] = _core
	return _core
}

// AttachCloser attaches a closer to the CoreBuilder. The closer will be called when
// the CoreBuilder is closed.
func (cb *CoreBuilder) AttachCloser(closer io.Closer) {
	cb.closers = append(cb.closers, closer)
}

// Close shuts down all other nodes in the cluster. It is not safe to call this method
// while the nodes are still in use.
func (cb *CoreBuilder) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, closer := range cb.closers {
		c.Exec(closer.Close)
	}
	for _, core_ := range cb.Cores {
		c.Exec(core_.Close)
	}
	return c.Error()
}

// WaitForTopologyToStabilize waits for all nodes in the cluster to be aware of each
// other.
func (cb *CoreBuilder) WaitForTopologyToStabilize() {
	for _, _c := range cb.Cores {
		c := _c
		gomega.Eventually(func() int {
			return len(c.Cluster.Nodes())
		}).Should(gomega.Equal(len(cb.Cores)))
	}
}

func (cb *CoreBuilder) peerAddresses() (addrs []address.Address) {
	for _, _core := range cb.Cores {
		addrs = append(addrs, _core.Config.AdvertiseAddress)
	}
	return
}

type StaticHostProvider struct {
	Node core.Node
}

var _ core.HostProvider = StaticHostProvider{}

func StaticHostKeyProvider(key core.NodeKey) StaticHostProvider {
	return StaticHostProvider{Node: core.Node{Key: key}}
}

func (s StaticHostProvider) Host() core.Node { return s.Node }

func (s StaticHostProvider) HostKey() core.NodeKey { return s.Node.Key }
