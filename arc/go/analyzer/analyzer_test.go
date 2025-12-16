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

var _ = Describe("Analyzer", func() {
	Describe("Duplicate Scope", func() {
		It("Should correctly diagnose a variable declaration that shadows a function", func() {
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
	})

	Describe("Declaration", func() {
		Describe("Local", func() {
			It("Should return an error diagnostic when a string is declared on an i32", func() {
				prog := MustSucceed(parser.Parse(`
					func cat() {
						my_var i32 := "dog"
					}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign str to i32"))
			})

			It("Should allow compatible types in local variable declaration", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42
				}
				`))
				ctx := context.CreateRoot(bCtx, ast, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			})

			It("Should infer types from an int literal", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
				Expect(varScope.ID).To(Equal(0))
				Expect(varScope.Name).To(Equal("x"))
				// Integer literals default to i64 after unification
				Expect(varScope.Type).To(Equal(types.I64()))
			})

			It("Should infer types from a float literal", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42.0
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
				Expect(varScope.ID).To(Equal(0))
				Expect(varScope.Name).To(Equal("x"))
				Expect(varScope.Type).To(Equal(types.F64()))
			})

			It("Should automatically cast an int literal to a floating point type", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 42
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
				Expect(varScope.ID).To(Equal(0))
				Expect(varScope.Name).To(Equal("x"))
				Expect(varScope.Type).To(Equal(types.F32()))
			})

			It("Should not allow assignment of a float literal to an int type", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42.0
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
				Expect(first.Message).To(ContainSubstring("does not satisfy float constraint"))
			})

			It("Should allow for variable declaration from a function parameter", func() {
				prog := MustSucceed(parser.Parse(`
					func testFunc(a i64) {
						b := a
					}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			})
		})
	})

	Describe("Assignment", func() {
		It("Should return an error diagnostic when the variable being assigned to was not declared", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat str := "abc"
					bob = cat
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
			Expect(first.Line).To(Equal(5))
			Expect(first.Column).To(Equal(5))
		})

		It("Should return an error diagnostic when the variable on the right hand side is not declared", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat str := "abc"
					cat = bob
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
		})

		It("Should return an error when assignment is attempted between incompatible types", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					v1 i32 := 1
					v2 str := "abc"
					v2 = v1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to variable of type str"))
		})
	})

	Describe("Control Flow", func() {
		It("Should return no diagnostics for a valid if statement", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					a f32 := 2.0
					if (a > 5) {
						return 1
					}
					return 2
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

		})

		It("Should return the correct symbol table for an if-else statement", func() {
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
			Expect(funcScope.ID).To(Equal(0))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(2))
			Expect(blocks[0].Children).To(BeEmpty())
			Expect(blocks[1].Children).To(BeEmpty())
		})

		It("Should return the correct symbol table for variables declared inside blocks", func() {
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
			Expect(funcScope.ID).To(Equal(0))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(1))
			firstBlock := blocks[0]
			Expect(firstBlock.Children).To(HaveLen(1))
			firstChild := firstBlock.Children[0]
			Expect(firstChild).ToNot(BeNil())
			Expect(firstChild.Name).To(Equal("b"))
		})

		It("Should return the correct symbol table for variables declared in blocks", func() {
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
			Expect(funcScope.ID).To(Equal(0))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(3))
			firstBlock := blocks[0]
			Expect(firstBlock.Children).To(BeEmpty())
			secondBlock := blocks[1]
			Expect(secondBlock.Children).To(HaveLen(1))
			secondBlockFirstChild := secondBlock.Children[0]
			Expect(secondBlockFirstChild.Name).To(Equal("c"))
			thirdBlock := blocks[2]
			Expect(thirdBlock.Children).To(BeEmpty())
		})
	})
})
