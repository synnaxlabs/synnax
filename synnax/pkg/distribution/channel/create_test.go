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
	"github.com/synnaxlabs/x/validate"
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
			err = mockCluster.Nodes[1].Channels.Create(ctx, &ch)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
				Expect(ch.Key().LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should not create the channel if it already exists by name", func() {
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
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
					err := mockCluster.Nodes[1].Channels.NewWriter(nil).Create(ctx, ch2)
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
				err := mockCluster.Nodes[1].Channels.Create(ctx, ch3)
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
				err := mockCluster.Nodes[1].Channels.Create(ctx, ch4)
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
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.NodeKey(1)))
			Expect(ch.Key().LocalKey()).To(Not(Equal(uint16(0))))
		})
		It("Should not create the channel if it already exists by name", func() {
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			k := ch.Key()
			ch.Leaseholder = 0
			ch.LocalKey = 0
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
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
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				// Try to create a new channel with the same name but different properties
				newCh := channel.Channel{
					Virtual:     true,
					Name:        "SG0001",
					DataType:    telem.Float32T,
					Leaseholder: 1,
				}

				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())

				var resChannels []channel.Channel
				err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(newCh.Key()).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resChannels).To(HaveLen(1))

				Expect(resChannels[0].Virtual).To(BeTrue())
				Expect(resChannels[0].DataType).To(Equal(telem.Float32T))

				err = mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(originalKey).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).To(MatchError(query.NotFound))
			})
			It("Should not overwrite the channel if it already exists by name and the new channel has the same properties as the old one", func() {
				ch := channel.Channel{
					IsIndex:     true,
					Name:        "SG0001",
					DataType:    telem.TimeStampT,
					Leaseholder: 1,
				}
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
				originalKey := ch.Key()

				newCh := channel.Channel{
					IsIndex:     true,
					Name:        "SG0001",         // Same name
					DataType:    telem.TimeStampT, // Same data type
					Leaseholder: 1,
				}

				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &newCh, channel.OverwriteIfNameExistsAndDifferentProperties())).To(Succeed())
				Expect(newCh.Key()).To(Equal(originalKey))

				var resChannels []channel.Channel
				err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(originalKey).Entries(&resChannels).Exec(ctx, nil)
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
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.Free))
			k := ch.Key()
			ch.LocalKey = 0
			ch.Leaseholder = 0
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Key()).To(Equal(k))
			Expect(ch.Key().Leaseholder()).To(Equal(aspen.Free))
		})
	})
	Context("Calculated Channels", func() {
		It("Should create a calculated channel without error", func() {
			baseCh := channel.Channel{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &baseCh)).To(Succeed())

			ch := channel.Channel{
				Name:       "SG0001",
				DataType:   telem.Float64T,
				Expression: "return 1",
				Requires:   []channel.Key{baseCh.Key()},
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
		})
		It("Should return an error if the requires field is empty", func() {
			ch := channel.Channel{
				Name:       "SG0001",
				DataType:   telem.Float64T,
				Expression: "return 1",
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(MatchError(validate.FieldError{
				Field:   "requires",
				Message: "calculated channels must require at least one channel",
			}))
		})
		It("Should return an error if the calculated channel depends on both a virtual and index channel", func() {
			vCH := channel.Channel{
				Name:     "SG0001",
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &vCH)).To(Succeed())
			idxCH := channel.Channel{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &idxCH)).To(Succeed())
			ch := channel.Channel{
				Name:       "SG0002",
				DataType:   telem.Float64T,
				Expression: "return 1",
				Requires:   []channel.Key{vCH.Key(), idxCH.Key()},
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(MatchError(validate.FieldError{
				Field:   "requires",
				Message: "cannot use a mix of virtual and non-virtual channels in calculations",
			}))
		})
		It("Should return an error if the calculated channel depends on a channel that does not exist", func() {
			ch := channel.Channel{
				Name:       "SG0001",
				DataType:   telem.Float64T,
				Expression: "return 1",
				Requires:   []channel.Key{111111111},
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(MatchError(query.NotFound))
		})
		It("Should return an error if all required channels do not share the same index", func() {
			idxCH1 := channel.Channel{
				Name:     "time1",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			idxCH2 := channel.Channel{
				Name:     "time2",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &idxCH1)).To(Succeed())
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &idxCH2)).To(Succeed())
			ch := channel.Channel{
				Name:       "SG0001",
				DataType:   telem.Float64T,
				Expression: "return 1",
				Requires:   []channel.Key{idxCH1.Key(), idxCH2.Key()},
			}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(MatchError(validate.FieldError{
				Field:   "requires",
				Message: "all required channels must share the same index",
			}))
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

			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch2)).To(Succeed())
		})
		It("Should update the channel name without error", func() {
			ch.Name = "SG0002"
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Name).To(Equal("SG0002"))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal("SG0002"))
		})
		It("Should update the channel expression without error", func() {
			ch.Requires = []channel.Key{ch2.Key()}
			ch.Expression = "sin(x)"
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Expression).To(Equal("sin(x)"))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Expression).To(Equal("sin(x)"))
		})
		It("Should update the requires without error", func() {
			ch.Requires = []channel.Key{ch2.Key()}
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch)).To(Succeed())
			Expect(ch.Requires).To(ContainElement(ch2.Key()))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Requires).To(ContainElement(ch2.Key()))
		})
		It("Should not update the channel if it already exists by name", func() {
			ch.Name = "SG0003"
			Expect(mockCluster.Nodes[1].Channels.Create(ctx, &ch, channel.RetrieveIfNameExists(true))).To(Succeed())
			Expect(ch.Name).To(Equal("SG0001"))

			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(ch.Key()).Entries(&resChannels).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal("SG0001"))
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
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &nonVirtualCh)).To(Succeed())
				originalKey := nonVirtualCh.Key()

				nonVirtualCh.Name = "UpdatedName"
				Expect(mockCluster.Nodes[1].Channels.Create(ctx, &nonVirtualCh)).To(Succeed())

				Expect(nonVirtualCh.Key()).ToNot(Equal(originalKey))

				var resChannels []channel.Channel
				err := mockCluster.Nodes[1].Channels.NewRetrieve().WhereKeys(originalKey, nonVirtualCh.Key()).Entries(&resChannels).Exec(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resChannels).To(HaveLen(2))

				Expect(resChannels[0].Name).To(Equal("NonVirtual"))
				Expect(resChannels[1].Name).To(Equal("UpdatedName"))
			})
	})

})
