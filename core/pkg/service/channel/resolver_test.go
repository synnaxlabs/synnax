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
		Expect(svc.Create(ctx, ch)).To(Succeed())

		sym := MustSucceed(svc.NewArcSymbolResolver(nil).Resolve(ctx, "resolver_test_ch"))
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
		Expect(svc.Create(ctx, ch)).To(Succeed())

		sym := MustSucceed(svc.NewArcSymbolResolver(nil).Resolve(ctx, "resolver_internal_ch"))
		Expect(sym.Renameable).To(BeFalse())
	})

	It("Should resolve a channel by numeric key", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "resolver_key_test_ch",
			Virtual:  true,
			DataType: telem.Int64T,
		}
		Expect(svc.Create(ctx, ch)).To(Succeed())

		sym := MustSucceed(svc.NewArcSymbolResolver(nil).Resolve(ctx, strconv.Itoa(int(ch.Key()))))
		Expect(sym.Name).To(Equal("resolver_key_test_ch"))
		Expect(sym.Kind).To(Equal(symbol.KindChannel))
		Expect(sym.Type).To(Equal(types.Chan(types.I64())))
	})

	It("Should return an error for a nonexistent symbol", func(ctx SpecContext) {
		Expect(svc.NewArcSymbolResolver(nil).Resolve(ctx, "does_not_exist_anywhere")).
			Error().To(MatchError(query.ErrNotFound))
	})
})
