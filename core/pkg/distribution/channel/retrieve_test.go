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
			Expect(n.Search.Initialize(ctx)).To(Succeed())
		}
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a set of channels", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     channel.NewRandomName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     channel.NewRandomName(),
				}}
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())

			var resChannels []channel.Channel

			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereNodeKey(1).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(len(created) + internalChannelCount))

			Eventually(func(g Gomega) {
				var resChannelsTwo []channel.Channel

				g.Expect(mockCluster.Nodes[2].Channel.
					NewRetrieve().
					WhereNodeKey(1).
					Entries(&resChannelsTwo).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(resChannelsTwo).To(HaveLen(len(created) + internalChannelCount))
			})

		})
		It("Should correctly retrieve a channel by its key", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     channel.NewRandomName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     channel.NewRandomName(),
				},
			}
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(created[0].Key()).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Key()).To(Equal(created[0].Key()))
		})
		It("Should correctly retrieve a channel by its name", func() {
			n := channel.NewRandomName()
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     n,
				},
			}
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereNames(n).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
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
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereNames("SG22.*").
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(2))
		})
		It("Should return a well formatted error if a channel cannot be found by its key", func() {
			var resChannels []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(435).
				Entries(&resChannels).
				Exec(ctx, nil)).To(MatchError(ContainSubstring("Channels with keys [435] not found")))
		})
		It("Should correctly filter channels by search term", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "a_completely_different_name",
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
				g.Expect(resChannels).To(HaveLen(1))
				g.Expect(resChannels[0].Name).To(Equal("catalina"))
			}).Should(Succeed())
		})

		It("Should return an error when retrieving a channel with a key of 0", func() {
			var resChannels []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(0).
				Entries(&resChannels).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})

	})
	Describe("WhereCalculated", func() {
		It("Should only return calculated channels", func() {
			base := channel.Channel{
				Virtual:  true,
				DataType: telem.Float32T,
				Name:     "wc_base",
			}
			calc := channel.Channel{
				Virtual:    true,
				DataType:   telem.Float32T,
				Name:       "wc_calc",
				Expression: "return wc_base * 2",
			}
			channels := []channel.Channel{base, calc}
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())

			var results []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereCalculated().
				Entries(&results).
				Exec(ctx, nil)).To(Succeed())
			for _, ch := range results {
				Expect(ch.IsCalculated()).To(BeTrue())
			}
			Expect(results).To(ContainElement(
				HaveField("Name", Equal("wc_calc")),
			))
		})

		It("Should return empty when no calculated channels exist in a fresh cluster", func() {
			freshCluster := mock.ProvisionCluster(ctx, 1)
			defer func() { Expect(freshCluster.Close()).To(Succeed()) }()
			base := channel.Channel{
				Virtual:  true,
				DataType: telem.Float32T,
				Name:     "wc_only_base",
			}
			Expect(freshCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &[]channel.Channel{base})).To(Succeed())

			var results []channel.Channel
			Expect(freshCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereCalculated().
				Entries(&results).
				Exec(ctx, nil)).To(Succeed())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("Exists", func() {
		It("Should return true if a channel exists", func() {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     channel.NewRandomName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     channel.NewRandomName(),
				},
			}
			Expect(mockCluster.Nodes[1].Channel.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())

			exists := MustSucceed(mockCluster.Nodes[1].Channel.
				NewRetrieve().
				WhereKeys(created[0].Key()).
				Exists(ctx, nil))
			Expect(exists).To(BeTrue())
		})
	})
})
