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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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
			ch.IsIndex = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.TimeStampT
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should not create the channel if it already exists by name", func() {
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
				Expect(ch.LocalKey).To(Equal(channel.LocalKey(3)))
			})
			It("Should create the channel in the cesium gorpDB", func() {
				channels := MustSucceed(mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey()))
				Expect(channels).To(HaveLen(1))
				cesiumCh := channels[0]
				Expect(cesiumCh.Key).To(Equal(ch.Key().StorageKey()))
				Expect(cesiumCh.DataType).To(Equal(telem.TimeStampT))
				Expect(cesiumCh.IsIndex).To(BeTrue())
			})
			It("Should create the channel without error", func() {
				ch.Leaseholder = 1
				ch.Name = channel.NewRandomName()
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.CreateWithoutGroupRelationship())).To(Succeed())
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
				entries := []ontology.Resource{}
				Expect(mockCluster.
					Nodes[1].
					Ontology.
					NewRetrieve().
					WhereIDs(ch.OntologyID()).
					TraverseTo(ontology.Parents).
					Entries(&entries).
					Exec(ctx, nil),
				).To(Succeed())
				Expect(entries).To(BeEmpty())
			})

		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(2)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should create the channel in cesium", func() {
				channels := MustSucceed(mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey()))
				Expect(channels).To(HaveLen(1))
				cesiumCh := channels[0]
				Expect(cesiumCh.DataType).To(Equal(telem.TimeStampT))
				Expect(cesiumCh.IsIndex).To(BeTrue())
			})
			It("Should not create the channel on another nodes time-series DB", func() {
				Expect(mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())).Error().To(MatchError(query.ErrNotFound))
			})
			It("Should assign a sequential key to the channels on each node",
				func() {
					ch2 := &channel.Channel{
						IsIndex:     true,
						Name:        channel.NewRandomName(),
						DataType:    telem.TimeStampT,
						Leaseholder: 1,
					}
					Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).Create(ctx, ch2)).To(Succeed())
					Expect(ch2.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
					Expect(ch2.Key().LocalKey()).To(Equal(channel.LocalKey(7)))
				})
			It("Should correctly create a virtual channel", func() {
				ch3 := &channel.Channel{
					Name:        channel.NewRandomName(),
					DataType:    telem.JSONT,
					Leaseholder: 2,
					Virtual:     true,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, ch3)).To(Succeed())
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
					Name:        channel.NewRandomName(),
					DataType:    telem.TimeStampT,
					Leaseholder: 2,
					IsIndex:     true,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, ch4)).To(Succeed())
				Expect(ch4.Key().Leaseholder()).To(Equal(aspen.NodeKey(2)))
				Expect(ch4.Key().LocalKey()).To(Equal(channel.LocalKey(9)))
				Expect(ch4.LocalIndex).To(Equal(channel.LocalKey(9)))
				channels := MustSucceed(mockCluster.Nodes[2].Storage.TS.RetrieveChannels(ctx, ch4.Key().StorageKey()))
				Expect(channels).To(HaveLen(1))
				Expect(channels[0].IsIndex).To(BeTrue())
			})
		})
		Context("Free", func() {
			BeforeEach(func() {
				ch.Leaseholder = cluster.NodeKeyFree
				ch.Virtual = true
			})
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKeyFree))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(5)))
				Expect(mockCluster.Nodes[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())).
					Error().To(MatchError(query.ErrNotFound))
			})
		})

		Context("error cases", func() {
			It("Should return an error if the name is invalid", func() {
				ch.Name = "invalid name"
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).
					To(MatchError(ContainSubstring("contains invalid characters")))
			})
			It("Should return an error if the name is a duplicate", func() {
				ch2 := channel.Channel{
					Name:        ch.Name,
					DataType:    telem.Float64T,
					Leaseholder: 1,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch2)).
					To(MatchError(ContainSubstring(fmt.Sprintf("channel with name '%s' already exists", ch.Name))))
			})
		})
	})
	Context("Multiple channels", func() {
		It("Should create multiple channels without error", func() {
			chs := []channel.Channel{
				{
					Name:        channel.NewRandomName(),
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
					IsIndex:     true,
				},
				{
					Name:        channel.NewRandomName(),
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
					IsIndex:     true,
				},
			}
			Expect(mockCluster.Nodes[1].Channel.CreateMany(ctx, &chs)).To(Succeed())
			Expect(chs[0].Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
			Expect(chs[0].Key().LocalKey()).To(Not(BeZero()))
			Expect(chs[1].Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
			Expect(chs[1].Key().LocalKey()).To(Not(BeZero()))
			Expect(chs[0].Key()).ToNot(Equal(chs[1].Key()))
		})
		It("Should return an error if the names are duplicates", func() {
			ch1 := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Leaseholder: 1,
				Virtual:     true,
			}
			ch2 := channel.Channel{
				Name:        ch1.Name,
				DataType:    telem.Float64T,
				Leaseholder: 1,
				Virtual:     true,
			}
			Expect(mockCluster.Nodes[1].Channel.CreateMany(ctx, &[]channel.Channel{ch1, ch2})).
				To(MatchError(ContainSubstring(fmt.Sprintf("duplicate channel name '%s' in request", ch1.Name))))
		})
	})
	Context("Creating if name doesn't exist", func() {
		var ch channel.Channel
		BeforeEach(func() {
			ch.IsIndex = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.TimeStampT
			ch.Leaseholder = 1
		})
		It("Should create the channel without error", func() {
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
			Expect(ch.Key().LocalKey()).To(Not(BeZero()))
		})
		It("Should not create the channel if it already exists by name", func() {
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			k := ch.Key()
			ch.Leaseholder = 0
			ch.LocalKey = 0
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
		})
		Describe("OverwriteIfNameExists", func() {

			It("Should overwrite the channel if it already exists by name and the new channel has different properties than the old one", func() {
				name := channel.NewRandomName()
				ch := channel.Channel{
					Virtual:     true,
					Name:        name,
					DataType:    telem.Float64T,
					Leaseholder: 1,
				}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				// Try to create a new channel with the same name but different
				// properties
				newCh := channel.Channel{
					Virtual:     true,
					Name:        name,
					DataType:    telem.Float32T,
					Leaseholder: 1,
				}

				Expect(mockCluster.Nodes[1].Channel.Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())

				var resChannels []channel.Channel
				Expect(mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(newCh.Key()).
					Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
				Expect(resChannels).To(HaveLen(1))

				Expect(resChannels[0].Virtual).To(BeTrue())
				Expect(resChannels[0].DataType).To(Equal(telem.Float32T))

				err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(originalKey).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).To(MatchError(query.ErrNotFound))
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
			ch.Leaseholder = cluster.NodeKeyFree
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKeyFree))
			k := ch.Key()
			ch.LocalKey = 0
			ch.Leaseholder = 0
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKeyFree))
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
			Expect(calcCh.Leaseholder).To(Equal(cluster.NodeKeyFree))
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
			Expect(indexCh.Leaseholder).To(Equal(cluster.NodeKeyFree))
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
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh2, channel.RetrieveIfNameExists())).To(Succeed())

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
	Context("Updating Calculated Channel Expression", func() {
		It("Should update expression when Create() called with existing key", func() {
			// 1. Create calculated channel
			calcCh := channel.Channel{
				Name:       "temperature_calc",
				DataType:   telem.Float64T,
				Expression: "return channel('sensor1') * 2.5",
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh)).To(Succeed())

			originalKey := calcCh.Key()
			originalIndexKey := calcCh.LocalIndex
			originalName := calcCh.Name

			// 2. Modify expression and call Create() again with same key
			calcCh.Expression = "return channel('sensor1') * 3.0 + 10"
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &calcCh)).To(Succeed())

			// 3. Verify key unchanged
			Expect(calcCh.Key()).To(Equal(originalKey))

			// 4. Verify index unchanged
			Expect(calcCh.LocalIndex).To(Equal(originalIndexKey))

			// 5. Retrieve and verify expression updated, other fields preserved
			var retrieved channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereKeys(originalKey).
				Entry(&retrieved).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrieved.Expression).To(Equal("return channel('sensor1') * 3.0 + 10"))
			Expect(retrieved.Name).To(Equal(originalName))
			Expect(retrieved.Key()).To(Equal(originalKey))
		})
	})
	Context("Updating a channel", func() {
		var ch channel.Channel
		var ch2 channel.Channel
		BeforeEach(func() {
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.Float64T
			ch.Virtual = true
			ch.Internal = false
			ch.Leaseholder = cluster.NodeKeyFree

			ch2.IsIndex = true
			ch2.Name = channel.NewRandomName()
			ch2.DataType = telem.TimeStampT
			ch2.Leaseholder = 1

			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch2)).To(Succeed())
		})
		It("Should update the channel name without error", func() {
			newName := channel.NewRandomName()
			ch.Name = newName
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Name).To(Equal(newName))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(newName))
		})
		It("Should not update the channel if it already exists by name", func() {
			existingName := ch2.Name
			ch.Name = existingName
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch, channel.RetrieveIfNameExists())).To(Succeed())
			Expect(ch.Name).To(Equal(existingName))

			var resChannels []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(existingName))
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

