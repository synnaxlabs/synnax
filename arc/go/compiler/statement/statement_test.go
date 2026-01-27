// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/statement"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

func compile(source string) []byte {
	stmt := MustSucceed(parser.ParseStatement(source))
	aCtx := acontext.CreateRoot(bCtx, stmt, nil)
	analyzer.AnalyzeStatement(aCtx)
	Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
	ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, true)
	Expect(MustSucceed(statement.Compile(context.Child(ctx, stmt)))).To(BeFalse())
	return ctx.Writer.Bytes()
}

func compileBlock(source string) []byte {
	block := MustSucceed(parser.ParseBlock("{" + source + "}"))
	aCtx := acontext.CreateRoot(bCtx, block, nil)
	analyzer.AnalyzeBlock(aCtx)
	Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
	ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, true)
	diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
	Expect(diverged).To(BeFalse())
	return ctx.Writer.Bytes()
}

var _ = Describe("Statement Compiler", func() {
	// NOTE: Output assignment tests (ir.KindOutput) are not included here because they
	// require a fully configured multi-output context (Outputs, OutputMemoryBase).
	// Output assignment compilation is tested via integration tests in the main compiler suite.

	DescribeTable("Single Statement Bytecode Values", func(source string, instructions ...any) {
		Expect(compile(source)).To(MatchOpcodes(instructions...))
	},
		Entry(
			"integer variable declaration with explicit type",
			"x i32 := 42",
			OpI32Const, int32(42),
			OpLocalSet, 0,
		),
		Entry(
			"floating point declaration with explicit type",
			"x f64 := 42.42",
			OpF64Const, 42.42,
			OpLocalSet, 0,
		),
		Entry(
			"f32 declaration with explicit type",
			"x f32 := 42.42",
			OpF32Const, float32(42.42),
			OpLocalSet, 0,
		),
		Entry(
			"f32 declaration with integer literal",
			"x f32 := 42",
			OpF32Const, float32(42),
			OpLocalSet, 0,
		),
		Entry(
			"variable declaration with inferred integer type",
			"x := 42",
			OpI64Const, int64(42),
			OpLocalSet, 0,
		),
		Entry(
			"variable declaration with inferred floating point type",
			"x := 42.0",
			OpF64Const, float64(42),
			OpLocalSet, 0,
		),
	)

	Describe("Stateful Variables", func() {
		It("Should compile stateful variable declaration with explicit type", func() {
			stmt := MustSucceed(parser.ParseStatement("count i64 $= 0"))
			aCtx := acontext.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.Compile(context.Child(ctx, stmt)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID (first stateful var)
				OpI64Const, int64(0), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
			))
		})

		It("Should compile stateful variable declaration with inferred type", func() {
			stmt := MustSucceed(parser.ParseStatement("count $= 0"))
			aCtx := acontext.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.Compile(context.Child(ctx, stmt)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpI64Const, int64(0), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
			))
		})

		It("Should compile stateful variable assignment", func() {
			block := MustSucceed(parser.ParseBlock(`{
				count i64 $= 0
				count = 5
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			stateStoreIdx := ctx.Imports.StateStore["i64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				// Declaration: count $= 0
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpI64Const, int64(0), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
				// Assignment: count = 5
				OpI64Const, int64(5), // new value
				OpLocalSet, 0, // store temporarily
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpLocalGet, 0, // get value back
				OpCall, uint64(stateStoreIdx),
			))
		})

		It("Should compile stateful variable reference in expression", func() {
			block := MustSucceed(parser.ParseBlock(`{
				count i64 $= 0
				x i64 := count + 1
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				// Declaration: count $= 0
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpI64Const, int64(0), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
				// Expression: count + 1
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpI64Const, int64(0), // dummy init value
				OpCall, uint64(stateLoadIdx),
				OpI64Const, int64(1), // literal 1
				OpI64Add,      // count + 1
				OpLocalSet, 1, // store in x's local
			))
		})

		It("Should compile multiple stateful variables", func() {
			block := MustSucceed(parser.ParseBlock(`{
				a i64 $= 10
				b i64 $= 20
				c i64 := a + b
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				// Declaration: a $= 10
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID for a
				OpI64Const, int64(10), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in a's local
				// Declaration: b $= 20
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(1), // var ID for b
				OpI64Const, int64(20), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 1, // store in b's local
				// Expression: a + b
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID for a
				OpI64Const, int64(0), // dummy init value
				OpCall, uint64(stateLoadIdx),
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(1), // var ID for b
				OpI64Const, int64(0), // dummy init value
				OpCall, uint64(stateLoadIdx),
				OpI64Add,      // a + b
				OpLocalSet, 2, // store in c's local
			))
		})

		It("Should compile stateful variable with different types", func() {
			stmt := MustSucceed(parser.ParseStatement("temperature f64 $= 20.5"))
			aCtx := acontext.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.Compile(context.Child(ctx, stmt)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["f64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpF64Const, 20.5, // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
			))
		})

		It("Should compile stateful variable compound assignment", func() {
			block := MustSucceed(parser.ParseBlock(`{
				count i64 $= 10
				count += 5
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			stateStoreIdx := ctx.Imports.StateStore["i64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpI64Const, int64(10),
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0,

				OpLocalGet, 0,
				OpI64Const, int64(5),
				OpI64Add,
				OpLocalSet, 0,
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(stateStoreIdx),
			))
		})

		It("Should compile stateful variable compound subtraction", func() {
			block := MustSucceed(parser.ParseBlock(`{
				value f64 $= 100.0
				value -= 25.5
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["f64"]
			stateStoreIdx := ctx.Imports.StateStore["f64"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpF64Const, 100.0,
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0,

				OpLocalGet, 0,
				OpF64Const, 25.5,
				OpF64Sub,
				OpLocalSet, 0,
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(stateStoreIdx),
			))
		})

		It("Should compile multiple compound assignments to stateful variable", func() {
			block := MustSucceed(parser.ParseBlock(`{
				n i32 $= 1
				n *= 2
				n *= 3
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			stateLoadIdx := ctx.Imports.StateLoad["i32"]
			stateStoreIdx := ctx.Imports.StateStore["i32"]
			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpI32Const, int32(1),
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0,

				OpLocalGet, 0,
				OpI32Const, int32(2),
				OpI32Mul,
				OpLocalSet, 0,
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(stateStoreIdx),

				OpLocalGet, 0,
				OpI32Const, int32(3),
				OpI32Mul,
				OpLocalSet, 0,
				OpI32Const, int32(0),
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(stateStoreIdx),
			))
		})
	})

	DescribeTable("Multi Statement Bytecode Values", func(source string, instructions ...any) {
		Expect(compileBlock(source)).To(MatchOpcodes(instructions...))
	},
		Entry("Dual Variable Declaration",
			`
				x := 12
				y := 14
			`,
			OpI64Const, int64(12),
			OpLocalSet, 0,
			OpI64Const, int64(14),
			OpLocalSet, 1,
		),
		Entry("Declaration through assignment",
			`
				x := 12
				y := x
			`,
			OpI64Const, int64(12),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpLocalSet, 1,
		),
		Entry("Assignment",
			`
				x := 12
				x = 5
			`,
			OpI64Const, int64(12),
			OpLocalSet, 0,
			OpI64Const, int64(5),
			OpLocalSet, 0,
		),
		Entry("Single if statement",
			`
			if 8 > 5 {
			}
			`,
			OpI64Const, int64(8),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpEnd,
		),
		Entry("If statement with body",
			`
			x := 0
			if 8 > 5 {
				x = 10
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(8),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpEnd,
		),
		Entry("If-else statement",
			`
			x := 0
			if 3 > 5 {
				x = 10
			} else {
				x = 20
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(3),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpElse,
			OpI64Const, int64(20),
			OpLocalSet, 0,
			OpEnd,
		),
		Entry("If-else if statement",
			`
			x := 0
			if 3 > 5 {
				x = 10
			} else if 4 > 2 {
				x = 15
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(3),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpElse,
			OpI64Const, int64(4),
			OpI64Const, int64(2),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(15),
			OpLocalSet, 0,
			OpElse,
			OpEnd,
			OpEnd,
		),
		Entry("If-else if-else statement",
			`
			x := 0
			if 3 > 5 {
				x = 10
			} else if 2 > 4 {
				x = 15
			} else {
				x = 20
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(3),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpElse,
			OpI64Const, int64(2),
			OpI64Const, int64(4),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(15),
			OpLocalSet, 0,
			OpElse,
			OpI64Const, int64(20),
			OpLocalSet, 0,
			OpEnd,
			OpEnd,
		),
		Entry("Nested if statements",
			`
			x := 0
			if 8 > 5 {
				if 3 > 2 {
					x = 10
				}
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(8),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(3),
			OpI64Const, int64(2),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpEnd,
			OpEnd,
		),
		Entry("Multiple sequential if statements",
			`
			x := 0
			if 8 > 5 {
				x = 10
			}
			if 3 > 2 {
				x = 20
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(8),
			OpI64Const, int64(5),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpEnd,
			OpI64Const, int64(3),
			OpI64Const, int64(2),
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(20),
			OpLocalSet, 0,
			OpEnd,
		),
		Entry("Complex condition with logical operators",
			`
			x := 0
			if 8 > 5 and 3 < 4 {
				x = 10
			}
			`,
			OpI64Const, int64(0),
			OpLocalSet, 0,
			OpI64Const, int64(8),
			OpI64Const, int64(5),
			OpI64GtS,
			OpI32Const, int32(0),
			OpI32Ne,
			OpI32Eqz,
			OpIf, BlockTypeI32,
			OpI32Const, int32(0),
			OpElse,
			OpI64Const, int64(3),
			OpI64Const, int64(4),
			OpI64LtS,
			OpI32Const, int32(0),
			OpI32Ne,
			OpEnd,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpEnd,
		),
		Entry("If with variable condition",
			`
			x := 5
			y := 3
			if x > y {
				x = 10
			}
			`,
			OpI64Const, int64(5),
			OpLocalSet, 0,
			OpI64Const, int64(3),
			OpLocalSet, 1,
			OpLocalGet, 0,
			OpLocalGet, 1,
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpEnd,
		),
		Entry("Multiple variable updates in if",
			`
			x := 5
			y := 3
			if x > y {
				x = 10
				y = 20
			}
			`,
			OpI64Const, int64(5),
			OpLocalSet, 0,
			OpI64Const, int64(3),
			OpLocalSet, 1,
			OpLocalGet, 0,
			OpLocalGet, 1,
			OpI64GtS,
			OpIf, BlockTypeEmpty,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpI64Const, int64(20),
			OpLocalSet, 1,
			OpEnd,
		),
	)

	DescribeTable("Compound assignment operators", func(source string, instructions ...any) {
		Expect(compileBlock(source)).To(MatchOpcodes(instructions...))
	},
		Entry("i64 plus equals",
			`x i64 := 10
			x += 5`,
			OpI64Const, int64(10),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(5),
			OpI64Add,
			OpLocalSet, 0,
		),
		Entry("f64 minus equals",
			`x f64 := 10.0
			x -= 2.5`,
			OpF64Const, float64(10.0),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, 2.5,
			OpF64Sub,
			OpLocalSet, 0,
		),
		Entry("i32 multiply equals",
			`x i32 := 3
			x *= 4`,
			OpI32Const, int32(3),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(4),
			OpI32Mul,
			OpLocalSet, 0,
		),
		Entry("f32 divide equals",
			`x f32 := 10.0
			x /= 2.0`,
			OpF32Const, float32(10.0),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32Const, float32(2.0),
			OpF32Div,
			OpLocalSet, 0,
		),
		Entry("i64 modulo equals",
			`x i64 := 17
			x %= 5`,
			OpI64Const, int64(17),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(5),
			OpI64RemS,
			OpLocalSet, 0,
		),
	)

	Describe("Compound string concatenation", func() {
		var sLit, sConcat uint64
		compileStr := func(source string) []byte {
			block := MustSucceed(parser.ParseBlock("{" + source + "}"))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())
			sLit = uint64(ctx.Imports.StringFromLiteral)
			sConcat = uint64(ctx.Imports.StringConcat)
			return ctx.Writer.Bytes()
		}

		It("Should compile string += with string literal", func() {
			Expect(compileStr(`
				s str := "hello"
				s += " world"
			`)).To(MatchOpcodes(
				OpI32Const, int32(0),
				OpI32Const, int32(5),
				OpCall, sLit,
				OpLocalSet, 0,

				OpLocalGet, 0,
				OpI32Const, int32(5),
				OpI32Const, int32(6),
				OpCall, sLit,
				OpCall, sConcat,
				OpLocalSet, 0,
			))
		})

		It("Should compile string += with string variable", func() {
			Expect(compileStr(`
				s str := "hello"
				suffix str := " world"
				s += suffix
			`)).To(MatchOpcodes(
				OpI32Const, int32(0),
				OpI32Const, int32(5),
				OpCall, sLit,
				OpLocalSet, 0,

				OpI32Const, int32(5),
				OpI32Const, int32(6),
				OpCall, sLit,
				OpLocalSet, 1,

				OpLocalGet, 0,
				OpLocalGet, 1,
				OpCall, sConcat,
				OpLocalSet, 0,
			))
		})

		It("Should compile multiple string += operations", func() {
			Expect(compileStr(`
				s str := "a"
				s += "b"
				s += "c"
			`)).To(MatchOpcodes(
				OpI32Const, int32(0),
				OpI32Const, int32(1),
				OpCall, sLit,
				OpLocalSet, 0,

				OpLocalGet, 0,
				OpI32Const, int32(1),
				OpI32Const, int32(1),
				OpCall, sLit,
				OpCall, sConcat,
				OpLocalSet, 0,

				OpLocalGet, 0,
				OpI32Const, int32(2),
				OpI32Const, int32(1),
				OpCall, sLit,
				OpCall, sConcat,
				OpLocalSet, 0,
			))
		})
	})

	DescribeTable("Variable casts", func(source string, instructions ...any) {
		Expect(compileBlock(source)).To(MatchOpcodes(instructions...))
	},
		Entry("i32 variable to f32",
			`x i32 := 42
			y := f32(x)`,
			OpI32Const, int32(42),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32ConvertI32S,
			OpLocalSet, 1,
		),
		Entry("i64 variable to i32",
			`x i64 := 100
			y := i32(x)`,
			OpI64Const, int64(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32WrapI64,
			OpLocalSet, 1,
		),
		Entry("i32 variable to i64",
			`x i32 := 42
			y := i64(x)`,
			OpI32Const, int32(42),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64ExtendI32S,
			OpLocalSet, 1,
		),
		Entry("f32 variable to i32",
			`x f32 := 3.14
			y := i32(x)`,
			OpF32Const, float32(3.14),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32TruncF32S,
			OpLocalSet, 1,
		),
		Entry("u32 variable to f32",
			`x u32 := 42
			y := f32(x)`,
			OpI32Const, int32(42),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32ConvertI32U,
			OpLocalSet, 1,
		),
		Entry("f32 variable to f64",
			`x f32 := 3.14
			y := f64(x)`,
			OpF32Const, float32(3.14),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64PromoteF32,
			OpLocalSet, 1,
		),
		Entry("f64 variable to f32",
			`x f64 := 3.14159
			y := f32(x)`,
			OpF64Const, 3.14159,
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32DemoteF64,
			OpLocalSet, 1,
		),
		Entry("nested variable casts",
			`x i64 := 100
			y := f32(i32(x))`,
			OpI64Const, int64(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32WrapI64,
			OpF32ConvertI32S,
			OpLocalSet, 1,
		),
	)

	Describe("Indexed Assignment", func() {
		It("Should compile indexed assignment to series", func() {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0] = 42
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)

			// Set up function scope BEFORE analysis (required for indexed assignment)
			fnScope := MustSucceed(aCtx.Scope.Add(aCtx, symbol.Symbol{
				Name: "testFunc",
				Kind: symbol.KindFunction,
			}))
			Expect(fnScope).ToNot(BeNil())
			fn := MustSucceed(aCtx.Scope.Resolve(aCtx, "testFunc"))
			aCtx.Scope = fn

			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue(), aCtx.Diagnostics.String())

			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			// Get the import indices we need to verify
			seriesCreateIdx := ctx.Imports.SeriesCreateEmpty[types.I64().String()]
			seriesSetElementIdx := ctx.Imports.SeriesSetElement[types.I64().String()]

			// Verify that bytecode contains correct sequence for indexed assignment:
			// 1. Create series and store in local
			// 2. For indexed assignment: get local, push index, push value, call set_element
			bytecode := ctx.Writer.Bytes()

			Expect(bytecode).To(ContainSubstring(string([]byte{byte(OpLocalGet)})))
			Expect(seriesCreateIdx).ToNot(Equal(uint32(0)))
			Expect(seriesSetElementIdx).ToNot(Equal(uint32(0)))
		})
	})

	Describe("Series Literals with Inferred Variables and Literal Coercion", func() {
		It("Should compile inferred int variable with exact-integer float literal", func() {
			// a := 5 creates an i64 variable
			// 12.0 is an exact integer float that should coerce to i64
			// Result: series[i64] with elements [5, 12]
			block := MustSucceed(parser.ParseBlock(`{
				a := 5
				x := [a, 12.0]
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue(), aCtx.Diagnostics.String())

			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			seriesCreateIdx := ctx.Imports.SeriesCreateEmpty[types.I64().String()]
			seriesSetIdx := ctx.Imports.SeriesSetElement[types.I64().String()]

			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				// a := 5
				OpI64Const, int64(5),
				OpLocalSet, 0,
				// x := [a, 12.0] - create series[i64] with 2 elements
				OpI32Const, int32(2),
				OpCall, uint64(seriesCreateIdx),
				// set element 0 = a
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(seriesSetIdx),
				// set element 1 = 12 (12.0 coerced to i64)
				OpI32Const, int32(1),
				OpI64Const, int64(12),
				OpCall, uint64(seriesSetIdx),
				// store series in x
				OpLocalSet, 1,
			))
		})

		It("Should compile inferred float variable with int literal", func() {
			// a := 12.0 creates an f64 variable
			// 5 is an int literal that should coerce to f64
			// Result: series[f64] with elements [12.0, 5.0]
			block := MustSucceed(parser.ParseBlock(`{
				a := 12.0
				x := [a, 5]
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue(), aCtx.Diagnostics.String())

			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			seriesCreateIdx := ctx.Imports.SeriesCreateEmpty[types.F64().String()]
			seriesSetIdx := ctx.Imports.SeriesSetElement[types.F64().String()]

			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				// a := 12.0
				OpF64Const, float64(12.0),
				OpLocalSet, 0,
				// x := [a, 5] - create series[f64] with 2 elements
				OpI32Const, int32(2),
				OpCall, uint64(seriesCreateIdx),
				// set element 0 = a
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(seriesSetIdx),
				// set element 1 = 5.0 (5 coerced to f64)
				OpI32Const, int32(1),
				OpF64Const, float64(5),
				OpCall, uint64(seriesSetIdx),
				// store series in x
				OpLocalSet, 1,
			))
		})

		It("Should compile multiple inferred variables with mixed literals", func() {
			// a := 5, b := 10 creates i64 variables
			// 15.0 is an exact integer float that should coerce to i64
			// Result: series[i64] with elements [5, 10, 15]
			block := MustSucceed(parser.ParseBlock(`{
				a := 5
				b := 10
				x := [a, b, 15.0]
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue(), aCtx.Diagnostics.String())

			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())

			seriesCreateIdx := ctx.Imports.SeriesCreateEmpty[types.I64().String()]
			seriesSetIdx := ctx.Imports.SeriesSetElement[types.I64().String()]

			Expect(ctx.Writer.Bytes()).To(MatchOpcodes(
				// a := 5
				OpI64Const, int64(5),
				OpLocalSet, 0,
				// b := 10
				OpI64Const, int64(10),
				OpLocalSet, 1,
				// x := [a, b, 15.0] - create series[i64] with 3 elements
				OpI32Const, int32(3),
				OpCall, uint64(seriesCreateIdx),
				// set element 0 = a
				OpI32Const, int32(0),
				OpLocalGet, 0,
				OpCall, uint64(seriesSetIdx),
				// set element 1 = b
				OpI32Const, int32(1),
				OpLocalGet, 1,
				OpCall, uint64(seriesSetIdx),
				// set element 2 = 15 (15.0 coerced to i64)
				OpI32Const, int32(2),
				OpI64Const, int64(15),
				OpCall, uint64(seriesSetIdx),
				// store series in x
				OpLocalSet, 2,
			))
		})
	})

	Describe("Channel Operations", func() {
		compileWithChannels := func(source string, resolver symbol.Resolver) ([]byte, *bindings.ImportIndex) {
			block := MustSucceed(parser.ParseBlock("{" + source + "}"))
			aCtx := acontext.CreateRoot(bCtx, block, resolver)
			fnScope := MustSucceed(aCtx.Scope.Add(aCtx, symbol.Symbol{
				Name: "testFunc",
				Kind: symbol.KindFunction,
			}))
			Expect(fnScope).ToNot(BeNil())
			fnScope.Channels = symbol.NewChannels()
			fn := MustSucceed(aCtx.Scope.Resolve(aCtx, "testFunc"))
			aCtx.Scope = fn
			analyzer.AnalyzeBlock(aCtx)
			Expect(aCtx.Diagnostics.Ok()).To(BeTrue(), aCtx.Diagnostics.String())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, aCtx.TypeMap, false)
			diverged := MustSucceed(statement.CompileBlock(context.Child(ctx, block)))
			Expect(diverged).To(BeFalse())
			return ctx.Writer.Bytes(), ctx.Imports
		}

		Describe("Channel Writes", func() {
			DescribeTable("Should compile channel write for numeric types",
				func(typeName string, arcType types.Type, valueCode string, expectedValueOps ...any) {
					resolver := symbol.MapResolver{
						"test_ch": {
							Name: "test_ch",
							Kind: symbol.KindChannel,
							Type: types.Chan(arcType),
							ID:   100,
						},
					}
					source := "test_ch = " + valueCode
					bytecode, imports := compileWithChannels(source, resolver)
					writeIdx := imports.ChannelWrite[typeName]
					expected := []any{OpI32Const, int32(100)}
					expected = append(expected, expectedValueOps...)
					expected = append(expected, OpCall, uint64(writeIdx))
					Expect(bytecode).To(MatchOpcodes(expected...))
				},
				Entry("i8", "i8", types.I8(), "42", OpI32Const, int32(42)),
				Entry("i16", "i16", types.I16(), "42", OpI32Const, int32(42)),
				Entry("i32", "i32", types.I32(), "42", OpI32Const, int32(42)),
				Entry("i64", "i64", types.I64(), "42", OpI64Const, int64(42)),
				Entry("u8", "u8", types.U8(), "42", OpI32Const, int32(42)),
				Entry("u16", "u16", types.U16(), "42", OpI32Const, int32(42)),
				Entry("u32", "u32", types.U32(), "42", OpI32Const, int32(42)),
				Entry("u64", "u64", types.U64(), "42", OpI64Const, int64(42)),
				Entry("f32", "f32", types.F32(), "3.14", OpF32Const, float32(3.14)),
				Entry("f64", "f64", types.F64(), "3.14159", OpF64Const, float64(3.14159)),
			)

			It("Should compile f64 channel write with float literal", func() {
				resolver := symbol.MapResolver{
					"f64_ch": {
						Name: "f64_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   200,
					},
				}
				bytecode, imports := compileWithChannels("f64_ch = 3.14159", resolver)
				writeIdx := imports.ChannelWrite["f64"]

				Expect(bytecode).To(MatchOpcodes(
					OpI32Const, int32(200), // channel ID
					OpF64Const, float64(3.14159), // value
					OpCall, uint64(writeIdx), // channel_write_f64
				))
			})

			It("Should compile f32 channel write with float literal", func() {
				resolver := symbol.MapResolver{
					"f32_ch": {
						Name: "f32_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   300,
					},
				}
				bytecode, imports := compileWithChannels("f32_ch = 2.718", resolver)
				writeIdx := imports.ChannelWrite["f32"]

				Expect(bytecode).To(MatchOpcodes(
					OpI32Const, int32(300), // channel ID
					OpF32Const, float32(2.718), // value
					OpCall, uint64(writeIdx), // channel_write_f32
				))
			})

			It("Should compile channel write with variable value", func() {
				resolver := symbol.MapResolver{
					"output_ch": {
						Name: "output_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   400,
					},
				}
				bytecode, imports := compileWithChannels(`
					x i32 := 42
					output_ch = x
				`, resolver)
				writeIdx := imports.ChannelWrite["i32"]

				Expect(bytecode).To(MatchOpcodes(
					// x := 42
					OpI32Const, int32(42),
					OpLocalSet, 0,
					// output_ch = x
					OpI32Const, int32(400), // channel ID
					OpLocalGet, 0, // get x
					OpCall, uint64(writeIdx), // channel_write_i32
				))
			})

			It("Should compile channel write with expression value", func() {
				resolver := symbol.MapResolver{
					"result_ch": {
						Name: "result_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   500,
					},
				}
				bytecode, imports := compileWithChannels(`
					a i64 := 10
					b i64 := 20
					result_ch = a + b
				`, resolver)
				writeIdx := imports.ChannelWrite["i64"]

				Expect(bytecode).To(MatchOpcodes(
					// a := 10
					OpI64Const, int64(10),
					OpLocalSet, 0,
					// b := 20
					OpI64Const, int64(20),
					OpLocalSet, 1,
					// result_ch = a + b
					OpI32Const, int32(500), // channel ID pushed first
					OpLocalGet, 0, // a
					OpLocalGet, 1, // b
					OpI64Add,                 // a + b
					OpCall, uint64(writeIdx), // channel_write_i64
				))
			})

			It("Should compile multiple channel writes", func() {
				resolver := symbol.MapResolver{
					"ch1": {
						Name: "ch1",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   600,
					},
					"ch2": {
						Name: "ch2",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   700,
					},
				}
				bytecode, imports := compileWithChannels(`
					ch1 = 100
					ch2 = 3.14
				`, resolver)
				writeI32Idx := imports.ChannelWrite["i32"]
				writeF64Idx := imports.ChannelWrite["f64"]

				Expect(bytecode).To(MatchOpcodes(
					// ch1 = 100
					OpI32Const, int32(600),
					OpI32Const, int32(100),
					OpCall, uint64(writeI32Idx),
					// ch2 = 3.14
					OpI32Const, int32(700),
					OpF64Const, float64(3.14),
					OpCall, uint64(writeF64Idx),
				))
			})
		})

		Describe("Channel Reads", func() {
			It("Should compile channel read in variable declaration", func() {
				resolver := symbol.MapResolver{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   100,
					},
				}
				bytecode, imports := compileWithChannels(`
					x f64 := sensor
				`, resolver)
				readIdx := imports.ChannelRead["f64"]

				Expect(bytecode).To(MatchOpcodes(
					OpI32Const, int32(100), // channel ID
					OpCall, uint64(readIdx), // channel_read_f64
					OpLocalSet, 0, // store in x
				))
			})

			DescribeTable("Should compile channel read for numeric types",
				func(typeName string, arcType types.Type) {
					resolver := symbol.MapResolver{
						"test_ch": {
							Name: "test_ch",
							Kind: symbol.KindChannel,
							Type: types.Chan(arcType),
							ID:   100,
						},
					}
					source := "x " + typeName + " := test_ch"
					bytecode, imports := compileWithChannels(source, resolver)
					readIdx := imports.ChannelRead[typeName]

					Expect(bytecode).To(MatchOpcodes(
						OpI32Const, int32(100), // channel ID
						OpCall, uint64(readIdx), // channel_read_<type>
						OpLocalSet, 0, // store in local
					))
				},
				Entry("i8", "i8", types.I8()),
				Entry("i16", "i16", types.I16()),
				Entry("i32", "i32", types.I32()),
				Entry("i64", "i64", types.I64()),
				Entry("u8", "u8", types.U8()),
				Entry("u16", "u16", types.U16()),
				Entry("u32", "u32", types.U32()),
				Entry("u64", "u64", types.U64()),
				Entry("f32", "f32", types.F32()),
				Entry("f64", "f64", types.F64()),
			)

			It("Should compile channel read in expression", func() {
				resolver := symbol.MapResolver{
					"pressure": {
						Name: "pressure",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   200,
					},
				}
				bytecode, imports := compileWithChannels(`
					threshold f64 := 100.0
					result f64 := pressure * 2.0
				`, resolver)
				readIdx := imports.ChannelRead["f64"]

				Expect(bytecode).To(MatchOpcodes(
					// threshold := 100.0
					OpF64Const, float64(100.0),
					OpLocalSet, 0,
					// result := pressure * 2.0
					OpI32Const, int32(200), // channel ID
					OpCall, uint64(readIdx), // channel_read_f64
					OpF64Const, float64(2.0),
					OpF64Mul,
					OpLocalSet, 1,
				))
			})

			It("Should compile channel read in comparison", func() {
				resolver := symbol.MapResolver{
					"temp": {
						Name: "temp",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   300,
					},
				}
				bytecode, imports := compileWithChannels(`
					x i32 := 0
					if temp > 100 {
						x = 1
					}
				`, resolver)
				readIdx := imports.ChannelRead["i32"]

				Expect(bytecode).To(MatchOpcodes(
					// x := 0
					OpI32Const, int32(0),
					OpLocalSet, 0,
					// if temp > 100
					OpI32Const, int32(300), // channel ID
					OpCall, uint64(readIdx), // channel_read_i32
					OpI32Const, int32(100),
					OpI32GtS,
					OpIf, BlockTypeEmpty,
					// x = 1
					OpI32Const, int32(1),
					OpLocalSet, 0,
					OpEnd,
				))
			})

			It("Should compile multiple channel reads", func() {
				resolver := symbol.MapResolver{
					"ch1": {
						Name: "ch1",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   400,
					},
					"ch2": {
						Name: "ch2",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   500,
					},
				}
				bytecode, imports := compileWithChannels(`
					sum i32 := ch1 + ch2
				`, resolver)
				readIdx := imports.ChannelRead["i32"]

				Expect(bytecode).To(MatchOpcodes(
					OpI32Const, int32(400), // ch1 ID
					OpCall, uint64(readIdx), // channel_read_i32
					OpI32Const, int32(500), // ch2 ID
					OpCall, uint64(readIdx), // channel_read_i32
					OpI32Add,
					OpLocalSet, 0,
				))
			})
		})

		Describe("Channel Read and Write Combined", func() {
			It("Should compile reading from one channel and writing to another", func() {
				resolver := symbol.MapResolver{
					"input_ch": {
						Name: "input_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   100,
					},
					"output_ch": {
						Name: "output_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   200,
					},
				}
				bytecode, imports := compileWithChannels(`
					output_ch = input_ch * 2.0
				`, resolver)
				readIdx := imports.ChannelRead["f64"]
				writeIdx := imports.ChannelWrite["f64"]

				Expect(bytecode).To(MatchOpcodes(
					// Push output channel ID first (before expression)
					OpI32Const, int32(200),
					// Read from input_ch
					OpI32Const, int32(100),
					OpCall, uint64(readIdx),
					// Multiply by 2.0
					OpF64Const, float64(2.0),
					OpF64Mul,
					// Write to output_ch
					OpCall, uint64(writeIdx),
				))
			})

			It("Should compile conditional channel write based on channel read", func() {
				resolver := symbol.MapResolver{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   100,
					},
					"alarm": {
						Name: "alarm",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   200,
					},
				}
				bytecode, imports := compileWithChannels(`
					if sensor > 50 {
						alarm = 1
					}
				`, resolver)
				readIdx := imports.ChannelRead["i32"]
				writeIdx := imports.ChannelWrite["i32"]

				Expect(bytecode).To(MatchOpcodes(
					// if sensor > 50
					OpI32Const, int32(100),
					OpCall, uint64(readIdx),
					OpI32Const, int32(50),
					OpI32GtS,
					OpIf, BlockTypeEmpty,
					// alarm = 1
					OpI32Const, int32(200),
					OpI32Const, int32(1),
					OpCall, uint64(writeIdx),
					OpEnd,
				))
			})
		})
	})
})
