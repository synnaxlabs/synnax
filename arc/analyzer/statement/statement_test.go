// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/result"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Statement", func() {
	createScope := func() *symbol.Scope {
		counter := 0
		return &symbol.Scope{Counter: &counter}
	}

	Describe("Variable Declaration", func() {
		Describe("Local Variables", func() {
			It("Should analyze a local variable with explicit type", func() {
				stmt := MustSucceed(text.ParseStatement(`x i32 := 42`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Resolve("x")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Type).To(Equal(types.I32{}))
			})

			It("Should infer type from initializer", func() {
				stmt := MustSucceed(text.ParseStatement(`x := 3.14`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Resolve("x")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Type).To(Equal(types.F64{}))
			})

			It("Should detect type mismatch", func() {
				stmt := MustSucceed(text.ParseStatement(`x i32 := "hello"`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("type mismatch: cannot assign string to i32"))
			})

			It("Should detect duplicate variable declaration", func() {
				stmt := MustSucceed(text.ParseBlock(`{
					x := 1
					x := 1
				}`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.AnalyzeBlock(scope, &res, stmt)
				Expect(ok).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("name x conflicts with existing symbol at line 2, col 5"))
			})

			It("Should detect undefined variable in initializer", func() {
				stmt := MustSucceed(text.ParseStatement(`x := y + 1`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: y"))
			})
		})

		Describe("Stateful Variables", func() {
			It("Should analyze a stateful variable", func() {
				stmt := MustSucceed(text.ParseStatement(`counter $= 0`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Resolve("counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Kind).To(Equal(symbol.KindStatefulVariable))
				Expect(sym.Type).To(Equal(types.I64{}))
			})

			It("Should analyze stateful variable with explicit type", func() {
				stmt := MustSucceed(text.ParseStatement(`total f32 $= 0.0`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Resolve("total")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Type).To(Equal(types.F32{}))
			})
		})
	})

	Describe("Assignment", func() {
		It("Should analyze assignment to existing variable", func() {
			stmt := MustSucceed(text.ParseStatement(`x = 42`))
			scope := createScope()
			_, _ = scope.Add(symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I64{}})
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should detect assignment to undefined variable", func() {
			stmt := MustSucceed(text.ParseStatement(`x = 42`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: x"))
		})

		It("Should detect type mismatch in assignment", func() {
			stmt := MustSucceed(text.ParseStatement(`x = "hello"`))
			scope := createScope()
			_, _ = scope.Add(symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32{}})
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			Expect(res.Diagnostics[0].Message).To(ContainSubstring("type mismatch"))
		})
	})

	Describe("If Statement", func() {
		It("Should analyze simple if statement", func() {
			stmt := MustSucceed(text.ParseStatement(`if 1 {
				x := 42
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should analyze if-else chain", func() {
			stmt := MustSucceed(text.ParseStatement(`if 0 {
				x := 1
			} else if 1 {
				y := 2
			} else {
				z := 3
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should detect undefined variable in condition", func() {
			stmt := MustSucceed(text.ParseStatement(`if x > 10 {
				y := 1
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: x"))
		})

		It("Should handle nested blocks with separate scopes", func() {
			stmt := MustSucceed(text.ParseStatement(`if 1 {
				x := 42
				if 1 {
					y := x + 1
				}
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})
	})

	Describe("Block", func() {
		It("Should analyze multiple statements in a block", func() {
			block := MustSucceed(text.ParseBlock(`{
				x := 1
				y := 2
				z := x + y
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.AnalyzeBlock(scope, &res, block)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should maintain variable visibility within block", func() {
			block := MustSucceed(text.ParseBlock(`{
				x := 1
				y := x + 2
				z := x + y
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.AnalyzeBlock(scope, &res, block)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should detect errors in block statements", func() {
			block := MustSucceed(text.ParseBlock(`{
				x := 1
				y := undefined
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.AnalyzeBlock(scope, &res, block)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: undefined"))
		})
	})

	Describe("Expression Statement", func() {
		It("Should analyze standalone expression", func() {
			stmt := MustSucceed(text.ParseStatement(`x + 1`))
			scope := createScope()
			_, _ = scope.Add(symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I64{}})
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should detect errors in standalone expression", func() {
			stmt := MustSucceed(text.ParseStatement(`undefined_var + 1`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
		})
	})

	Describe("Channel Operations", func() {
		var scope *symbol.Scope

		BeforeEach(func() {
			resolver := symbol.MapResolver{
				"sensor":      symbol.Symbol{Kind: symbol.KindChannel, Type: types.Chan{ValueType: types.F64{}}},
				"output":      symbol.Symbol{Kind: symbol.KindChannel, Type: types.Chan{ValueType: types.F64{}}},
				"int_chan":    symbol.Symbol{Kind: symbol.KindChannel, Type: types.Chan{ValueType: types.I32{}}},
				"string_chan": symbol.Symbol{Kind: symbol.KindChannel, Type: types.Chan{ValueType: types.String{}}},
			}
			scope = symbol.CreateRoot(resolver)
		})

		Describe("Channel Writes", func() {
			It("Should analyze basic channel write with arrow", func() {
				stmt := MustSucceed(text.ParseStatement(`42.0 -> output`))
				res := result.Result{Symbols: scope}
				Expect(statement.Analyze(scope, &res, stmt)).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
			})

			It("Should analyze channel write with recv operator", func() {
				stmt := MustSucceed(text.ParseStatement(`output <- 42.0`))
				res := result.Result{Symbols: scope}
				Expect(statement.Analyze(scope, &res, stmt)).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
			})

			It("Should detect type mismatch in channel write", func() {
				stmt := MustSucceed(text.ParseStatement(`"hello" -> output`))
				res := result.Result{Symbols: scope}
				Expect(statement.Analyze(scope, &res, stmt)).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("type mismatch: cannot write string to channel of type f64"))
			})

			It("Should analyze channel write with variable", func() {
				stmt := MustSucceed(text.ParseStatement(`value -> output`))
				_, _ = scope.Add(symbol.Symbol{Name: "value", Kind: symbol.KindVariable, Type: types.F64{}})
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
			})

			It("Should detect undefined channel in write", func() {
				stmt := MustSucceed(text.ParseStatement(`42.0 -> undefined_channel`))
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: undefined_channel"))
			})
		})

		Describe("Channel Reads", func() {
			It("Should analyze blocking channel read", func() {
				stmt := MustSucceed(text.ParseStatement(`value := <-sensor`))
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))

				// Verify the variable has the correct type
				varScope, err := scope.Resolve("value")
				Expect(err).ToNot(HaveOccurred())
				Expect(varScope.Type).To(Equal(types.F64{}))
			})

			It("Should analyze non-blocking channel read", func() {
				stmt := MustSucceed(text.ParseStatement(`current := sensor`))
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))

				// Verify the variable has the correct type
				varScope, err := scope.Resolve("current")
				Expect(err).ToNot(HaveOccurred())
				Expect(varScope.Type).To(Equal(types.F64{}))
			})

			It("Should detect undefined channel in read", func() {
				stmt := MustSucceed(text.ParseStatement(`value := <-undefined_channel`))
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined"))
			})
		})

	})

	Describe("Mixed Type Scenarios", func() {
		It("Should handle complex nested structures", func() {
			stmt := MustSucceed(text.ParseStatement(`if 1 {
				x := 10
				y $= 20
				if x < y {
					z := x + y
					z = z * 2
				}
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should properly track types through assignments", func() {
			block := MustSucceed(text.ParseBlock(`{
				x i32 := 10
				y := x
				z := y + 5
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.AnalyzeBlock(scope, &res, block)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should return an error when a variable of an incorrect type is assigned to another variable", func() {
			block := MustSucceed(text.ParseBlock(`{
				x i32 := 10
				y f32 := x
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			Expect(statement.AnalyzeBlock(scope, &res, block)).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			first := res.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to f32"))
		})
	})
})
