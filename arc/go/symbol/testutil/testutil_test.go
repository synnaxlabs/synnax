// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("StaticResolver", func() {
	Describe("Resolve", func() {
		It("Should return the symbol matching the given name", func(ctx SpecContext) {
			r := StaticResolver{
				{Name: "pi", Kind: symbol.KindInput, Type: types.F64()},
				{Name: "tau", Kind: symbol.KindInput, Type: types.F64()},
			}
			sym := MustSucceed(r.Resolve(ctx, "tau"))
			Expect(sym.Name).To(Equal("tau"))
			Expect(sym.Kind).To(Equal(symbol.KindInput))
		})

		It(
			"Should return query.ErrNotFound when the name is absent",
			func(ctx SpecContext) {
				r := StaticResolver{
					{Name: "pi", Kind: symbol.KindInput, Type: types.F64()},
				}
				Expect(r.Resolve(ctx, "phi")).Error().To(SatisfyAll(
					MatchError(query.ErrNotFound),
					MatchError(ContainSubstring("symbol phi not found")),
				))
			},
		)

		It(
			"Should return query.ErrNotFound for an empty resolver",
			func(ctx SpecContext) {
				r := StaticResolver{}
				Expect(
					r.Resolve(ctx, "anything"),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should return a copy so callers cannot mutate the resolver via the result",
			func(ctx SpecContext) {
				r := StaticResolver{
					{Name: "pi", Kind: symbol.KindInput, Type: types.F64()},
				}
				sym := MustSucceed(r.Resolve(ctx, "pi"))
				sym.Name = "mutated"
				again := MustSucceed(r.Resolve(ctx, "pi"))
				Expect(again.Name).To(Equal("pi"))
			},
		)
	})

	Describe("Search", func() {
		It(
			"Should return entries whose name has the given prefix",
			func(ctx SpecContext) {
				r := StaticResolver{
					{Name: "pi", Kind: symbol.KindInput},
					{Name: "print", Kind: symbol.KindFunction},
					{Name: "tau", Kind: symbol.KindInput},
				}
				results := MustSucceed(r.Search(ctx, "p"))
				Expect(results).To(HaveLen(2))
				names := []string{results[0].Name, results[1].Name}
				Expect(names).To(ConsistOf("pi", "print"))
			},
		)

		It(
			"Should fuzzy-match entries within Levenshtein distance 2 when the term is long enough",
			func(ctx SpecContext) {
				r := StaticResolver{
					{Name: "temperature", Kind: symbol.KindVariable},
					{Name: "humidity", Kind: symbol.KindVariable},
				}
				results := MustSucceed(r.Search(ctx, "temperatur"))
				Expect(results).To(HaveLen(1))
				Expect(results[0].Name).To(Equal("temperature"))
			},
		)

		It(
			"Should skip fuzzy matching when the term is 2 chars or shorter",
			func(ctx SpecContext) {
				r := StaticResolver{{Name: "temperature", Kind: symbol.KindVariable}}
				Expect(MustSucceed(r.Search(ctx, "tx"))).To(BeEmpty())
			},
		)

		It("Should return all entries for an empty term", func(ctx SpecContext) {
			r := StaticResolver{
				{Name: "foo", Kind: symbol.KindVariable},
				{Name: "bar", Kind: symbol.KindVariable},
			}
			Expect(MustSucceed(r.Search(ctx, ""))).To(HaveLen(2))
		})

		It("Should return an empty slice when nothing matches", func(ctx SpecContext) {
			r := StaticResolver{{Name: "foo", Kind: symbol.KindVariable}}
			Expect(MustSucceed(r.Search(ctx, "xyz"))).To(BeEmpty())
		})
	})

	Describe("Add", func() {
		It("Should append a symbol so it becomes resolvable", func(ctx SpecContext) {
			r := StaticResolver{}
			r.Add(
				symbol.Symbol{
					Name: "sensor",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			)
			sym := MustSucceed(r.Resolve(ctx, "sensor"))
			Expect(sym.Kind).To(Equal(symbol.KindChannel))
		})

		It("Should be visible through a captured variable", func(ctx SpecContext) {
			r := StaticResolver{}
			read := func() (*symbol.Symbol, error) { return r.Resolve(ctx, "ch") }
			Expect(read()).Error().To(MatchError(query.ErrNotFound))
			r.Add(
				symbol.Symbol{
					Name: "ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
				},
			)
			sym := MustSucceed(read())
			Expect(sym.Name).To(Equal("ch"))
		})
	})

	Describe("Remove", func() {
		It("Should delete the entry with the given name", func(ctx SpecContext) {
			r := StaticResolver{
				{Name: "a", Kind: symbol.KindVariable},
				{Name: "b", Kind: symbol.KindVariable},
			}
			r.Remove("a")
			Expect(r.Resolve(ctx, "a")).Error().To(MatchError(query.ErrNotFound))
			Expect(MustSucceed(r.Resolve(ctx, "b")).Name).To(Equal("b"))
		})

		It("Should preserve order of the remaining entries", func(ctx SpecContext) {
			r := StaticResolver{
				{Name: "a", Kind: symbol.KindVariable},
				{Name: "b", Kind: symbol.KindVariable},
				{Name: "c", Kind: symbol.KindVariable},
			}
			r.Remove("b")
			results := MustSucceed(r.Search(ctx, ""))
			names := []string{results[0].Name, results[1].Name}
			Expect(names).To(Equal([]string{"a", "c"}))
		})

		It("Should be a no-op when no entry matches", func(_ SpecContext) {
			r := StaticResolver{{Name: "a", Kind: symbol.KindVariable}}
			r.Remove("missing")
			Expect(r).To(HaveLen(1))
		})

		It(
			"Should remove only the first matching entry when duplicates exist",
			func(ctx SpecContext) {
				r := StaticResolver{
					{Name: "dup", Kind: symbol.KindVariable, Type: types.I32()},
					{Name: "dup", Kind: symbol.KindVariable, Type: types.I64()},
				}
				r.Remove("dup")
				Expect(r).To(HaveLen(1))
				Expect(MustSucceed(r.Resolve(ctx, "dup")).Type).To(Equal(types.I64()))
			},
		)
	})
})

var _ = Describe("NewRoot", func() {
	It("Should attach STL symbols to the ambient prelude", func() {
		root := NewRoot(nil)
		Expect(root.Parent).ToNot(BeNil())
		Expect(root.Parent.Kind).To(Equal(symbol.KindAmbient))
		Expect(root.Parent.Children()).ToNot(BeEmpty())
	})

	It("Should attach extras as additional ambient siblings of the root", func() {
		extra := symbol.Symbol{
			Name: "sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
		}
		root := NewRoot(nil, extra)
		Expect(root.Parent.FindChild("sensor")).ToNot(BeNil())
	})

	It(
		"Should consult the given resolver on root.Resolve misses",
		func(bCtx SpecContext) {
			r := StaticResolver{
				{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			root := NewRoot(r)
			Expect(MustSucceed(root.Resolve(bCtx, "ch")).Name).To(Equal("ch"))
		},
	)
})

var _ = Describe("NewGraphRoot", func() {
	It(
		"Should add KindModuleAlias children to the root for each ambient module",
		func() {
			root := NewGraphRoot(nil)
			hasAnyAlias := false
			for _, child := range root.Children() {
				if child.Kind == symbol.KindModuleAlias {
					hasAnyAlias = true
					break
				}
			}
			Expect(hasAnyAlias).To(BeTrue())
		},
	)
})

var _ = Describe("FirstChildOfKind", func() {
	It("Should return the first child with the matching Kind", func(bCtx SpecContext) {
		root := NewRoot(nil)
		fnSym := MustSucceed(
			root.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}),
		)
		MustSucceed(
			root.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			),
		)
		Expect(
			MustSucceed(FirstChildOfKind(root, symbol.KindFunction)),
		).To(Equal(fnSym))
	})

	It("Should return query.ErrNotFound when no child matches", func(bCtx SpecContext) {
		root := NewRoot(nil)
		MustSucceed(
			root.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			),
		)
		Expect(
			FirstChildOfKind(root, symbol.KindFunction),
		).Error().
			To(MatchError(query.ErrNotFound))
	})

	It(
		"Should return the first match in insertion order when multiple children share the Kind",
		func(bCtx SpecContext) {
			root := NewRoot(nil)
			first := MustSucceed(
				root.Add(bCtx, symbol.Symbol{Name: "f1", Kind: symbol.KindFunction}),
			)
			MustSucceed(
				root.Add(bCtx, symbol.Symbol{Name: "f2", Kind: symbol.KindFunction}),
			)
			Expect(
				MustSucceed(FirstChildOfKind(root, symbol.KindFunction)),
			).To(Equal(first))
		},
	)
})
