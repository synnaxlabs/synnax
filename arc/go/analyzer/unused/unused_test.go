// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unused_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

// analyze parses and analyzes src, returning the accumulated diagnostics.
func analyze(bCtx SpecContext, src string, resolver symbol.Resolver) diagnostics.Diagnostics {
	prog := MustSucceed(parser.Parse(src))
	ctx := context.CreateRoot(bCtx, prog, resolver)
	analyzer.AnalyzeProgram(ctx)
	return *ctx.Diagnostics
}

// warningsWithCode returns only diagnostics with the given code.
func warningsWithCode(diags diagnostics.Diagnostics, code diagnostics.ErrorCode) []diagnostics.Diagnostic {
	var out []diagnostics.Diagnostic
	for _, d := range diags {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

// expectWarning parses and analyzes src, asserting that exactly one warning
// with the given code was emitted whose message contains messageSubstring.
// Other diagnostics with different codes are ignored.
func expectWarning(
	bCtx SpecContext,
	src string,
	resolver symbol.Resolver,
	code diagnostics.ErrorCode,
	messageSubstring string,
) {
	diags := analyze(bCtx, src, resolver)
	matches := warningsWithCode(diags, code)
	ExpectWithOffset(1, matches).To(HaveLen(1), diags.String())
	ExpectWithOffset(1, matches[0].Message).To(ContainSubstring(messageSubstring))
}

// expectNoWarning parses and analyzes src, asserting that no warning with the
// given code was emitted. Other diagnostics may be present.
func expectNoWarning(
	bCtx SpecContext,
	src string,
	resolver symbol.Resolver,
	code diagnostics.ErrorCode,
) {
	diags := analyze(bCtx, src, resolver)
	ExpectWithOffset(1, warningsWithCode(diags, code)).To(BeEmpty(), diags.String())
}

var _ = Describe("Unused Variable (ARC5101)", func() {
	Describe("emits warning for", func() {
		It("a local variable that is never read", func(bCtx SpecContext) {
			expectWarning(bCtx, `
				func test() i32 {
					x := 42
					return 0
				}
			`, nil, codes.UnusedVariable, "unused variable 'x'")
		})

		It("a stateful variable that is never read or written", func(bCtx SpecContext) {
			expectWarning(bCtx, `
				func test() i32 {
					count i64 $= 0
					return 1
				}
			`, nil, codes.UnusedVariable, "unused stateful variable 'count'")
		})

		It("a channel alias that is never read", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			expectWarning(bCtx, `
				func test() {
					sensor_ref := sensor
				}
			`, resolver, codes.UnusedVariable, "unused channel alias 'sensor_ref'")
		})
	})

	Describe("does not warn for", func() {
		It("a variable that is written but never read (deferred to unused-assignment rule)", func(bCtx SpecContext) {
			// R1 treats any reference as "used" so reassignment patterns like
			// x i32 := 1; if cond { x = 2 }; return x do not get a false
			// warning on the intermediate write. Detecting writes whose values
			// are never read is a separate future rule.
			expectNoWarning(bCtx, `
				func test() {
					x i32 := 1
					x = 2
				}
			`, nil, codes.UnusedVariable)
		})

		It("a variable used in the return expression", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				func test() i32 {
					x := 42
					return x
				}
			`, nil, codes.UnusedVariable)
		})

		It("a stateful variable read in an expression", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				func counter(trigger i64) i64 {
					count i64 $= 0
					count = count + 1
					return count
				}
			`, nil, codes.UnusedVariable)
		})

		It("a channel alias that is read in an expression", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			expectNoWarning(bCtx, `
				func test() f32 {
					sensor_ref := sensor
					return sensor_ref
				}
			`, resolver, codes.UnusedVariable)
		})

		It("a variable whose name begins with an underscore", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				func test() {
					_placeholder := 42
				}
			`, nil, codes.UnusedVariable)
		})

		It("a stateful variable whose name begins with an underscore", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				func test() {
					_count i64 $= 0
				}
			`, nil, codes.UnusedVariable)
		})

		It("a declaration whose initializer failed to analyze", func(bCtx SpecContext) {
			// The undefined symbol in the initializer poisons x's type, so the
			// unused pass does not pile on a warning about x.
			expectNoWarning(bCtx, `
				func test() {
					x := undefined_symbol
				}
			`, nil, codes.UnusedVariable)
		})

		It("a function parameter that is never read", func(bCtx SpecContext) {
			// Parameters encode the function's contract; R1 does not flag them.
			expectNoWarning(bCtx, `
				func test(x i32, y i32) i32 {
					return y
				}
			`, nil, codes.UnusedVariable)
		})
	})

	It("emits distinct warnings for multiple unused variables in one function", func(bCtx SpecContext) {
		diags := analyze(bCtx, `
			func test() {
				a := 1
				b := 2
			}
		`, nil)
		warns := warningsWithCode(diags, codes.UnusedVariable)
		Expect(warns).To(HaveLen(2), diags.String())
		Expect(warns[0].Message).To(ContainSubstring("unused variable 'a'"))
		Expect(warns[1].Message).To(ContainSubstring("unused variable 'b'"))
	})
})

