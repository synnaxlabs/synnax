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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelmock "github.com/synnaxlabs/synnax/pkg/service/channel/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rename", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.ProvisionCluster(context.Background(), 3)
		for _, n := range mockCluster.Nodes {
			channelmock.ChannelService(n)
		}
	})
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func(ctx SpecContext) {
			ch.Virtual = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.Float64T
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &ch)).To(Succeed())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should rename the channel without error", func(ctx SpecContext) {
				name := channel.NewRandomName()
				Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(channelmock.ChannelService(mockCluster.Nodes[1]).NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should rename the channel without error", func(ctx SpecContext) {
				name := channel.NewRandomName()
				Expect(channelmock.ChannelService(mockCluster.Nodes[2]).Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(channelmock.ChannelService(mockCluster.Nodes[2]).NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
		Context("new name is invalid", func() {
			It("Should return an error", func(ctx SpecContext) {
				Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Rename(ctx, ch.Key(), "invalid name", false)).To(MatchError(ContainSubstring("contains invalid characters")))
			})
		})
		Context("new name is a duplicate", func() {
			It("Should return an error", func(ctx SpecContext) {
				secondCh := channel.Channel{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Float64T,
				}
				Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Create(ctx, &secondCh)).To(Succeed())
				Expect(channelmock.ChannelService(mockCluster.Nodes[1]).Rename(ctx, ch.Key(), secondCh.Name, false)).
					To(MatchError(ContainSubstring("channel with name '%s' already exists", secondCh.Name)))
			})
		})
	})
	Context("Multiple channels", func() {
		It("Should rename the channels without error", func(ctx SpecContext) {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Int64T,
				},
				{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Float32T,
				},
				{
					Name:        channel.NewRandomName(),
					DataType:    telem.StringT,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				},
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			names := []string{channel.NewRandomName(), channel.NewRandomName(), channel.NewRandomName()}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).RenameMany(
				ctx,
				keys,
				names,
				false,
			)).To(Succeed())
			var resChannels []channel.Channel
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).NewRetrieve().Where(channel.MatchKeys(keys...)).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(channel.KeysFromChannels(resChannels)).To(Equal(keys))
			Expect(resChannels[0].Name).To(Equal(names[0]))
			Expect(resChannels[1].Name).To(Equal(names[1]))
			Expect(resChannels[2].Name).To(Equal(names[2]))
		})
	})

	Context("Map Rename", func() {
		It("Should rename channels using a map of old names to new names", func(ctx SpecContext) {
			id := channel.NewRandomName()
			ch1 := channel.Channel{
				Name:     fmt.Sprintf("young_fermat_%s", id),
				Virtual:  true,
				DataType: telem.Int64T,
			}
			ch2 := channel.Channel{
				Name:     fmt.Sprintf("young_laplace_%s", id),
				Virtual:  true,
				DataType: telem.Float32T,
			}
			ch3 := channel.Channel{
				Name:        fmt.Sprintf("young_newton_%s", id),
				DataType:    telem.StringT,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			}
			channels := []channel.Channel{ch1, ch2, ch3}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).CreateMany(ctx, &channels)).To(Succeed())
			nameMap := map[string]string{
				ch1.Name: fmt.Sprintf("old_fermat_%s", id),
				ch2.Name: fmt.Sprintf("old_laplace_%s", id),
				ch3.Name: fmt.Sprintf("old_newton_%s", id),
			}
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).MapRename(ctx, nameMap, false)).To(Succeed())
			var resChannels []channel.Channel
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).NewRetrieve().
				Where(channel.MatchNames(lo.Keys(nameMap)...)).
				Entries(&resChannels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(resChannels).To(BeEmpty())
			Expect(channelmock.ChannelService(mockCluster.Nodes[1]).NewRetrieve().
				Where(channel.MatchNames(lo.Values(nameMap)...)).
				Entries(&resChannels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(resChannels).To(HaveLen(3))
		})
	})
})
