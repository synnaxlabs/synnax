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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/signal"
	"time"
)

var _ = Describe("Open", func() {
	Context("Valid Properties", func() {

		var (
			gossipNet *fmock.Network[gossip.Message, gossip.Message]
			pledgeNet *fmock.Network[pledge.Request, pledge.Response]
		)

		BeforeEach(func() {
			gossipNet = fmock.NewNetwork[gossip.Message, gossip.Message]()
			pledgeNet = fmock.NewNetwork[pledge.Request, pledge.Response]()
		})

		Context("Name Cluster", func() {

			It("Should correctly join the Cluster", func() {

				By("Initializing the Cluster correctly")
				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne, err := cluster.Open(
					ctx,
					cluster.Config{
						HostAddress: gossipT1.Address,
						Pledge: pledge.Config{
							Peers:           []address.Address{},
							TransportClient: pledgeNet.UnaryClient(),
							TransportServer: pledgeT1,
						},
						Gossip: gossip.Config{
							TransportClient: gossipNet.UnaryClient(),
							TransportServer: gossipT1,
							Interval:        100 * time.Millisecond,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterOne.Host().Key).To(Equal(node.Key(1)))

				By("Pledging a new node to the Cluster")
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)
				clusterTwo, err := cluster.Open(
					ctx,
					cluster.Config{
						HostAddress: gossipT2.Address,
						Pledge: pledge.Config{
							Peers:           []address.Address{gossipT1.Address},
							TransportServer: pledgeT2,
							TransportClient: pledgeNet.UnaryClient(),
						},
						Gossip: gossip.Config{
							TransportServer: gossipT2,
							TransportClient: gossipNet.UnaryClient(),
							Interval:        100 * time.Millisecond,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwo.Host().Key).To(Equal(node.Key(2)))
				By("Converging Cluster state through gossip")
				Eventually(clusterOne.Nodes).Should(HaveLen(2))
				Eventually(clusterTwo.Nodes).Should(HaveLen(2))

				Expect(clusterOne.Close()).To(Succeed())
				Expect(clusterTwo.Close()).To(Succeed())
			})

		})

		Context("Existing Cluster in Storage", func() {

			It("Should restart Cluster activities using the persisted state", func() {

				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne, err := cluster.Open(
					ctx,
					cluster.Config{
						HostAddress: gossipT1.Address,
						Pledge: pledge.Config{
							Peers:           []address.Address{},
							TransportClient: pledgeNet.UnaryClient(),
							TransportServer: pledgeT1,
						},
						Gossip: gossip.Config{
							TransportClient: gossipNet.UnaryClient(),
							TransportServer: gossipT1,
							Interval:        100 * time.Millisecond,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterOne.Host().Key).To(Equal(node.Key(1)))

				kvDB := memkv.New()
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)

				clusterTwoConfig := cluster.Config{
					HostAddress: gossipT2.Address,
					Pledge: pledge.Config{
						Peers:           []address.Address{gossipT1.Address},
						TransportClient: pledgeNet.UnaryClient(),
						TransportServer: pledgeT2,
					},
					Gossip: gossip.Config{
						TransportClient: gossipNet.UnaryClient(),
						TransportServer: gossipT2,
						Interval:        100 * time.Millisecond,
					},
					StorageKey:           []byte("Cluster-join-test-storage"),
					Storage:              kvDB,
					StorageFlushInterval: cluster.FlushOnEvery,
					Codec:                &binary.MsgPackCodec{},
				}
				clusterTwo, err := cluster.Open(ctx, clusterTwoConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwo.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwo.Close()).To(Succeed())

				clusterTwoAgain, err := cluster.Open(ctx, clusterTwoConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwoAgain.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwoAgain.Nodes()).To(HaveLen(2))

				Expect(clusterOne.Close()).To(Succeed())
				Expect(clusterTwoAgain.Close()).To(Succeed())
				Expect(kvDB.Close()).To(Succeed())
			})

		})

	})

	Context("Invalid Properties", func() {
		It("Should return an error ", func() {
			cfg := cluster.Config{}
			ctx, cancel := signal.Isolated()
			defer cancel()
			_, err := cluster.Open(ctx, cfg)
			Expect(err).To(HaveOccurred())
		})
	})
})
