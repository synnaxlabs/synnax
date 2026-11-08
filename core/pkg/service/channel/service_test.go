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
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("ShouldValidateNames", func() {
		It("Should report true by default", func(ctx SpecContext) {
			Expect(svc.ShouldValidateNames()).To(BeTrue())
		})
		It(
			"Should report false when name validation is disabled",
			func(ctx SpecContext) {
				noValidateSvc := openService(
					ctx,
					testutil.NewNode(ctx),
					channel.ServiceConfig{
						ValidateNames: new(false),
					},
				)
				Expect(noValidateSvc.ShouldValidateNames()).To(BeFalse())
			},
		)
	})

	Describe("NewArcSymbolResolver", func() {
		It(
			"Should resolve a channel created through the service by name",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:     "arc_resolver_by_name",
					DataType: telem.Float32T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
				sym := MustSucceed(svc.NewArcSymbolResolver(nil).Resolve(ctx, ch.Name))
				Expect(sym.Name).To(Equal(ch.Name))
				Expect(sym.ID).To(Equal(int(ch.Key())))
			},
		)
		It("Should fuzzy-search channels by name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "catalina",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
			Expect(svc.NewArcSymbolResolver(nil).Search(ctx, ch.Name)).
				To(ContainElement(HaveField("Name", ch.Name)))
		})
	})

	Describe("CalculationInputs", func() {
		create := func(ctx SpecContext, expression string) channel.Channel {
			GinkgoHelper()
			ch := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: expression,
			}
			Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
			return ch
		}
		It(
			"Should return the channels a calculation reads through other calculations",
			func(ctx SpecContext) {
				base := create(ctx, "")
				inner := create(ctx, "return "+base.Name+" * 2")
				outer := create(ctx, "return "+inner.Name+" + 1")
				Expect(svc.CalculationInputs(ctx, channel.Keys{outer.Key()})).
					To(ConsistOf(inner.Key(), base.Key()))
			},
		)
		It("Should return nothing for a channel that is not calculated", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			Expect(svc.CalculationInputs(ctx, channel.Keys{base.Key()})).To(BeEmpty())
		})
		It("Should leave out the requested keys", func(ctx SpecContext) {
			base := create(ctx, "")
			calc := create(ctx, "return "+base.Name+" * 2")
			Expect(svc.CalculationInputs(ctx, channel.Keys{calc.Key(), base.Key()})).
				To(BeEmpty())
		})
		It("Should return nothing for a calculation that fails analysis", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			calc := create(ctx, "return "+base.Name+" * 2")
			Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
			Expect(svc.CalculationInputs(ctx, channel.Keys{calc.Key()})).To(BeEmpty())
		})
		It(
			"Should return nothing for a key that has no channel",
			func(ctx SpecContext) {
				missing := distchannel.NewKey(1, 99999)
				Expect(svc.CalculationInputs(ctx, channel.Keys{missing})).To(BeEmpty())
			},
		)
	})

	Describe("NewRetrieve", func() {
		It(
			"Should retrieve a channel created through the service",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
				var retrieved channel.Channel
				Expect(svc.NewRetrieve().Where(
					channel.MatchKeys(ch.Key()),
				).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
				Expect(retrieved.Name).To(Equal(ch.Name))
			},
		)
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
				},
			)
			defer disconnect()
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})

	Describe("NewWriter", func() {
		It(
			"Should create a writer that infers types for calculated channels",
			func(ctx SpecContext) {
				w := svc.NewWriter(nil)
				ch := channel.Channel{
					Name:       UniqueChannelName(),
					Expression: "return 1 + 1",
					Virtual:    true,
				}
				Expect(w.Create(ctx, &ch)).To(Succeed())
				Expect(ch.DataType).To(Equal(telem.Int64T))
			},
		)
	})
})
