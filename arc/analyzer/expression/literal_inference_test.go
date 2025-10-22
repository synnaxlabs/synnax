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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Literal Type Inference", func() {
	var (
		bCtx         context.Context
		testResolver symbol.MapResolver
	)

	BeforeEach(func() {
		bCtx = context.Background()
		testResolver = symbol.MapResolver{
			"abc": symbol.Symbol{
				Name: "abc",
				Kind: symbol.KindVariable,
				Type: types.F32(),
			},
			"xyz": symbol.Symbol{
				Name: "xyz",
				Kind: symbol.KindVariable,
				Type: types.I32(),
			},
			"sensor": symbol.Symbol{
				Name: "sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
			},
			"integer_sensor": symbol.Symbol{
				Name: "integer_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I8()),
			},
		}
	})

	Describe("Numeric literals should adapt to context", func() {
		It("should allow 2 + abc where abc is f32", func() {
			program := MustSucceed(parser.Parse(`
func test{} () {
	result f32
} {
	result = 2 + abc
}
`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow abc + 2 where abc is f32", func() {
			program := MustSucceed(parser.Parse(`
func test{} () {
	result f32
} {
	result = abc + 2
}
`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow 2.5 + abc where abc is f32", func() {
			program := MustSucceed(parser.Parse(`
func test{} () {
	result f32
} {
	result = 2.5 + abc
}
`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow 5 + xyz where xyz is i32", func() {
			program := MustSucceed(parser.Parse(`
func test{} () {
	result i32
} {
	result = 5 + xyz
}
`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should infer correct type for expressions with multiple literals", func() {
			program := MustSucceed(parser.Parse(`
func test{} () {
	result f32
} {
	result = 2 + abc + 3
}
`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should infer the correct type for channel and literal operations", func() {
			program := MustSucceed(parser.Parse(`
			func cat() f64 {
				return 2 * sensor
			}
			`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should infer the correct type for channel and several literal operations", func() {
			program := MustSucceed(parser.Parse(`
			func cat() f64 {
				return 2 * sensor * 3.0 * sensor
			}
			`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should infer the correct type for channel and several literal operations", func() {
			program := MustSucceed(parser.Parse(`
			func cat() f64 {
				return 2.2 * integer_sensor
			}
			`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			// Accept either "f64 and i8" or "i8 and f64" (order may vary)
			errorMsg := ctx.Diagnostics.Error().Error()
			Expect(errorMsg).To(Or(
				ContainSubstring("types f64 and i8 are not unifiable"),
				ContainSubstring("types i8 and f64 are not unifiable"),
			))
		})

		It("Should infer the correct type for the direct return of a channel as an i8", func() {
			program := MustSucceed(parser.Parse(`
			func cat() i8 {
				return integer_sensor
			}
			`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})
})
