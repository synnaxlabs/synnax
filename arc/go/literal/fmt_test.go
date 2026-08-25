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
				{
					Text:          "x",
					Spec:          ".2f",
					IsPlaceholder: true,
					Start:         0,
					End:           7,
					SpecOffset:    2,
				},
			}),
		Entry("placeholder with integer spec", "{n:d}",
			[]literal.FmtStrSegment{
				{
					Text:          "n",
					Spec:          "d",
					IsPlaceholder: true,
					Start:         0,
					End:           5,
					SpecOffset:    2,
				},
			}),
		Entry("placeholder with padded integer spec", "{n:05d}",
			[]literal.FmtStrSegment{
				{
					Text:          "n",
					Spec:          "05d",
					IsPlaceholder: true,
					Start:         0,
					End:           7,
					SpecOffset:    2,
				},
			}),
		Entry("placeholder with arithmetic expression", "{a + b}",
			[]literal.FmtStrSegment{
				{Text: "a + b", IsPlaceholder: true, Start: 0, End: 7, SpecOffset: -1},
			}),
		Entry("placeholder with rightmost colon splitting expr from spec",
			"{a:b:.2f}",
			[]literal.FmtStrSegment{
				{
					Text:          "a:b",
					Spec:          ".2f",
					IsPlaceholder: true,
					Start:         0,
					End:           9,
					SpecOffset:    4,
				},
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
				{
					Text:          "a",
					Spec:          ".2f",
					IsPlaceholder: true,
					Start:         2,
					End:           9,
					SpecOffset:    4,
				},
				{Text: " b=", Start: 9, End: 12, SpecOffset: -1},
				{
					Text:          "b",
					Spec:          "d",
					IsPlaceholder: true,
					Start:         12,
					End:           17,
					SpecOffset:    14,
				},
			}),
		Entry(`doubled opening brace is literal`, `{{`,
			[]literal.FmtStrSegment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
			}),
		Entry(`doubled closing brace is literal`, `}}`,
			[]literal.FmtStrSegment{
				{Text: "}", Start: 0, End: 2, SpecOffset: -1},
			}),
		Entry(`bare closing brace is literal`, `}`,
			[]literal.FmtStrSegment{
				{Text: "}", Start: 0, End: 1, SpecOffset: -1},
			}),
		Entry(`doubled open with doubled close`, `{{ }}`,
			[]literal.FmtStrSegment{
				{Text: "{ }", Start: 0, End: 5, SpecOffset: -1},
			}),
		Entry(`doubled braces around literal`, `{{hello}}`,
			[]literal.FmtStrSegment{
				{Text: "{hello}", Start: 0, End: 9, SpecOffset: -1},
			}),
		Entry(`doubled braces around placeholder`, `{{{x}}}`,
			[]literal.FmtStrSegment{
				{Text: "{", Start: 0, End: 2, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 2, End: 5, SpecOffset: -1},
				{Text: "}", Start: 5, End: 7, SpecOffset: -1},
			}),
		Entry(`doubled braces mixed with placeholder`,
			`pre {{ {x} }} post`,
			[]literal.FmtStrSegment{
				{Text: "pre { ", Start: 0, End: 7, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 7, End: 10, SpecOffset: -1},
				{Text: " } post", Start: 10, End: 18, SpecOffset: -1},
			}),
		Entry(`literal backslash before non-brace`, `a\nb`,
			[]literal.FmtStrSegment{
				{Text: `a\nb`, Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry(`literal backslash before close brace`, `a\}b`,
			[]literal.FmtStrSegment{
				{Text: `a\}b`, Start: 0, End: 4, SpecOffset: -1},
			}),
		Entry(`placeholder adjacent to bare close`, `{x}}`,
			[]literal.FmtStrSegment{
				{Text: "x", IsPlaceholder: true, Start: 0, End: 3, SpecOffset: -1},
				{Text: "}", Start: 3, End: 4, SpecOffset: -1},
			}),
		Entry(`doubled brace spanning newlines`,
			"line1 {{\nline2}}",
			[]literal.FmtStrSegment{
				{Text: "line1 {\nline2}", Start: 0, End: 16, SpecOffset: -1},
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
		Entry("empty placeholder", "{}",
			[]literal.FmtStrSegment{
				{Text: "", IsPlaceholder: true, Start: 0, End: 2, SpecOffset: -1},
			}),
		Entry("empty placeholder surrounded by text", "pre {} post",
			[]literal.FmtStrSegment{
				{Text: "pre ", Start: 0, End: 4, SpecOffset: -1},
				{Text: "", IsPlaceholder: true, Start: 4, End: 6, SpecOffset: -1},
				{Text: " post", Start: 6, End: 11, SpecOffset: -1},
			}),
		Entry("placeholder with balanced nested braces", "{f({a: 1})}",
			[]literal.FmtStrSegment{
				{
					Text:          "f({a: 1})",
					IsPlaceholder: true,
					Start:         0,
					End:           11,
					SpecOffset:    -1,
				},
			}),
		Entry("placeholder with balanced nested braces and spec",
			"{f({a: 1}):d}",
			[]literal.FmtStrSegment{
				{
					Text: "f({a: 1})", Spec: "d", IsPlaceholder: true,
					Start: 0, End: 13, SpecOffset: 10,
				},
			}),
		Entry("placeholder with index expression", "{arr[0]}",
			[]literal.FmtStrSegment{
				{Text: "arr[0]", IsPlaceholder: true, Start: 0, End: 8, SpecOffset: -1},
			}),
		Entry("placeholder with slice expression", "{arr[0:5]}",
			[]literal.FmtStrSegment{
				{
					Text:          "arr[0:5]",
					IsPlaceholder: true,
					Start:         0,
					End:           10,
					SpecOffset:    -1,
				},
			}),
		Entry("placeholder with slice expression and spec", "{arr[0:5]:s}",
			[]literal.FmtStrSegment{
				{
					Text: "arr[0:5]", Spec: "s", IsPlaceholder: true,
					Start: 0, End: 12, SpecOffset: 9,
				},
			}),
		// Adjacency tests: a literal backslash followed by a placeholder. The
		// expected behavior (matching Python's rf"...") is that the backslash
		// stays literal and the {expr} is interpolated. This is required for
		// Windows-style paths like rf"C:\logs\{name}.txt" to work.
		Entry(`backslash immediately before placeholder`, `\{x}`,
			[]literal.FmtStrSegment{
				{Text: `\`, Start: 0, End: 1, SpecOffset: -1},
				{Text: "x", IsPlaceholder: true, Start: 1, End: 4, SpecOffset: -1},
			}),
		Entry(`Windows path with placeholder after final backslash`,
			`C:\logs\{name}.txt`,
			[]literal.FmtStrSegment{
				{Text: `C:\logs\`, Start: 0, End: 8, SpecOffset: -1},
				{Text: "name", IsPlaceholder: true, Start: 8, End: 14, SpecOffset: -1},
				{Text: ".txt", Start: 14, End: 18, SpecOffset: -1},
			}),
	)

	DescribeTable("error cases",
		func(body, errSubstr string) {
			Expect(
				literal.FmtStrParse(body),
			).Error().
				To(MatchError(ContainSubstring(errSubstr)))
		},
		Entry("lone opening brace", "{", "unmatched '{'"),
		Entry("opening brace with body, no close", "{foo", "unmatched '{'"),
		Entry("opening brace with another opening inside", "{foo{bar}",
			"unmatched '{'"),
		Entry("opening brace before valid placeholder", "{ {x}",
			"unmatched '{'"),
		Entry("opening brace at end of literal", "literal {",
			"unmatched '{'"),
		Entry("placeholder starting with format spec", "{:.2f}",
			"expression before ':'"),
		Entry("placeholder with empty spec after colon", "{x:}",
			"format spec after ':' is empty"),
	)
})

var _ = Describe("StripQuotes", func() {
	DescribeTable("strips quotes and peels prefix",
		func(input, expectedBody string, expectedFlags literal.StringFlags) {
			body, flags, ok := literal.StripQuotes(input)
			Expect(ok).To(BeTrue())
			Expect(body).To(Equal(expectedBody))
			Expect(flags).To(Equal(expectedFlags))
		},
		Entry("plain double-quoted", `"hello"`, "hello", literal.StringFlags{}),
		Entry("empty double-quoted", `""`, "", literal.StringFlags{}),
		Entry("plain backtick", "`hello`", "hello",
			literal.StringFlags{Multi: true}),
		Entry("empty backtick", "``", "",
			literal.StringFlags{Multi: true}),
		Entry("backtick with newline", "`a\nb`", "a\nb",
			literal.StringFlags{Multi: true}),
		Entry("backtick body containing double quote", "`a\"b`", `a"b`,
			literal.StringFlags{Multi: true}),
		Entry("raw double-quoted", `r"path"`, "path",
			literal.StringFlags{Raw: true}),
		Entry("raw backtick", "r`path`", "path",
			literal.StringFlags{Raw: true, Multi: true}),
		Entry("format double-quoted", `f"hi {x}"`, "hi {x}",
			literal.StringFlags{Format: true}),
		Entry("format backtick", "f`hi {x}`", "hi {x}",
			literal.StringFlags{Format: true, Multi: true}),
		Entry("rf double-quoted", `rf"hi"`, "hi",
			literal.StringFlags{Raw: true, Format: true}),
		Entry("fr double-quoted", `fr"hi"`, "hi",
			literal.StringFlags{Raw: true, Format: true}),
		Entry("rf backtick", "rf`hi`", "hi",
			literal.StringFlags{Raw: true, Format: true, Multi: true}),
		Entry("body with embedded escape sequence", `"a\nb"`, `a\nb`,
			literal.StringFlags{}),
	)

	DescribeTable("rejects malformed input",
		func(input string) {
			body, flags, ok := literal.StripQuotes(input)
			Expect(ok).To(BeFalse())
			Expect(body).To(BeEmpty())
			Expect(flags).To(Equal(literal.StringFlags{}))
		},
		Entry("empty string", ""),
		Entry("single quote", `"`),
		Entry("missing leading quote", `hi"`),
		Entry("missing trailing quote", `"hi`),
		Entry("plain text no delimiters", "hello"),
		Entry("single backtick", "`"),
		Entry("duplicate r prefix", `rr"hi"`),
		Entry("duplicate f prefix", `ff"hi"`),
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
		{"i8", types.I8()},
		{"i16", types.I16()},
		{"i32", types.I32()},
		{"i64", types.I64()},
		{"u8", types.U8()},
		{"u16", types.U16()},
		{"u32", types.U32()},
		{"u64", types.U64()},
	}
	floatTypes := []namedType{{"f32", types.F32()}, {"f64", types.F64()}}
	stringType := namedType{"string", types.String()}

	var validArgs []any
	validArgs = append(validArgs, func(spec string, t types.Type) {
		Expect(literal.FmtStrValidateSpec(spec, t)).To(Succeed())
	})
	// Integer verbs across every integer type.
	for _, verb := range []string{"d", "b", "o", "O", "x", "X", "c"} {
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
	allTypes := append(
		append(append([]namedType{}, intTypes...), floatTypes...),
		stringType,
	)

	var invalidArgs []any
	invalidArgs = append(
		invalidArgs,
		func(spec string, t types.Type, errSubstr string) {
			Expect(literal.FmtStrValidateSpec(spec, t)).
				To(MatchError(ContainSubstring(errSubstr)))
		},
	)
	// Integer-only verbs rejected on every non-integer type.
	for _, verb := range []string{"d", "o", "O", "c", "b", "x", "X"} {
		for _, nt := range nonIntTypes {
			invalidArgs = append(invalidArgs,
				Entry(verb+" on "+nt.name, verb, nt.t, "invalid format spec"))
		}
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
	invalidArgs = append(
		invalidArgs,
		Entry("unknown verb on int", "z", types.I32(), "invalid format spec"),
		Entry("unknown verb on float", "z", types.F64(), "invalid format spec"),
		Entry(
			"blacklisted verb with flag on i32",
			"+v",
			types.I32(),
			"invalid format spec",
		),
		Entry(
			"blacklisted verb with width on f64",
			"5v",
			types.F64(),
			"invalid format spec",
		),
		Entry(
			"trailing chars after float verb",
			"f.2",
			types.F64(),
			"invalid format spec",
		),
		Entry(
			"trailing chars after integer verb",
			"d5",
			types.I32(),
			"invalid format spec",
		),
		Entry(
			"trailing chars after string verb",
			"sx",
			types.String(),
			"invalid format spec",
		),
		Entry("precision before width", ".2-5f", types.F64(), "invalid format spec"),
		Entry("flag after width", "5+d", types.I32(), "invalid format spec"),
		Entry("missing verb", "5", types.I32(), "invalid format spec"),
		Entry("only precision no verb", ".2", types.F64(), "invalid format spec"),
		Entry("precision without digits", ".f", types.F64(), "invalid format spec"),
	)
	DescribeTable("error cases", invalidArgs...)

	It("validates against the constraint of a constrained type variable", func() {
		intConstraint := types.IntegerConstraint()
		Expect(
			literal.FmtStrValidateSpec("d", types.Variable("T", &intConstraint)),
		).To(Succeed())
	})

	It("rejects a spec invalid for the variable's constraint", func() {
		stringConstraint := types.String()
		Expect(
			literal.FmtStrValidateSpec(".2f", types.Variable("T", &stringConstraint)),
		).
			To(MatchError(ContainSubstring("invalid format spec")))
	})

	It("errors on an unconstrained type variable", func() {
		Expect(literal.FmtStrValidateSpec("d", types.Variable("T", nil))).
			To(MatchError(ContainSubstring("cannot format type")))
	})

	It("errors on a non-formattable type kind", func() {
		Expect(literal.FmtStrValidateSpec("d", types.Chan(types.I32()))).
			To(MatchError(ContainSubstring("cannot format type")))
	})
})
