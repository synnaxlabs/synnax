// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcsymbol "github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("DefaultResolverModules", func() {
	It("Should return a non-empty list of modules", func() {
		modules := symbol.DefaultResolverModules()
		Expect(modules).ToNot(BeEmpty())
		Expect(len(modules)).To(Equal(14))
	})
})

var _ = Describe("CreateResolver", func() {
	It("Should resolve an STL symbol", func() {
		resolver := symbol.CreateResolver(dist.Channel)
		sym, err := resolver.Resolve(ctx, "set_status")
		Expect(err).ToNot(HaveOccurred())
		Expect(sym.Name).To(Equal("set_status"))
		Expect(sym.Kind).To(Equal(arcsymbol.KindFunction))
	})

	It("Should resolve a channel by name", func() {
		ch := &channel.Channel{
			Name:     "resolver_test_ch",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

		resolver := symbol.CreateResolver(dist.Channel)
		sym, err := resolver.Resolve(ctx, "resolver_test_ch")
		Expect(err).ToNot(HaveOccurred())
		Expect(sym.Name).To(Equal("resolver_test_ch"))
		Expect(sym.Kind).To(Equal(arcsymbol.KindChannel))
		Expect(sym.Type).To(Equal(types.Chan(types.F32())))
		Expect(sym.ID).To(Equal(int(ch.Key())))
	})

	It("Should resolve a channel by numeric key", func() {
		ch := &channel.Channel{
			Name:     "resolver_key_test_ch",
			Virtual:  true,
			DataType: telem.Int64T,
		}
		Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

		resolver := symbol.CreateResolver(dist.Channel)
		sym, err := resolver.Resolve(ctx, strconv.Itoa(int(ch.Key())))
		Expect(err).ToNot(HaveOccurred())
		Expect(sym.Name).To(Equal("resolver_key_test_ch"))
		Expect(sym.Kind).To(Equal(arcsymbol.KindChannel))
		Expect(sym.Type).To(Equal(types.Chan(types.I64())))
	})

	It("Should return an error for a nonexistent symbol", func() {
		resolver := symbol.CreateResolver(dist.Channel)
		_, err := resolver.Resolve(ctx, "does_not_exist_anywhere")
		Expect(err).To(MatchError(query.ErrNotFound))
	})

	It("Should use custom modules when provided", func() {
		resolver := symbol.CreateResolver(dist.Channel, symbol.DefaultResolverModules()...)
		sym, err := resolver.Resolve(ctx, "set_status")
		Expect(err).ToNot(HaveOccurred())
		Expect(sym.Name).To(Equal("set_status"))
	})
})
