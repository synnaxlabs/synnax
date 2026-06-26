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
	aspentransportmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	storagemock "github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

// Cluster is an in-memory, multi-node distribution cluster for use in tests and
// benchmarks. Every node shares the same in-process transport and storage backing, so
// no real network or disk is involved. Cluster is not safe for concurrent use.
type Cluster struct {
	storage *storagemock.Cluster
	// Nodes maps each provisioned node's host key to its Node.
	Nodes       map[node.Key]Node
	writerNet   *mock.FramerWriterNetwork
	iterNet     *mock.FramerIteratorNetwork
	channelNet  *mock.ChannelNetwork
	relayNet    *mock.FramerRelayNetwork
	deleteNet   *mock.FramerDeleterNetwork
	aspenNet    *aspentransportmock.Network
	addrFactory *address.Factory
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
func NewCluster(ctx context.Context, n int) *Cluster {
	return testutil.DeferClose(OpenCluster(ctx, n))
}

// OpenCluster opens an n-node in-memory cluster. The caller owns teardown: close the
// returned Cluster to tear down all nodes and their storage.
func OpenCluster(ctx context.Context, n int) *Cluster {
	c := &Cluster{
		storage:     storagemock.NewCluster(),
		writerNet:   mock.NewWriterNetwork(),
		iterNet:     mock.NewIteratorNetwork(),
		channelNet:  mock.NewChannelNetwork(),
		relayNet:    mock.NewRelayNetwork(),
		deleteNet:   mock.NewDeleterNetwork(),
		aspenNet:    aspentransportmock.NewNetwork(),
		addrFactory: address.NewLocalFactory(0),
		Nodes:       make(map[node.Key]Node),
	}
	for range n {
		c.Provision(ctx)
	}
	return c
}

// Provision provisions a new Node in the cluster and returns it. The optional overrides
// are layered on top of the base distribution.LayerConfig, allowing a caller to tweak
// distribution-layer behavior (e.g. name validation) for a single node.
func (c *Cluster) Provision(
	ctx context.Context, overrides ...distribution.LayerConfig,
) Node {
	var (
		peers        = c.addrFactory.Generated()
		addr         = c.addrFactory.Next()
		storageLayer = c.storage.Provision(ctx)
		cfg          = distribution.LayerConfig{
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
		}
		cfgs              = append([]distribution.LayerConfig{cfg}, overrides...)
		distributionLayer = testutil.MustSucceed(distribution.OpenLayer(ctx, cfgs...))
	)
	node := Node{Layer: distributionLayer, Storage: storageLayer}
	c.Nodes[distributionLayer.Cluster.HostKey()] = node
	for _, node := range c.Nodes {
		gomega.Eventually(func() int {
			return len(node.Cluster.Nodes())
		}).Should(gomega.Equal(len(c.Nodes)))
	}
	return node
}

// Close tears down every node in the cluster along with the shared storage, returning
// the joined error of all teardown failures.
func (c *Cluster) Close() error {
	var err error
	for _, node := range c.Nodes {
		err = errors.Join(err, node.Close())
	}
	return errors.Join(err, c.storage.Close())
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
