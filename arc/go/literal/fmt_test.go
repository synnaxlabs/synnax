// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Parse", func() {
	DescribeTable("valid bodies",
		func(body string, expected []literal.FmtStrSegment) {
			Expect(MustSucceed(literal.FmtStrParse(body))).To(Equal(expected))
		},
		Entry("empty body", "", []literal.FmtStrSegment(nil)),
		Entry("plain literal", "hello",
			[]literal.FmtStrSegment{
				{Text: "hello", Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry("literal with newlines", "line1\nline2\nline3",
			[]literal.FmtStrSegment{
				{Text: "line1\nline2\nline3", Start: 0, End: 17, SpecOffset: -1},
			}),
		Entry("literal with tabs and CR", "a\tb\rc",
			[]literal.FmtStrSegment{
				{Text: "a\tb\rc", Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry("literal with unicode", "héllo 世界 🚀",
			[]literal.FmtStrSegment{
				{Text: "héllo 世界 🚀", Start: 0, End: 18, SpecOffset: -1},
			}),
		Entry("literal with bare percent", "50% off",
			[]literal.FmtStrSegment{
				{Text: "50% off", Start: 0, End: 7, SpecOffset: -1},
			}),
		Entry("placeholder only", "{x}",
			[]literal.FmtStrSegment{
				{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
			}),
		Entry("literal then placeholder", "pre {x}", []literal.FmtStrSegment{
			{Text: "pre ", Start: 0, End: 4, SpecOffset: -1},
			{Text: "x", IsPlaceholder: true, Start: 4, End: 7, SpecOffset: -1},
		}),
		Entry("placeholder then literal", "{x} post", []literal.FmtStrSegment{
			{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
			{Text: " post", Start: 3, End: 8, SpecOffset: -1},
		}),
		Entry("literal surrounding placeholder", "pre {x} post",
			[]literal.FmtStrSegment{
				{Text: "pre ", Start: 0, End: 4, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 4, End: 7, SpecOffset: -1},
				{Text: " post", Start: 7, End: 12, SpecOffset: -1},
			}),
		Entry("two placeholders separated by literal", "{a} {b}",
			[]literal.FmtStrSegment{
				{Text: "a", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: " ", Start: 3, End: 4, SpecOffset: -1},
				{Text: "b", IsPlaceholder: true, Start: 4, End: 7, SpecOffset: -1},
			}),
		Entry("two adjacent placeholders", "{a}{b}",
			[]literal.FmtStrSegment{
				{Text: "a", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: "b", IsPlaceholder: true, Start: 3, End: 6, SpecOffset: -1},
			}),
		Entry("three placeholders mixed with literal", "x={x} y={y} z={z}",
			[]literal.FmtStrSegment{
				{Text: "x=", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
				{Text: " y=", Start: 5, End: 8, SpecOffset: -1},
				{Text: "y", IsPlaceholder: true, Start: 8, End: 11, SpecOffset: -1},
				{Text: " z=", Start: 11, End: 14, SpecOffset: -1},
				{Text: "z", IsPlaceholder: true, Start: 14, End: 17, SpecOffset: -1},
			}),
		Entry("placeholder spanning newlines in surrounding text",
			"line1\n{x}\nline2",
			[]literal.FmtStrSegment{
				{Text: "line1\n", Start: 0, End: 6, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 6, End: 9, SpecOffset: -1},
				{Text: "\nline2", Start: 9, End: 15, SpecOffset: -1},
			}),
		Entry("placeholder with float spec", "{x:.2f}",
			[]literal.FmtStrSegment{
				{Text: "x", Spec: ".2f", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: 2},
			}),
		Entry("placeholder with integer spec", "{n:d}",
			[]literal.FmtStrSegment{
				{Text: "n", Spec: "d", IsPlaceholder: true, Start: 0, End: 5, SpecOffset: 2},
			}),
		Entry("placeholder with padded integer spec", "{n:05d}",
			[]literal.FmtStrSegment{
				{Text: "n", Spec: "05d", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: 2},
			}),
		Entry("placeholder with arithmetic expression", "{a + b}",
			[]literal.FmtStrSegment{
				{Text: "a + b", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: -1},
			}),
		Entry("placeholder with rightmost colon splitting expr from spec",
			"{a:b:.2f}",
			[]literal.FmtStrSegment{
				{Text: "a:b", Spec: ".2f", IsPlaceholder: true, Start: 0, End: 9, SpecOffset: 4},
			}),
		Entry("placeholder with function call", "{len(x)}",
			[]literal.FmtStrSegment{
				{Text: "len(x)", IsPlaceholder: true, Start: 0, End: 8, SpecOffset: -1},
			}),
		Entry("placeholder with member access", "{a.b}",
			[]literal.FmtStrSegment{
				{Text: "a.b", IsPlaceholder: true, Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry("multiple placeholders each with spec",
			"a={a:.2f} b={b:d}",
			[]literal.FmtStrSegment{
				{Text: "a=", Start: 0, End: 2, SpecOffset: -1},
				{Text: "a", Spec: ".2f", IsPlaceholder: true, Start: 2, End: 9, SpecOffset: 4},
				{Text: " b=", Start: 9, End: 12, SpecOffset: -1},
				{Text: "b", Spec: "d", IsPlaceholder: true, Start: 12, End: 17, SpecOffset: 14},
			}),
		Entry(`escaped opening brace`, `\{`,
			[]literal.FmtStrSegment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
			}),
		Entry(`bare closing brace is literal`, `}`,
			[]literal.FmtStrSegment{
				{Text: "}", Start: 0, End: 1, SpecOffset: -1},
			}),
		Entry(`escaped open with bare close`, `\{ }`,
			[]literal.FmtStrSegment{
				{Text: "{ }", Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry(`escaped open around literal with bare close`, `\{hello}`,
			[]literal.FmtStrSegment{
				{Text: "{hello}", Start: 0, End: 8, SpecOffset: -1},
			}),
		Entry(`escaped open and bare close around placeholder`, `\{{x}}`,
			[]literal.FmtStrSegment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
				{Text: "}", Start: 5, End: 6, SpecOffset: -1},
			}),
		Entry(`escaped open mixed with placeholder and bare close`,
			`pre \{ {x} } post`,
			[]literal.FmtStrSegment{
				{Text: "pre { ", Start: 0, End: 7, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 7, End: 10, SpecOffset: -1},
				{Text: " } post", Start: 10, End: 17, SpecOffset: -1},
			}),
		Entry(`literal backslash before non-brace`, `a\nb`,
			[]literal.FmtStrSegment{
				{Text: `a\nb`, Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry(`literal backslash before close brace`, `a\}b`,
			[]literal.FmtStrSegment{
				{Text: `a\}b`, Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry(`escaped brace adjacent to placeholder`, `\{{x}`,
			[]literal.FmtStrSegment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
			}),
		Entry(`placeholder adjacent to bare close`, `{x}}`,
			[]literal.FmtStrSegment{
				{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: "}", Start: 3, End: 4, SpecOffset: -1},
			}),
		Entry(`escaped open spanning newlines with bare close`,
			"line1 \\{\nline2}",
			[]literal.FmtStrSegment{
				{Text: "line1 {\nline2}", Start: 0, End: 15, SpecOffset: -1},
			}),
		Entry("bare close prefix", "}foo",
			[]literal.FmtStrSegment{
				{Text: "}foo", Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry("bare close suffix", "foo}",
			[]literal.FmtStrSegment{
				{Text: "foo}", Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry("bare close in middle", "foo}bar",
			[]literal.FmtStrSegment{
				{Text: "foo}bar", Start: 0, End: 7, SpecOffset: -1},
			}),
	)

	DescribeTable("error cases",
		func(body, errSubstr string) {
			Expect(literal.FmtStrParse(body)).Error().To(MatchError(ContainSubstring(errSubstr)))
		},
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
		Entry("placeholder starting with format spec", "{:.2f}",
			"expression before ':'"),
		Entry("placeholder with empty spec after colon", "{x:}",
			"format spec after ':' is empty"),
	)
})

var _ = Describe("StripDelimiters", func() {
	DescribeTable("strips matching backtick delimiters",
		func(input, expectedBody string) {
			body, ok := literal.FmtStrStripDelimiters(input)
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
			body, ok := literal.FmtStrStripDelimiters(input)
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
			expr, spec := MustSucceed2(literal.FmtStrSplitSpec(body))
			Expect(expr).To(Equal(expectedExpr))
			Expect(spec).To(Equal(expectedSpec))
		},
		Entry("no colon", "x", "x", ""),
		Entry("identifier with float spec", "x:.2f", "x", ".2f"),
		Entry("identifier with integer spec", "n:d", "n", "d"),
		Entry("identifier with padded spec", "n:05d", "n", "05d"),
		Entry("multiple colons picks rightmost",
			"a:b:c", "a:b", "c"),
	)

	DescribeTable("error cases",
		func(body, errSubstr string) {
			Expect(literal.FmtStrSplitSpec(body)).Error().To(MatchError(ContainSubstring(errSubstr)))
		},
		Entry("lone colon", ":", "expression before ':'"),
		Entry("colon then spec only", ":d", "expression before ':'"),
		Entry("expr with bare trailing colon", "x:",
			"format spec after ':' is empty"),
	)
})

var _ = Describe("HasPlaceholder", func() {
	It("returns true when any segment is a placeholder", func() {
		segs := MustSucceed(literal.FmtStrParse("hello {x}"))
		Expect(literal.FmtStrHasPlaceholder(segs)).To(BeTrue())
	})

	It("returns false when no segment is a placeholder", func() {
		segs := MustSucceed(literal.FmtStrParse("plain literal"))
		Expect(literal.FmtStrHasPlaceholder(segs)).To(BeFalse())
	})

	It("returns false for an empty segment slice", func() {
		Expect(literal.FmtStrHasPlaceholder(nil)).To(BeFalse())
	})
})

var _ = Describe("ValidateSpec", func() {
	type namedType struct {
		name string
		t    types.Type
	}
	intTypes := []namedType{
		{"i8", types.I8()}, {"i16", types.I16()},
		{"i32", types.I32()}, {"i64", types.I64()},
		{"u8", types.U8()}, {"u16", types.U16()},
		{"u32", types.U32()}, {"u64", types.U64()},
	}
	floatTypes := []namedType{{"f32", types.F32()}, {"f64", types.F64()}}
	stringType := namedType{"string", types.String()}

	var validArgs []any
	validArgs = append(validArgs, func(spec string, t types.Type) {
		Expect(literal.FmtStrValidateSpec(spec, t)).To(Succeed())
	})
	// Integer verbs across every integer type.
	for _, verb := range []string{"d", "b", "o", "x", "X", "c"} {
		for _, it := range intTypes {
			validArgs = append(validArgs, Entry(verb+" on "+it.name, verb, it.t))
		}
	}
	// Float verbs across every float type.
	for _, verb := range []string{"f", "e", "E", "g", "G"} {
		for _, ft := range floatTypes {
			validArgs = append(validArgs, Entry(verb+" on "+ft.name, verb, ft.t))
		}
	}
	// Go's fmt accepts x and b on floats (hex/binary scientific form); pin so a
	// future validator change cannot regress.
	for _, verb := range []string{"x", "b"} {
		for _, ft := range floatTypes {
			validArgs = append(validArgs,
				Entry(verb+" on "+ft.name+" (Go-fmt cross-type)", verb, ft.t))
		}
	}
	// String verbs.
	for _, verb := range []string{"s", "q"} {
		validArgs = append(validArgs,
			Entry(verb+" on "+stringType.name, verb, stringType.t))
	}
	// Flags, width, precision, and constants.
	validArgs = append(validArgs,
		Entry("padded decimal on i32", "05d", types.I32()),
		Entry("signed decimal on i32", "+d", types.I32()),
		Entry("decimal on integer constant", "d", types.IntegerConstraint()),
		Entry("hex on integer constant", "05x", types.IntegerConstraint()),
		Entry("float on float constant", ".2f", types.FloatConstraint()),
		Entry("empty spec on string skips check", "", types.String()),
	)
	DescribeTable("valid specs", validArgs...)

	nonIntTypes := append(append([]namedType{}, floatTypes...), stringType)
	nonFloatTypes := append(append([]namedType{}, intTypes...), stringType)
	nonStringTypes := append(append([]namedType{}, intTypes...), floatTypes...)
	allTypes := append(append(append([]namedType{}, intTypes...), floatTypes...), stringType)

	var invalidArgs []any
	invalidArgs = append(invalidArgs, func(spec string, t types.Type, errSubstr string) {
		err := literal.FmtStrValidateSpec(spec, t)
		Expect(err).To(MatchError(ContainSubstring(errSubstr)))
	})
	// Integer-only verbs rejected on every non-integer type.
	for _, verb := range []string{"d", "o", "c"} {
		for _, nt := range nonIntTypes {
			invalidArgs = append(invalidArgs,
				Entry(verb+" on "+nt.name, verb, nt.t, "invalid format spec"))
		}
	}
	// b, x, X are valid on int and float, but rejected on string (x/X blocked).
	for _, verb := range []string{"b", "x", "X"} {
		invalidArgs = append(invalidArgs,
			Entry(verb+" on "+stringType.name, verb, stringType.t, "invalid format spec"))
	}
	// Float verbs rejected on every non-float type.
	for _, verb := range []string{"f", "e", "E", "g", "G"} {
		for _, nt := range nonFloatTypes {
			invalidArgs = append(invalidArgs,
				Entry(verb+" on "+nt.name, verb, nt.t, "invalid format spec"))
		}
	}
	// String verbs rejected on every non-string type.
	for _, verb := range []string{"s", "q"} {
		for _, nt := range nonStringTypes {
			invalidArgs = append(invalidArgs,
				Entry(verb+" on "+nt.name, verb, nt.t, "invalid format spec"))
		}
	}
	// Blacklisted verbs rejected on every type.
	for _, verb := range []string{"v", "T", "U"} {
		for _, nt := range allTypes {
			invalidArgs = append(invalidArgs,
				Entry(verb+" on "+nt.name, verb, nt.t, "invalid format spec"))
		}
	}
	// Spec-shape and malformed-spec error cases.
	invalidArgs = append(invalidArgs,
		Entry("unknown verb on int", "z", types.I32(), "invalid format spec"),
		Entry("unknown verb on float", "z", types.F64(), "invalid format spec"),
		Entry("blacklisted verb with flag on i32", "+v", types.I32(), "invalid format spec"),
		Entry("blacklisted verb with width on f64", "5v", types.F64(), "invalid format spec"),
		Entry("trailing chars after float verb", "f.2", types.F64(), "invalid format spec"),
		Entry("trailing chars after integer verb", "d5", types.I32(), "invalid format spec"),
		Entry("trailing chars after string verb", "sx", types.String(), "invalid format spec"),
		Entry("precision before width", ".2-5f", types.F64(), "invalid format spec"),
		Entry("flag after width", "5+d", types.I32(), "invalid format spec"),
		Entry("missing verb", "5", types.I32(), "invalid format spec"),
		Entry("only precision no verb", ".2", types.F64(), "invalid format spec"),
		Entry("precision without digits", ".f", types.F64(), "invalid format spec"),
	)
	DescribeTable("error cases", invalidArgs...)

	It("validates against the constraint of a constrained type variable", func() {
		intConstraint := types.IntegerConstraint()
		Expect(literal.FmtStrValidateSpec("d", types.Variable("T", &intConstraint))).To(Succeed())
	})

	It("rejects a spec invalid for the variable's constraint", func() {
		stringConstraint := types.String()
		err := literal.FmtStrValidateSpec(".2f", types.Variable("T", &stringConstraint))
		Expect(err).To(MatchError(ContainSubstring("invalid format spec")))
	})

	It("errors on an unconstrained type variable", func() {
		err := literal.FmtStrValidateSpec("d", types.Variable("T", nil))
		Expect(err).To(MatchError(ContainSubstring("cannot format type")))
	})

	It("errors on a non-formattable type kind", func() {
		err := literal.FmtStrValidateSpec("d", types.Chan(types.I32()))
		Expect(err).To(MatchError(ContainSubstring("cannot format type")))
	})
})
