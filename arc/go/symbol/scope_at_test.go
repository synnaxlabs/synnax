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
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/x/testutil"
)

// blocksOf returns each block of tree, outermost first.
func blocksOf(tree antlr.Tree) []parser.IBlockContext {
	var out []parser.IBlockContext
	if b, ok := tree.(parser.IBlockContext); ok {
		out = append(out, b)
	}
	for _, child := range tree.GetChildren() {
		out = append(out, blocksOf(child)...)
	}
	return out
}

var _ = Describe("ScopeAt", func() {
	var outer, inner parser.IBlockContext
	BeforeEach(func() {
		block := MustSucceed(parser.ParseBlock(
			"{\n\tx := 1\n\tif x > 0 {\n\t\ty := 2\n\t}\n}",
		))
		blocks := blocksOf(block)
		Expect(blocks).To(HaveLen(2))
		outer, inner = blocks[0], blocks[1]
	})
	// scopes returns a root that holds a function over the outer block, which holds a
	// block over the inner one.
	scopes := func(ctx SpecContext) (root, fn, block *symbol.Symbol) {
		GinkgoHelper()
		root = symbol.NewRoot(nil, nil)
		fn = MustSucceed(root.Add(ctx, symbol.Symbol{
			Name: "f",
			Kind: symbol.KindFunction,
			AST:  outer,
		}))
		block = MustSucceed(fn.Add(ctx, symbol.Symbol{
			Name: "if",
			Kind: symbol.KindBlock,
			AST:  inner,
		}))
		return root, fn, block
	}

	It("Should return the deepest scope whose source holds the position", func(
		ctx SpecContext,
	) {
		root, _, block := scopes(ctx)
		Expect(symbol.ScopeAt(root, 4, 2)).To(BeIdenticalTo(block))
	})

	It("Should return the scope around a position outside its inner scopes", func(
		ctx SpecContext,
	) {
		root, fn, _ := scopes(ctx)
		Expect(symbol.ScopeAt(root, 2, 1)).To(BeIdenticalTo(fn))
	})

	It("Should return the root for a position that no scope holds", func(
		ctx SpecContext,
	) {
		root, _, _ := scopes(ctx)
		Expect(symbol.ScopeAt(root, 9, 0)).To(BeIdenticalTo(root))
	})

	It("Should return nil for a nil root", func() {
		Expect(symbol.ScopeAt(nil, 1, 0)).To(BeNil())
	})

	It("Should skip an internal scope", func(ctx SpecContext) {
		root := symbol.NewRoot(nil, nil)
		MustSucceed(root.Add(ctx, symbol.Symbol{
			Name:     "f",
			Kind:     symbol.KindFunction,
			AST:      outer,
			Internal: true,
		}))
		Expect(symbol.ScopeAt(root, 2, 1)).To(BeIdenticalTo(root))
	})

	It("Should skip the members of an imported module", func() {
		member := &symbol.Symbol{Name: "f", Kind: symbol.KindFunction, AST: outer}
		module := (&symbol.Symbol{Name: "m", Kind: symbol.KindModule}).AddChild(member)
		root := symbol.NewRoot(nil, nil)
		root.AddChild(&symbol.Symbol{
			Name:   "m",
			Kind:   symbol.KindModuleAlias,
			Target: module,
		})
		Expect(symbol.ScopeAt(root, 2, 1)).To(BeIdenticalTo(root))
	})
})
