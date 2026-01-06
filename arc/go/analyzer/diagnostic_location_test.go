// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Diagnostic Locations", func() {

	Describe("Undefined Symbol Errors", func() {
		It("Should report correct location for undefined variable in assignment", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x := undefined_var
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined_var"))
			Expect(diag.Line).To(Equal(3))
			Expect(diag.Column).To(Equal(6))
			Expect(diag.Severity).To(Equal(diagnostics.Error))
		})

		It("Should report correct location for undefined variable on left side of assignment", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x i32 := 1
	undefined_target = x
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined_target"))
			Expect(diag.Line).To(Equal(4))
			Expect(diag.Column).To(Equal(1))
		})

		It("Should report correct location for undefined function call", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	result := unknownFunc(1, 2)
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: unknownFunc"))
			Expect(diag.Line).To(Equal(3))
		})
	})

	Describe("Type Mismatch Errors", func() {
		It("Should report correct location for type mismatch in variable declaration", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x i32 := "hello"
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("type mismatch"))
			Expect(diag.Line).To(Equal(3))
			Expect(diag.Severity).To(Equal(diagnostics.Error))
		})

		It("Should report correct location for type mismatch in assignment", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x i32 := 10
	x = "hello"
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("type mismatch"))
			Expect(diag.Line).To(Equal(4))
		})

		It("Should report correct location for type mismatch in binary expression", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x i32 := 10
	y f32 := 20.5
	z := x + y
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("type mismatch"))
			Expect(diag.Line).To(Equal(5))
		})
	})

	Describe("Duplicate Declaration Errors", func() {
		It("Should report correct location for duplicate variable declaration", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x := 1
	x := 2
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("name x conflicts"))
			Expect(diag.Line).To(Equal(4))
			Expect(diag.Column).To(Equal(1))
		})

		It("Should report correct location for duplicate function declaration", func() {
			prog := MustSucceed(parser.Parse(`
func myFunc() {
}

func myFunc() {
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("name myFunc conflicts"))
			Expect(diag.Line).To(Equal(5))
		})

		It("Should report correct location for duplicate parameter name", func() {
			prog := MustSucceed(parser.Parse(`
func test(x i32, x i32) {
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("name x conflicts"))
			Expect(diag.Line).To(Equal(2))
		})
	})

	Describe("Return Statement Errors", func() {
		It("Should report correct location for missing return", func() {
			prog := MustSucceed(parser.Parse(`
func test() i64 {
	x := 42
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("must return a value"))
		})

		It("Should report correct location for unexpected return value", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	return 42
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("unexpected return value"))
			Expect(diag.Line).To(Equal(3))
		})
	})

	Describe("Operator Errors", func() {
		It("Should report correct location for invalid arithmetic on strings", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x := "hello" + "world"
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("cannot use str in + operation"))
			Expect(diag.Line).To(Equal(3))
		})

		It("Should report correct location for invalid unary operator", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x := "hello"
	y := -x
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("operator - not supported"))
			Expect(diag.Line).To(Equal(4))
		})

		It("Should report correct location for invalid logical operation", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	x i32 := 10
	y i32 := 20
	z := x and y
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("cannot use i32 in and operation"))
			Expect(diag.Line).To(Equal(5))
		})
	})

	Describe("Nested Scope Errors", func() {
		It("Should report correct location for error in nested if block", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	if 1 {
		if 1 {
			x := undefined
		}
	}
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined"))
			Expect(diag.Line).To(Equal(5))
		})

		It("Should report correct location for error in else block", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	if 1 {
		x := 1
	} else {
		y := undefined
	}
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined"))
			Expect(diag.Line).To(Equal(6))
		})
	})

	Describe("Error Recovery", func() {
		// Note: The analyzer currently stops analysis after encountering the first error
		// in a statement, but may report errors across independent statements.
		It("Should report first error with correct location", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	a := undefined1
	b := undefined2
}
`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			// Analyzer stops after first error
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined1"))
			Expect(diag.Line).To(Equal(3))
		})
	})
})
