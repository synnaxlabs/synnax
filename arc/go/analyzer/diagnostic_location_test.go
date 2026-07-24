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
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Diagnostic Locations", func() {
	type diagnosticCase struct {
		source               string
		expectedMsg          string
		expectedLine         int
		expectedCharacter    int
		expectedEndLine      int
		expectedEndCharacter int
		expectedSev          protocol.DiagnosticSeverity
	}

	runDiagnosticTest := func(bCtx SpecContext, tc diagnosticCase) {
		prog := MustSucceed(parser.Parse(tc.source))
		ctx := context.NewRoot(bCtx, prog, nil)
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeFalse())
		Expect(*ctx.Diagnostics).To(HaveLen(1))

		diag := (*ctx.Diagnostics)[0]
		Expect(diag.Message).To(ContainSubstring(tc.expectedMsg))
		if tc.expectedLine > 0 {
			Expect(diag.Range.Start.Line).To(Equal(uint32(tc.expectedLine)))
		}
		if tc.expectedCharacter > 0 {
			Expect(diag.Range.Start.Character).To(Equal(uint32(tc.expectedCharacter)))
		}
		if tc.expectedEndLine > 0 {
			Expect(diag.Range.End.Line).To(Equal(uint32(tc.expectedEndLine)))
		}
		if tc.expectedEndCharacter > 0 {
			Expect(diag.Range.End.Character).To(Equal(uint32(tc.expectedEndCharacter)))
		}
		if tc.expectedSev != 0 {
			Expect(diag.Severity).To(Equal(tc.expectedSev))
		}
	}

	DescribeTable("Undefined Symbol Errors",
		runDiagnosticTest,
		Entry("undefined variable in assignment",
			diagnosticCase{
				source: `
func test() {
	x := undefined_var
}`,
				expectedMsg:       "undefined symbol: undefined_var",
				expectedLine:      2,
				expectedCharacter: 6,
				expectedSev:       protocol.DiagnosticSeverityError,
			}),
		Entry("undefined variable on left side of assignment",
			diagnosticCase{
				source: `
func test() {
	x i32 := 1
	undefined_target = x
}`,
				expectedMsg:       "undefined symbol: undefined_target",
				expectedLine:      3,
				expectedCharacter: 1,
				expectedSev:       0,
			}),
		Entry("undefined function call",
			diagnosticCase{
				source: `
func test() {
	result := unknownFunc(1, 2)
}`,
				expectedMsg:       "undefined symbol: unknownFunc",
				expectedLine:      2,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
	)

	DescribeTable("Type Mismatch Errors",
		runDiagnosticTest,
		Entry("type mismatch in variable declaration",
			diagnosticCase{
				source: `
func test() {
	x i32 := "hello"
}`,
				expectedMsg:       "type mismatch",
				expectedLine:      2,
				expectedCharacter: -1,
				expectedSev:       protocol.DiagnosticSeverityError,
			}),
		Entry("type mismatch in assignment",
			diagnosticCase{
				source: `
func test() {
	x i32 := 10
	x = "hello"
}`,
				expectedMsg:       "type mismatch",
				expectedLine:      3,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
		Entry("type mismatch in binary expression",
			diagnosticCase{
				source: `
func test() {
	x i32 := 10
	y f32 := 20.5
	z := x + y
}`,
				expectedMsg:       "type mismatch",
				expectedLine:      4,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
	)

	DescribeTable("Duplicate Declaration Errors",
		runDiagnosticTest,
		Entry("duplicate variable declaration",
			diagnosticCase{
				source: `
func test() {
	x := 1
	x := 2
}`,
				expectedMsg:       "name x conflicts",
				expectedLine:      3,
				expectedCharacter: 1,
				expectedSev:       0,
			}),
		Entry("duplicate function declaration",
			diagnosticCase{
				source: `
func myFunc() {
}

func myFunc() {
}`,
				expectedMsg:       "name myFunc conflicts",
				expectedLine:      4,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
		Entry("duplicate parameter name",
			diagnosticCase{
				source: `
func test(x i32, x i32) {
}`,
				expectedMsg:       "name x conflicts",
				expectedLine:      1,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
	)

	DescribeTable("Return Statement Errors",
		runDiagnosticTest,
		Entry("missing return value",
			diagnosticCase{
				source: `
func test() i64 {
	x := 42
}`,
				expectedMsg:       "must return a value",
				expectedLine:      -1,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
		Entry("unexpected return value",
			diagnosticCase{
				source: `
func test() {
	return 42
}`,
				expectedMsg:       "cannot return a value from a function with no return type",
				expectedLine:      2,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
	)

	DescribeTable("Operator Errors",
		runDiagnosticTest,
		Entry("invalid arithmetic on strings",
			diagnosticCase{
				source: `
func test() {
	x := "hello" + 12
}`,
				expectedMsg:       "is not compatible with",
				expectedLine:      2,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
		Entry("invalid unary operator",
			diagnosticCase{
				source: `
func test() {
	x := "hello"
	y := -x
}`,
				expectedMsg:       "operator - not supported",
				expectedLine:      3,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
		Entry("invalid logical operation",
			diagnosticCase{
				source: `
func test() {
	x i32 := 10
	y i32 := 20
	z := x and y
}`,
				expectedMsg:       "cannot use i32 in and operation",
				expectedLine:      4,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
	)

	DescribeTable("Nested Scope Errors",
		runDiagnosticTest,
		Entry("error in nested if block",
			diagnosticCase{
				source: `
func test() {
	if 1 {
		if 1 {
			x := undefined
		}
	}
}`,
				expectedMsg:       "undefined symbol: undefined",
				expectedLine:      4,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
		Entry("error in else block",
			diagnosticCase{
				source: `
func test() {
	if 1 {
		x := 1
	} else {
		y := undefined
	}
}`,
				expectedMsg:       "undefined symbol: undefined",
				expectedLine:      5,
				expectedCharacter: -1,
				expectedSev:       0,
			}),
	)

	Describe("Error Recovery", func() {
		It("Should report all independent errors with correct locations", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
func test() {
	a := undefined1
	b := undefined2
}`))
			ctx := context.NewRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(2))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined1"))
			Expect(diag.Range.Start.Line).To(Equal(uint32(2)))

			diag2 := (*ctx.Diagnostics)[1]
			Expect(diag2.Message).To(ContainSubstring("undefined symbol: undefined2"))
			Expect(diag2.Range.Start.Line).To(Equal(uint32(3)))
		})
	})

	DescribeTable("Diagnostic End Range",
		runDiagnosticTest,
		Entry("undefined variable should span the identifier",
			diagnosticCase{
				source:               "func test() {\n\tx := undefined_var\n}",
				expectedMsg:          "undefined symbol: undefined_var",
				expectedLine:         1,
				expectedCharacter:    6,
				expectedEndLine:      1,
				expectedEndCharacter: 19,
				expectedSev:          protocol.DiagnosticSeverityError,
			}),
		Entry("short identifier should have correct end character",
			diagnosticCase{
				source:               "func test() {\n\tx := y\n}",
				expectedMsg:          "undefined symbol: y",
				expectedLine:         1,
				expectedCharacter:    6,
				expectedEndLine:      1,
				expectedEndCharacter: 7,
				expectedSev:          protocol.DiagnosticSeverityError,
			}),
		Entry("multiline expression should span correctly",
			diagnosticCase{
				source:               "func test() {\n\tx := undefined_symbol +\n\t\t1\n}",
				expectedMsg:          "undefined symbol: undefined_symbol",
				expectedLine:         1,
				expectedCharacter:    6,
				expectedEndLine:      1,
				expectedEndCharacter: 22,
				expectedSev:          protocol.DiagnosticSeverityError,
			}),
	)
})
