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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const internalChannelCount = 1

var _ = Describe("Retrieve", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func() {
		mockCluster = mock.ProvisionCluster(ctx, 2)
		for _, n := range mockCluster.Nodes {
			Expect(n.Ontology.InitializeSearchIndex(ctx)).To(Succeed())
		}
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a set of channels", func() {
			ch1 := channel.Channel{
				Virtual:  true,
				DataType: telem.Float32T,
				Name:     "SG02",
			}
			ch2 := channel.Channel{
				Virtual:  true,
				DataType: telem.Float32T,
				Name:     "SG03",
			}
			created := []channel.Channel{ch1, ch2}
			err := mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())

			var resChannels []channel.Channel

			err = mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereNodeKey(1).
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(len(created) + internalChannelCount))

			Eventually(func(g Gomega) {
				var resChannelsTwo []channel.Channel

				err = mockCluster.Nodes[2].Channel.
					NewRetrieve().
					WhereNodeKey(1).
					Entries(&resChannelsTwo).
					Exec(ctx, nil)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(resChannelsTwo).To(HaveLen(len(created) + internalChannelCount))
			})

		})
		It("Should correctly retrieve a channel by its key", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "SG02",
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "SG03",
				},
			}
			err := mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel

			err = mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(created[0].Key()).
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Key()).To(Equal(created[0].Key()))
		})
		It("Should correctly retrieve a channel by its name", func() {
			n := uuid.New().String()
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     n,
				},
			}
			err := mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel

			err = mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereNames(n).
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(n))
		})
		It("Should correctly retrieve channels by regex expression", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "SG222",
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "SG223",
				},
			}
			err := mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel

			err = mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereNames("SG22.*").
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(2))
		})
		It("Should return a well formatted error if a channel cannot be found by its key", func() {
			var resChannels []channel.Channel
			err := mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(435).
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Channels with keys [435] not found"))

		})
		It("Should correctly filter channels by search term", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "YXG-----222",
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "catalina",
				},
			}
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
			Eventually(func(g Gomega) {
				var resChannels []channel.Channel
				g.Expect(mockCluster.Nodes[1].Channel.
					NewRetrieve().
					Search("catalina").
					Entries(&resChannels).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(len(resChannels)).To(BeNumerically(">", 0))
				g.Expect(resChannels[0].Name).To(Equal("catalina"))
			}).Should(Succeed())
		})

		It("Should return an error when retrieving a channel with a key of 0", func() {
			var resChannels []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(0).
				Entries(&resChannels).
				Exec(ctx, nil)).To(HaveOccurredAs(query.NotFound))
		})

	})
	Describe("Exists", func() {
		It("Should return true if a channel exists", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "SG02",
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "SG03",
				},
			}
			err := mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())

			exists, err := mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(created[0].Key()).
				Exists(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

})
