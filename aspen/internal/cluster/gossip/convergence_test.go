// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gossip_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/internal/node"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/rand"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Convergence", func() {
	var net *freightertestutil.Network[gossip.Message, gossip.Message]
	BeforeEach(func() {
		net = freightertestutil.NewNetwork[gossip.Message, gossip.Message]()
	})
	DescribeTable(
		"Should converge the store",
		func(ctx SpecContext, nodeCount, initialViewCount, cycles int) {
			group := make(node.Group)
			configs := make(map[node.Key]gossip.Config)
			for i := 1; i <= nodeCount; i++ {
				server := net.UnaryServer("")
				n := node.Node{Key: node.Key(i), Address: server.Address()}
				group[n.Key] = n
				configs[n.Key] = gossip.Config{
					TransportServer: server,
					TransportClient: net.UnaryClient(),
				}
			}
			var (
				gossips []*gossip.Gossip
				stores  []store.Store
			)
			for _, n := range group {
				subNodes := rand.SubMap(group.WhereNot(n.Key), initialViewCount)
				subNodes[n.Key] = n
				s := store.New(ctx)
				s.SetState(ctx, store.State{Nodes: subNodes, HostKey: n.Key})
				cfg := configs[n.Key]
				cfg.Store = s
				g := MustSucceed(gossip.New(cfg))
				gossips = append(gossips, g)
				stores = append(stores, s)
			}
			for range cycles {
				wg := sync.WaitGroup{}
				for _, g := range gossips {
					wg.Add(1)
					go func(g *gossip.Gossip) {
						defer GinkgoRecover()
						defer wg.Done()
						Expect(g.GossipOnce(ctx)).To(Succeed())
					}(g)
				}
				wg.Wait()
			}
			for _, s := range stores {
				Expect(s.CopyState().Nodes).To(HaveLen(nodeCount))
			}
		},
		Entry("across 4 nodes in 10 cycles", 4, 2, 10),
		Entry("across 10 nodes in 10 cycles", 10, 2, 10),
		Entry("across 30 nodes in 10 cycles", 30, 2, 10),
		Entry("across 100 nodes in 10 cycles", 100, 5, 10),
	)
})
