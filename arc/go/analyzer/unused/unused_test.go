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

// expectUnused parses and analyzes src, asserting that exactly one ARC5101
// warning was emitted whose message contains the given substring.
func expectUnused(bCtx SpecContext, src string, messageSubstring string) {
	diags := analyze(bCtx, src, nil)
	warns := diags.Warnings()
	ExpectWithOffset(1, warns).To(HaveLen(1), diags.String())
	ExpectWithOffset(1, warns[0].Code).To(Equal(codes.UnusedVariable))
	ExpectWithOffset(1, warns[0].Message).To(ContainSubstring(messageSubstring))
}

// expectNoUnused parses and analyzes src, asserting that no ARC5101 warning
// was emitted. Other diagnostics may be present.
func expectNoUnused(bCtx SpecContext, src string, resolver symbol.Resolver) {
	diags := analyze(bCtx, src, resolver)
	for _, d := range diags {
		ExpectWithOffset(1, d.Code).ToNot(Equal(codes.UnusedVariable), diags.String())
	}
}

var _ = Describe("Unused Variable (ARC5101)", func() {
	Describe("emits warning for", func() {
		It("a local variable that is never read", func(bCtx SpecContext) {
			expectUnused(bCtx, `
				func test() i32 {
					x := 42
					return 0
				}
			`, "unused variable 'x'")
		})

		It("a stateful variable that is never read or written", func(bCtx SpecContext) {
			expectUnused(bCtx, `
				func test() i32 {
					count i64 $= 0
					return 1
				}
			`, "unused stateful variable 'count'")
		})

		It("a channel alias that is never read", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			diags := analyze(bCtx, `
				func test() {
					sensor_ref := sensor
				}
			`, resolver)
			warns := diags.Warnings()
			Expect(warns).To(HaveLen(1), diags.String())
			Expect(warns[0].Code).To(Equal(codes.UnusedVariable))
			Expect(warns[0].Message).To(ContainSubstring("unused channel alias 'sensor_ref'"))
		})

	})

	Describe("does not warn for", func() {
		It("a variable that is written but never read (deferred to unused-assignment rule)", func(bCtx SpecContext) {
			// R1 treats any reference as "used" so reassignment patterns like
			// x i32 := 1; if cond { x = 2 }; return x do not get a false
			// warning on the intermediate write. Detecting writes whose values
			// are never read is a separate future rule.
			expectNoUnused(bCtx, `
				func test() {
					x i32 := 1
					x = 2
				}
			`, nil)
		})
		It("a variable used in the return expression", func(bCtx SpecContext) {
			expectNoUnused(bCtx, `
				func test() i32 {
					x := 42
					return x
				}
			`, nil)
		})

		It("a stateful variable read in an expression", func(bCtx SpecContext) {
			expectNoUnused(bCtx, `
				func counter(trigger i64) i64 {
					count i64 $= 0
					count = count + 1
					return count
				}
			`, nil)
		})

		It("a channel alias that is read in an expression", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			expectNoUnused(bCtx, `
				func test() f32 {
					sensor_ref := sensor
					return sensor_ref
				}
			`, resolver)
		})

		It("a variable whose name begins with an underscore", func(bCtx SpecContext) {
			expectNoUnused(bCtx, `
				func test() {
					_placeholder := 42
				}
			`, nil)
		})

		It("a stateful variable whose name begins with an underscore", func(bCtx SpecContext) {
			expectNoUnused(bCtx, `
				func test() {
					_count i64 $= 0
				}
			`, nil)
		})

		It("a declaration whose initializer failed to analyze", func(bCtx SpecContext) {
			// The undefined symbol in the initializer poisons x's type, so the
			// unused pass does not pile on a warning about x.
			expectNoUnused(bCtx, `
				func test() {
					x := undefined_symbol
				}
			`, nil)
		})

		It("a function parameter that is never read", func(bCtx SpecContext) {
			// Parameters encode the function's contract; R1 does not flag them.
			expectNoUnused(bCtx, `
				func test(x i32, y i32) i32 {
					return y
				}
			`, nil)
		})
	})

	It("emits distinct warnings for multiple unused variables in one function", func(bCtx SpecContext) {
		diags := analyze(bCtx, `
			func test() {
				a := 1
				b := 2
			}
		`, nil)
		warns := diags.Warnings()
		Expect(warns).To(HaveLen(2), diags.String())
		Expect(warns[0].Message).To(ContainSubstring("unused variable 'a'"))
		Expect(warns[1].Message).To(ContainSubstring("unused variable 'b'"))
	})
})
