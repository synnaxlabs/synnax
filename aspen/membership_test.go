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
	stdnet "net"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/mock"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Membership", Serial, Ordered, func() {
	Describe("Bootstrap Cluster", func() {
		It("Should correctly bootstrap a cluster", func(ctx SpecContext) {
			db := MustSucceed(aspen.Open(
				ctx,
				"",
				ephemeralAddress,
				[]aspen.Address{},
				aspen.Bootstrap(),
				aspen.InMemory(),
			))

			By("Assigning a valid Name of 1")
			Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))

			By("Adding itself to the node list")
			Expect(db.Cluster.Nodes()).To(HaveLen(1))

			By("By setting its state to healthy")
			Expect(db.Cluster.Host().State).To(Equal(aspen.NodeStateHealthy))

			Expect(db.Close()).To(Succeed())
		})

		It(
			"Should correctly bootstrap a cluster with peers provided",
			func(ctx SpecContext) {
				db := MustSucceed(aspen.Open(
					ctx,
					"",
					ephemeralAddress,
					[]aspen.Address{unreachableAddress},
					aspen.InMemory(),
					aspen.Bootstrap(),
				))
				defer func() { Expect(db.Close()).To(Succeed()) }()

				By("Assigning a valid Name of 1")
				Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))
			},
		)

		It(
			"Should correctly join a node that is already looking for peers",
			func(ctx SpecContext) {
				wg := sync.WaitGroup{}
				wg.Add(1)
				// The pledging node must know where the bootstrapper will be before the
				// bootstrapper opens, so hold the address until it is needed.
				reserved := MustSucceed(stdnet.Listen("tcp", ephemeralAddress.String()))
				bootstrapAddr := address.Address(reserved.Addr().String())
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					db := MustSucceed(aspen.Open(
						ctx,
						"",
						ephemeralAddress,
						[]aspen.Address{bootstrapAddr},
						aspen.InMemory(),
					))
					defer func() { Expect(db.Close()).To(Succeed()) }()

					By("Assigning a unique Name of 2")
					Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(2)))
				}()
				Expect(reserved.Close()).To(Succeed())
				db := MustSucceed(aspen.Open(
					ctx,
					"",
					bootstrapAddr,
					[]aspen.Address{},
					aspen.InMemory(),
					aspen.Bootstrap(),
				))

				By("Assigning a unique Name of 1")
				Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))
				wg.Wait()

				By("Safely closing the database")
				Expect(db.Close()).To(Succeed())
			},
		)
	})

	Describe("Concurrent Pledges", func() {
		It(
			"Should correctly join many nodes to the cluster concurrently",
			func(ctx SpecContext) {
				numPledges := 9
				bootstrapper := MustSucceed(aspen.Open(
					ctx,
					"",
					ephemeralAddress,
					[]aspen.Address{},
					aspen.InMemory(),
					aspen.Bootstrap(),
				))
				peers := []aspen.Address{bootstrapper.Cluster.Host().Address}
				wg := sync.WaitGroup{}
				wg.Add(numPledges)
				var (
					ids = make([]aspen.NodeKey, numPledges)
					dbs = make([]*aspen.DB, numPledges)
				)
				for i := range numPledges {
					go func(i int) {
						defer GinkgoRecover()
						defer wg.Done()
						db := MustSucceed(aspen.Open(
							ctx,
							"",
							ephemeralAddress,
							peers,
							aspen.InMemory(),
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
						builder := &mock.Builder{
							DataDir: "./testdata",
							DefaultOptions: []aspen.Option{
								aspen.WithPropagationConfig(propConfig),
							},
							Nodes: make(map[aspen.NodeKey]mock.NodeInfo),
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
							[]aspen.Address{},
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
				)
			})
		})
	})
})
