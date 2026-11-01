// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/testutil"
	transporttestutil "github.com/synnaxlabs/aspen/transport/testutil"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// specTimeout bounds every spec in this file. Aspen retries pledges until its context
// is cancelled, so a node that never joins would otherwise hang until the suite ends.
const specTimeout = 30 * time.Second

// fastPledgeConfig keeps the pledge retry interval flat. The default scale grows it by
// 25% per failed pledge, so a node that loses early rounds under load waits minutes.
var fastPledgeConfig = aspen.PropagationConfig{
	PledgeRetryInterval: 10 * time.Millisecond,
	PledgeRetryScale:    1,
}

var _ = Describe("Membership", Serial, Ordered, func() {
	Describe("Bootstrap Cluster", func() {
		It("Should correctly bootstrap a cluster", func(ctx SpecContext) {
			db := MustSucceed(aspen.Open(
				ctx,
				GinkgoT().TempDir(),
				"localhost:0",
				[]address.Address{},
			))

			By("Assigning a valid Name of 1")
			Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))

			By("Adding itself to the node list")
			Expect(db.Cluster.Nodes()).To(HaveLen(1))

			By("By setting its state to healthy")
			Expect(db.Cluster.Host().State).To(Equal(aspen.NodeStateHealthy))

			Expect(db.Close()).To(Succeed())
		}, SpecTimeout(specTimeout))

		It(
			"Should correctly join a node that is already looking for peers",
			func(ctx SpecContext) {
				// The in-memory network fixes the bootstrapper's address before it
				// opens,
				// so the pledging node can look for it first.
				var (
					network       = transporttestutil.NewNetwork()
					bootstrapAddr = address.Address("localhost:1")
					pledgerKV     = DeferClose(memkv.New())
					wg            sync.WaitGroup
				)
				wg.Add(1)
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					db := MustSucceed(aspen.Open(
						ctx,
						"",
						"localhost:2",
						[]address.Address{bootstrapAddr},
						aspen.WithEngine(pledgerKV),
						aspen.WithTransport(network.NewTransport("localhost:2")),
						aspen.WithPropagationConfig(fastPledgeConfig),
					))
					defer func() { Expect(db.Close()).To(Succeed()) }()

					By("Assigning a unique Name of 2")
					Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(2)))
				}()
				db := MustSucceed(aspen.Open(
					ctx,
					"",
					bootstrapAddr,
					[]address.Address{},
					aspen.WithEngine(DeferClose(memkv.New())),
					aspen.WithTransport(network.NewTransport(bootstrapAddr)),
				))

				By("Assigning a unique Name of 1")
				Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))
				wg.Wait()

				By("Safely closing the database")
				Expect(db.Close()).To(Succeed())
			},
			SpecTimeout(specTimeout),
		)
	})

	Describe("Concurrent Pledges", func() {
		It(
			"Should correctly join many nodes to the cluster concurrently",
			func(ctx SpecContext) {
				numPledges := 9
				bootstrapper := MustSucceed(aspen.Open(
					ctx,
					GinkgoT().TempDir(),
					"localhost:0",
					[]address.Address{},
				))
				peers := []address.Address{bootstrapper.Cluster.Host().Address}
				wg := sync.WaitGroup{}
				wg.Add(numPledges)
				var (
					ids = make([]aspen.NodeKey, numPledges)
					dbs = make([]*aspen.DB, numPledges)
				)
				for i := range numPledges {
					dir := GinkgoT().TempDir()
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						db := MustSucceed(aspen.Open(
							ctx,
							dir,
							"localhost:0",
							peers,
							aspen.WithPropagationConfig(fastPledgeConfig),
						))
						ids[i] = db.Cluster.HostKey()
						dbs[i] = db
					}(i)
				}
				wg.Wait()

				By("Assigning a unique Name to each node")
				ids = append(ids, bootstrapper.Cluster.HostKey())
				Expect(lo.Uniq(ids)).To(HaveLen(len(ids)))

				By("Safely closing the database")
				Expect(bootstrapper.Close()).To(Succeed())
				for _, db := range dbs {
					Expect(db.Close()).To(Succeed())
				}
			},
			SpecTimeout(specTimeout),
		)
	})

	Describe("Joining, Dying, and Rejoining", func() {
		Context("Persisted storage", func() {
			Context("Single node death", func() {
				It(
					"Should correctly handle a single node dying and rejoining",
					func(ctx SpecContext) {
						propConfig := aspen.PropagationConfig{
							PledgeRetryInterval:   10 * time.Millisecond,
							PledgeRetryScale:      1,
							ClusterGossipInterval: 50 * time.Millisecond,
						}
						builder := &testutil.Builder{
							DataDir: "./testdata",
							DefaultOptions: []aspen.Option{
								aspen.WithPropagationConfig(propConfig),
							},
							Nodes: make(map[aspen.NodeKey]testutil.NodeInfo),
						}
						defer func() {
							Expect(builder.Cleanup()).To(Succeed())
						}()

						By("Forking the databases")
						for range 3 {
							MustSucceed(builder.New(ctx))
						}

						By("Assigning the correct generation")
						node := builder.Nodes[2]
						Expect(
							node.DB.Cluster.Host().Heartbeat.Generation,
						).To(Equal(uint32(0)))

						By("Closing the database")
						Expect(node.DB.Close()).To(Succeed())

						By("Opening the database again")
						db := MustSucceed(aspen.Open(
							ctx,
							node.Dir,
							node.Addr,
							[]address.Address{},
							builder.DefaultOptions...,
						))

						By("Assigning the correct Name")
						Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(2)))

						By("Incrementing the heartbeat generation")
						Expect(
							db.Cluster.Host().Heartbeat.Generation,
						).To(Equal(uint32(1)))

						By("Propagating the incremented heartbeat to other nodes")
						ctx1 := builder.Nodes[1]
						Eventually(func(g Gomega) {
							n2, err := ctx1.DB.Cluster.Node(2)
							g.Expect(err).ToNot(HaveOccurred())
							g.Expect(n2.State).To(Equal(aspen.NodeStateHealthy))
							g.Expect(n2.Heartbeat.Generation).To(Equal(uint32(1)))
						}).Should(Succeed())

						By("Closing the databases")
						Expect(builder.Nodes[1].DB.Close()).To(Succeed())
						Expect(builder.Nodes[3].DB.Close()).To(Succeed())
						Expect(db.Close()).To(Succeed())
					},
					SpecTimeout(specTimeout),
				)
			})
		})
	})
})
