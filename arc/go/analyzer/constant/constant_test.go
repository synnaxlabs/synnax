// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constant_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func analyzeProgram(src string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	prog := MustSucceed(parser.Parse(src))
	ctx := context.CreateRoot(bCtx, prog, resolver)
	analyzer.AnalyzeProgram(ctx)
	return ctx
}

func analyzeExpectSuccess(src string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	ctx := analyzeProgram(src, resolver)
	ExpectWithOffset(1, *ctx.Diagnostics).To(BeEmpty(), ctx.Diagnostics.String())
	return ctx
}

func analyzeExpectError(src string, resolver symbol.Resolver, msgMatcher OmegaMatcher) context.Context[parser.IProgramContext] {
	ctx := analyzeProgram(src, resolver)
	ExpectWithOffset(1, *ctx.Diagnostics).To(HaveLen(1))
	ExpectWithOffset(1, (*ctx.Diagnostics)[0].Message).To(msgMatcher)
	ExpectWithOffset(1, (*ctx.Diagnostics)[0].Severity).To(Equal(diagnostics.SeverityError))
	return ctx
}

var _ = Describe("Global Constant Analyzer", func() {
	Describe("CollectDeclarations", func() {
		Describe("basic declaration collection", func() {
			It("should handle empty program", func() {
				ctx := analyzeExpectSuccess(``, nil)
				Expect(ctx.Scope.Children).To(BeEmpty())
			})

			It("should parse global constant with explicit type", func() {
				ctx := analyzeExpectSuccess(`MAX_PRESSURE f64 := 500.0`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(1))
				c := ctx.Scope.Children[0]
				Expect(c.Name).To(Equal("MAX_PRESSURE"))
				Expect(c.Kind).To(Equal(symbol.KindGlobalConstant))
				Expect(c.Type).To(Equal(types.F64()))
				Expect(c.DefaultValue).To(Equal(500.0))
			})

			It("should infer i64 from integer literal", func() {
				ctx := analyzeExpectSuccess(`COUNT := 42`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Name).To(Equal("COUNT"))
				Expect(c.Kind).To(Equal(symbol.KindGlobalConstant))
				Expect(c.Type).To(Equal(types.I64()))
				Expect(c.DefaultValue).To(Equal(int64(42)))
			})

			It("should infer f64 from float literal", func() {
				ctx := analyzeExpectSuccess(`PI := 3.14159`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Type).To(Equal(types.F64()))
				Expect(c.DefaultValue).To(Equal(3.14159))
			})

			It("should support unit literals", func() {
				ctx := analyzeExpectSuccess(`TIMEOUT := 100ms`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Type.Kind).To(Equal(types.KindI64))
				Expect(c.DefaultValue).To(BeNumerically("==", int64(100*telem.Millisecond)))
			})

			It("should support explicit type with unit literal", func() {
				ctx := analyzeExpectSuccess(`DELAY i64 := 500ms`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Type.Kind).To(Equal(types.KindI64))
				Expect(c.DefaultValue).To(BeNumerically("==", int64(500*telem.Millisecond)))
			})

			It("should parse string constant", func() {
				ctx := analyzeExpectSuccess(`MESSAGE := "hello"`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Type).To(Equal(types.String()))
				Expect(c.DefaultValue).To(Equal("hello"))
			})

			It("should collect multiple constants", func() {
				ctx := analyzeExpectSuccess(`
					MAX := 100
					MIN := 0
					PI := 3.14
				`, nil)
				consts := ctx.Scope.FilterChildrenByKind(symbol.KindGlobalConstant)
				Expect(consts).To(HaveLen(3))
			})

			It("should coexist with functions", func() {
				ctx := analyzeExpectSuccess(`
					MAX := 100
					func foo() {}
				`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(2))
				consts := ctx.Scope.FilterChildrenByKind(symbol.KindGlobalConstant)
				funcs := ctx.Scope.FilterChildrenByKind(symbol.KindFunction)
				Expect(consts).To(HaveLen(1))
				Expect(funcs).To(HaveLen(1))
			})
		})

		Describe("type coercion", func() {
			It("should coerce integer to explicit f64", func() {
				ctx := analyzeExpectSuccess(`VALUE f64 := 42`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Type).To(Equal(types.F64()))
				Expect(c.DefaultValue).To(Equal(float64(42)))
			})

			It("should coerce integer to explicit i32", func() {
				ctx := analyzeExpectSuccess(`VALUE i32 := 42`, nil)
				c := ctx.Scope.Children[0]
				Expect(c.Type).To(Equal(types.I32()))
				Expect(c.DefaultValue).To(Equal(int32(42)))
			})
		})

		Describe("error conditions", func() {
			It("should reject duplicate names", func() {
				analyzeExpectError(`
					X := 1
					X := 2
				`, nil, ContainSubstring("conflicts with existing symbol"))
			})

			It("should reject shadowing function names", func() {
				analyzeExpectError(`
					foo := 1
					func foo() {}
				`, nil, ContainSubstring("conflicts with existing symbol"))
			})

			It("should reject overflow in explicit type", func() {
				analyzeExpectError(`VALUE i8 := 128`, nil, ContainSubstring("out of range for i8"))
			})

			It("should reject non-integer float to integer type", func() {
				analyzeExpectError(`VALUE i32 := 3.14`, nil, ContainSubstring("cannot convert non-integer float"))
			})
		})
	})

	Describe("Usage in Functions", func() {
		It("should resolve constant in function body", func() {
			ctx := analyzeExpectSuccess(`
				MAX := 100
				func check(x i64) i64 {
					return x + MAX
				}
			`, nil)
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "check"))
			Expect(funcScope.Kind).To(Equal(symbol.KindFunction))
		})

		It("should use constant in expression", func() {
			ctx := analyzeExpectSuccess(`
				PI := 3.14159
				func area(r f64) f64 {
					return r * r * PI
				}
			`, nil)
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "area"))
			Expect(funcScope.Kind).To(Equal(symbol.KindFunction))
		})

		It("should allow constant in condition", func() {
			ctx := analyzeExpectSuccess(`
				THRESHOLD := 100
				func check(x i64) i64 {
					if (x > THRESHOLD) {
						return 1
					}
					return 0
				}
			`, nil)
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "check"))
			Expect(funcScope.Kind).To(Equal(symbol.KindFunction))
		})
	})
})
