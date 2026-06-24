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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

type Cluster struct {
	storage     *mock.Cluster
	Nodes       map[node.Key]Node
	net         *tmock.Network
	aspenNet    *aspentransmock.Network
	addrFactory *address.Factory
}

// MustOpenCluster opens an n-node in-memory cluster and registers its teardown via
// testutil.DeferClose, so callers running inside a Ginkgo spec do not need to close it
// themselves. It must be called from within a Ginkgo node; use OpenCluster from plain
// Go tests and benchmarks, where DeferCleanup is unavailable.
//
// The teardown runs at the scope where MustOpenCluster is called. When called from a
// BeforeAll or BeforeSuite, the close fires after ShouldNotLeakGoroutinesPerSpec's
// per-spec check, so a spec that spawns a goroutine against this cluster (e.g. an
// observer) must stop it itself rather than relying on cluster teardown.
func MustOpenCluster(ctx context.Context, n int) *Cluster {
	return testutil.DeferClose(OpenCluster(ctx, n))
}

// OpenCluster opens an n-node in-memory cluster. The caller owns teardown: close the
// returned Cluster to tear down all nodes and their storage.
func OpenCluster(ctx context.Context, n int) *Cluster {
	c := newCluster()
	for range n {
		c.Provision(ctx)
	}
	return c
}

func newCluster() *Cluster {
	return &Cluster{
		storage:     mock.NewCluster(),
		net:         tmock.NewNetwork(),
		aspenNet:    aspentransmock.NewNetwork(),
		addrFactory: address.NewLocalFactory(0),
		Nodes:       make(map[node.Key]Node),
	}
}

// Provision provisions a new Node in the cluster and returns it. The optional
// overrides are layered on top of the base distribution.LayerConfig, allowing a caller
// to tweak distribution-layer behavior (e.g. name validation) for a single node.
func (c *Cluster) Provision(ctx context.Context, overrides ...distribution.LayerConfig) Node {
	var (
		peers        = c.addrFactory.Generated()
		addr         = c.addrFactory.Next()
		storageLayer = c.storage.Provision(ctx)
		cfgs         = append([]distribution.LayerConfig{{
			Storage:          storageLayer,
			Transport:        c.net.New(addr, 1),
			AspenTransport:   c.aspenNet.NewTransport(),
			AdvertiseAddress: addr,
			PeerAddresses:    peers,
			AspenOptions: []aspen.Option{
				aspen.WithPropagationConfig(aspen.FastPropagationConfig),
			},
		}}, overrides...)
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

func (c *Cluster) Close() error {
	var err error
	for _, node := range c.Nodes {
		err = errors.Join(err, node.Close())
	}
	return errors.Join(err, c.storage.Close())
}
