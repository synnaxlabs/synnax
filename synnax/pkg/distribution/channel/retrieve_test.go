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
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("getAttributes", Ordered, func() {
	var (
		services map[aspen.NodeKey]channel.Service
		builder  *mock.CoreBuilder
	)
	BeforeAll(func() { builder, services = provisionServices() })
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("Retrieve", func() {
		It("Should correctly retrieve a set of channels", func() {
			ch1 := channel.Channel{
				Rate:     25 * telem.Hz,
				DataType: telem.Float32T,
				Name:     "SG02",
			}
			ch2 := channel.Channel{
				Rate:     25 * telem.Hz,
				DataType: telem.Float32T,
				Name:     "SG03",
			}
			created := []channel.Channel{ch1, ch2}
			err := services[1].NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())

			var resChannels []channel.Channel

			err = services[1].
				NewRetrieve().
				WhereNodeKey(1).
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(len(created)))

			Eventually(func(g Gomega) {
				var resChannelsTwo []channel.Channel

				err = services[2].
					NewRetrieve().
					WhereNodeKey(1).
					Entries(&resChannelsTwo).
					Exec(ctx, nil)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(resChannelsTwo).To(HaveLen(len(created)))
			})

		})
		It("Should correctly retrieve a channel by its key", func() {
			created := []channel.Channel{
				{
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG02",
				},
				{
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG03",
				},
			}
			err := services[1].NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel

			err = services[1].
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
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     n,
				},
			}
			err := services[1].NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel

			err = services[1].
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
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG222",
				},
				{
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG223",
				},
			}
			err := services[1].NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel

			err = services[1].
				NewRetrieve().
				WhereNames("SG22.*").
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(2))
		})
		It("Should return a well formatted error if a channel cannot be found by its key", func() {
			var resChannels []channel.Channel
			err := services[1].
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
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG-----222",
				},
				{
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG-----223",
				},
			}
			err := services[1].NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())
			var resChannels []channel.Channel
			err = services[1].
				NewRetrieve().
				Search("SG-----222").
				Entries(&resChannels).
				Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(resChannels)).To(BeNumerically(">", 0))
			Expect(resChannels[0].Name).To(Equal("SG-----222"))

		})
	})
	Describe("Exists", func() {
		It("Should return true if a channel exists", func() {
			created := []channel.Channel{
				{
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG02",
				},
				{
					Rate:     25 * telem.Hz,
					DataType: telem.Float32T,
					Name:     "SG03",
				},
			}
			err := services[1].NewWriter(nil).CreateMany(ctx, &created)
			Expect(err).ToNot(HaveOccurred())

			exists, err := services[1].
				NewRetrieve().
				WhereKeys(created[0].Key()).
				Exists(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
})
