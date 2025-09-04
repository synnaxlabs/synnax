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
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/analyzer/statement"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
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
				stmt := MustSucceed(parser.ParseStatement(`x i32 := 42`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Get("x")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Symbol.Type).To(Equal(types.I32{}))
			})

			It("Should infer type from initializer", func() {
				stmt := MustSucceed(parser.ParseStatement(`x := 3.14`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Get("x")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Symbol.Type).To(Equal(types.F64{}))
			})

			It("Should detect type mismatch", func() {
				stmt := MustSucceed(parser.ParseStatement(`x i32 := "hello"`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeFalse())
				Expect(res.Diagnostics).To(HaveLen(1))
				Expect(res.Diagnostics[0].Message).To(ContainSubstring("type mismatch: cannot assign string to i32"))
			})

			It("Should detect duplicate variable declaration", func() {
				stmt := MustSucceed(parser.ParseBlock(`{
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
				stmt := MustSucceed(parser.ParseStatement(`x := y + 1`))
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
				stmt := MustSucceed(parser.ParseStatement(`counter $= 0`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Get("counter")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Symbol.Kind).To(Equal(symbol.KindStatefulVariable))
				Expect(sym.Symbol.Type).To(Equal(types.I64{}))
			})

			It("Should analyze stateful variable with explicit type", func() {
				stmt := MustSucceed(parser.ParseStatement(`total f32 $= 0.0`))
				scope := createScope()
				res := result.Result{Symbols: scope}
				ok := statement.Analyze(scope, &res, stmt)
				Expect(ok).To(BeTrue())
				Expect(res.Diagnostics).To(HaveLen(0))
				sym, err := scope.Get("total")
				Expect(err).ToNot(HaveOccurred())
				Expect(sym.Symbol.Type).To(Equal(types.F32{}))
			})
		})
	})

	Describe("Assignment", func() {
		It("Should analyze assignment to existing variable", func() {
			stmt := MustSucceed(parser.ParseStatement(`x = 42`))
			scope := createScope()
			_, _ = scope.AddSymbol("x", symbol.KindVariable, types.I64{}, nil)
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should detect assignment to undefined variable", func() {
			stmt := MustSucceed(parser.ParseStatement(`x = 42`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			Expect(res.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: x"))
		})

		It("Should detect type mismatch in assignment", func() {
			stmt := MustSucceed(parser.ParseStatement(`x = "hello"`))
			scope := createScope()
			_, _ = scope.AddSymbol("x", symbol.KindVariable, types.I32{}, nil)
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
			Expect(res.Diagnostics[0].Message).To(ContainSubstring("type mismatch"))
		})
	})

	Describe("If Statement", func() {
		It("Should analyze simple if statement", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 1 {
				x := 42
			}`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should analyze if-else chain", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 0 {
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
			stmt := MustSucceed(parser.ParseStatement(`if x > 10 {
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
			stmt := MustSucceed(parser.ParseStatement(`if 1 {
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
			block := MustSucceed(parser.ParseBlock(`{
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
			block := MustSucceed(parser.ParseBlock(`{
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
			block := MustSucceed(parser.ParseBlock(`{
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
			stmt := MustSucceed(parser.ParseStatement(`x + 1`))
			scope := createScope()
			_, _ = scope.AddSymbol("x", symbol.KindVariable, types.I64{}, nil)
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeTrue())
			Expect(res.Diagnostics).To(HaveLen(0))
		})

		It("Should detect errors in standalone expression", func() {
			stmt := MustSucceed(parser.ParseStatement(`undefined_var + 1`))
			scope := createScope()
			res := result.Result{Symbols: scope}
			ok := statement.Analyze(scope, &res, stmt)
			Expect(ok).To(BeFalse())
			Expect(res.Diagnostics).To(HaveLen(1))
		})
	})

	Describe("Mixed Type Scenarios", func() {
		It("Should handle complex nested structures", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 1 {
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
			block := MustSucceed(parser.ParseBlock(`{
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
			block := MustSucceed(parser.ParseBlock(`{
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
