// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"time"
)

var _ = Describe("Open", func() {
	Context("Valid Properties", func() {

		var (
			gossipNet *mock.Network[gossip.Message, gossip.Message]
			pledgeNet *mock.Network[pledge.Request, pledge.Response]
		)

		BeforeEach(func() {
			gossipNet = mock.NewNetwork[gossip.Message, gossip.Message]()
			pledgeNet = mock.NewNetwork[pledge.Request, pledge.Response]()
		})

		Context("Name Cluster", func() {

			It("Should correctly join the Cluster", func(ctx SpecContext) {

				By("Initializing the Cluster correctly")
				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne := MustSucceed(cluster.Open(
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
				))
				Expect(clusterOne.Host().Key).To(Equal(node.Key(1)))

				By("Pledging a new node to the Cluster")
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)
				clusterTwo := MustSucceed(cluster.Open(
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
				))
				Expect(clusterTwo.Host().Key).To(Equal(node.Key(2)))
				By("Converging Cluster state through gossip")
				Eventually(clusterOne.Nodes).Should(HaveLen(2))
				Eventually(clusterTwo.Nodes).Should(HaveLen(2))

				Expect(clusterOne.Close()).To(Succeed())
				Expect(clusterTwo.Close()).To(Succeed())
			})

		})

		Context("Existing Cluster in Storage", func() {

			It("Should restart Cluster activities using the persisted state", func(ctx SpecContext) {

				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne := MustSucceed(cluster.Open(
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
				))
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
					Codec:                msgpack.Codec,
				}
				clusterTwo := MustSucceed(cluster.Open(ctx, clusterTwoConfig))
				Expect(clusterTwo.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwo.Close()).To(Succeed())

				clusterTwoAgain := MustSucceed(cluster.Open(ctx, clusterTwoConfig))
				Expect(clusterTwoAgain.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwoAgain.Nodes()).To(HaveLen(2))

				Expect(clusterOne.Close()).To(Succeed())
				Expect(clusterTwoAgain.Close()).To(Succeed())
				Expect(kvDB.Close()).To(Succeed())
			})

			It("Should recover state written by msgpack (v0.39 to v0.53 upgrade)", func(ctx SpecContext) {
				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne := MustSucceed(cluster.Open(
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
				))

				kvDB := memkv.New()
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)
				storageKey := []byte("msgpack-upgrade-test")

				// Simulate a v0.39-v0.53 server that wrote state as msgpack.
				oldConfig := cluster.Config{
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
					StorageKey:           storageKey,
					Storage:              kvDB,
					StorageFlushInterval: cluster.FlushOnEvery,
					Codec:                msgpack.Codec,
				}
				clusterTwo := MustSucceed(cluster.Open(ctx, oldConfig))
				Expect(clusterTwo.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwo.Close()).To(Succeed())

				// Reopen with the default codec (JSON primary, msgpack+gob fallback),
				// simulating an upgrade to v0.54+.
				gossipT3 := gossipNet.UnaryServer(gossipT2.Address)
				pledgeT3 := pledgeNet.UnaryServer(gossipT3.Address)
				upgradedConfig := cluster.Config{
					HostAddress: gossipT3.Address,
					Pledge: pledge.Config{
						Peers:           []address.Address{gossipT1.Address},
						TransportClient: pledgeNet.UnaryClient(),
						TransportServer: pledgeT3,
					},
					Gossip: gossip.Config{
						TransportClient: gossipNet.UnaryClient(),
						TransportServer: gossipT3,
						Interval:        100 * time.Millisecond,
					},
					StorageKey:           storageKey,
					Storage:              kvDB,
					StorageFlushInterval: cluster.FlushOnEvery,
				}
				clusterTwoAgain := MustSucceed(cluster.Open(ctx, upgradedConfig))
				Expect(clusterTwoAgain.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwoAgain.Nodes()).To(HaveLen(2))

				Expect(clusterOne.Close()).To(Succeed())
				Expect(clusterTwoAgain.Close()).To(Succeed())
				Expect(kvDB.Close()).To(Succeed())
			})

			It("Should recover state written by gob (pre-v0.39 upgrade)", func(ctx SpecContext) {
				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne := MustSucceed(cluster.Open(
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
				))

				kvDB := memkv.New()
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)
				storageKey := []byte("gob-upgrade-test")

				// Simulate a pre-v0.39 server that wrote state as gob.
				oldConfig := cluster.Config{
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
					StorageKey:           storageKey,
					Storage:              kvDB,
					StorageFlushInterval: cluster.FlushOnEvery,
					Codec:                gob.Codec,
				}
				clusterTwo := MustSucceed(cluster.Open(ctx, oldConfig))
				Expect(clusterTwo.Host().Key).To(Equal(node.Key(2)))
				Expect(clusterTwo.Close()).To(Succeed())

				// Reopen with the default codec, simulating an upgrade to v0.54+.
				gossipT3 := gossipNet.UnaryServer(gossipT2.Address)
				pledgeT3 := pledgeNet.UnaryServer(gossipT3.Address)
				upgradedConfig := cluster.Config{
					HostAddress: gossipT3.Address,
					Pledge: pledge.Config{
						Peers:           []address.Address{gossipT1.Address},
						TransportClient: pledgeNet.UnaryClient(),
						TransportServer: pledgeT3,
					},
					Gossip: gossip.Config{
						TransportClient: gossipNet.UnaryClient(),
						TransportServer: gossipT3,
						Interval:        100 * time.Millisecond,
					},
					StorageKey:           storageKey,
					Storage:              kvDB,
					StorageFlushInterval: cluster.FlushOnEvery,
				}
				clusterTwoAgain := MustSucceed(cluster.Open(ctx, upgradedConfig))
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
