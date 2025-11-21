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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rename", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func() { mockCluster = mock.ProvisionCluster(ctx, 3) })
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Context("Single channel", func() {
		var ch channel.Channel
		JustBeforeEach(func() {
			ch.Virtual = true
			ch.Name = RandomName()
			ch.DataType = telem.Float64T
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &ch)).To(Succeed())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should rename the channel without error", func() {
				name := RandomName()
				Expect(mockCluster.Nodes[1].Channel.Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
					WhereKeys(ch.Key()).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should rename the channel without error", func() {
				name := RandomName()
				Expect(mockCluster.Nodes[2].Channel.Rename(ctx, ch.Key(), name, false)).To(Succeed())
				var resCh channel.Channel
				Expect(mockCluster.Nodes[2].Channel.NewRetrieve().
					WhereKeys(ch.Key()).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())
				Expect(resCh.Name).To(Equal(name))
			})
		})
	})
	Context("Multiple channels", func() {
		It("Should rename the channels without error", func() {
			channels := []channel.Channel{
				{
					Name:     RandomName(),
					Virtual:  true,
					DataType: telem.Int64T,
				},
				{
					Name:     RandomName(),
					Virtual:  true,
					DataType: telem.Float32T,
				},
				{
					Name:        RandomName(),
					DataType:    telem.StringT,
					Leaseholder: cluster.Free,
					Virtual:     true,
				},
			}
			Expect(mockCluster.Nodes[1].Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			names := []string{RandomName(), RandomName(), RandomName()}
			Expect(mockCluster.Nodes[1].Channel.RenameMany(
				ctx,
				keys,
				names,
				false,
			)).To(Succeed())
			var resChannels []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().WhereKeys(keys...).Entries(&resChannels).Exec(ctx, nil)).To(Succeed())
			Expect(channel.KeysFromChannels(resChannels)).To(Equal(keys))
			Expect(resChannels[0].Name).To(Equal(names[0]))
			Expect(resChannels[1].Name).To(Equal(names[1]))
			Expect(resChannels[2].Name).To(Equal(names[2]))
		})
	})

	Context("Map Rename", func() {
		It("Should rename channels using a map of old names to new names", func() {
			id := RandomName()
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
				Leaseholder: cluster.Free,
				Virtual:     true,
			}
			channels := []channel.Channel{ch1, ch2, ch3}
			Expect(mockCluster.Nodes[1].Channel.CreateMany(ctx, &channels)).To(Succeed())
			nameMap := map[string]string{
				ch1.Name: fmt.Sprintf("old_fermat_%s", id),
				ch2.Name: fmt.Sprintf("old_laplace_%s", id),
				ch3.Name: fmt.Sprintf("old_newton_%s", id),
			}
			Expect(mockCluster.Nodes[1].Channel.MapRename(ctx, nameMap, false)).To(Succeed())
			var resChannels []channel.Channel
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereNames(lo.Keys(nameMap)...).
				Entries(&resChannels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(resChannels).To(HaveLen(0))
			Expect(mockCluster.Nodes[1].Channel.NewRetrieve().
				WhereNames(lo.Values(nameMap)...).
				Entries(&resChannels).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(resChannels).To(HaveLen(3))
		})
	})
})
