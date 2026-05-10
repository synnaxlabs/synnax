// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmtstring_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Parse", func() {
	DescribeTable("valid bodies",
		func(body string, expected []fmtstring.Segment) {
			Expect(MustSucceed(fmtstring.Parse(body))).To(Equal(expected))
		},
		Entry("empty body", "", []fmtstring.Segment(nil)),
		Entry("plain literal", "hello",
			[]fmtstring.Segment{
				{Text: "hello", Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry("literal with newlines", "line1\nline2\nline3",
			[]fmtstring.Segment{
				{Text: "line1\nline2\nline3", Start: 0, End: 17, SpecOffset: -1},
			}),
		Entry("literal with tabs and CR", "a\tb\rc",
			[]fmtstring.Segment{
				{Text: "a\tb\rc", Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry("literal with unicode", "héllo 世界 🚀",
			[]fmtstring.Segment{
				{Text: "héllo 世界 🚀", Start: 0, End: 18, SpecOffset: -1},
			}),
		Entry("literal with bare percent", "50% off",
			[]fmtstring.Segment{
				{Text: "50% off", Start: 0, End: 7, SpecOffset: -1},
			}),
		Entry("placeholder only", "{x}",
			[]fmtstring.Segment{
				{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
			}),
		Entry("literal then placeholder", "pre {x}", []fmtstring.Segment{
			{Text: "pre ", Start: 0, End: 4, SpecOffset: -1},
			{Text: "x", IsPlaceholder: true, Start: 4, End: 7, SpecOffset: -1},
		}),
		Entry("placeholder then literal", "{x} post", []fmtstring.Segment{
			{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
			{Text: " post", Start: 3, End: 8, SpecOffset: -1},
		}),
		Entry("literal surrounding placeholder", "pre {x} post",
			[]fmtstring.Segment{
				{Text: "pre ", Start: 0, End: 4, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 4, End: 7, SpecOffset: -1},
				{Text: " post", Start: 7, End: 12, SpecOffset: -1},
			}),
		Entry("two placeholders separated by literal", "{a} {b}",
			[]fmtstring.Segment{
				{Text: "a", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: " ", Start: 3, End: 4, SpecOffset: -1},
				{Text: "b", IsPlaceholder: true, Start: 4, End: 7, SpecOffset: -1},
			}),
		Entry("two adjacent placeholders", "{a}{b}",
			[]fmtstring.Segment{
				{Text: "a", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: "b", IsPlaceholder: true, Start: 3, End: 6, SpecOffset: -1},
			}),
		Entry("three placeholders mixed with literal", "x={x} y={y} z={z}",
			[]fmtstring.Segment{
				{Text: "x=", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
				{Text: " y=", Start: 5, End: 8, SpecOffset: -1},
				{Text: "y", IsPlaceholder: true, Start: 8, End: 11, SpecOffset: -1},
				{Text: " z=", Start: 11, End: 14, SpecOffset: -1},
				{Text: "z", IsPlaceholder: true, Start: 14, End: 17, SpecOffset: -1},
			}),
		Entry("placeholder spanning newlines in surrounding text",
			"line1\n{x}\nline2",
			[]fmtstring.Segment{
				{Text: "line1\n", Start: 0, End: 6, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 6, End: 9, SpecOffset: -1},
				{Text: "\nline2", Start: 9, End: 15, SpecOffset: -1},
			}),
		Entry("placeholder with float spec", "{x%.2f}",
			[]fmtstring.Segment{
				{Text: "x", Spec: ".2f", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: 2},
			}),
		Entry("placeholder with integer spec", "{n%d}",
			[]fmtstring.Segment{
				{Text: "n", Spec: "d", IsPlaceholder: true, Start: 0, End: 5, SpecOffset: 2},
			}),
		Entry("placeholder with padded integer spec", "{n%05d}",
			[]fmtstring.Segment{
				{Text: "n", Spec: "05d", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: 2},
			}),
		Entry("placeholder with arithmetic expression", "{a + b}",
			[]fmtstring.Segment{
				{Text: "a + b", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: -1},
			}),
		Entry("placeholder with modulo (whitespace around %)", "{a % b}",
			[]fmtstring.Segment{
				{Text: "a % b", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: -1},
			}),
		Entry("placeholder with modulo and trailing format spec", "{a % b%.2f}",
			[]fmtstring.Segment{
				{Text: "a % b", Spec: ".2f", IsPlaceholder: true, Start: 0, End: 11, SpecOffset: 6},
			}),
		Entry("placeholder with function call", "{len(x)}",
			[]fmtstring.Segment{
				{Text: "len(x)", IsPlaceholder: true, Start: 0, End: 8, SpecOffset: -1},
			}),
		Entry("placeholder with member access", "{a.b}",
			[]fmtstring.Segment{
				{Text: "a.b", IsPlaceholder: true, Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry("multiple placeholders each with spec",
			"a={a%.2f} b={b%d}",
			[]fmtstring.Segment{
				{Text: "a=", Start: 0, End: 2, SpecOffset: -1},
				{Text: "a", Spec: ".2f", IsPlaceholder: true, Start: 2, End: 9, SpecOffset: 4},
				{Text: " b=", Start: 9, End: 12, SpecOffset: -1},
				{Text: "b", Spec: "d", IsPlaceholder: true, Start: 12, End: 17, SpecOffset: 14},
			}),
		Entry(`escaped opening brace`, `\{`,
			[]fmtstring.Segment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
			}),
		Entry(`escaped closing brace`, `\}`,
			[]fmtstring.Segment{
				{Text: "}", Start: 0, End: 2, SpecOffset: -1},
			}),
		Entry(`both escaped braces`, `\{ \}`,
			[]fmtstring.Segment{
				{Text: "{ }", Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry(`escaped braces around literal`, `\{hello\}`,
			[]fmtstring.Segment{
				{Text: "{hello}", Start: 0, End: 9, SpecOffset: -1},
			}),
		Entry(`escaped braces around placeholder`, `\{{x}\}`,
			[]fmtstring.Segment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
				{Text: "}", Start: 5, End: 7, SpecOffset: -1},
			}),
		Entry(`escaped brace mixed with placeholder`, `pre \{ {x} \} post`,
			[]fmtstring.Segment{
				{Text: "pre { ", Start: 0, End: 7, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 7, End: 10, SpecOffset: -1},
				{Text: " } post", Start: 10, End: 18, SpecOffset: -1},
			}),
		Entry(`literal backslash before non-brace`, `a\nb`,
			[]fmtstring.Segment{
				{Text: `a\nb`, Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry(`escaped brace adjacent to placeholder`, `\{{x}`,
			[]fmtstring.Segment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
			}),
		Entry(`placeholder adjacent to escaped brace`, `{x}\}`,
			[]fmtstring.Segment{
				{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: "}", Start: 3, End: 5, SpecOffset: -1},
			}),
		Entry(`escaped braces spanning newlines`, "line1 \\{\nline2\\}",
			[]fmtstring.Segment{
				{Text: "line1 {\nline2}", Start: 0, End: 16, SpecOffset: -1},
			}),
	)

	DescribeTable("error cases",
		func(body, errSubstr string) {
			Expect(fmtstring.Parse(body)).Error().To(MatchError(ContainSubstring(errSubstr)))
		},
		Entry("lone closing brace", "}", "unmatched '}'"),
		Entry("closing brace as prefix", "}foo", "unmatched '}'"),
		Entry("closing brace as suffix", "foo}", "unmatched '}'"),
		Entry("closing brace in middle", "foo}bar", "unmatched '}'"),
		Entry("closing brace after placeholder", "{x}}", "unmatched '}'"),
		Entry("lone opening brace", "{", "unmatched '{'"),
		Entry("opening brace with body, no close", "{foo", "unmatched '{'"),
		Entry("opening brace with another opening inside", "{foo{bar}",
			"unmatched '{'"),
		Entry("opening brace before valid placeholder", "{ {x}",
			"unmatched '{'"),
		Entry("two opening braces in a row", "{{", "unmatched '{'"),
		Entry("opening brace at end of literal", "literal {",
			"unmatched '{'"),
		Entry("empty placeholder", "{}",
			"placeholder '{}' must contain an expression"),
		Entry("empty placeholder surrounded by text", "pre {} post",
			"placeholder '{}' must contain an expression"),
		Entry("placeholder starting with format spec", "{%.2f}",
			"expression before '%'"),
		Entry("placeholder with empty spec after percent", "{x%}",
			"format spec after '%' is empty"),
	)
})

var _ = Describe("StripDelimiters", func() {
	DescribeTable("strips matching backtick delimiters",
		func(input, expectedBody string) {
			body, ok := fmtstring.StripDelimiters(input)
			Expect(ok).To(BeTrue())
			Expect(body).To(Equal(expectedBody))
		},
		Entry("empty body", "``", ""),
		Entry("simple body", "`hello`", "hello"),
		Entry("body with placeholder", "`{x}`", "{x}"),
		Entry("body with embedded newline", "`a\nb`", "a\nb"),
		Entry("body with escaped backtick", "`a\\`b`", "a\\`b"),
	)

	DescribeTable("rejects malformed delimiters",
		func(input string) {
			body, ok := fmtstring.StripDelimiters(input)
			Expect(ok).To(BeFalse())
			Expect(body).To(BeEmpty())
		},
		Entry("empty string", ""),
		Entry("single backtick", "`"),
		Entry("missing leading backtick", "hi`"),
		Entry("missing trailing backtick", "`hi"),
		Entry("double-quoted string", `"hi"`),
		Entry("plain text no delimiters", "hello"),
	)
})

var _ = Describe("SplitSpec", func() {
	DescribeTable("valid splits",
		func(body, expectedExpr, expectedSpec string) {
			expr, spec := MustSucceed2(fmtstring.SplitSpec(body))
			Expect(expr).To(Equal(expectedExpr))
			Expect(spec).To(Equal(expectedSpec))
		},
		Entry("no percent", "x", "x", ""),
		Entry("identifier with float spec", "x%.2f", "x", ".2f"),
		Entry("identifier with integer spec", "n%d", "n", "d"),
		Entry("identifier with padded spec", "n%05d", "n", "05d"),
		Entry("modulo expression with whitespace", "a % b", "a % b", ""),
		Entry("expr with bare trailing percent and space", "a %", "a %", ""),
		Entry("expr with leading-space percent", "a% b", "a% b", ""),
		Entry("multiple percents picks rightmost",
			"a%b%c", "a%b", "c"),
		Entry("modulo plus trailing spec", "a % b%.2f", "a % b", ".2f"),
	)

	DescribeTable("error cases",
		func(body, errSubstr string) {
			Expect(fmtstring.SplitSpec(body)).Error().To(MatchError(ContainSubstring(errSubstr)))
		},
		Entry("lone percent", "%", "expression before '%'"),
		Entry("percent then spec only", "%d", "expression before '%'"),
		Entry("expr with bare trailing percent", "x%",
			"format spec after '%' is empty"),
	)
})

var _ = Describe("HasPlaceholder", func() {
	It("returns true when any segment is a placeholder", func() {
		segs := MustSucceed(fmtstring.Parse("hello {x}"))
		Expect(fmtstring.HasPlaceholder(segs)).To(BeTrue())
	})

	It("returns false when no segment is a placeholder", func() {
		segs := MustSucceed(fmtstring.Parse("plain literal"))
		Expect(fmtstring.HasPlaceholder(segs)).To(BeFalse())
	})

	It("returns false for an empty segment slice", func() {
		Expect(fmtstring.HasPlaceholder(nil)).To(BeFalse())
	})
})

var _ = Describe("ValidateSpec", func() {
	DescribeTable("valid specs",
		func(spec string, t types.Type) {
			Expect(fmtstring.ValidateSpec(spec, t)).To(Succeed())
		},
		Entry("float decimal on f64", ".2f", types.F64()),
		Entry("float exponential on f32", "e", types.F32()),
		Entry("float general on f64", "g", types.F64()),
		Entry("integer decimal on i32", "d", types.I32()),
		Entry("integer hex on u32", "x", types.U32()),
		Entry("integer octal on u64", "o", types.U64()),
		Entry("integer binary on i64", "b", types.I64()),
		Entry("padded integer on i32", "05d", types.I32()),
		Entry("signed flag on i32", "+d", types.I32()),
		Entry("decimal on i8", "d", types.I8()),
		Entry("decimal on u8", "d", types.U8()),
		Entry("decimal on i16", "d", types.I16()),
		Entry("decimal on u16", "d", types.U16()),
		Entry("type verb on i32", "T", types.I32()),
		Entry("type verb on f64", "T", types.F64()),
		Entry("type verb on string", "T", types.String()),
		Entry("default verb on i32", "v", types.I32()),
		Entry("default verb on string", "v", types.String()),
		Entry("string verb on string", "s", types.String()),
		Entry("quoted on string", "q", types.String()),
		Entry("hex of bytes on string", "x", types.String()),
		Entry("decimal on integer constant", "d", types.IntegerConstraint()),
		Entry("hex on integer constant", "05x", types.IntegerConstraint()),
		Entry("float on float constant", ".2f", types.FloatConstraint()),
		Entry("empty spec on string skips check", "", types.String()),
	)

	DescribeTable("error cases",
		func(spec string, t types.Type, errSubstr string) {
			err := fmtstring.ValidateSpec(spec, t)
			Expect(err).To(MatchError(ContainSubstring(errSubstr)))
		},
		Entry("unknown verb on int", "z", types.I32(), "invalid format spec"),
		Entry("unknown verb on float", "z", types.F64(), "invalid format spec"),
		Entry("integer-only verb on float", "d", types.F64(), "invalid format spec"),
		Entry("integer-only verb %o on float", "o", types.F64(), "invalid format spec"),
		Entry("decimal on string", "d", types.String(), "invalid format spec"),
		Entry("float verb on string", ".2f", types.String(), "invalid format spec"),
	)

	It("validates against the constraint of a constrained type variable", func() {
		intConstraint := types.IntegerConstraint()
		Expect(fmtstring.ValidateSpec("d", types.Variable("T", &intConstraint))).To(Succeed())
	})

	It("rejects a spec invalid for the variable's constraint", func() {
		stringConstraint := types.String()
		err := fmtstring.ValidateSpec(".2f", types.Variable("T", &stringConstraint))
		Expect(err).To(MatchError(ContainSubstring("invalid format spec")))
	})

	It("errors on an unconstrained type variable", func() {
		err := fmtstring.ValidateSpec("d", types.Variable("T", nil))
		Expect(err).To(MatchError(ContainSubstring("cannot format type")))
	})

	It("errors on a non-formattable type kind", func() {
		err := fmtstring.ValidateSpec("d", types.Chan(types.I32()))
		Expect(err).To(MatchError(ContainSubstring("cannot format type")))
	})
})
