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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var channels = []symbol.Symbol{
	{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
	{Name: "ich", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 11},
}

func analyze(bCtx SpecContext, source string) diagnostics.Diagnostics {
	GinkgoHelper()
	prog := MustSucceed(parser.Parse(source))
	ctx := context.NewRoot(bCtx, prog, NewRoot(nil, channels...))
	analyzer.AnalyzeProgram(ctx)
	return *ctx.Diagnostics
}

func inFunc(body string) string { return "func f() {\n" + body + "\n}" }

func expectDiagnostic(
	diags diagnostics.Diagnostics,
	code diagnostics.ErrorCode,
	message string,
) {
	GinkgoHelper()
	Expect(diags).To(HaveLen(1), diags.String())
	Expect(diags[0].Code).To(Equal(code))
	Expect(diags[0].Message).To(Equal(message))
}

// bounds holds the decimal edges of an integer type.
type bounds struct {
	typ                      string
	belowMin, min, minPlus1  string
	maxMinus1, max, aboveMax string
}

var integerBounds = []bounds{
	{"i8", "-129", "-128", "-127", "126", "127", "128"},
	{"i16", "-32769", "-32768", "-32767", "32766", "32767", "32768"},
	{
		"i32",
		"-2147483649", "-2147483648", "-2147483647",
		"2147483646", "2147483647", "2147483648",
	},
	{
		"i64",
		"-9223372036854775809", "-9223372036854775808", "-9223372036854775807",
		"9223372036854775806", "9223372036854775807", "9223372036854775808",
	},
	{"u8", "-1", "0", "1", "254", "255", "256"},
	{"u16", "-1", "0", "1", "65534", "65535", "65536"},
	{"u32", "-1", "0", "1", "4294967294", "4294967295", "4294967296"},
	{
		"u64",
		"-1", "0", "1",
		"18446744073709551614", "18446744073709551615", "18446744073709551616",
	},
}

func boundEntries(build func(b bounds) TableEntry) []TableEntry {
	entries := make([]TableEntry, len(integerBounds))
	for i, b := range integerBounds {
		entries[i] = build(b)
	}
	return entries
}

var _ = Describe("Check", func() {
	DescribeTable(
		"operators",
		func(bCtx SpecContext, body string, code diagnostics.ErrorCode, msg string) {
			expectDiagnostic(analyze(bCtx, inFunc(body)), code, msg)
		},
		Entry("+", "x i8 := 100 + 28", codes.ConstantOverflow.Code,
			"constant 128 overflows i8"),
		Entry("-", "x i8 := -100 - 29", codes.ConstantOverflow.Code,
			"constant -129 overflows i8"),
		Entry("*", "x i8 := 16 * 8", codes.ConstantOverflow.Code,
			"constant 128 overflows i8"),
		Entry(
			"/ of the minimum by -1",
			"x i8 := -128 / -1",
			codes.ConstantOverflow.Code,
			"constant 128 overflows i8",
		),
		Entry("/ by zero", "x := 5 / 0", codes.DivisionByZero.Code,
			"integer division by zero"),
		Entry("% by zero", "x := 5 % 0", codes.DivisionByZero.Code,
			"integer modulo by zero"),
		Entry("unary - of the signed minimum", "x := -i8(-128)",
			codes.ConstantOverflow.Code, "constant 128 overflows i8"),
		Entry("unary - of an unsigned value", "x := -u8(1)",
			codes.ConstantOverflow.Code, "constant -1 overflows u8"),
		Entry("unary - of an unsigned literal", "x u8 := -1",
			codes.ConstantOverflow.Code, "constant -1 overflows u8"),
		Entry("^", "x i8 := 2 ^ 7", codes.ConstantOverflow.Code,
			"constant 128 overflows i8"),
		Entry("^ with a large exponent", "x := 2 ^ 100", codes.ConstantOverflow.Code,
			"constant 2^100 overflows i64"),
		Entry("a double-negated literal in a cast", "x := i8(--100) + i8(100)",
			codes.ConstantOverflow.Code, "constant 200 overflows i8"),
		Entry("an intermediate result", "x i8 := 100 + 100 - 100",
			codes.ConstantOverflow.Code, "constant 200 overflows i8"),
		Entry("i64 minimum divided by -1", "x := (-9223372036854775807 - 1) / -1",
			codes.ConstantOverflow.Code, "constant 9223372036854775808 overflows i64"),
	)

	DescribeTable("upper bound",
		func(bCtx SpecContext, b bounds) {
			Expect(analyze(bCtx, inFunc(fmt.Sprintf(
				"x %s := %s + 1", b.typ, b.maxMinus1,
			)))).To(BeEmpty())
			expectDiagnostic(
				analyze(bCtx, inFunc(fmt.Sprintf("x %s := %s + 1", b.typ, b.max))),
				codes.ConstantOverflow.Code,
				fmt.Sprintf("constant %s overflows %s", b.aboveMax, b.typ),
			)
		},
		boundEntries(func(b bounds) TableEntry { return Entry(b.typ, b) }),
	)

	DescribeTable("lower bound",
		func(bCtx SpecContext, b bounds) {
			Expect(analyze(bCtx, inFunc(fmt.Sprintf(
				"x %s := %s - 1", b.typ, b.minPlus1,
			)))).To(BeEmpty())
			expectDiagnostic(
				analyze(bCtx, inFunc(fmt.Sprintf("x %s := %s - 2", b.typ, b.minPlus1))),
				codes.ConstantOverflow.Code,
				fmt.Sprintf("constant %s overflows %s", b.belowMin, b.typ),
			)
		},
		boundEntries(func(b bounds) TableEntry { return Entry(b.typ, b) }),
	)

	DescribeTable("literal bounds",
		func(bCtx SpecContext, b bounds) {
			Expect(analyze(bCtx, inFunc(fmt.Sprintf(
				"x %s := %s\ny %s := %s", b.typ, b.min, b.typ, b.max,
			)))).To(BeEmpty())
			expectDiagnostic(
				analyze(bCtx, inFunc(fmt.Sprintf("x %s := %s", b.typ, b.aboveMax))),
				codes.ConstantOverflow.Code,
				fmt.Sprintf("constant %s overflows %s", b.aboveMax, b.typ),
			)
		},
		boundEntries(func(b bounds) TableEntry { return Entry(b.typ, b) }),
	)

	DescribeTable("positions",
		func(bCtx SpecContext, template string) {
			expectDiagnostic(
				analyze(bCtx, fmt.Sprintf(template, "100 + 100")),
				codes.ConstantOverflow.Code,
				"constant 200 overflows i8",
			)
			expectDiagnostic(
				analyze(bCtx, fmt.Sprintf(template, "1 / 0")),
				codes.DivisionByZero.Code,
				"integer division by zero",
			)
		},
		Entry("flow variable", "x i8 := %s"),
		Entry("function variable", inFunc("x i8 := %s")),
		Entry("stateful variable", inFunc("x i8 $= %s")),
		Entry("stage variable", `sequence main {
    stage a {
        x i8 := %s
        true => next
    }
}
true => main`),
		Entry("call input", `func g(a i8) {v f32} i8 {
    return a
}
ch -> g(a=%s)`),
		Entry("return value", "func f() i8 {\n    return %s\n}"),
	)

	It("Should check format string placeholders", func(bCtx SpecContext) {
		expectDiagnostic(
			analyze(bCtx, inFunc(`s := f"{i8(100) + i8(100)}"`)),
			codes.ConstantOverflow.Code,
			"constant 200 overflows i8",
		)
		expectDiagnostic(
			analyze(bCtx, inFunc(`s := f"{1 / 0}"`)),
			codes.DivisionByZero.Code,
			"integer division by zero",
		)
	})

	It("Should report each failed operation in one expression", func(bCtx SpecContext) {
		diags := analyze(bCtx, "func g(a i64) i64 {\n    return a\n}\n"+
			inFunc("x := (1 / 0) + g(2 % 0) + -(100 * 100000000000000000)"))
		Expect(diags).To(HaveLen(3), diags.String())
		Expect(diags[0].Message).To(Equal("integer division by zero"))
		Expect(diags[1].Message).To(Equal("integer modulo by zero"))
		Expect(diags[2].Message).To(Equal(
			"constant 10000000000000000000 overflows i64",
		))
	})

	DescribeTable("nested parentheses",
		func(bCtx SpecContext, body string, code diagnostics.ErrorCode, msg string) {
			expectDiagnostic(analyze(bCtx, inFunc(body)), code, msg)
		},
		Entry("division by zero", "x := (1 + (2 * (3 / (0))))",
			codes.DivisionByZero.Code, "integer division by zero"),
		Entry("overflow", "x i8 := ((64) + ((32 + (32))))",
			codes.ConstantOverflow.Code, "constant 128 overflows i8"),
		Entry("constant part of a non-constant expression", "x i32 := ich + (1 / 0)",
			codes.DivisionByZero.Code, "integer division by zero"),
	)

	DescribeTable("accepted expressions",
		func(bCtx SpecContext, body string) {
			Expect(analyze(bCtx, inFunc(body))).To(BeEmpty())
		},
		Entry("a power in range", "x i8 := 2 ^ 6"),
		Entry("a negative remainder", "x := 7 % -3"),
		Entry("nested parentheses in range", "x i8 := ((64) + ((31 + (32))))"),
		Entry("float division by zero", "x := 1.0 / 0.0"),
		Entry("integer literals with a float type", "x f32 := 1 / 0"),
		Entry("a large float power", "x := 2.0 ^ 2000.0"),
		Entry("a narrowing cast", "x := u8(300)"),
		Entry("a literal above i64 in a u64 cast", "x := u64(18446744073709551615)"),
		Entry("a negated literal in a cast", "x := i8(-128)"),
		Entry("a double-negated literal in a cast", "x := i8(--100) - i8(100)"),
		Entry("a channel read divided by zero", "x := ich / 0"),
		Entry("a channel read that can overflow", "x i32 := ich + 2147483647"),
		Entry("a variable divisor of zero", "y := 0\nx := 1 / y"),
	)
})
