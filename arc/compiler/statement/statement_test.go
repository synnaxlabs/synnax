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
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/statement"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

func compile(source string) []byte {
	stmt := MustSucceed(parser.ParseStatement(source))
	aCtx := acontext.CreateRoot(bCtx, stmt, nil)
	Expect(analyzer.AnalyzeStatement(aCtx)).To(BeTrue())
	ctx := context.CreateRoot(bCtx, aCtx.Scope, true)
	Expect(statement.Compile(context.Child(ctx, stmt))).To(Succeed())
	return ctx.Writer.Bytes()
}

func compileBlock(source string) []byte {
	block := MustSucceed(parser.ParseBlock("{" + source + "}"))
	aCtx := acontext.CreateRoot(bCtx, block, nil)
	Expect(analyzer.AnalyzeBlock(aCtx)).To(BeTrue())
	ctx := context.CreateRoot(bCtx, aCtx.Scope, true)
	Expect(statement.CompileBlock(context.Child(ctx, block))).To(Succeed())
	return ctx.Writer.Bytes()
}

var _ = Describe("Statement Compiler", func() {
	// NOTE: Output assignment tests (ir.KindOutput) are not included here because they
	// require a fully configured multi-output context (Outputs, OutputMemoryBase).
	// Output assignment compilation is tested via integration tests in the main compiler suite.

	DescribeTable("Single Statement Bytecode Values", func(source string, instructions ...any) {
		Expect(compile(source)).To(Equal(WASM(instructions...)))
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
			Expect(analyzer.AnalyzeStatement(aCtx)).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, false)
			Expect(statement.Compile(context.Child(ctx, stmt))).To(Succeed())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			expected := WASM(
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID (first stateful var)
				OpI64Const, int64(0), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("Should compile stateful variable declaration with inferred type", func() {
			stmt := MustSucceed(parser.ParseStatement("count $= 0"))
			aCtx := acontext.CreateRoot(bCtx, stmt, nil)
			Expect(analyzer.AnalyzeStatement(aCtx)).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, false)
			Expect(statement.Compile(context.Child(ctx, stmt))).To(Succeed())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			expected := WASM(
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpI64Const, int64(0), // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("Should compile stateful variable assignment", func() {
			block := MustSucceed(parser.ParseBlock(`{
				count i64 $= 0
				count = 5
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			Expect(analyzer.AnalyzeBlock(aCtx)).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, false)
			Expect(statement.CompileBlock(context.Child(ctx, block))).To(Succeed())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			stateStoreIdx := ctx.Imports.StateStore["i64"]
			expected := WASM(
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
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("Should compile stateful variable reference in expression", func() {
			block := MustSucceed(parser.ParseBlock(`{
				count i64 $= 0
				x i64 := count + 1
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			Expect(analyzer.AnalyzeBlock(aCtx)).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, false)
			Expect(statement.CompileBlock(context.Child(ctx, block))).To(Succeed())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			expected := WASM(
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
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("Should compile multiple stateful variables", func() {
			block := MustSucceed(parser.ParseBlock(`{
				a i64 $= 10
				b i64 $= 20
				c i64 := a + b
			}`))
			aCtx := acontext.CreateRoot(bCtx, block, nil)
			Expect(analyzer.AnalyzeBlock(aCtx)).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, false)
			Expect(statement.CompileBlock(context.Child(ctx, block))).To(Succeed())

			stateLoadIdx := ctx.Imports.StateLoad["i64"]
			expected := WASM(
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
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("Should compile stateful variable with different types", func() {
			stmt := MustSucceed(parser.ParseStatement("temperature f64 $= 20.5"))
			aCtx := acontext.CreateRoot(bCtx, stmt, nil)
			Expect(analyzer.AnalyzeStatement(aCtx)).To(BeTrue())
			ctx := context.CreateRoot(bCtx, aCtx.Scope, false)
			Expect(statement.Compile(context.Child(ctx, stmt))).To(Succeed())

			stateLoadIdx := ctx.Imports.StateLoad["f64"]
			expected := WASM(
				OpI32Const, int32(0), // func ID
				OpI32Const, int32(0), // var ID
				OpF64Const, 20.5, // init value
				OpCall, uint64(stateLoadIdx),
				OpLocalSet, 0, // store in local
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})
	})

	DescribeTable("Multi Statement Bytecode Values", func(source string, instructions ...any) {
		Expect(compileBlock(source)).To(Equal(WASM(instructions...)))
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
			if 8 > 5 && 3 < 4 {
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
})
