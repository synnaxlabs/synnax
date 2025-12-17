// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"context"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/kv/kvmock"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("txn", func() {
	var builder *kvmock.Builder

	BeforeEach(func() {
		builder = kvmock.NewBuilder(
			kv.Config{
				RecoveryThreshold: 12,
				GossipInterval:    10 * time.Millisecond,
			},
			cluster.Config{
				Gossip: gossip.Config{Interval: 10 * time.Millisecond},
				Pledge: pledge.Config{RetryInterval: 10 * time.Millisecond},
			},
		)
	})

	AfterEach(func() {
		Expect(builder.Close()).To(Succeed())
	})

	Describe("StreamServer", func() {

		It("Should open a new database without error", func() {
			kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv).ToNot(BeNil())
		})

	})

	Describe("SetNode", func() {

		Describe("Gateway Leaseholder", func() {

			It("Should commit the operation to storage", func() {
				kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv).ToNot(BeNil())
				Expect(kv.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
				v, closer, err := kv.Get(ctx, []byte("key"))
				Expect(err).ToNot(HaveOccurred())
				Expect(v).To(Equal([]byte("value")))
				Expect(closer.Close()).To(Succeed())
			})

			It("Should propagate the operation to other members of the cluster",
				func() {
					kv1, err := builder.New(ctx, kv.Config{
						Instrumentation: Instrumentation("kv1"),
					}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
					Eventually(func(g Gomega) {
						v, closer, err := kv2.Get(ctx, []byte("key"))
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(v).To(Equal([]byte("value")))
						Expect(closer.Close()).To(Succeed())
					}).Should(Succeed())
				})
			It("Should forward an update to the Leaseholder", func() {
				kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
				Eventually(func(g Gomega) {
					v, closer, err := kv2.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					g.Expect(kv2.Set(ctx, []byte("key"), []byte("value2"))).To(Succeed())
					Expect(closer.Close()).To(Succeed())
				}).Should(Succeed())
				Expect(func(g Gomega) {
					v, closer, err := kv1.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value2")))
					Expect(closer.Close()).To(Succeed())
					v, closer, err = kv1.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value2")))
					Expect(closer.Close()).To(Succeed())
				})
			})

			It("Should return an error when attempting to transfer the lease",
				func() {
					kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					_, err = builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
					err = kv1.Set(ctx, []byte("key"), []byte("value2"), node.Key(2))
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, kv.ErrLeaseNotTransferable)).To(BeTrue())
				})

		})

		Describe("Peers Leaseholder", func() {
			It("Should commit the operation to storage", func() {
				kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				waitForClusterStateToConverge(builder)
				Expect(kv1.Set(ctx, []byte("key"), []byte("value"), node.Key(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, closer, err := kv2.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					Expect(closer.Close()).To(Succeed())
				}).Should(Succeed())
			})

			It("Should return an error if the lease option is not a node Name", func() {
				kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Set(ctx, []byte("key"), []byte("value"), "2")).To(HaveOccurred())
			})
		})

	})

	Describe("Tx", func() {
		It("Should execute a set of operations", func() {
			kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv).ToNot(BeNil())
			txn := kv.OpenTx()
			Expect(txn.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(txn.Set(ctx, []byte("key2"), []byte("value2"))).To(Succeed())
			Expect(txn.Commit(ctx)).To(Succeed())
			v, closer, err := kv.Get(ctx, []byte("key"))
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal([]byte("value")))
			Expect(closer.Close()).To(Succeed())
			v, closer, err = kv.Get(ctx, []byte("key2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal([]byte("value2")))
			Expect(closer.Close()).To(Succeed())
		})

	})

	Describe("delete", func() {

		Describe("Gateway Leaseholder", func() {
			It("Should apply the operation to storage", func() {
				kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv).ToNot(BeNil())
				Expect(kv.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
				v, closer, err := kv.Get(ctx, []byte("key"))
				Expect(err).ToNot(HaveOccurred())
				Expect(v).To(Equal([]byte("value")))
				Expect(kv.Delete(ctx, []byte("key"))).To(Succeed())
				Expect(closer.Close()).To(Succeed())
				v, closer, err = kv.Get(ctx, []byte("key"))
				Expect(err).To(HaveOccurred())
				Expect(v).To(BeNil())
				Expect(closer).To(BeNil())
			})
		})

		Describe("Peer Leaseholder", func() {
			It("Should apply the operation to storage", func() {
				kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				waitForClusterStateToConverge(builder)
				Expect(kv1.Set(ctx, []byte("key"), []byte("value"), node.Key(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, closer, err := kv2.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					Expect(closer.Close()).To(Succeed())
				}).Should(Succeed())
			})
		})

	})

	Describe("Request Recovery", func() {
		It("Should stop propagating an operation after a set threshold of"+
			" redundant broadcasts", func() {
			kv1, err := builder.New(ctx, kv.Config{
				GossipInterval:    20 * time.Millisecond,
				RecoveryThreshold: 2,
			}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			_, err = builder.New(ctx, kv.Config{
				GossipInterval:    20 * time.Millisecond,
				RecoveryThreshold: 2,
			}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Eventually(func() int {
				return len(builder.OpNet.Entries)
			}).
				WithPolling(250 * time.Millisecond).
				WithTimeout(500 * time.Millisecond).
				Should(BeElementOf([]int{5, 6, 7}))
		})
	})

	Describe("Observable", func() {
		It("Should allow for a caller to listen to key-value changes", func() {
			kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv).ToNot(BeNil())
			var accumulated []xkv.Change
			kv.OnChange(func(ctx context.Context, r xkv.TxReader) {
				accumulated = slices.Collect(r)
			})
			Expect(kv.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(accumulated).To(HaveLen(1))
				g.Expect(accumulated[0].Value).To(Equal([]byte("value")))
			}).Should(Succeed())
		})
	})

	Describe("Recovery", func() {
		It("Should recover the state of the key-value store", func() {
			kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(kv1.Set(ctx, []byte("key2"), []byte("value2"))).To(Succeed())
			Expect(kv1.Set(ctx, []byte("key3"), []byte("value3"))).To(Succeed())
			kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Eventually(func(g Gomega) {
				v, closer, err := kv2.Get(ctx, []byte("key"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value")))
				Expect(closer.Close()).To(Succeed())
				v, closer, err = kv2.Get(ctx, []byte("key2"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value2")))
				Expect(closer.Close()).To(Succeed())
				v, closer, err = kv2.Get(ctx, []byte("key3"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value3")))
				Expect(closer.Close()).To(Succeed())
			})
		})

		It("Should correctly recover delete operations", func() {
			kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(kv1.Delete(ctx, []byte("key"))).To(Succeed())
			kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Eventually(func(g Gomega) {
				v, closer, err := kv2.Get(ctx, []byte("key"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value")))
				Expect(closer.Close()).To(Succeed())
			})
		})
	})
})

func waitForClusterStateToConverge(builder *kvmock.Builder) {
	Eventually(func(g Gomega) {
		_, err := builder.ClusterAPIs[1].Resolve(2)
		g.Expect(err).ToNot(HaveOccurred())
	}).Should(Succeed())
}
