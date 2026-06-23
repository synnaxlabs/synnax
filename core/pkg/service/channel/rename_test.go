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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rename", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		services    map[node.Key]*channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.MustOpenCluster(context.Background(), 3)
		services = make(map[node.Key]*channel.Service)
		for k, n := range mockCluster.Nodes {
			services[k] = openService(ctx, n)
		}
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func(ctx SpecContext) {
			ch.Virtual = true
			ch.Name = channel.NewRandomName()
			ch.DataType = telem.Float64T
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should rename the channel without error", func(ctx SpecContext) {
				name := channel.NewRandomName()
				Expect(services[1].Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(services[1].NewRetrieve().
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
				Expect(services[2].Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(services[2].NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
		Context("new name is invalid", func() {
			It("Should return an error", func(ctx SpecContext) {
				Expect(services[1].Rename(ctx, ch.Key(), "invalid name", false)).To(MatchError(ContainSubstring("contains invalid characters")))
			})
		})
		Context("new name is a duplicate", func() {
			It("Should return an error", func(ctx SpecContext) {
				secondCh := channel.Channel{
					Name:     channel.NewRandomName(),
					Virtual:  true,
					DataType: telem.Float64T,
				}
				Expect(services[1].Create(ctx, &secondCh)).To(Succeed())
				Expect(services[1].Rename(ctx, ch.Key(), secondCh.Name, false)).
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
			Expect(services[1].CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			names := []string{channel.NewRandomName(), channel.NewRandomName(), channel.NewRandomName()}
			Expect(services[1].RenameMany(
				ctx,
				keys,
				names,
				false,
			)).To(Succeed())
			var resChannels []channel.Channel
			Expect(services[1].NewRetrieve().Where(channel.MatchKeys(keys...)).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(channel.KeysFromChannels(resChannels)).To(Equal(keys))
			Expect(resChannels[0].Name).To(Equal(names[0]))
			Expect(resChannels[1].Name).To(Equal(names[1]))
			Expect(resChannels[2].Name).To(Equal(names[2]))
		})
	})

})
