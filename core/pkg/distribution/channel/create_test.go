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
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Create", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func() {
		mockCluster = mock.ProvisionCluster(ctx, 2)
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func() {
			var err error
			ch.IsIndex = true
			ch.Name = "SG01"
			ch.DataType = telem.TimeStampT
			err = mockCluster.Nodes[1].Channel.Create(ctx, &ch)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should not create the channel if it already exists by name", func() {
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
				Expect(ch.LocalKey).To(Equal(channel.LocalKey(3)))
			})
			It("Should create the channel in the cesium gorpDB", func() {
				channels, err := mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().StorageKey()))
				Expect(cesiumCH.DataType).To(Equal(telem.TimeStampT))
				Expect(cesiumCH.IsIndex).To(BeTrue())
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(2)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should create the channel in cesium", func() {
				channels, err := mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.DataType).To(Equal(telem.TimeStampT))
				Expect(cesiumCH.IsIndex).To(BeTrue())
			})
			It("Should not create the channel on another nodes time-series DB", func() {
				channels, err := mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
				Expect(err).To(MatchError(query.NotFound))
				Expect(channels).To(HaveLen(0))
			})
			It("Should assign a sequential key to the channels on each node",
				func() {
					ch2 := &channel.Channel{
						IsIndex:     true,
						Name:        "SG01",
						DataType:    telem.TimeStampT,
						Leaseholder: 1,
					}
					err := mockCluster.Nodes[1].Channel.NewWriter(nil).Create(ctx, ch2)
					Expect(err).To(BeNil())
					Expect(ch2.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
					Expect(ch2.Key().LocalKey()).To(Equal(channel.LocalKey(5)))
				})
			It("Should correctly create a virtual channel", func() {
				ch3 := &channel.Channel{
					Name:        "SG01",
					DataType:    telem.JSONT,
					Leaseholder: 2,
					Virtual:     true,
				}
				err := mockCluster.Nodes[1].Channel.Create(ctx, ch3)
				Expect(err).To(BeNil())
				Expect(ch3.Key().Leaseholder()).To(Equal(aspen.NodeKey(2)))
				Eventually(func(g Gomega) {
					channels, err := mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch3.Key().StorageKey())
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(channels).To(HaveLen(1))
					g.Expect(channels[0].DataType).To(Equal(telem.JSONT))
					g.Expect(channels[0].Virtual).To(BeTrue())
				})
			})
			It("Should create an index channel", func() {
				ch4 := &channel.Channel{
					Name:        "SG01",
					DataType:    telem.TimeStampT,
					Leaseholder: 2,
					IsIndex:     true,
				}
				err := mockCluster.Nodes[1].Channel.Create(ctx, ch4)
				Expect(err).To(BeNil())
				Expect(ch4.Key().Leaseholder()).To(Equal(aspen.NodeKey(2)))
				Expect(ch4.Key().LocalKey()).To(Equal(channel.LocalKey(9)))
				Expect(ch4.LocalIndex).To(Equal(channel.LocalKey(9)))
				channels, err := mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch4.Key().StorageKey())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				Expect(channels[0].IsIndex).To(BeTrue())
			})
		})
		Context("Free", func() {
			BeforeEach(func() {
				ch.Leaseholder = cluster.Free
				ch.Virtual = true
			})
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.Free))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(5)))
				channels, err := mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
				Expect(err).To(MatchError(query.NotFound))
				Expect(channels).To(HaveLen(0))
			})
		})
	})
	Context("Creating if name doesn't exist", func() {
		var ch channel.Channel
		BeforeEach(func() {
			ch.IsIndex = true
			ch.Name = "SG0001"
			ch.DataType = telem.TimeStampT
			ch.Leaseholder = 1
		})
		It("Should create the channel without error", func() {
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
			Expect(ch.Key().LocalKey()).To(Not(Equal(uint16(0))))
		})
		It("Should not create the channel if it already exists by name", func() {
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			k := ch.Key()
			ch.Leaseholder = 0
			ch.LocalKey = 0
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
		})
		Describe("OverwriteIfNameExists", func() {

			It("Should overwrite the channel if it already exists by name and the new channel has different properties than the old one", func() {
				ch := channel.Channel{
					Virtual:     true,
					Name:        "SG0001",
					DataType:    telem.Float64T,
					Leaseholder: 1,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				// Try to create a new channel with the same name but different properties
				newCh := channel.Channel{
					Virtual:     true,
					Name:        "SG0001",
					DataType:    telem.Float32T,
					Leaseholder: 1,
				}

				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())

				var resChannels []channel.Channel
				err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(newCh.Key()).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resChannels).To(HaveLen(1))

				Expect(resChannels[0].Virtual).To(BeTrue())
				Expect(resChannels[0].DataType).To(Equal(telem.Float32T))

				err = mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(originalKey).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).To(MatchError(query.NotFound))
			})
			It("Should not overwrite the channel if it already exists by name and the new channel has the same properties as the old one", func() {
				ch := channel.Channel{
					IsIndex:     true,
					Name:        "SG0001",
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				newCh := channel.Channel{
					IsIndex:     true,
					Name:        "SG0001",         // Same name
					DataType:    telem.TimeStampT, // Same data type
					Leaseholder: 1,
				}

				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())
				Expect(newCh.Key()).To(Equal(originalKey))

				var resChannels []channel.Channel
				err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(originalKey).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resChannels).To(HaveLen(1))
				Expect(resChannels[0].IsIndex).To(BeTrue())
				Expect(resChannels[0].DataType).To(Equal(telem.TimeStampT))
			})
		})
		It("Should not create a free channel if it already exists by name", func() {
			ch.Name = "SG0002"
			ch.Virtual = true
			ch.Leaseholder = cluster.Free
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.Free))
			k := ch.Key()
			ch.LocalKey = 0
			ch.Leaseholder = 0
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.Free))
		})
	})
	Context("Calculated Channel with Auto-Created Index", func() {
		It("Should automatically create an index channel for calculated channels", func() {
			calcCh := channel.Channel{
				Name:       "calculated_temp",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh)).To(Succeed())

			// Verify calculated channel properties
			Expect(calcCh.Leaseholder).To(Equal(cluster.Free))
			Expect(calcCh.Virtual).To(BeTrue())
			Expect(calcCh.LocalIndex).ToNot(BeZero())

			// Verify index channel was created
			indexName := "calculated_temp_time"
			var indexChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereNames(indexName).
				Entries(&indexChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(indexChannels).To(HaveLen(1))

			indexCh := indexChannels[0]
			Expect(indexCh.IsIndex).To(BeTrue())
			Expect(indexCh.DataType).To(Equal(telem.TimeStampT))
			Expect(indexCh.Virtual).To(BeTrue())
			Expect(indexCh.Leaseholder).To(Equal(cluster.Free))
			Expect(indexCh.LocalKey).To(Equal(calcCh.LocalIndex))
		})

		It("Should reject calculated channel with manually-specified index", func() {
			calcCh := channel.Channel{
				Name:       "calculated_bad",
				DataType:   telem.Float64T,
				LocalIndex: channel.LocalKey(999),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			err := mockCluster.Nodes[1].Channel.Create(ctx, &calcCh)
			Expect(err).To(MatchError(ContainSubstring("calculated channels cannot specify an index manually")))
		})

		It("Should retrieve existing calculated channel with its index", func() {
			calcCh := channel.Channel{
				Name:       "calculated_retrieve_test",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh)).To(Succeed())
			originalKey := calcCh.Key()
			originalIndexKey := calcCh.LocalIndex

			// Try to create again with RetrieveIfNameExists
			calcCh2 := channel.Channel{
				Name:       "calculated_retrieve_test",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh2, channel.RetrieveIfNameExists(true))).To(Succeed())

			// Should return existing channel with same index
			Expect(calcCh2.Key()).To(Equal(originalKey))
			Expect(calcCh2.LocalIndex).To(Equal(originalIndexKey))
		})

		It("Should handle batch create with calculated and regular channels", func() {
			// Create index channels for regular channels first
			indexCh1 := channel.Channel{
				Name:        "regular1_idx",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			indexCh2 := channel.Channel{
				Name:        "regular2_idx",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &indexCh1)).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &indexCh2)).To(Succeed())

			// Now create channels with proper indexes
			channels := []channel.Channel{
				{Name: "regular1", DataType: telem.Float64T, Leaseholder: 1, LocalIndex: indexCh1.LocalKey},
				{Name: "calculated1", DataType: telem.Float64T, Expression: "return 1 + 1"},
				{Name: "regular2", DataType: telem.Int32T, Leaseholder: 1, LocalIndex: indexCh2.LocalKey},
				{Name: "calculated2", DataType: telem.Float32T, Expression: "return 1 + 2"},
			}
			for i := range channels {
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channels[i])).To(Succeed())
			}

			// Check calculated channels have auto-created indexes
			Expect(channels[1].LocalIndex).ToNot(BeZero())
			Expect(channels[3].LocalIndex).ToNot(BeZero())

			// Check regular channels have the indexes we specified
			Expect(channels[0].LocalIndex).To(Equal(indexCh1.LocalKey))
			Expect(channels[2].LocalIndex).To(Equal(indexCh2.LocalKey))

			// Verify auto-created index channels exist for calculated channels
			var indexChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereNames("calculated1_time", "calculated2_time").
				Entries(&indexChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(indexChannels).To(HaveLen(2))
		})

		It("Should create internal index for internal calculated channel", func() {
			calcCh := channel.Channel{
				Name:       "internal_calculated",
				DataType:   telem.Float64T,
				Expression: "return 1 + 1",
				Internal:   true,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh)).To(Succeed())

			// Verify index is also internal
			indexName := "internal_calculated_time"
			var indexChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereNames(indexName).
				Entries(&indexChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(indexChannels).To(HaveLen(1))
			Expect(indexChannels[0].Internal).To(BeTrue())
		})
	})
	Context("Updating a channel", func() {
		var ch channel.Channel
		var ch2 channel.Channel
		BeforeEach(func() {
			ch.Name = "SG0001"
			ch.DataType = telem.Float64T
			ch.Virtual = true
			ch.Internal = false
			ch.Leaseholder = cluster.Free

			ch2.IsIndex = true
			ch2.Name = "SG0003"
			ch2.DataType = telem.TimeStampT
			ch2.Leaseholder = 1

			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch2)).To(Succeed())
		})
		It("Should update the channel name without error", func() {
			ch.Name = "SG0002"
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Name).To(Equal("SG0002"))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal("SG0002"))
		})
		It("Should not update the channel if it already exists by name", func() {
			ch.Name = "SG0003"
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Name).To(Equal("SG0003"))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal("SG0003"))
		})
		It("Should assign a new key when attempting to update a non-virtual channel",
			func() {
				// Create initial non-virtual channel
				nonVirtualCh := channel.Channel{
					IsIndex:     true,
					Name:        "NonVirtual",
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
					Virtual:     false,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &nonVirtualCh)).To(Succeed())
				originalKey := nonVirtualCh.Key()

				nonVirtualCh.Name = "UpdatedName"
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &nonVirtualCh)).To(Succeed())

				Expect(nonVirtualCh.Key()).ToNot(Equal(originalKey))

				var resChannels []channel.Channel
				err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(originalKey, nonVirtualCh.Key()).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resChannels).To(HaveLen(2))

				Expect(resChannels[0].Name).To(Equal("NonVirtual"))
				Expect(resChannels[1].Name).To(Equal("UpdatedName"))
			})
	})

})
