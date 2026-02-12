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
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

type searchTracker struct {
	inner        symbol.Resolver
	searchCalled bool
}

func (s *searchTracker) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return s.inner.Resolve(ctx, name)
}

func (s *searchTracker) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	s.searchCalled = true
	return s.inner.Search(ctx, term)
}

func resolveErr(scope *symbol.Scope, name string) error {
	_, err := scope.Resolve(bCtx, name)
	return err
}

var _ = Describe("Symbol Suggestions", func() {
	Describe("LevenshteinDistance", func() {
		DescribeTable("should calculate correct edit distance",
			func(a, b string, expected int) {
				Expect(compare.LevenshteinDistance(a, b)).To(Equal(expected))
			},
			Entry("identical strings", "hello", "hello", 0),
			Entry("empty strings", "", "", 0),
			Entry("first empty", "", "abc", 3),
			Entry("second empty", "abc", "", 3),
			Entry("single substitution", "cat", "bat", 1),
			Entry("single insertion", "cat", "cats", 1),
			Entry("single deletion", "cats", "cat", 1),
			Entry("multiple edits", "kitten", "sitting", 3),
			Entry("completely different", "abc", "xyz", 3),
			Entry("typo - missing letter", "temperatur", "temperature", 1),
			Entry("typo - extra letter", "temperaturee", "temperature", 1),
			Entry("typo - swapped letters", "teh", "the", 2),
			Entry("case sensitive", "Hello", "hello", 1),
		)
	})

	Describe("SuggestSimilar", func() {
		It("should suggest similar symbol names", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "temperature", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "pressure", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "humidity", Kind: symbol.KindVariable}))

			suggestions := root.SuggestSimilar(bCtx, "temperatur", 2)
			Expect(suggestions).To(ContainElement("temperature"))
		})

		It("should return empty slice when no similar symbols exist", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable}))

			suggestions := root.SuggestSimilar(bCtx, "temperature", 2)
			Expect(suggestions).To(BeEmpty())
		})

		It("should respect maxSuggestions limit", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "cat", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "bat", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "rat", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "hat", Kind: symbol.KindVariable}))

			suggestions := root.SuggestSimilar(bCtx, "mat", 2)
			Expect(suggestions).To(HaveLen(2))
		})

		It("should search parent scopes", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "globalVar", Kind: symbol.KindVariable}))

			child := MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "block", Kind: symbol.KindBlock}))

			suggestions := child.SuggestSimilar(bCtx, "globalVa", 2)
			Expect(suggestions).To(ContainElement("globalVar"))
		})

		It("should sort suggestions by distance", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "test", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "tests", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "testing", Kind: symbol.KindVariable}))

			suggestions := root.SuggestSimilar(bCtx, "tset", 3)
			Expect(len(suggestions)).To(BeNumerically(">=", 1))
			Expect(suggestions[0]).To(Equal("test"))
		})

		It("should not include exact matches", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "temperature", Kind: symbol.KindVariable}))

			suggestions := root.SuggestSimilar(bCtx, "temperature", 2)
			Expect(suggestions).NotTo(ContainElement("temperature"))
		})
	})

	Describe("Resolve with suggestions", func() {
		It("should return UndefinedSymbolError with lazy hint", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "temperature", Kind: symbol.KindVariable}))

			Expect(root.Resolve(bCtx, "temperatur")).Error().
				To(MatchError(ContainSubstring("undefined symbol: temperatur")))

			var undefinedErr *symbol.UndefinedSymbolError
			Expect(errors.As(resolveErr(root, "temperatur"), &undefinedErr)).To(BeTrue())
			Expect(undefinedErr.GetHint()).To(ContainSubstring("did you mean: temperature?"))
		})

		It("should not include suggestions when none are close enough", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable}))

			Expect(root.Resolve(bCtx, "unknownSymbol")).Error().
				To(MatchError("undefined symbol: unknownSymbol"))

			var undefinedErr *symbol.UndefinedSymbolError
			Expect(errors.As(resolveErr(root, "unknownSymbol"), &undefinedErr)).To(BeTrue())
			Expect(undefinedErr.GetHint()).To(BeEmpty())
		})
	})

	Describe("Lazy suggestion performance", func() {
		It("should not trigger Search on global resolver during Add", func() {
			tracker := &searchTracker{
				inner: symbol.MapResolver{
					"builtin_fn": {Name: "builtin_fn", Kind: symbol.KindFunction},
				},
			}
			root := symbol.CreateRootScope(tracker)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "alpha", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "beta", Kind: symbol.KindVariable}))
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "gamma", Kind: symbol.KindVariable}))
			Expect(tracker.searchCalled).To(BeFalse())
		})

		It("should provide suggestions via diagnostics.Error with HintProvider", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "temperature", Kind: symbol.KindVariable}))

			d := diagnostics.Error(resolveErr(root, "temperatur"), nil)
			Expect(d.Message).To(Equal("undefined symbol: temperatur"))
			Expect(d.Notes).To(HaveLen(1))
			Expect(d.Notes[0].Message).To(ContainSubstring("did you mean: temperature?"))
		})

		It("should not add notes via diagnostics.Error when no suggestions exist", func() {
			root := symbol.CreateRootScope(nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable}))

			d := diagnostics.Error(resolveErr(root, "unknownSymbol"), nil)
			Expect(d.Message).To(Equal("undefined symbol: unknownSymbol"))
			Expect(d.Notes).To(BeEmpty())
		})
	})
})
