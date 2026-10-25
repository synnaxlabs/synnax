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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", Ordered, func() {
	var (
		svc    *channel.Service
		writer channel.Writer
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		svc, _ = openService(ctx, mock.NewNode(ctx))
		writer = svc.NewWriter(nil)
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a set of channels", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
			}
			Expect(writer.CreateMany(ctx, &created)).To(Succeed())

			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchLeaseholders(1)).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(len(created)))
		})
		It("Should correctly retrieve a channel by its key", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
			}
			Expect(writer.CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchKeys(created[0].Key())).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Key()).To(Equal(created[0].Key()))
		})
		It("Should correctly retrieve a channel by its name", func(ctx SpecContext) {
			n := UniqueChannelName()
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     n,
				},
			}
			Expect(writer.CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchNames(n)).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(n))
		})
		It(
			"Should correctly retrieve channels by regex expression",
			func(ctx SpecContext) {
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
				Expect(writer.CreateMany(ctx, &created)).To(Succeed())
				var resChannels []channel.Channel

				Expect(svc.
					NewRetrieve().
					Where(channel.MatchNames("SG22.*")).
					Entries(&resChannels).
					Exec(ctx, nil)).To(Succeed())
				Expect(resChannels).To(HaveLen(2))
			},
		)
		It(
			"Should return a well formatted error if a channel cannot be found by its key",
			func(ctx SpecContext) {
				var resChannels []channel.Channel
				Expect(svc.
					NewRetrieve().
					Where(channel.MatchKeys(435)).
					Entries(&resChannels).
					Exec(ctx, nil),
				).To(MatchError(ContainSubstring("Channels with keys [435] not found")))
			},
		)
		It("Should correctly filter channels by search term", func(ctx SpecContext) {
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
			Expect(writer.CreateMany(ctx, &created)).To(Succeed())
			Eventually(func(g Gomega) {
				var resChannels []channel.Channel
				g.Expect(svc.
					NewRetrieve().
					Search("catalina").
					Entries(&resChannels).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(resChannels).To(HaveLen(1))
				g.Expect(resChannels[0].Name).To(Equal("catalina"))
			}).Should(Succeed())
		})

		It(
			"Should return an error when retrieving a channel with a key of 0",
			func(ctx SpecContext) {
				var resChannels []channel.Channel
				Expect(svc.
					NewRetrieve().
					Where(channel.MatchKeys(0)).
					Entries(&resChannels).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("MatchCalculated", func() {
		It("Should only return calculated channels", func(ctx SpecContext) {
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
			Expect(writer.Create(ctx, &base)).To(Succeed())
			Expect(writer.Create(ctx, &calc)).To(Succeed())

			var results []channel.Channel
			Expect(svc.
				NewRetrieve().
				Where(channel.MatchCalculated()).
				Entries(&results).
				Exec(ctx, nil)).To(Succeed())
			for _, ch := range results {
				Expect(ch.IsCalculated()).To(BeTrue())
			}
			Expect(results).To(ContainElement(
				HaveField("Name", Equal("wc_calc")),
			))
		})

		It(
			"Should return empty when no calculated channels exist on a fresh node",
			func(ctx SpecContext) {
				freshSvc, _ := openService(ctx, mock.NewNode(ctx))
				base := channel.Channel{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     "wc_only_base",
				}
				Expect(freshSvc.NewWriter(nil).
					CreateMany(ctx, &[]channel.Channel{base})).To(Succeed())

				var results []channel.Channel
				Expect(freshSvc.
					NewRetrieve().
					Where(channel.MatchCalculated()).
					Entries(&results).
					Exec(ctx, nil)).To(Succeed())
				Expect(results).To(BeEmpty())
			},
		)
	})

	Describe("MatchIndexes", func() {
		createIndexed := func(ctx SpecContext, prefix string) (
			channel.Channel, []channel.Channel,
		) {
			GinkgoHelper()
			idx := channel.Channel{
				Name: prefix + "_time", DataType: telem.TimestampT, IsIndex: true,
			}
			Expect(writer.Create(ctx, &idx)).To(Succeed())
			data := []channel.Channel{
				{
					Name:       prefix + "_a",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				},
				{
					Name:       prefix + "_b",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				},
			}
			Expect(writer.CreateMany(ctx, &data)).To(Succeed())
			return idx, data
		}
		retrieveKeys := func(ctx SpecContext, keys ...channel.Key) channel.Keys {
			GinkgoHelper()
			var results []channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchIndexes(keys...)).
				Entries(&results).
				Exec(ctx, nil)).To(Succeed())
			return channel.KeysFromChannels(results)
		}

		It("Should match every channel indexed by the key", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, "mi_stored")
			Expect(retrieveKeys(ctx, idx.Key())).To(ConsistOf(
				idx.Key(), data[0].Key(), data[1].Key(),
			))
		})

		It("Should match a calculated channel and its index", func(ctx SpecContext) {
			base := channel.Channel{
				Name: "mi_base", DataType: telem.Float32T, Virtual: true,
			}
			calc := channel.Channel{
				Name:       "mi_calc",
				DataType:   telem.Float32T,
				Expression: "return mi_base * 2",
			}
			Expect(writer.Create(ctx, &base)).To(Succeed())
			Expect(writer.Create(ctx, &calc)).To(Succeed())
			Expect(retrieveKeys(ctx, calc.Index())).To(ConsistOf(
				calc.Index(), calc.Key(),
			))
		})

		It("Should match across several index keys", func(ctx SpecContext) {
			first, firstData := createIndexed(ctx, "mi_first")
			second, secondData := createIndexed(ctx, "mi_second")
			Expect(retrieveKeys(ctx, first.Key(), second.Key())).To(ConsistOf(
				first.Key(), firstData[0].Key(), firstData[1].Key(),
				second.Key(), secondData[0].Key(), secondData[1].Key(),
			))
		})

		It("Should reject a same local index on another leaseholder", func(
			ctx SpecContext,
		) {
			idx, _ := createIndexed(ctx, "mi_twin")
			Expect(retrieveKeys(ctx, channel.NewKey(2, idx.LocalKey))).To(BeEmpty())
		})

		It("Should match nothing when given no keys", func(ctx SpecContext) {
			createIndexed(ctx, "mi_none")
			Expect(retrieveKeys(ctx)).To(BeEmpty())
		})

		It("Should stop matching a channel once it is deleted", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, "mi_deleted")
			Expect(writer.Delete(ctx, data[0].Key(), false)).To(Succeed())
			Expect(retrieveKeys(ctx, idx.Key())).To(ConsistOf(idx.Key(), data[1].Key()))
		})
	})

	Describe("Exists", func() {
		It("Should return true if a channel exists", func(ctx SpecContext) {
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     UniqueChannelName(),
				},
			}
			Expect(writer.CreateMany(ctx, &created)).To(Succeed())

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchKeys(created[0].Key())).
				Exists(ctx, nil),
			).To(BeTrue())
		})
	})
})
