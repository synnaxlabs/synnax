// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/kv/kvmock"
	"github.com/synnaxlabs/aspen/internal/node"
	kvx "github.com/synnaxlabs/x/kv"
	"time"
)

var _ = Describe("txn", func() {
	var (
		builder *kvmock.Builder
	)

	BeforeEach(func() {
		builder = kvmock.NewBuilder(
			kv.Config{
				RecoveryThreshold: 12,
				GossipInterval:    100 * time.Millisecond,
			},
			cluster.Config{
				Gossip: gossip.Config{Interval: 50 * time.Millisecond},
				Pledge: pledge.Config{RetryInterval: 50 * time.Millisecond},
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
				Expect(kvx.Set(ctx, kv, []byte("key"), []byte("value"))).To(Succeed())
				v, err := kvx.Get(ctx, kv, []byte("key"))
				Expect(err).ToNot(HaveOccurred())
				Expect(v).To(Equal([]byte("value")))
			})

			It("Should propagate the operation to other members of the cluster",
				func() {
					kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					Expect(kvx.Set(ctx, kv1, []byte("key"), []byte("value"))).To(Succeed())
					Eventually(func(g Gomega) {
						v, err := kvx.Get(ctx, kv2, []byte("key"))
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(v).To(Equal([]byte("value")))
					}).Should(Succeed())
				})

			It("Should forward an update to the Leaseholder", func() {
				kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kvx.Set(ctx, kv1, []byte("key"), []byte("value"))).To(Succeed())
				Eventually(func(g Gomega) {
					v, err := kvx.Get(ctx, kv2, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					g.Expect(kvx.Set(ctx, kv2, []byte("key"), []byte("value2"))).To(Succeed())
				}).Should(Succeed())
				Expect(func(g Gomega) {
					v, err := kvx.Get(ctx, kv1, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value2")))
					v, err = kvx.Get(ctx, kv2, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value2")))
				})
			})

			It("Should return an error when attempting to transfer the leaseAlloc",
				func() {
					kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					_, err = builder.New(ctx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					Expect(kvx.Set(ctx, kv1, []byte("key"), []byte("value"))).To(Succeed())
					err = kvx.Set(ctx, kv1, []byte("key"), []byte("value2"), node.ID(2))
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
				Expect(kvx.Set(ctx, kv1, []byte("key"), []byte("value"), node.ID(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, err := kvx.Get(ctx, kv2, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
				}).Should(Succeed())
			})

			It("Should return an error if the lease option is not a node ID", func() {
				kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kvx.Set(ctx, kv, []byte("key"), []byte("value"), "2")).To(HaveOccurred())
			})

		})

	})

	Describe("TypedWriter", func() {
		It("Should execute a writer of operations", func() {
			kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv).ToNot(BeNil())
			txn := kv.NewWriter(ctx)
			Expect(txn.Set([]byte("key"), []byte("value"))).To(Succeed())
			Expect(txn.Set([]byte("key2"), []byte("value2"))).To(Succeed())
			Expect(txn.Commit()).To(Succeed())
			v, err := kvx.Get(ctx, kv, []byte("key"))
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal([]byte("value")))
			v, err = kvx.Get(ctx, kv, []byte("key2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal([]byte("value2")))
		})

	})

	Describe("Delete", func() {

		Describe("Gateway Leaseholder", func() {

			It("Should applyToAndCommit the operation to storage", func() {
				kv, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv).ToNot(BeNil())
				Expect(kvx.Set(ctx, kv, []byte("key"), []byte("value"))).To(Succeed())
				v, err := kvx.Get(ctx, kv, []byte("key"))
				Expect(err).ToNot(HaveOccurred())
				Expect(v).To(Equal([]byte("value")))
				Expect(kvx.Delete(ctx, kv, []byte("key"))).To(Succeed())
				v, err = kvx.Get(ctx, kv, []byte("key"))
				Expect(err).To(HaveOccurred())
				Expect(v).To(BeNil())
			})

		})

		Describe("Peers Leaseholder", func() {

			It("Should apply the operation to storage", func() {
				kv1, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(ctx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				waitForClusterStateToConverge(builder)
				Expect(kvx.Set(ctx, kv1, []byte("key"), []byte("value"), node.ID(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, err := kvx.Get(ctx, kv2, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
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
			Expect(kvx.Set(ctx, kv1, []byte("key"), []byte("value"))).To(Succeed())
			Eventually(func() int {
				return len(builder.OpNet.Entries)
			}).
				WithPolling(250 * time.Millisecond).
				WithTimeout(500 * time.Millisecond).
				Should(BeElementOf([]int{5, 6, 7}))
		})

	})

})

func waitForClusterStateToConverge(builder *kvmock.Builder) {
	Eventually(func(g Gomega) {
		_, err := builder.ClusterAPIs[1].Resolve(2)
		g.Expect(err).ToNot(HaveOccurred())
	}).Should(Succeed())
}
