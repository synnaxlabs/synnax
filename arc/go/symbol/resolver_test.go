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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MapResolver", func() {
	Describe("Resolve", func() {
		It("Should resolve existing symbol", func() {
			resolver := symbol.MapResolver{
				"pi":    symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
				"count": symbol.Symbol{Name: "count", Kind: symbol.KindVariable, Type: types.I32()},
			}
			sym := MustSucceed(resolver.Resolve(bCtx, "pi"))
			Expect(sym.Name).To(Equal("pi"))
			Expect(sym.Kind).To(Equal(symbol.KindConfig))
			Expect(sym.Type).To(Equal(types.F64()))
		})

		It("Should return error for non-existent symbol", func() {
			resolver := symbol.MapResolver{
				"x": symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			}
			Expect(resolver.Resolve(bCtx, "y")).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should work with empty resolver", func() {
			resolver := symbol.MapResolver{}
			Expect(resolver.Resolve(bCtx, "anything")).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Search", func() {
		It("Should resolve all symbols matching prefix", func() {
			resolver := symbol.MapResolver{
				"pi":      symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
				"count":   symbol.Symbol{Name: "count", Kind: symbol.KindVariable, Type: types.I32()},
				"counter": symbol.Symbol{Name: "counter", Kind: symbol.KindVariable, Type: types.I32()},
				"max":     symbol.Symbol{Name: "max", Kind: symbol.KindFunction, Type: types.F64()},
			}
			symbols := MustSucceed(resolver.Search(bCtx, "count"))
			Expect(symbols).To(HaveLen(2))

			names := []string{symbols[0].Name, symbols[1].Name}
			Expect(names).To(ContainElements("count", "counter"))
		})

		It("Should return empty slice for non-matching search", func() {
			resolver := symbol.MapResolver{
				"pi":    symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
				"count": symbol.Symbol{Name: "count", Kind: symbol.KindVariable, Type: types.I32()},
			}
			symbols := MustSucceed(resolver.Search(bCtx, "completely_different_name"))
			Expect(symbols).To(BeEmpty())
		})

		It("Should return all symbols for empty prefix", func() {
			resolver := symbol.MapResolver{
				"pi":    symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
				"count": symbol.Symbol{Name: "count", Kind: symbol.KindVariable, Type: types.I32()},
			}
			symbols := MustSucceed(resolver.Search(bCtx, ""))
			Expect(symbols).To(HaveLen(2))
		})

		It("Should work with empty resolver", func() {
			resolver := symbol.MapResolver{}
			symbols := MustSucceed(resolver.Search(bCtx, "anything"))
			Expect(symbols).To(BeEmpty())
		})
	})
})

var _ = Describe("CompoundResolver", func() {
	Describe("Resolve", func() {
		It("Should resolve from first matching resolver", func() {
			resolver1 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()},
			}
			resolver2 := symbol.MapResolver{
				"bar": symbol.Symbol{Name: "bar", Kind: symbol.KindVariable, Type: types.String()},
			}
			compound := symbol.CompoundResolver{resolver1, resolver2}
			sym := MustSucceed(compound.Resolve(bCtx, "bar"))
			Expect(sym.Name).To(Equal("bar"))
			Expect(sym.Type).To(Equal(types.String()))
		})
		It("Should prioritize first resolver when multiple match", func() {
			resolver1 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()},
			}
			resolver2 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.String()},
			}
			compound := symbol.CompoundResolver{resolver1, resolver2}
			sym := MustSucceed(compound.Resolve(bCtx, "foo"))
			Expect(sym.Type).To(Equal(types.I32()))
		})
		It("Should return error when no resolver matches", func() {
			resolver1 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()},
			}
			compound := symbol.CompoundResolver{resolver1}
			_, err := compound.Resolve(bCtx, "nonexistent")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Search", func() {
		It("Should resolve from all sub-resolvers", func() {
			resolver1 := symbol.MapResolver{
				"foo":    symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()},
				"foobar": symbol.Symbol{Name: "foobar", Kind: symbol.KindVariable, Type: types.I32()},
			}
			resolver2 := symbol.MapResolver{
				"food": symbol.Symbol{Name: "food", Kind: symbol.KindVariable, Type: types.String()},
			}
			compound := symbol.CompoundResolver{resolver1, resolver2}

			symbols := MustSucceed(compound.Search(bCtx, "foo"))
			Expect(symbols).To(HaveLen(3))

			names := []string{symbols[0].Name, symbols[1].Name, symbols[2].Name}
			Expect(names).To(ContainElements("foo", "foobar", "food"))
		})

		It("Should deduplicate symbols by name (first wins)", func() {
			resolver1 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()},
			}
			resolver2 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.String()},
			}
			compound := symbol.CompoundResolver{resolver1, resolver2}

			symbols := MustSucceed(compound.Search(bCtx, "foo"))
			Expect(symbols).To(HaveLen(1))
			Expect(symbols[0].Type).To(Equal(types.I32())) // First resolver wins
		})

		It("Should return empty slice when no resolvers match", func() {
			resolver1 := symbol.MapResolver{
				"foo": symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()},
			}
			compound := symbol.CompoundResolver{resolver1}

			symbols := MustSucceed(compound.Search(bCtx, "completely_different_name"))
			Expect(symbols).To(BeEmpty())
		})
	})
})
