// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/aspen"
	aspentransmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

type Node struct {
	*distribution.Layer
	Storage *storage.Layer
	// channels is the cluster-shared in-memory channel metadata store, bound as this
	// node's channel retriever. It lets distribution-layer tests create and resolve
	// channels without depending on the service layer.
	channels *ChannelStore
	// owner is non-nil only for the Node returned by OpenNode, where the node owns its
	// single-node cluster. When set, Close tears down the whole cluster (including
	// storage) rather than just this node's layer.
	owner *Cluster
}

// Close closes the node's distribution layer. For a Node returned by OpenNode it also
// closes the underlying single-node cluster, including its storage. This intentionally
// shadows the embedded Layer.Close so that OpenNode's caller can tear everything down
// through the returned node.
func (n Node) Close() error {
	if n.owner != nil {
		return n.owner.Close()
	}
	return n.Layer.Close()
}

type Cluster struct {
	storage     *mock.Cluster
	channels    *ChannelStore
	Nodes       map[node.Key]Node
	writerNet   *tmock.FramerWriterNetwork
	iterNet     *tmock.FramerIteratorNetwork
	channelNet  *tmock.ChannelNetwork
	relayNet    *tmock.FramerRelayNetwork
	deleteNet   *tmock.FramerDeleterNetwork
	aspenNet    *aspentransmock.Network
	addrFactory *address.Factory
	cfg         distribution.LayerConfig
}

// NewCluster opens an n-node in-memory cluster and registers its teardown via
// testutil.DeferClose, so callers running inside a Ginkgo spec do not need to close it
// themselves. It must be called from within a Ginkgo node; use OpenCluster from plain
// Go tests and benchmarks, where DeferCleanup is unavailable.
//
// The teardown runs at the scope where NewCluster is called. When called from a
// BeforeAll or BeforeSuite, the close fires after ShouldNotLeakGoroutinesPerSpec's
// per-spec check, so a spec that spawns a goroutine against this cluster (e.g. an
// observer) must stop it itself rather than relying on cluster teardown.
func NewCluster(ctx context.Context, n int, cfgs ...distribution.LayerConfig) *Cluster {
	return testutil.DeferClose(OpenCluster(ctx, n, cfgs...))
}

// NewNode opens a single-node in-memory cluster and registers its teardown via
// testutil.DeferClose, returning the node. Like NewCluster, it must be called from
// within a Ginkgo node; use OpenNode from plain Go tests and benchmarks.
func NewNode(ctx context.Context, cfgs ...distribution.LayerConfig) Node {
	return testutil.DeferClose(OpenNode(ctx, cfgs...))
}

// OpenCluster opens an n-node in-memory cluster, provisioning every node with the
// provided configs. The caller owns teardown: close the returned Cluster to tear down
// all nodes and their storage.
func OpenCluster(ctx context.Context, n int, cfgs ...distribution.LayerConfig) *Cluster {
	c := newCluster(cfgs...)
	for range n {
		c.Provision(ctx)
	}
	return c
}

// OpenNode opens a single-node in-memory cluster and returns its node. The caller owns
// teardown: closing the returned node tears down the whole cluster, including its
// storage.
func OpenNode(ctx context.Context, cfgs ...distribution.LayerConfig) Node {
	c := OpenCluster(ctx, 1, cfgs...)
	n := c.Nodes[node.KeyBootstrapper]
	n.owner = c
	return n
}

func newCluster(cfgs ...distribution.LayerConfig) *Cluster {
	// NOTE: We don't use config.New here because it returns a zero-value when
	// validation fails (which it will since we don't have required fields).
	// Instead, we manually merge the configs to preserve caller-provided optional
	// values (e.g. AspenOptions, Instrumentation).
	var cfg distribution.LayerConfig
	for _, c := range cfgs {
		cfg = cfg.Override(c)
	}
	return &Cluster{
		cfg:         cfg,
		storage:     mock.NewCluster(),
		channels:    newChannelStore(),
		writerNet:   tmock.NewWriterNetwork(),
		iterNet:     tmock.NewIteratorNetwork(),
		channelNet:  tmock.NewChannelNetwork(),
		relayNet:    tmock.NewRelayNetwork(),
		deleteNet:   tmock.NewDeleterNetwork(),
		aspenNet:    aspentransmock.NewNetwork(),
		addrFactory: address.NewLocalFactory(0),
		Nodes:       make(map[node.Key]Node),
	}
}

func (c *Cluster) Provision(
	ctx context.Context,
	cfgs ...distribution.LayerConfig,
) Node {
	var (
		peers             = c.addrFactory.Generated()
		addr              = c.addrFactory.Next()
		storageLayer      = c.storage.Provision(ctx)
		distributionLayer = testutil.MustSucceed(distribution.OpenLayer(ctx, append([]distribution.LayerConfig{{
			Storage: storageLayer,
			FrameTransport: mockFramerTransport{
				iter:    c.iterNet.New(addr, 1),
				writer:  c.writerNet.New(addr, 1),
				relay:   c.relayNet.New(addr, 1),
				deleter: c.deleteNet.New(addr),
			},
			ChannelTransport: c.channelNet.New(addr),
			AspenTransport:   c.aspenNet.NewTransport(),
			AdvertiseAddress: addr,
			PeerAddresses:    peers,
			AspenOptions: []aspen.Option{
				aspen.WithPropagationConfig(aspen.FastPropagationConfig),
			},
		}, c.cfg}, cfgs...)...))
	)
	node := Node{Layer: distributionLayer, Storage: storageLayer, channels: c.channels}
	c.Nodes[distributionLayer.Cluster.HostKey()] = node
	c.WaitForTopologyToStabilize()
	return node
}

// WaitForTopologyToStabilize waits for all nodes in the cluster to be aware of each
// other.
func (c *Cluster) WaitForTopologyToStabilize() {
	for _, node := range c.Nodes {
		gomega.Eventually(func() int {
			return len(node.Cluster.Nodes())
		}).Should(gomega.Equal(len(c.Nodes)))
	}
}

func (b *Cluster) Close() error {
	var err error
	for _, node := range b.Nodes {
		err = errors.Join(err, node.Close())
	}
	return errors.Join(err, b.storage.Close())
}

type mockFramerTransport struct {
	iter    iterator.Transport
	writer  writer.Transport
	relay   relay.Transport
	deleter deleter.Transport
}

var _ framer.Transport = (*mockFramerTransport)(nil)

func (m mockFramerTransport) Iterator() iterator.Transport { return m.iter }

func (m mockFramerTransport) Writer() writer.Transport { return m.writer }

func (m mockFramerTransport) Relay() relay.Transport { return m.relay }

func (m mockFramerTransport) Deleter() deleter.Transport { return m.deleter }

type StaticHostProvider struct{ Node node.Node }

var _ node.HostProvider = StaticHostProvider{}

func StaticHostKeyProvider(key node.Key) StaticHostProvider {
	return StaticHostProvider{Node: node.Node{Key: key}}
}

func (s StaticHostProvider) Host() node.Node { return s.Node }

func (s StaticHostProvider) HostKey() node.Key { return s.Node.Key }
