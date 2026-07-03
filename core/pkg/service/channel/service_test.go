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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Group", func() {
		It("Should return the Channels group channels are created under", func(ctx SpecContext) {
			g := svc.Group()
			Expect(g.Key).ToNot(BeZero())
			Expect(g.Name).To(Equal("Channels"))
		})
	})

	Describe("ShouldValidateNames", func() {
		It("Should report true by default", func(ctx SpecContext) {
			Expect(svc.ShouldValidateNames()).To(BeTrue())
		})
		It("Should report false when name validation is disabled", func(ctx SpecContext) {
			noValidateSvc := openService(ctx, mock.NewNode(ctx), channel.ServiceConfig{
				ValidateNames: new(false),
			})
			Expect(noValidateSvc.ShouldValidateNames()).To(BeFalse())
		})
	})

	Describe("NewArcSymbolResolver", func() {
		It("Should resolve a channel created through the service by name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "arc_resolver_by_name",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			sym := MustSucceed(svc.NewArcSymbolResolver(nil).Resolve(ctx, ch.Name))
			Expect(sym.Name).To(Equal(ch.Name))
			Expect(sym.ID).To(Equal(int(ch.Key())))
		})
		It("Should fuzzy-search channels by name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "catalina",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Eventually(func(g Gomega) {
				results := MustSucceed(
					svc.NewArcSymbolResolver(nil).Search(ctx, ch.Name),
				)
				names := make([]string, len(results))
				for i, sym := range results {
					names[i] = sym.Name
				}
				g.Expect(names).To(ContainElement(ch.Name))
			}).Should(Succeed())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a channel created through the service", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			var retrieved channel.Channel
			Expect(svc.NewRetrieve().Where(
				channel.MatchKeys(ch.Key()),
			).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(ch.Name))
		})
	})

	Describe("Observe", func() {
		It("Should return a non-nil observable", func(ctx SpecContext) {
			obs := svc.Observe()
			Expect(obs).ToNot(BeNil())
		})
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			disconnect := svc.Observe().OnChange(
				func(context.Context, gorp.TxReader[channel.Key, channel.Channel]) {
					called.Store(true)
				})
			defer disconnect()
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(svc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})

	Describe("NewWriter", func() {
		It("Should create a writer that infers types for calculated channels", func(ctx SpecContext) {
			w := svc.NewWriter(nil)
			ch := channel.Channel{
				Name:       channel.NewRandomName(),
				Expression: "return 1 + 1",
				Virtual:    true,
			}
			Expect(w.Create(ctx, &ch)).To(Succeed())
			Expect(ch.DataType).To(Equal(telem.Int64T))
		})
	})
})
