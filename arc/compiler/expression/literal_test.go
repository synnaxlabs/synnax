// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Literal Compilation", func() {
	DescribeTable("should compile literals correctly",
		expectExpression,

		// Integer Literals
		Entry(
			"integer literals as i64",
			"42",
			types.I64{},
			OpI64Const,
			int64(42),
		),

		Entry(
			"zero",
			"0",
			types.I64{},
			OpI64Const,
			int64(0),
		),

		Entry(
			"large integers",
			"1000000",
			types.I64{},
			OpI64Const,
			int64(1000000),
		),

		// Float Literals
		Entry(
			"float literals as f64",
			"3.14",
			types.F64{},
			OpF64Const,
			float64(3.14),
		),

		Entry(
			"simple decimals",
			"2.5",
			types.F64{},
			OpF64Const,
			float64(2.5),
		),

		Entry(
			"float with leading dot",
			".5",
			types.F64{},
			OpF64Const,
			float64(0.5),
		),

		Entry(
			"float with trailing dot",
			"1.",
			types.F64{},
			OpF64Const,
			float64(1.0),
		),

		// Parenthesized Expressions
		Entry(
			"parenthesized integer",
			"(42)",
			types.I64{},
			OpI64Const,
			int64(42),
		),

		Entry(
			"nested parentheses",
			"((42))",
			types.I64{},
			OpI64Const,
			int64(42),
		),

		Entry(
			"parenthesized float",
			"(3.14)",
			types.F64{},
			OpF64Const,
			float64(3.14),
		),
	)
})
