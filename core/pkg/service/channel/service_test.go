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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Service", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		services    map[node.Key]*channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.MustOpenCluster(ctx, 1)
		services = make(map[node.Key]*channel.Service)
		for k, n := range mockCluster.Nodes {
			services[k] = openService(ctx, n)
		}
	})

	Describe("CountExternalNonVirtual", func() {
		It("Should return zero for empty database", func(ctx SpecContext) {
			count := services[1].CountExternalNonVirtual()
			Expect(count).To(BeEquivalentTo(0))
		})

		It("Should count external non-virtual channels", func(ctx SpecContext) {
			initialCount := services[1].CountExternalNonVirtual()

			// Create an index channel (external, non-virtual)
			indexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &indexCh)).To(Succeed())

			// Create a data channel (external, non-virtual)
			dataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  indexCh.LocalKey,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &dataCh)).To(Succeed())

			// Count should increase by 2
			Expect(services[1].CountExternalNonVirtual()).To(Equal(initialCount + 2))
		})

		It("Should not count virtual channels", func(ctx SpecContext) {
			initialCount := services[1].CountExternalNonVirtual()

			// Create a virtual channel (external, but virtual)
			virtualCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			}
			Expect(services[1].Create(ctx, &virtualCh)).To(Succeed())

			// Count should NOT increase
			Expect(services[1].CountExternalNonVirtual()).To(Equal(initialCount))
		})

		It("Should not count internal channels", func(ctx SpecContext) {
			initialCount := services[1].CountExternalNonVirtual()

			// Create an internal index channel
			internalIndexCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(services[1].Create(ctx, &internalIndexCh)).To(Succeed())

			// Create an internal data channel
			internalDataCh := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  internalIndexCh.LocalKey,
				Leaseholder: 1,
				Internal:    true,
			}
			Expect(services[1].Create(ctx, &internalDataCh)).To(Succeed())

			// Count should NOT increase
			Expect(services[1].CountExternalNonVirtual()).To(Equal(initialCount))
		})
	})

	Describe("Observe", func() {
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			disconnect := services[1].Observe().OnChange(func(ctx context.Context, _ gorp.TxReader[channel.Key, channel.Channel]) {
				called.Store(true)
			})
			DeferCleanup(disconnect)
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})
})

var _ = Describe("Service Passthrough", func() {
	Describe("Group", func() {
		It("Should return a valid group", func(ctx SpecContext) {
			g := svc.Group()
			Expect(g.Key).ToNot(BeZero())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a channel created through the service", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			var retrieved channel.Channel
			Expect(svc.NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(ch.Name))
		})
	})

	Describe("Observe", func() {
		It("Should return a non-nil observable", func(ctx SpecContext) {
			obs := svc.Observe()
			Expect(obs).ToNot(BeNil())
		})
	})

	Describe("CountExternalNonVirtual", func() {
		It("Should return a count", func(ctx SpecContext) {
			count := svc.CountExternalNonVirtual()
			Expect(count).To(BeNumerically(">=", 0))
		})
	})

	Describe("Delete", func() {
		It("Should delete a channel by key", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(svc.Delete(ctx, ch.Key(), false)).To(Succeed())
		})
	})

	Describe("DeleteMany", func() {
		It("Should delete multiple channels by key", func(ctx SpecContext) {
			channels := []channel.Channel{
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			keys := []channel.Key{channels[0].Key(), channels[1].Key()}
			Expect(svc.DeleteMany(ctx, keys, false)).To(Succeed())
		})
	})

	Describe("DeleteByName", func() {
		It("Should delete a channel by name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			Expect(svc.DeleteByName(ctx, ch.Name, false)).To(Succeed())
		})
	})

	Describe("DeleteManyByNames", func() {
		It("Should delete multiple channels by name", func(ctx SpecContext) {
			channels := []channel.Channel{
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			names := []string{channels[0].Name, channels[1].Name}
			Expect(svc.DeleteManyByNames(ctx, names, false)).To(Succeed())
		})
	})

	Describe("Rename", func() {
		It("Should rename a channel", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			newName := channel.NewRandomName()
			Expect(svc.Rename(ctx, ch.Key(), newName, false)).To(Succeed())
			var retrieved channel.Channel
			Expect(svc.NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(newName))
		})
	})

	Describe("RenameMany", func() {
		It("Should rename multiple channels", func(ctx SpecContext) {
			channels := []channel.Channel{
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
				{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true},
			}
			Expect(svc.CreateMany(ctx, &channels)).To(Succeed())
			newNames := []string{channel.NewRandomName(), channel.NewRandomName()}
			keys := []channel.Key{channels[0].Key(), channels[1].Key()}
			Expect(svc.RenameMany(ctx, keys, newNames, false)).To(Succeed())
			var r0 channel.Channel
			Expect(svc.NewRetrieve().Where(channel.MatchKeys(keys[0])).Entry(&r0).Exec(ctx, nil)).To(Succeed())
			Expect(r0.Name).To(Equal(newNames[0]))
		})
	})

	Describe("MapRename", func() {
		It("Should rename channels via old-to-new name map", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			newName := channel.NewRandomName()
			Expect(svc.MapRename(ctx, map[string]string{ch.Name: newName}, false)).To(Succeed())
			var retrieved channel.Channel
			Expect(svc.NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(newName))
		})
	})
})