var _ = Describe("Unused Global Constant (ARC5103)", func() {
	Describe("emits warning for", func() {
		It("a global constant that is never referenced", func(bCtx SpecContext) {
			expectWarning(bCtx, `
				MAX_PRESSURE := 500.0
				func _noop() {}
			`, nil, codes.UnusedGlobalConstant, "unused global constant 'MAX_PRESSURE'")
		})
	})

	Describe("does not warn for", func() {
		It("a constant referenced in a function body", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				MAX := 100
				func _check(x i64) i64 {
					return x + MAX
				}
			`, nil, codes.UnusedGlobalConstant)
		})

		It("a constant referenced in a conditional", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				THRESHOLD := 100
				func _check(x i64) i64 {
					if x > THRESHOLD {
						return 1
					}
					return 0
				}
			`, nil, codes.UnusedGlobalConstant)
		})

		It("a constant whose name begins with an underscore", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				_RESERVED := 42
			`, nil, codes.UnusedGlobalConstant)
		})
	})
})

var _ = Describe("Uncalled Function (ARC5102)", func() {
	Describe("emits warning for", func() {
		It("a function that is never called anywhere", func(bCtx SpecContext) {
			expectWarning(bCtx, `
				func helper(x i32) i32 {
					return x * 2
				}
			`, nil, codes.UncalledFunction, "uncalled function 'helper'")
		})

		It("a function only called from another uncalled function", func(bCtx SpecContext) {
			// Both helper and wrapper are unreferenced from any entry point.
			// (Call-graph reachability composition arrives with R4/R5; for
			// now both are independently flagged as never-called.)
			diags := analyze(bCtx, `
				func helper(x i32) i32 {
					return x * 2
				}
				func wrapper(x i32) i32 {
					return helper(x)
				}
			`, nil)
			warns := warningsWithCode(diags, codes.UncalledFunction)
			Expect(warns).To(HaveLen(1), diags.String())
			Expect(warns[0].Message).To(ContainSubstring("uncalled function 'wrapper'"))
		})
	})

	Describe("does not warn for", func() {
		It("a function called via expression from another (reachable) function", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"src":  {Name: "src", Kind: symbol.KindChannel, Type: types.Chan(types.I32())},
				"sink": {Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.I32())},
			}
			// caller is reached via the top-level flow; helper is reached via
			// caller's expression-level call. Neither should be flagged.
			expectNoWarning(bCtx, `
				func helper(x i32) i32 {
					return x * 2
				}
				func caller(v i32) i32 {
					return helper(v)
				}
				src -> caller{} -> sink
			`, resolver, codes.UncalledFunction)
		})

		It("a function called via a top-level flow statement", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"src":  {Name: "src", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
				"sink": {Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			expectNoWarning(bCtx, `
				func double(v f32) f32 {
					return v * 2
				}
				src -> double{} -> sink
			`, resolver, codes.UncalledFunction)
		})

		It("a function called inside a stage body", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"trigger": {Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				"sensor":  {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			expectNoWarning(bCtx, `
				func over(v f32) u8 {
					return v > 10
				}
				sequence main {
					stage check {
						sensor -> over{} => next
					}
					stage done {}
				}
				trigger => main
			`, resolver, codes.UncalledFunction)
		})

		It("a function whose name begins with an underscore", func(bCtx SpecContext) {
			expectNoWarning(bCtx, `
				func _reserved(x i32) i32 {
					return x
				}
			`, nil, codes.UncalledFunction)
		})
	})
})
