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
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Diagnostic Locations", func() {
	type diagnosticCase struct {
		source         string
		expectedMsg    string
		expectedLine   int
		expectedColumn int // -1 means don't check
		expectedSev    diagnostics.Severity
	}

	runDiagnosticTest := func(tc diagnosticCase) {
		prog := MustSucceed(parser.Parse(tc.source))
		ctx := context.CreateRoot(bCtx, prog, nil)
		Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
		Expect(*ctx.Diagnostics).To(HaveLen(1))

		diag := (*ctx.Diagnostics)[0]
		Expect(diag.Message).To(ContainSubstring(tc.expectedMsg))
		if tc.expectedLine >= 0 {
			Expect(diag.Line).To(Equal(tc.expectedLine))
		}
		if tc.expectedColumn >= 0 {
			Expect(diag.Column).To(Equal(tc.expectedColumn))
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
				expectedMsg:    "undefined symbol: undefined_var",
				expectedLine:   3,
				expectedColumn: 6,
				expectedSev:    diagnostics.Error,
			}),
		Entry("undefined variable on left side of assignment",
			diagnosticCase{
				source: `
func test() {
	x i32 := 1
	undefined_target = x
}`,
				expectedMsg:    "undefined symbol: undefined_target",
				expectedLine:   4,
				expectedColumn: 1,
				expectedSev:    0,
			}),
		Entry("undefined function call",
			diagnosticCase{
				source: `
func test() {
	result := unknownFunc(1, 2)
}`,
				expectedMsg:    "undefined symbol: unknownFunc",
				expectedLine:   3,
				expectedColumn: -1,
				expectedSev:    0,
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
				expectedMsg:    "type mismatch",
				expectedLine:   3,
				expectedColumn: -1,
				expectedSev:    diagnostics.Error,
			}),
		Entry("type mismatch in assignment",
			diagnosticCase{
				source: `
func test() {
	x i32 := 10
	x = "hello"
}`,
				expectedMsg:    "type mismatch",
				expectedLine:   4,
				expectedColumn: -1,
				expectedSev:    0,
			}),
		Entry("type mismatch in binary expression",
			diagnosticCase{
				source: `
func test() {
	x i32 := 10
	y f32 := 20.5
	z := x + y
}`,
				expectedMsg:    "type mismatch",
				expectedLine:   5,
				expectedColumn: -1,
				expectedSev:    0,
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
				expectedMsg:    "name x conflicts",
				expectedLine:   4,
				expectedColumn: 1,
				expectedSev:    0,
			}),
		Entry("duplicate function declaration",
			diagnosticCase{
				source: `
func myFunc() {
}

func myFunc() {
}`,
				expectedMsg:    "name myFunc conflicts",
				expectedLine:   5,
				expectedColumn: -1,
				expectedSev:    0,
			}),
		Entry("duplicate parameter name",
			diagnosticCase{
				source: `
func test(x i32, x i32) {
}`,
				expectedMsg:    "name x conflicts",
				expectedLine:   2,
				expectedColumn: -1,
				expectedSev:    0,
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
				expectedMsg:    "must return a value",
				expectedLine:   -1,
				expectedColumn: -1,
				expectedSev:    0,
			}),
		Entry("unexpected return value",
			diagnosticCase{
				source: `
func test() {
	return 42
}`,
				expectedMsg:    "unexpected return value",
				expectedLine:   3,
				expectedColumn: -1,
				expectedSev:    0,
			}),
	)

	DescribeTable("Operator Errors",
		runDiagnosticTest,
		Entry("invalid arithmetic on strings",
			diagnosticCase{
				source: `
func test() {
	x := "hello" + "world"
}`,
				expectedMsg:    "cannot use str in + operation",
				expectedLine:   3,
				expectedColumn: -1,
				expectedSev:    0,
			}),
		Entry("invalid unary operator",
			diagnosticCase{
				source: `
func test() {
	x := "hello"
	y := -x
}`,
				expectedMsg:    "operator - not supported",
				expectedLine:   4,
				expectedColumn: -1,
				expectedSev:    0,
			}),
		Entry("invalid logical operation",
			diagnosticCase{
				source: `
func test() {
	x i32 := 10
	y i32 := 20
	z := x and y
}`,
				expectedMsg:    "cannot use i32 in and operation",
				expectedLine:   5,
				expectedColumn: -1,
				expectedSev:    0,
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
				expectedMsg:    "undefined symbol: undefined",
				expectedLine:   5,
				expectedColumn: -1,
				expectedSev:    0,
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
				expectedMsg:    "undefined symbol: undefined",
				expectedLine:   6,
				expectedColumn: -1,
				expectedSev:    0,
			}),
	)

	Describe("Error Recovery", func() {
		It("Should report first error with correct location", func() {
			prog := MustSucceed(parser.Parse(`
func test() {
	a := undefined1
	b := undefined2
}`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined1"))
			Expect(diag.Line).To(Equal(3))
		})
	})
})
