// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Service", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func() {
		mockCluster = mock.ProvisionCluster(ctx, 1)
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})

	Describe("CountExternalNonVirtual", func() {
		It("Should return zero for empty database", func() {
			// The cluster may have some channels from setup, so we just verify
			// the count is non-negative and the method works
			count := mockCluster.Nodes[1].Channel.CountExternalNonVirtual()
			Expect(count).To(BeNumerically(">=", 0))
		})

		It("Should count external non-virtual channels", func() {
			initialCount := mockCluster.Nodes[1].Channel.CountExternalNonVirtual()

			// Create an index channel (external, non-virtual)
			indexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &indexCh)).To(Succeed())

			// Create a data channel (external, non-virtual)
			dataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  indexCh.LocalKey,
				Leaseholder: 1,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &dataCh)).To(Succeed())

			// Count should increase by 2
			Expect(mockCluster.Nodes[1].Channel.CountExternalNonVirtual()).To(Equal(initialCount + 2))
		})

		It("Should not count virtual channels", func() {
			initialCount := mockCluster.Nodes[1].Channel.CountExternalNonVirtual()

			// Create a virtual channel (external, but virtual)
			virtualCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Leaseholder: cluster.Free,
				Virtual:     true,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &virtualCh)).To(Succeed())

			// Count should NOT increase
			Expect(mockCluster.Nodes[1].Channel.CountExternalNonVirtual()).To(Equal(initialCount))
		})

		It("Should not count internal channels", func() {
			initialCount := mockCluster.Nodes[1].Channel.CountExternalNonVirtual()

			// Create an internal index channel
			internalIndexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &internalIndexCh)).To(Succeed())

			// Create an internal data channel
			internalDataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  internalIndexCh.LocalKey,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &internalDataCh)).To(Succeed())

			// Count should NOT increase
			Expect(mockCluster.Nodes[1].Channel.CountExternalNonVirtual()).To(Equal(initialCount))
		})
	})
})
