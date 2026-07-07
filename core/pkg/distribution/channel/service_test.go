// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagemock "github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", Ordered, func() {
	var dist mock.Node
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = mock.NewNode(ctx)
	})

	Describe("CountExternalNonVirtual", func() {
		It("Should return zero for empty database", func(ctx SpecContext) {
			count := dist.Channel.CountExternalNonVirtual()
			Expect(count).To(BeEquivalentTo(0))
		})

		It("Should count external non-virtual channels", func(ctx SpecContext) {
			initialCount := dist.Channel.CountExternalNonVirtual()

			// Create an index channel (external, non-virtual)
			indexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &indexCh)).To(Succeed())

			// Create a data channel (external, non-virtual)
			dataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  indexCh.LocalKey,
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &dataCh)).To(Succeed())

			// Count should increase by 2
			Expect(dist.Channel.CountExternalNonVirtual()).To(Equal(initialCount + 2))
		})

		It("Should not count virtual channels", func(ctx SpecContext) {
			initialCount := dist.Channel.CountExternalNonVirtual()

			// Create a virtual channel (external, but virtual)
			virtualCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			}
			Expect(dist.Channel.Create(ctx, &virtualCh)).To(Succeed())

			// Count should NOT increase
			Expect(dist.Channel.CountExternalNonVirtual()).To(Equal(initialCount))
		})

		It("Should not count internal channels", func(ctx SpecContext) {
			initialCount := dist.Channel.CountExternalNonVirtual()

			// Create an internal index channel
			internalIndexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(dist.Channel.Create(ctx, &internalIndexCh)).To(Succeed())

			// Create an internal data channel
			internalDataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  internalIndexCh.LocalKey,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(dist.Channel.Create(ctx, &internalDataCh)).To(Succeed())

			// Count should NOT increase
			Expect(dist.Channel.CountExternalNonVirtual()).To(Equal(initialCount))
		})
	})

	Describe("Observe", func() {
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			disconnect := dist.Channel.Observe().OnChange(
				func(context.Context, gorp.TxReader[channel.Key, channel.Channel]) {
					called.Store(true)
				})
			defer disconnect()
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(dist.Channel.Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})
})

var _ = Describe("Startup Free Storage Registration", func() {
	It("Should register free channels found in the channel table at startup", func(ctx SpecContext) {
		freshStorageCluster := DeferClose(storagemock.NewCluster())
		c := mock.NewCluster(ctx, 1)
		n := c.Nodes[1]
		ch := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		Expect(n.Channel.Create(ctx, &ch)).To(Succeed())

		// Restart the node's distribution layer with the same key-value store but a
		// fresh time-series engine, mirroring a process restart: the channel table
		// survives on disk while transient channel registrations do not.
		Expect(n.Layer.Close()).To(Succeed())
		freshStorage := freshStorageCluster.Provision(ctx)
		c.Provision(ctx, distribution.LayerConfig{
			Storage: &storage.Layer{KV: n.Storage.KV, TS: freshStorage.TS},
		})
		stored := MustSucceed(
			freshStorage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		)
		Expect(stored.Transient).To(BeTrue())
	})
})
