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
	"strconv"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NewArcSymbolResolver", func() {
	It("Should resolve a channel by name", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "resolver_test_ch",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(svc.NewWriter(nil).Create(ctx, ch)).To(Succeed())

		sym := MustSucceed(
			svc.NewArcSymbolResolver(nil).Resolve(ctx, "resolver_test_ch"),
		)
		Expect(sym.Name).To(Equal("resolver_test_ch"))
		Expect(sym.Kind).To(Equal(symbol.KindChannel))
		Expect(sym.Type).To(Equal(types.Chan(types.F32())))
		Expect(sym.ID).To(Equal(int(ch.Key())))
		Expect(sym.Renameable).To(BeTrue())
	})

	It("Should mark internal channels as not Renameable", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "resolver_internal_ch",
			Virtual:  true,
			Internal: true,
			DataType: telem.Float32T,
		}
		Expect(svc.NewWriter(nil).Create(ctx, ch)).To(Succeed())

		sym := MustSucceed(
			svc.NewArcSymbolResolver(nil).Resolve(ctx, "resolver_internal_ch"),
		)
		Expect(sym.Renameable).To(BeFalse())
	})

	It("Should resolve a channel by numeric key", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "resolver_key_test_ch",
			Virtual:  true,
			DataType: telem.Int64T,
		}
		Expect(svc.NewWriter(nil).Create(ctx, ch)).To(Succeed())

		sym := MustSucceed(
			svc.NewArcSymbolResolver(nil).Resolve(ctx, strconv.Itoa(int(ch.Key()))),
		)
		Expect(sym.Name).To(Equal("resolver_key_test_ch"))
		Expect(sym.Kind).To(Equal(symbol.KindChannel))
		Expect(sym.Type).To(Equal(types.Chan(types.I64())))
	})

	It("Should return an error for a nonexistent symbol", func(ctx SpecContext) {
		Expect(svc.NewArcSymbolResolver(nil).Resolve(ctx, "does_not_exist_anywhere")).
			Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("NewConsistentResolver", func() {
	var ch *channel.Channel
	BeforeEach(func(ctx SpecContext) {
		ch = &channel.Channel{
			Name:     "consistent_resolver_ch",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(svc.NewWriter(nil).Create(ctx, ch)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, ch.Key(), false)).To(Succeed())
		})
	})

	It(
		"Should resolve a renamed channel by the name it first had",
		func(ctx SpecContext) {
			r := channel.NewConsistentResolver(svc.NewArcSymbolResolver(nil))
			Expect(r.Resolve(ctx, "consistent_resolver_ch")).
				To(HaveField("ID", int(ch.Key())))
			Expect(svc.NewWriter(nil).Rename(
				ctx, ch.Key(), "consistent_resolver_ch_renamed", false,
			)).To(Succeed())
			Expect(r.Resolve(ctx, "consistent_resolver_ch")).
				To(HaveField("ID", int(ch.Key())))
		},
	)

	It(
		"Should keep reporting a missing name after it is created",
		func(ctx SpecContext) {
			r := channel.NewConsistentResolver(svc.NewArcSymbolResolver(nil))
			Expect(r.Resolve(ctx, "consistent_resolver_later")).Error().
				To(MatchError(query.ErrNotFound))
			later := &channel.Channel{
				Name:     "consistent_resolver_later",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(svc.NewWriter(nil).Create(ctx, later)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, later.Key(), false)).To(Succeed())
			})
			Expect(r.Resolve(ctx, "consistent_resolver_later")).Error().
				To(MatchError(query.ErrNotFound))
			Expect(channel.NewConsistentResolver(svc.NewArcSymbolResolver(nil)).
				Resolve(ctx, "consistent_resolver_later")).
				To(HaveField("ID", int(later.Key())))
		},
	)

	It("Should return a separate symbol for each lookup", func(ctx SpecContext) {
		r := channel.NewConsistentResolver(svc.NewArcSymbolResolver(nil))
		first := MustSucceed(r.Resolve(ctx, "consistent_resolver_ch"))
		first.Name = "changed"
		Expect(r.Resolve(ctx, "consistent_resolver_ch")).
			To(HaveField("Name", "consistent_resolver_ch"))
	})

	It("Should resolve the same name from many goroutines", func(ctx SpecContext) {
		r := channel.NewConsistentResolver(svc.NewArcSymbolResolver(nil))
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				defer GinkgoRecover()
				Expect(r.Resolve(ctx, "consistent_resolver_ch")).
					To(HaveField("ID", int(ch.Key())))
				Expect(r.Resolve(ctx, "consistent_resolver_missing")).Error().
					To(MatchError(query.ErrNotFound))
			})
		}
		wg.Wait()
	})

	It("Should search through the wrapped resolver", func(ctx SpecContext) {
		r := channel.NewConsistentResolver(svc.NewArcSymbolResolver(nil))
		Eventually(func(g Gomega) {
			g.Expect(r.Search(ctx, "consistent_resolver")).
				To(ContainElement(HaveField("ID", int(ch.Key()))))
		}).Should(Succeed())
	})
})
