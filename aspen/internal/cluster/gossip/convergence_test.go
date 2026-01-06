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
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/rand"
	"sync"
)

type convergenceVars struct {
	nodeCount            int
	convergenceThreshold int
	initialViewCount     int
}

var progressiveConvergence = []convergenceVars{
	{
		nodeCount:            4,
		initialViewCount:     2,
		convergenceThreshold: 10,
	},
	{
		nodeCount:            10,
		initialViewCount:     2,
		convergenceThreshold: 10,
	},
	{
		nodeCount:            30,
		initialViewCount:     2,
		convergenceThreshold: 10,
	},
	{
		nodeCount:            100,
		initialViewCount:     5,
		convergenceThreshold: 10,
	},
}

var _ = Describe("Convergence", func() {
	var (
		net *fmock.Network[gossip.Message, gossip.Message]
	)
	BeforeEach(func() {
		net = fmock.NewNetwork[gossip.Message, gossip.Message]()
	})
	p := alamos.NewParametrize(alamos.IterVars(progressiveConvergence))
	p.Template(func(i int, values convergenceVars) {
		It(fmt.Sprintf("Should converge store across %v nodes in %v cycles",
			values.nodeCount,
			values.convergenceThreshold,
		), func() {
			group := make(node.Group)
			configs := make(map[node.Key]gossip.Config)
			for i := 1; i <= values.nodeCount; i++ {
				server := net.UnaryServer("")
				n := node.Node{Key: node.Key(i), Address: server.Address}
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
				subNodes := rand.SubMap(group.WhereNot(n.Key), values.initialViewCount)
				subNodes[n.Key] = n
				s := store.New(ctx)
				s.SetState(ctx, store.State{Nodes: subNodes, HostKey: n.Key})
				cfg := configs[n.Key]
				cfg.Store = s
				g, err := gossip.New(cfg)
				Expect(err).ToNot(HaveOccurred())
				gossips = append(gossips, g)
				stores = append(stores, s)
			}
			ctx := context.Background()
			for range values.convergenceThreshold {
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
				Expect(s.CopyState().Nodes).To(HaveLen(values.nodeCount))
			}
		})
	})
	p.Construct()
})
