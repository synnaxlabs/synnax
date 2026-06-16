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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelmock "github.com/synnaxlabs/synnax/pkg/service/channel/mock"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Service", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.ProvisionCluster(context.Background(), 1)
		for _, n := range mockCluster.Nodes {
			channelmock.ChannelService(n)
		}
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})

	Describe("CountExternalNonVirtual", func() {
		It("Should return zero for empty database", func(ctx SpecContext) {
			count := channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()
			Expect(count).To(BeEquivalentTo(0))
		})

		It("Should count external non-virtual channels", func(ctx SpecContext) {
			initialCount := channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()

			// Create an index channel (external, non-virtual)
			indexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &indexCh)).To(Succeed())

			// Create a data channel (external, non-virtual)
			dataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  indexCh.LocalKey,
				Leaseholder: 1,
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &dataCh)).To(Succeed())

			// Count should increase by 2
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()).To(Equal(initialCount + 2))
		})

		It("Should not count virtual channels", func(ctx SpecContext) {
			initialCount := channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()

			// Create a virtual channel (external, but virtual)
			virtualCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &virtualCh)).To(Succeed())

			// Count should NOT increase
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()).To(Equal(initialCount))
		})

		It("Should not count internal channels", func(ctx SpecContext) {
			initialCount := channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()

			// Create an internal index channel
			internalIndexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &internalIndexCh)).To(Succeed())

			// Create an internal data channel
			internalDataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  internalIndexCh.LocalKey,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &internalDataCh)).To(Succeed())

			// Count should NOT increase
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).CountExternalNonVirtual()).To(Equal(initialCount))
		})
	})

	Describe("Observe", func() {
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			channelmock.ChannelService(mockCluster.Nodes[1]).Observe().OnChange(func(ctx context.Context, _ gorp.TxReader[channel.Key, channel.Channel]) {
				called.Store(true)
			})
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})
})
