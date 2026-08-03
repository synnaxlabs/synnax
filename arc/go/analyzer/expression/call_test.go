// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// These specs exercise AnalyzeCall through the parens (positional) surface form. The
// brace form, named-argument matching, and the externallySatisfied trigger path are
// driven by the flow analyzer and are covered in the flow package's specs.
var _ = Describe("AnalyzeCall", func() {
	Describe("Argument count", func() {
		DescribeTable("valid argument counts",
			func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
			Entry("exact positional arguments", `
				func add(x i32, y i32) i32 { return x + y }
				func main() { result := add(10, 20) }
			`),
			Entry("required param with optional omitted", `
				func add(x i64, y i64 = 0) i64 { return x + y }
				func main() { result := add(10) }
			`),
			Entry("all optional params omitted", `
				func defaults(a i64 = 1, b i64 = 2) i64 { return a + b }
				func main() { result := defaults() }
			`),
		)

		DescribeTable("invalid argument counts",
			func(ctx SpecContext, code string, expectedMsg string) {
				expectFailure(ctx, code, nil, expectedMsg)
			},
			Entry("missing required argument", `
				func add(x i32, y i32) i32 { return x + y }
				func main() { result := add(5) }
			`, "missing required argument for parameter 'y' of func 'add'"),
			Entry("missing the only required argument", `
				func getValue(x i32) i32 { return x }
				func main() { result := getValue() }
			`, "missing required argument for parameter 'x' of func 'getValue'"),
			Entry("too many arguments", `
				func add(x i32, y i32) i32 { return x + y }
				func main() { result := add(5, 10, 15) }
			`, "too many arguments for func 'add': expected at most 2"),
		)
	})

	Describe("Argument types", func() {
		DescribeTable("type mismatches",
			func(ctx SpecContext, code string, expectedMsg string) {
				expectFailure(ctx, code, nil, expectedMsg)
			},
			Entry("string literal to i32 parameter", `
				func add(x i32, y i32) i32 { return x + y }
				func main() { result := add(5, "hello") }
			`, "argument 'y' of 'add'"),
			Entry("i32 variable to f32 parameter", `
				func process(x f32) f32 { return x * 2.0 }
				func main() {
					x i32 := 5
					result := process(x)
				}
			`, "argument 'x' of 'process'"),
		)
	})

	Describe("Channel arguments", func() {
		It("Should dereference a channel argument to a value-typed parameter", func(ctx SpecContext) {
			expectSuccess(ctx, `
				func read(x f32) f32 { return x }
				func main() { result := read(temp) }
			`, []symbol.Symbol{
				{Name: "temp", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			})
		})
	})
})
