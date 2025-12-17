// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Analyzer Integration", func() {

	Describe("Cross-Scope Symbol Resolution", func() {
		It("Should diagnose a variable declaration that shadows a function name", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					dog := 1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
		})

		It("Should allow variable declaration from a function parameter", func() {
			prog := MustSucceed(parser.Parse(`
				func testFunc(a i64) {
					b := a
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
		})

		It("Should resolve variables across nested scopes", func() {
			prog := MustSucceed(parser.Parse(`
				func outer() i64 {
					x i64 := 10
					if x > 5 {
						y := x + 1
						return y
					}
					return x
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
		})
	})

	Describe("Type Unification Results", func() {
		It("Should resolve integer literal to i64 after unification", func() {
			prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.I64()))
		})

		It("Should resolve float literal to f64 after unification", func() {
			prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42.0
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.F64()))
		})

		It("Should unify integer literal with explicit f32 type", func() {
			prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 42
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.F32()))
		})
	})

	Describe("Symbol Table Structure", func() {
		It("Should build correct symbol table for if-else statement", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					if 3 > 5 {
						return 1
					} else {
						return 2
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(2))
			Expect(blocks[0].Children).To(BeEmpty())
			Expect(blocks[1].Children).To(BeEmpty())
		})

		It("Should build correct symbol table for variables in nested blocks", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					a f32 := 2.0
					if (a > 5) {
						b := 2
						return b
					}
					return 2
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(1))
			Expect(blocks[0].Children).To(HaveLen(1))
			Expect(blocks[0].Children[0].Name).To(Equal("b"))
		})

		It("Should build correct symbol table for if-else-if chain", func() {
			prog := MustSucceed(parser.Parse(`
				func dog(b i64) i64 {
					a i64 := 2
					if b == a {
						return 2
					} else if a > b {
						c i64 := 5
						return c
					} else {
						return 3
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(3))
			Expect(blocks[0].Children).To(BeEmpty())
			Expect(blocks[1].Children).To(HaveLen(1))
			Expect(blocks[1].Children[0].Name).To(Equal("c"))
			Expect(blocks[2].Children).To(BeEmpty())
		})
	})

	Describe("Multi-Function Programs", func() {
		It("Should analyze program with multiple functions", func() {
			prog := MustSucceed(parser.Parse(`
				func add(x i64, y i64) i64 {
					return x + y
				}

				func multiply(a i64, b i64) i64 {
					return a * b
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			addFunc := MustSucceed(ctx.Scope.Resolve(ctx, "add"))
			Expect(addFunc.Name).To(Equal("add"))
			mulFunc := MustSucceed(ctx.Scope.Resolve(ctx, "multiply"))
			Expect(mulFunc.Name).To(Equal("multiply"))
		})

		It("Should detect undefined function in call expression", func() {
			prog := MustSucceed(parser.Parse(`
				func caller() i64 {
					return unknownFunc(5)
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: unknownFunc"))
		})
	})
})
