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
	"github.com/synnaxlabs/arc/analyzer/text"
	"github.com/synnaxlabs/arc/compiler/core"
	"github.com/synnaxlabs/arc/compiler/statement"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	. "github.com/synnaxlabs/x/testutil"
)

func compile(source string) []byte {
	stmt := MustSucceed(text.ParseStatement(source))
	result := text.AnalyzeStatement(stmt, text.Options{})
	Expect(result.Ok()).To(BeTrue())
	ctx := core.NewContext(result.Symbols, true)
	Expect(statement.Compile(ctx, stmt)).To(Succeed())
	return ctx.Writer.Bytes()
}

func compileBlock(source string) []byte {
	block := MustSucceed(text.ParseBlock("{" + source + "}"))
	result := text.AnalyzeBlock(block, text.Options{})
	Expect(result.Ok()).To(BeTrue())
	ctx := core.NewContext(result.Symbols, true)
	Expect(statement.CompileBlock(ctx, block)).To(Succeed())
	return ctx.Writer.Bytes()
}

var _ = Describe("Statement Compiler", func() {
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