var _ = Context("Name Validation Disabled", func() {
	Describe("Channel Creation", Ordered, func() {
		var mockCluster *mock.Cluster
		BeforeAll(func() {
			mockCluster = mock.ProvisionCluster(ctx, 1, distribution.Config{
				ValidateChannelNames: config.False(),
			})
		})
		AfterAll(func() {
			Expect(mockCluster.Close()).To(Succeed())
		})
		It("Should create a channel with spaces in the name", func() {
			ch := channel.Channel{
				Name:        "my channel with spaces",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(BeZero())
			var retrieved channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereKeys(ch.Key()).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("my channel with spaces"))
		})
		It("Should create a channel with special characters in the name", func() {
			ch := channel.Channel{
				Name:        "sensor!@#$%",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: cluster.NodeKeyFree,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(BeZero())
			var retrieved channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereKeys(ch.Key()).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("sensor!@#$%"))
		})
		It("Should create a channel with a name starting with a digit", func() {
			ch := channel.Channel{
				Name:        "1sensor",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(BeZero())
			var retrieved channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereKeys(ch.Key()).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("1sensor"))
		})
		It("Should still reject empty names", func() {
			ch := channel.Channel{
				Name:        "",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: cluster.NodeKeyFree,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should allow renaming to a name with special characters", func() {
			ch := channel.Channel{
				Name:        "original_name",
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Rename(ctx, ch.Key(), "new name with spaces!", false)).To(Succeed())
			var retrieved channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereKeys(ch.Key()).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("new name with spaces!"))
		})
	})
})
