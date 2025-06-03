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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Delete", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		limit       int
	)
	BeforeAll(func() {
		limit = 5
		mockCluster = mock.ProvisionCluster(ctx, 2)
		mockCluster.Provision(ctx, distribution.Config{
			TestingIntOverflowCheck: channel.FixedOverflowChecker(limit),
		})
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Describe("Channels Deletion", func() {
		Context("Single Channel", func() {
			var (
				idxCh, ch channel.Channel
			)
			JustBeforeEach(func() {
				idxCh.Name = "SG01_time"
				idxCh.DataType = telem.TimeStampT
				idxCh.IsIndex = true
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &idxCh)).To(Succeed())
				ch.Name = "SG01"
				ch.DataType = telem.Float64T
				ch.LocalIndex = idxCh.LocalKey
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
			})
			Context("Node is local", func() {
				BeforeEach(func() {
					idxCh.Leaseholder = 1
					ch.Leaseholder = 1
				})
				It("Should not allow deletion of index channel with dependent channels", func() {
					Expect(mockCluster.Nodes[1].Channels.Delete(ctx, idxCh.Key(), true)).ToNot(Succeed())
				})
				It("Should delete the channel without error", func() {
					Expect(mockCluster.Nodes[1].Channels.DeleteMany(ctx, channel.Keys{idxCh.Key(), ch.Key()}, true)).To(Succeed())
				})
				It("Should not be able to retrieve the channel after deletion", func() {
					Expect(mockCluster.Nodes[1].Channels.Delete(ctx, ch.Key(), true)).To(Succeed())
					exists, err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
				It("Should not be able to retrieve the channel from the storage KV", func() {
					Expect(mockCluster.Nodes[1].Channels.Delete(ctx, ch.Key(), true)).To(Succeed())
					channels, err := mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
					Expect(err).To(MatchError(cesium.ErrChannelNotFound))
					Expect(channels).To(BeEmpty())
				})
			})

			Context("Node is remote", func() {
				BeforeEach(func() {
					idxCh.Leaseholder = 2
					ch.Leaseholder = 2
				})
				It("Should not allow deletion of index channel with dependent channels", func() {
					Expect(mockCluster.Nodes[1].Channels.Delete(ctx, idxCh.Key(), true)).ToNot(Succeed())
				})
				It("Should delete the channel without error", func() {
					Expect(mockCluster.Nodes[1].Channels.DeleteMany(ctx, []channel.Key{idxCh.Key(), ch.Key()}, true)).To(Succeed())
				})
				It("Should not be able to retrieve the channel after deletion", func() {
					Expect(mockCluster.Nodes[1].Channels.Delete(ctx, ch.Key(), true)).To(Succeed())
					exists, err := mockCluster.Nodes[2].Channels.NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
					Eventually(func(g Gomega) {
						exists, err = mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(exists).To(BeFalse())
					}).Should(Succeed())
				})
				It("Should not be able to retrieve the channel from the storage KV", func() {
					Expect(mockCluster.Nodes[1].Channels.Delete(ctx, ch.Key(), true)).To(Succeed())
					channels, err := mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
					Expect(err).To(MatchError(cesium.ErrChannelNotFound))
					Expect(channels).To(BeEmpty())
				})
			})
		})
	})

	Context("Channels Limit", func() {

		It("Should allow creating channels after deleting some to stay under the limit", func() {
			// Create channels up to the limit
			channels := make([]channel.Channel, int(limit))
			for i := range limit {
				ch := channel.Channel{
					IsIndex:     true,
					DataType:    telem.TimeStampT,
					Name:        fmt.Sprintf("LimitTest%d", i),
					Leaseholder: 3,
				}
				Expect(mockCluster.Nodes[3].Channels.Create(ctx, &ch)).To(Succeed())
				channels[i] = ch
			}

			// Try to create one more channel over the limit
			overLimitCh := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        "OverLimit",
				Leaseholder: 3,
			}
			err := mockCluster.Nodes[3].Channels.Create(ctx, &overLimitCh)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("channel limit exceeded"))

			// Delete one channel
			writer := mockCluster.Nodes[3].Channels.NewWriter(nil)
			Expect(writer.Delete(ctx, channels[0].Key(), false)).To(Succeed())

			// Now we should be able to create a new channel
			newCh := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        "NewAfterDelete",
				Leaseholder: 3,
			}
			Expect(mockCluster.Nodes[3].Channels.Create(ctx, &newCh)).To(Succeed())

			// Try to create one more channel (should fail again)
			anotherCh := channel.Channel{
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Name:        "AnotherOverLimit",
				Leaseholder: 3,
			}
			err = mockCluster.Nodes[3].Channels.Create(ctx, &anotherCh)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("channel limit exceeded"))
		})
	})
})
