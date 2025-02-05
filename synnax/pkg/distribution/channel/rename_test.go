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
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rename", Ordered, func() {
	var (
		services map[core.NodeKey]channel.Service
		builder  *mock.CoreBuilder
	)
	BeforeAll(func() { builder, services = provisionServices() })
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func() {
			var err error
			ch.Rate = 5 * telem.Hz
			ch.Name = "SG01"
			ch.DataType = telem.Float64T
			err = services[1].Create(ctx, &ch)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should rename the channel without error", func() {
				Expect(services[1].Rename(ctx, ch.Key(), "SG03", false)).To(Succeed())
				Eventually(func(g Gomega) {
					var resCh channel.Channel
					g.Expect(services[1].NewRetrieve().
						WhereKeys(ch.Key()).
						Entry(&resCh).Exec(ctx, nil),
					).To(Succeed())
					g.Expect(resCh.Name).To(Equal("SG03"))
				}).Should(Succeed())
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should rename the channel without error", func() {
				Expect(services[2].Rename(ctx, ch.Key(), "SG03", false)).To(Succeed())
				Eventually(func(g Gomega) {
					var resCh channel.Channel
					g.Expect(services[2].NewRetrieve().
						WhereKeys(ch.Key()).
						Entry(&resCh).
						Exec(ctx, nil)).To(Succeed())
					g.Expect(resCh.Name).To(Equal("SG03"))
				}).Should(Succeed())
			})
		})
	})
	Context("Multiple channels", func() {
		It("Should rename the channels without error", func() {
			ch1 := channel.Channel{
				Name:     "fermat",
				Rate:     2 * telem.Hz,
				DataType: telem.Int64T,
			}
			ch2 := channel.Channel{
				Name:     "laplace",
				Rate:     2 * telem.Hz,
				DataType: telem.Float32T,
			}
			ch3 := channel.Channel{
				Name:        "newton",
				DataType:    telem.StringT,
				Leaseholder: core.Free,
				Virtual:     true,
			}
			channels := []channel.Channel{ch1, ch2, ch3}
			Expect(services[1].CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			Expect(services[1].RenameMany(
				ctx,
				keys,
				[]string{"fermat1", "laplace1", "newton1"},
				false,
			)).To(Succeed())

			Eventually(func(g Gomega) {
				var resChannels []channel.Channel
				g.Expect(services[1].NewRetrieve().WhereKeys(keys...).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
				Expect(channel.KeysFromChannels(resChannels)).To(Equal(keys))
				g.Expect(resChannels[0].Name).To(Equal("fermat1"))
				g.Expect(resChannels[1].Name).To(Equal("laplace1"))
				g.Expect(resChannels[2].Name).To(Equal("newton1"))
			}).Should(Succeed())
		})
	})
})
