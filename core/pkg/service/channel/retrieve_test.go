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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const internalChannelCount = 1

var _ = Describe("Retrieve", Ordered, func() {
	var svc *channel.Service
	BeforeAll(func(ctx SpecContext) {
		n := mock.NewNode(ctx)
		svc = openService(ctx, n)
		Expect(n.Search.Initialize(ctx)).To(Succeed())
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a set of channels", func(ctx SpecContext) {
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
			Expect(svc.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())

			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchLeaseholders(1)).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(len(created) + internalChannelCount))
		})
		It("Should correctly retrieve a channel by its key", func(ctx SpecContext) {
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
			Expect(svc.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
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
			n := channel.NewRandomName()
			created := []channel.Channel{
				{
					Virtual:  true,
					DataType: telem.Float32T,
					Name:     n,
				},
			}
			Expect(svc.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchNames(n)).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Name).To(Equal(n))
		})
		It("Should correctly retrieve channels by regex expression", func(ctx SpecContext) {
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
			Expect(svc.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
			var resChannels []channel.Channel

			Expect(svc.
				NewRetrieve().
				Where(channel.MatchNames("SG22.*")).
				Entries(&resChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(resChannels).To(HaveLen(2))
		})
		It("Should return a well formatted error if a channel cannot be found by its key", func(ctx SpecContext) {
			var resChannels []channel.Channel
			Expect(svc.
				NewRetrieve().
				Where(channel.MatchKeys(435)).
				Entries(&resChannels).
				Exec(ctx, nil)).To(MatchError(ContainSubstring("Channels with keys [435] not found")))
		})
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
			Expect(svc.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())
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

		It("Should return an error when retrieving a channel with a key of 0", func(ctx SpecContext) {
			var resChannels []channel.Channel
			Expect(svc.
				NewRetrieve().
				Where(channel.MatchKeys(0)).
				Entries(&resChannels).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})

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
			Expect(svc.Create(ctx, &base)).To(Succeed())
			Expect(svc.Create(ctx, &calc)).To(Succeed())

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

		It("Should return empty when no calculated channels exist on a fresh node", func(ctx SpecContext) {
			freshSvc := openService(ctx, mock.NewNode(ctx))
			base := channel.Channel{
				Virtual:  true,
				DataType: telem.Float32T,
				Name:     "wc_only_base",
			}
			Expect(freshSvc.NewWriter(nil).CreateMany(ctx, &[]channel.Channel{base})).To(Succeed())

			var results []channel.Channel
			Expect(freshSvc.
				NewRetrieve().
				Where(channel.MatchCalculated()).
				Entries(&results).
				Exec(ctx, nil)).To(Succeed())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("Exists", func() {
		It("Should return true if a channel exists", func(ctx SpecContext) {
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
			Expect(svc.NewWriter(nil).CreateMany(ctx, &created)).To(Succeed())

			exists := MustSucceed(svc.
				NewRetrieve().
				Where(channel.MatchKeys(created[0].Key())).
				Exists(ctx, nil))
			Expect(exists).To(BeTrue())
		})
	})
})
