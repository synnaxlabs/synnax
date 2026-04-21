// Copyright 2026 Synnax Labs, Inc.
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
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
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
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("txn", func() {
	var builder *kvmock.Builder

	BeforeEach(func() {
		builder = DeferClose(kvmock.NewBuilder(
			kv.Config{
				RecoveryThreshold: 12,
				GossipInterval:    10 * time.Millisecond,
			},
			cluster.Config{
				Gossip: gossip.Config{Interval: 10 * time.Millisecond},
				Pledge: pledge.Config{RetryInterval: 10 * time.Millisecond},
			},
		))
	})

	Describe("StreamServer", func() {

		It("Should open a new database without error", func(ctx SpecContext) {
			kv := MustOpen(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv).ToNot(BeNil())
		})

	})

	Describe("SetNode", func() {

		Describe("Gateway Leaseholder", func() {

			It("Should commit the operation to storage", func(ctx SpecContext) {
				kv := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				Expect(kv).ToNot(BeNil())
				Expect(kv.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
				v, closer := MustSucceed2(kv.Get(ctx, []byte("key")))
				Expect(v).To(Equal([]byte("value")))
				Expect(closer.Close()).To(Succeed())
			})

			It("Should propagate the operation to other members of the cluster",
				func(ctx SpecContext) {
					kv1 := MustSucceed(builder.New(ctx, kv.Config{
						Instrumentation: Instrumentation("kv1"),
					}, cluster.Config{}))
					kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
					Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
					Eventually(func(g Gomega) {
						v, closer, err := kv2.Get(ctx, []byte("key"))
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(v).To(Equal([]byte("value")))
						g.Expect(closer.Close()).To(Succeed())
					}).Should(Succeed())
				})
			It("Should forward an update to the Leaseholder", func(ctx SpecContext) {
				kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
				Eventually(func(g Gomega) {
					v, closer, err := kv2.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					g.Expect(kv2.Set(ctx, []byte("key"), []byte("value2"))).To(Succeed())
					g.Expect(closer.Close()).To(Succeed())
				}).Should(Succeed())
				v, closer := MustSucceed2(kv1.Get(ctx, []byte("key")))
				Expect(v).To(Equal([]byte("value2")))
				Expect(closer.Close()).To(Succeed())
				v, closer = MustSucceed2(kv1.Get(ctx, []byte("key")))
				Expect(v).To(Equal([]byte("value2")))
				Expect(closer.Close()).To(Succeed())
			})

			It("Should return an error when attempting to transfer the lease",
				func(ctx SpecContext) {
					kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
					MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
					Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
					err := kv1.Set(ctx, []byte("key"), []byte("value2"), node.Key(2))
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, kv.ErrLeaseNotTransferable)).To(BeTrue())
				})

		})

		Describe("Peers Leaseholder", func() {
			It("Should commit the operation to storage", func(ctx SpecContext) {
				kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				waitForClusterStateToConverge(builder)
				Expect(kv1.Set(ctx, []byte("key"), []byte("value"), node.Key(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, closer, err := kv2.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					g.Expect(closer.Close()).To(Succeed())
				}).Should(Succeed())
			})

			It("Should return an error if the lease option is not a node Name", func(ctx SpecContext) {
				kv := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				Expect(kv.Set(ctx, []byte("key"), []byte("value"), "2")).To(HaveOccurred())
			})
		})

	})

	Describe("Tx", func() {
		It("Should execute a set of operations", func(ctx SpecContext) {
			kv := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv).ToNot(BeNil())
			txn := kv.OpenTx()
			Expect(txn.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(txn.Set(ctx, []byte("key2"), []byte("value2"))).To(Succeed())
			Expect(txn.Commit(ctx)).To(Succeed())
			v, closer := MustSucceed2(kv.Get(ctx, []byte("key")))
			Expect(v).To(Equal([]byte("value")))
			Expect(closer.Close()).To(Succeed())
			v, closer = MustSucceed2(kv.Get(ctx, []byte("key2")))
			Expect(v).To(Equal([]byte("value2")))
			Expect(closer.Close()).To(Succeed())
		})

	})

	Describe("delete", func() {

		Describe("Gateway Leaseholder", func() {
			It("Should apply the operation to storage", func(ctx SpecContext) {
				kv := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				Expect(kv).ToNot(BeNil())
				Expect(kv.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
				v, closer := MustSucceed2(kv.Get(ctx, []byte("key")))
				Expect(v).To(Equal([]byte("value")))
				Expect(kv.Delete(ctx, []byte("key"))).To(Succeed())
				Expect(closer.Close()).To(Succeed())
				Expect(kv.Get(ctx, []byte("key"))).Error().To(HaveOccurred())
			})
		})

		It("Should delete a key written directly to the engine without a digest", func(ctx SpecContext) {
			engine := DeferClose(memkv.New())
			kv := MustSucceed(
				builder.New(ctx, kv.Config{Engine: engine}, cluster.Config{}),
			)
			Expect(engine.Set(ctx, []byte("direct-key"), []byte("direct-value"))).
				To(Succeed())
			v, closer := MustSucceed2(kv.Get(ctx, []byte("direct-key")))
			Expect(v).To(Equal([]byte("direct-value")))
			Expect(closer.Close()).To(Succeed())
			Expect(kv.Delete(ctx, []byte("direct-key"))).To(Succeed())
			Expect(kv.Get(ctx, []byte("direct-key"))).Error().
				To(MatchError(query.ErrNotFound))
		})

		Describe("Peer Leaseholder", func() {
			It("Should apply the operation to storage", func(ctx SpecContext) {
				kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
				waitForClusterStateToConverge(builder)
				Expect(kv1.Set(ctx, []byte("key"), []byte("value"), node.Key(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, closer, err := kv2.Get(ctx, []byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					g.Expect(closer.Close()).To(Succeed())
				}).Should(Succeed())
			})
		})

	})

	Describe("Request Recovery", func() {
		It("Should stop propagating an operation after a set threshold of"+
			" redundant broadcasts", func(ctx SpecContext) {
			kv1 := MustSucceed(builder.New(ctx, kv.Config{
				GossipInterval:    20 * time.Millisecond,
				RecoveryThreshold: 2,
			}, cluster.Config{}))
			MustSucceed(builder.New(ctx, kv.Config{
				GossipInterval:    20 * time.Millisecond,
				RecoveryThreshold: 2,
			}, cluster.Config{}))
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Eventually(func() int {
				return builder.OpNet.EntryCount()
			}).
				WithPolling(250 * time.Millisecond).
				WithTimeout(500 * time.Millisecond).
				Should(BeElementOf([]int{5, 6, 7}))
		})
	})

	Describe("Observable", func() {
		It("Should allow for a caller to listen to key-value changes", func(ctx SpecContext) {
			kv := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv).ToNot(BeNil())
			var mu sync.Mutex
			var accumulated []xkv.Change
			kv.OnChange(func(ctx context.Context, r xkv.TxReader) {
				mu.Lock()
				defer mu.Unlock()
				accumulated = slices.Collect(r)
			})
			Expect(kv.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Eventually(func(g Gomega) {
				mu.Lock()
				defer mu.Unlock()
				g.Expect(accumulated).To(HaveLen(1))
				g.Expect(accumulated[0].Value).To(Equal([]byte("value")))
			}).Should(Succeed())
		})

		It("Should not stall writes when an observer handler is slow", func(ctx SpecContext) {
			db := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))

			gate := make(chan struct{})
			db.OnChange(func(ctx context.Context, r xkv.TxReader) {
				<-gate
			})
			defer close(gate)

			// The pipeline has ~500 items of total buffer capacity (5 channels
			// at capacity 100 on the critical path). We write more than that to
			// guarantee we'd hit the clog if it exists.
			totalWrites := 700
			var completed atomic.Int64
			go func() {
				defer GinkgoRecover()
				for i := range totalWrites {
					key := fmt.Appendf(nil, "key-%d", i)
					err := db.Set(ctx, key, []byte("v"))
					if err != nil {
						return
					}
					completed.Add(1)
				}
			}()

			// All writes should complete even though the observer is blocked.
			Eventually(func() int64 {
				return completed.Load()
			}, 5*time.Second, 50*time.Millisecond).Should(
				Equal(int64(totalWrites)),
			)
		})
	})

	Describe("Recovery", func() {
		It("Should recover the state of the key-value store", func(ctx SpecContext) {
			kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(kv1.Set(ctx, []byte("key2"), []byte("value2"))).To(Succeed())
			Expect(kv1.Set(ctx, []byte("key3"), []byte("value3"))).To(Succeed())
			kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Eventually(func(g Gomega) {
				v, closer, err := kv2.Get(ctx, []byte("key"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value")))
				g.Expect(closer.Close()).To(Succeed())
				v, closer, err = kv2.Get(ctx, []byte("key2"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value2")))
				g.Expect(closer.Close()).To(Succeed())
				v, closer, err = kv2.Get(ctx, []byte("key3"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value3")))
				g.Expect(closer.Close()).To(Succeed())
			})
		})

		It("Should persist digests during recovery so recovered keys can be deleted", func(ctx SpecContext) {
			kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Eventually(func(g Gomega) {
				v, closer, err := kv2.Get(ctx, []byte("key"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value")))
				g.Expect(closer.Close()).To(Succeed())
			}).Should(Succeed())
			Expect(kv1.Delete(ctx, []byte("key"))).To(Succeed())
			Eventually(func(g Gomega) {
				_, closer, err := kv2.Get(ctx, []byte("key"))
				if closer != nil {
					Expect(closer.Close()).To(Succeed())
				}
				g.Expect(err).To(MatchError(query.ErrNotFound))
			}).Should(Succeed())
		})

		It("Should correctly recover delete operations", func(ctx SpecContext) {
			kv1 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Expect(kv1.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(kv1.Delete(ctx, []byte("key"))).To(Succeed())
			kv2 := MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
			Eventually(func(g Gomega) {
				v, closer, err := kv2.Get(ctx, []byte("key"))
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(v).To(Equal([]byte("value")))
				g.Expect(closer.Close()).To(Succeed())
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
