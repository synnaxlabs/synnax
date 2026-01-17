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
	"github.com/synnaxlabs/aspen/mock"
	"github.com/synnaxlabs/x/address"
	xnet "github.com/synnaxlabs/x/net"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Membership", Serial, Ordered, func() {
	Describe("Bootstrap Cluster", func() {

		It("Should correctly bootstrap a cluster", func() {
			db, err := aspen.Open(
				ctx,
				"",
				"localhost:22546",
				[]aspen.Address{},
				aspen.Bootstrap(),
				aspen.InMemory(),
			)

			By("Opening without error")
			Expect(err).ToNot(HaveOccurred())

			By("Assigning a valid Name of 1")
			Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))

			By("Adding itself to the node list")
			Expect(db.Cluster.Nodes()).To(HaveLen(1))

			By("By setting its state to healthy")
			Expect(db.Cluster.Host().State).To(Equal(aspen.NodeStateHealthy))

			Expect(db.Close()).To(Succeed())
		})

		It("Should correctly bootstrap a cluster with peers provided", func() {
			addr1 := address.Newf("localhost:%v", MustSucceed(xnet.FindOpenPort()))
			db, err := aspen.Open(
				ctx,
				"",
				addr1,
				[]aspen.Address{"localhost:22547"},
				aspen.InMemory(),
				aspen.Bootstrap(),
			)
			defer func() { Expect(db.Close()).To(Succeed()) }()

			By("Opening without error")
			Expect(err).ToNot(HaveOccurred())

			By("Assigning a valid Name of 1")
			Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))
		})

		It("Should correctly join a node that is already looking for peers", func() {
			wg := sync.WaitGroup{}
			wg.Add(1)
			addr1 := address.Newf("localhost:%v", MustSucceed(xnet.FindOpenPort()))
			addr2 := address.Newf("localhost:%v", MustSucceed(xnet.FindOpenPort()))
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
				defer cancel()
				db, err := aspen.Open(
					ctx,
					"",
					addr1,
					[]aspen.Address{addr2},
					aspen.InMemory(),
				)
				defer func() { Expect(db.Close()).To(Succeed()) }()

				By("Joining the second node to the cluster without error")
				Expect(err).ToNot(HaveOccurred())

				By("Assigning a unique Name of 2")
				Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(2)))
			}()
			db, err := aspen.Open(
				ctx,
				"",
				addr2,
				[]aspen.Address{},
				aspen.InMemory(),
				aspen.Bootstrap(),
			)

			By("Joining the first node to the cluster without error")
			Expect(err).ToNot(HaveOccurred())

			By("Assigning a unique Name of 1")
			Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(1)))
			wg.Wait()

			By("Safely closing the database")
			Expect(db.Close()).To(Succeed())
		})

	})

	Describe("Concurrent Pledges", func() {

		It("Should correctly join many nodes to the cluster concurrently", func() {
			numNodes := 10
			wg := sync.WaitGroup{}
			wg.Add(numNodes)
			var (
				addresses = address.NewLocalFactory(22546).NextN(numNodes)
				ids       = make([]aspen.NodeKey, numNodes)
				dbs       = make([]*aspen.DB, numNodes)
			)
			for i := range numNodes {
				go func(i int) {
					defer GinkgoRecover()
					defer wg.Done()
					opts := []aspen.Option{aspen.InMemory()}
					if i == 0 {
						opts = append(opts, aspen.Bootstrap())
					}
					db, err := aspen.Open(
						context.TODO(), "",
						addresses[i],
						addresses,
						opts...,
					)
					ids[i] = db.Cluster.HostKey()
					dbs[i] = db
					By("Joining the node to the cluster without error")
					Expect(err).ToNot(HaveOccurred())
				}(i)
			}
			wg.Wait()

			By("Assigning a unique Name to each node")
			Expect(len(lo.Uniq(ids))).To(Equal(len(ids)))

			By("Safely closing the database")
			for _, db := range dbs {
				Expect(db.Close()).To(Succeed())
			}
		})

	})

	Describe("Joining, Dying, and Rejoining", func() {
		Context("Persisted storage", func() {
			Context("Single node death", func() {
				It("Should correctly handle a single node dying and rejoining", func() {
					propConfig := aspen.PropagationConfig{
						PledgeRetryInterval:   10 * time.Millisecond,
						PledgeRetryScale:      1,
						ClusterGossipInterval: 50 * time.Millisecond,
					}
					builder := &mock.Builder{
						PortRangeStart: 22546,
						DataDir:        "./testdata",
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
						_, err := builder.New(ctx)
						Expect(err).ToNot(HaveOccurred())
					}

					By("Assigning the correct generation")
					node := builder.Nodes[2]
					Expect(node.DB.Cluster.Host().Heartbeat.Generation).To(Equal(uint32(0)))

					By("Closing the database")
					Expect(node.DB.Close()).To(Succeed())

					By("Opening the database again")
					db, err := aspen.Open(
						context.TODO(),
						node.Dir,
						node.Addr,
						[]aspen.Address{},
						builder.DefaultOptions...,
					)
					Expect(err).ToNot(HaveOccurred())

					By("Assigning the correct Name")
					Expect(db.Cluster.HostKey()).To(Equal(aspen.NodeKey(2)))

					By("Incrementing the heartbeat generation")
					Expect(db.Cluster.Host().Heartbeat.Generation).To(Equal(uint32(1)))

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
				})
			})
		})
	})

})
