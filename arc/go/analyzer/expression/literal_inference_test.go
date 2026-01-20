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
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Literal Type Inference", func() {
	var testResolver symbol.MapResolver

	BeforeEach(func() {
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
		It("Should allow comparison of f32 variable with integer literal", func() {
			expectSuccess(`
				func testFunc() {
					x f32 := 10
					z := x > 5
				}
			`, nil)
		})

		It("Should reject comparison of i32 variable with float literal", func() {
			expectFailure(`
				func testFunc() {
					x i32 := 10
					z := x > 5.0
				}
			`, nil, "type mismatch")
		})

		It("should allow 2 + abc where abc is f32", func() {
			expectSuccess(`
				func test{} () f32 {
					return 2 + abc
				}
			`, testResolver)
		})

		It("should allow abc + 2 where abc is f32", func() {
			expectSuccess(`
				func test{} () f32 {
					return abc + 2
				}
			`, testResolver)
		})

		It("should allow 2.5 + abc where abc is f32", func() {
			expectSuccess(`
				func test{} () f32 {
					return 2.5 + abc
				}
			`, testResolver)
		})

		It("should allow 5 + xyz where xyz is i32", func() {
			expectSuccess(`
				func test{} () i32 {
					return 5 + xyz
				}
			`, testResolver)
		})

		It("should infer correct type for expressions with multiple literals", func() {
			expectSuccess(`
				func test{} () f32 {
					return 2 + abc + 3
				}
			`, testResolver)
		})

		It("Should infer the correct type for channel and literal operations", func() {
			expectSuccess(`
				func cat() f64 {
					return 2 * sensor
				}
			`, testResolver)
		})

		It("Should infer the correct type for channel and literal operations in power expressions", func() {
			expectSuccess(`
				func cat() f64 {
					return sensor ^ 2
				}
			`, testResolver)
		})

		It("Should infer the correct type for channel and several literal operations", func() {
			expectSuccess(`
				func cat() f64 {
					return 2 * sensor * 3.0 * sensor
				}
			`, testResolver)
		})

		It("Should reject float literal with incompatible integer channel type", func() {
			program := MustSucceed(parser.Parse(`
				func cat() f64 {
					return 2.2 * integer_sensor
				}
			`))
			ctx := acontext.CreateRoot(bCtx, program, testResolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			errorMsg := ctx.Diagnostics.Error()
			Expect(errorMsg).To(Or(
				ContainSubstring("is not compatible with"),
				ContainSubstring("type mismatch"),
			))
		})

		It("Should infer the correct type for the direct return of a channel as an i8", func() {
			expectSuccess(`
				func cat() i8 {
					return integer_sensor
				}
			`, testResolver)
		})
	})

	Describe("Power operator regression tests (SY-3207)", func() {
		BeforeEach(func() {
			testResolver["f32_sensor"] = symbol.Symbol{
				Name: "f32_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
			}
			testResolver["f64_sensor"] = symbol.Symbol{
				Name: "f64_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
			}
			testResolver["i32_sensor"] = symbol.Symbol{
				Name: "i32_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I32()),
			}
		})

		It("Should infer integer literal as f32 in power expression with f32 channel", func() {
			expectSuccess(`
				func test() f32 {
					return f32_sensor ^ 2
				}
			`, testResolver)
		})

		It("Should infer integer literal as f64 in power expression with f64 channel", func() {
			expectSuccess(`
				func test() f64 {
					return f64_sensor ^ 3
				}
			`, testResolver)
		})

		It("Should infer integer literal as i32 in power expression with i32 channel", func() {
			expectSuccess(`
				func test() i32 {
					return i32_sensor ^ 2
				}
			`, testResolver)
		})

		It("Should handle float literal as exponent with float channel", func() {
			expectSuccess(`
				func test() f64 {
					return f64_sensor ^ 2.5
				}
			`, testResolver)
		})

		It("Should handle chained power operations with literals", func() {
			expectSuccess(`
				func test() f32 {
					return f32_sensor ^ 2 ^ 3
				}
			`, testResolver)
		})

		It("Should handle power in complex expression with literals", func() {
			expectSuccess(`
				func test() f32 {
					return 2 * f32_sensor ^ 2 + 3
				}
			`, testResolver)
		})
	})
})
