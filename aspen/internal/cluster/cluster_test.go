// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"time"
)

var _ = Describe("Cluster", func() {
	var (
		builder    *clustermock.Builder
		clusterCtx signal.Context
		shutdown   context.CancelFunc
	)
	BeforeEach(func() {
		clusterCtx, shutdown = signal.WithCancel(ctx)
		builder = clustermock.NewBuilder(cluster.Config{
			Gossip: gossip.Config{Interval: 5 * time.Millisecond},
			Pledge: pledge.Config{RetryInterval: 1 * time.Millisecond},
		})
	})

	AfterEach(func() {
		shutdown()
		Expect(clusterCtx.Err()).To(HaveOccurredAs(context.Canceled))
	})

	Describe("Node", func() {

		It("Should return a node by its Name", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			c2, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() node.Key {
				n, _ := c2.Node(c1.HostKey())
				return n.Key
			}).Should(Equal(c1.HostKey()))
			Eventually(func() node.Key {
				n, _ := c1.Node(c2.HostKey())
				return n.Key
			}).Should(Equal(c2.HostKey()))
		})

	})

	Describe("Resolve", func() {

		It("Should resolve the address of a node by its Name", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			c2, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() address.Address {
				addr, _ := c1.Resolve(c2.HostKey())
				return addr
			}).Should(Equal(address.Address("localhost:1")))
			Eventually(func() address.Address {
				addr, _ := c2.Resolve(c1.HostKey())
				return addr
			}).Should(Equal(address.Address("localhost:0")))
		})

	})

})
