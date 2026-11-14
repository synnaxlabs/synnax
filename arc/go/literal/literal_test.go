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
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// Helper to extract literal from expression text
func getLiteral(text string) parser.ILiteralContext {
	GinkgoHelper()
	expr := MustSucceed(parser.ParseExpression(text))
	return parser.GetLiteral(expr)
}

// Helper to parse a numeric literal from text
func parseNumeric(text string, targetType types.Type) (literal.ParsedValue, error) {
	GinkgoHelper()
	lit := getLiteral(text)
	numLit := lit.NumericLiteral()
	return literal.ParseNumeric(numLit, targetType)
}

var _ = Describe("Literal Parser", func() {
	// Note: Negative numbers like "-1" are unary expressions, not literals.
	// They should be tested through the full analyzer, not the literal parser.

	DescribeTable(
		"Numeric literal parsing",
		func(
			input string,
			targetType types.Type,
			shouldSucceed bool,
			expectedValue any,
			expectedType types.Type,
			errorSubstring string,
		) {
			parsed, err := parseNumeric(input, targetType)
			if shouldSucceed {
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(expectedValue))
				Expect(parsed.Type).To(Equal(expectedType))
			} else {
				Expect(err).To(MatchError(ContainSubstring(errorSubstring)))
			}
		},
		// i8 tests
		Entry("i8 max value", "127", types.I8(), true, int8(127), types.I8(), ""),
		Entry(
			"i8 out of range",
			"128",
			types.I8(),
			false,
			nil,
			types.Type{},
			"out of range for i8",
		),
		Entry("i8 zero", "0", types.I8(), true, int8(0), types.I8(), ""),
		// i16 tests
		Entry(
			"i16 max value",
			"32767",
			types.I16(),
			true,
			int16(32767),
			types.I16(),
			"",
		),
		Entry(
			"i16 out of range",
			"32768",
			types.I16(),
			false,
			nil,
			types.Type{},
			"out of range for i16",
		),
		// i32 tests
		Entry(
			"i32 max value",
			"2147483647",
			types.I32(),
			true,
			int32(2147483647),
			types.I32(),
			"",
		),
		// i64 tests
		Entry("i64 typical value", "42", types.I64(), true, int64(42), types.I64(), ""),
		// u8 tests
		Entry("u8 max value", "255", types.U8(), true, uint8(255), types.U8(), ""),
		Entry("u8 zero", "0", types.U8(), true, uint8(0), types.U8(), ""),
		// u16 tests
		Entry(
			"u16 max value",
			"65535",
			types.U16(),
			true,
			uint16(65535),
			types.U16(),
			"",
		),
		// u32 tests
		Entry(
			"u32 typical value",
			"100",
			types.U32(),
			true,
			uint32(100),
			types.U32(),
			"",
		),
		// u64 tests
		Entry(
			"u64 typical value",
			"100",
			types.U64(),
			true,
			uint64(100),
			types.U64(),
			"",
		),
		// Float tests
		Entry("f32 literal", "3.14", types.F32(), true, float32(3.14), types.F32(), ""),
		Entry("f64 literal", "3.14159", types.F64(), true, 3.14159, types.F64(), ""),
		// Value type inference
		Entry(
			"integer defaults to i64",
			"42",
			types.Type{},
			true,
			int64(42),
			types.I64(),
			"",
		),
		Entry(
			"float defaults to f64",
			"3.14",
			types.Type{},
			true,
			3.14,
			types.F64(),
			"",
		),
		// Float to integer conversions (exact)
		Entry(
			"exact float to i32 (3.0)",
			"3.0",
			types.I32(),
			true,
			int32(3),
			types.I32(),
			"",
		),
		Entry(
			"exact float to i32 (0.0)",
			"0.0",
			types.I32(),
			true,
			int32(0),
			types.I32(),
			"",
		),
		// Float to integer conversions (non-exact - should fail)
		Entry(
			"non-exact float to i32 (3.14)",
			"3.14",
			types.I32(),
			false,
			nil,
			types.Type{},
			"cannot convert non-integer float",
		),
		Entry(
			"non-exact float to u8 (3.5)",
			"3.5",
			types.U8(),
			false,
			nil,
			types.Type{},
			"cannot convert non-integer float",
		),
		// Integer to float conversions
		Entry("int to f32", "42", types.F32(), true, float32(42), types.F32(), ""),
		Entry("int to f64", "42", types.F64(), true, float64(42), types.F64(), ""),
	)

	Describe("Unit literals", func() {
		Context("With no target type", func() {
			DescribeTable("Should keep the number as written",
				func(text string, expected any, kind types.Kind) {
					parsed := MustSucceed(literal.Parse(getLiteral(text), types.Type{}))
					Expect(parsed.Value).To(Equal(expected))
					Expect(parsed.Type.Kind).To(Equal(kind))
					Expect(parsed.Type.Unit).ToNot(BeNil())
				},
				Entry("an integer duration", "10s", telem.TimeSpan(10), types.KindI64),
				Entry("an integer mass", "100kg", int64(100), types.KindI64),
				Entry("an integer pressure", "100psi", int64(100), types.KindI64),
				Entry("a float length", "5.0km", 5.0, types.KindF64),
				Entry("a float mass", "100.5kg", 100.5, types.KindF64),
			)
		})

		Context("With target type having a unit (scale conversion)", func() {
			It("Should convert 300ms to TimeSpan (nanoseconds)", func() {
				lit := getLiteral("300ms")
				parsed := MustSucceed(literal.Parse(lit, types.TimeSpan()))
				// 300ms = 300 * (1e-3 / 1e-9) = 300,000,000 ns
				Expect(parsed.Value).To(Equal(telem.Millisecond * 300))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should convert 5s to TimeSpan (nanoseconds)", func() {
				lit := getLiteral("5s")
				parsed := MustSucceed(literal.Parse(lit, types.TimeSpan()))
				// 5s = 5 * (1 / 1e-9) = 5,000,000,000 ns
				Expect(parsed.Value).To(Equal(telem.Second * 5))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should convert 1us to TimeSpan (nanoseconds)", func() {
				lit := getLiteral("1us")
				parsed := MustSucceed(literal.Parse(lit, types.TimeSpan()))
				// 1us = 1 * (1e-6 / 1e-9) = 1000 ns
				Expect(parsed.Value).To(Equal(telem.Microsecond))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})
		})

		Context("With a type variable target", func() {
			DescribeTable(
				"Should give the kind of the constraint",
				func(text string, constraint types.Type, expected any) {
					lit := getLiteral(text)
					name := lit.NumericLiteral().IDENTIFIER().GetText()
					target := types.Variable("lit", &constraint)
					target.Unit = MustBeOk(units.Resolve(name))
					Expect(
						MustSucceed(literal.Parse(lit, target)).Value,
					).To(Equal(expected))
				},
				Entry("integer", "10khz", types.IntegerConstraint(), int64(10)),
				Entry(
					"integer duration",
					"5s",
					types.IntegerConstraint(),
					telem.TimeSpan(5),
				),
				Entry(
					"exact integer float",
					"2.0khz",
					types.ExactIntegerFloatConstraint(),
					float64(2),
				),
				Entry("float", "14.7psi", types.FloatConstraint(), 14.7),
			)
		})

		Context("With a target with no unit", func() {
			DescribeTable("Should keep the number as written",
				func(text string, target types.Type, expected any) {
					expr := MustSucceed(parser.ParseExpression(text))
					Expect(MustSucceed(literal.ParseConst(expr, target)).Value).
						To(Equal(expected))
				},
				Entry("a float", "14.7psi", types.F64(), 14.7),
				Entry("a narrowed float", "14.7psi", types.F32(), float32(14.7)),
				Entry("an integer", "100psi", types.I32(), int32(100)),
				Entry("a duration", "5s", types.I64(), telem.TimeSpan(5)),
				Entry("a negated duration", "-5s", types.I64(), telem.TimeSpan(-5)),
				Entry("an unsigned integer", "5khz", types.U64(), uint64(5)),
			)

			It("Should return an error for a float into an integer", func() {
				expr := MustSucceed(parser.ParseExpression("14.7psi"))
				Expect(literal.ParseConst(expr, types.I64())).
					Error().To(MatchError(ContainSubstring("fractional part")))
			})
		})

		Context("With an integer target in another unit", func() {
			DescribeTable("Should drop the part of a unit that the target cannot hold",
				func(text, unit string, expected any, truncated bool) {
					target := types.I64()
					target.Unit = MustBeOk(units.Resolve(unit))
					parsed := MustSucceed(literal.Parse(getLiteral(text), target))
					Expect(parsed.Value).To(Equal(expected))
					Expect(parsed.Truncated).To(Equal(truncated))
				},
				Entry("a larger unit with a remainder",
					"90s", "min", telem.TimeSpan(1), true),
				Entry("a larger unit with no remainder",
					"120s", "min", telem.TimeSpan(2), false),
				Entry("a factor that is not whole", "5psi", "Pa", int64(34473), true),
				Entry("a whole factor", "5s", "ms", telem.TimeSpan(5000), false),
				Entry("a float that a whole factor makes whole",
					"1.5s", "ms", telem.TimeSpan(1500), false),
			)

			It("Should return an error for a float that a whole factor leaves in parts",
				func() {
					target := types.I64()
					target.Unit = MustBeOk(units.Resolve("ms"))
					Expect(literal.Parse(getLiteral("1.0005s"), target)).
						Error().To(MatchError(ContainSubstring("fractional part")))
				})
		})

		Context(
			"ParseNumeric Go-like constant semantics (error on fractional)",
			func() {
				It("Should error when converting non-exact value to i64", func() {
					lit := getLiteral("100.5kg")
					Expect(literal.ParseNumeric(lit.NumericLiteral(), types.I64())).
						Error().To(MatchError(ContainSubstring("fractional part")))
				})

				It("Should succeed when converting exact float literal to i32", func() {
					lit := getLiteral("1.0km")
					parsed := MustSucceed(
						literal.ParseNumeric(lit.NumericLiteral(), types.I32()),
					)
					Expect(parsed.Value).To(Equal(int32(1)))
				})
			},
		)

		Context("Error cases", func() {
			It("Should return error for unknown units", func() {
				lit := getLiteral("5foobar")
				Expect(literal.Parse(lit, types.Type{})).
					Error().To(MatchError(ContainSubstring("unknown unit")))
			})

			It("Should return error for range overflow", func() {
				lit := getLiteral("300psi")
				Expect(literal.Parse(lit, types.I8())).
					Error().To(MatchError(ContainSubstring("out of range")))
			})
		})
	})

	Describe("String literals", func() {
		It("Should parse simple string", func() {
			lit := getLiteral(`"hello"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("hello"))
			Expect(parsed.Type).To(Equal(types.String()))
		})

		It("Should parse empty string", func() {
			lit := getLiteral(`""`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal(""))
			Expect(parsed.Type).To(Equal(types.String()))
		})

		It("Should parse string with spaces", func() {
			lit := getLiteral(`"hello world"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("hello world"))
		})

		It("Should handle newline escape", func() {
			lit := getLiteral(`"line1\nline2"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("line1\nline2"))
		})

		It("Should handle tab escape", func() {
			lit := getLiteral(`"col1\tcol2"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("col1\tcol2"))
		})

		It("Should handle carriage return escape", func() {
			lit := getLiteral(`"line1\rline2"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("line1\rline2"))
		})

		It("Should handle escaped quote", func() {
			lit := getLiteral(`"say \"hello\""`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal(`say "hello"`))
		})

		It("Should handle escaped backslash", func() {
			lit := getLiteral(`"path\\to\\file"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal(`path\to\file`))
		})

		It("Should handle unicode escape", func() {
			lit := getLiteral(`"A\u0042C"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("ABC"))
		})

		It("Should infer string type when no target type specified", func() {
			lit := getLiteral(`"hello"`)
			parsed := MustSucceed(literal.Parse(lit, types.Type{}))
			Expect(parsed.Value).To(Equal("hello"))
			Expect(parsed.Type).To(Equal(types.String()))
		})

		It("Should return error when assigning string to non-string type", func() {
			lit := getLiteral(`"hello"`)
			Expect(literal.Parse(lit, types.I32())).
				Error().To(MatchError(ContainSubstring("cannot assign string to")))
		})

		It(
			"Should return error for malformed string literal (missing closing quote)",
			func() {
				Expect(
					literal.ParseString(`"hello`, types.String()),
				).Error().To(MatchError(ContainSubstring("invalid string literal")))
			},
		)
	})

	Describe("Raw and multi-line string literals", func() {
		DescribeTable(
			"ParseString happy path",
			func(input string, target types.Type, expected string) {
				parsed := MustSucceed(literal.ParseString(input, target))
				Expect(parsed.Value).To(Equal(expected))
				Expect(parsed.Type).To(Equal(types.String()))
			},
			Entry("empty raw", `r""`, types.String(), ""),
			Entry("simple raw", `r"hello"`, types.String(), "hello"),
			Entry("raw with spaces", `r"hello world"`, types.String(), "hello world"),
			Entry("raw backslash-n verbatim", `r"a\nb"`, types.String(), `a\nb`),
			Entry(
				"raw backslash-t verbatim",
				`r"col1\tcol2"`,
				types.String(),
				`col1\tcol2`,
			),
			Entry(
				"raw escaped quote preserves backslash",
				`r"\""`,
				types.String(),
				`\"`,
			),
			Entry(
				"raw windows path",
				`r"C:\Users\path"`,
				types.String(),
				`C:\Users\path`,
			),
			Entry("empty multi", "``", types.String(), ""),
			Entry("simple multi", "`hello`", types.String(), "hello"),
			Entry(
				"multi with real newline",
				"`line1\nline2`",
				types.String(),
				"line1\nline2",
			),
			Entry("multi three-line literal", "`a\nb\nc`", types.String(), "a\nb\nc"),
			Entry(
				"multi indentation preserved",
				"`a\n    b`",
				types.String(),
				"a\n    b",
			),
			Entry("multi tab char preserved", "`a\tb`", types.String(), "a\tb"),
			Entry("multi unicode", "`°C`", types.String(), "°C"),
			Entry(
				"multi processes escape sequences",
				"`a\\nb`",
				types.String(),
				"a\nb",
			),
			Entry("raw multi verbatim escapes", "r`a\\nb`", types.String(), `a\nb`),
			Entry(
				"raw multi with real newline",
				"r`line1\nline2`",
				types.String(),
				"line1\nline2",
			),
			Entry("no target type infers string from raw", `r"hi"`, types.Type{}, "hi"),
			Entry(
				"format single preserves placeholders in body",
				`f"hi {x}"`,
				types.String(),
				"hi {x}",
			),
			Entry(
				"format single processes standard escapes",
				`f"a\nb {x}"`,
				types.String(),
				"a\nb {x}",
			),
			Entry(
				"format multi preserves placeholders in body",
				"f`v={x}\nt={t}`",
				types.String(),
				"v={x}\nt={t}",
			),
			Entry(
				"format multi processes escapes outside placeholders",
				"f`a\\tb {x}`",
				types.String(),
				"a\tb {x}",
			),
			Entry(
				"rf preserves placeholder and backslash verbatim",
				`rf"C:\path\{x}"`,
				types.String(),
				`C:\path\{x}`,
			),
			Entry(
				"fr (order-flipped) behaves identically to rf",
				`fr"hi {x}"`,
				types.String(),
				`hi {x}`,
			),
			Entry(
				"rf multi preserves placeholder, real newline, and backslash",
				"rf`v={x}\nraw=\\n`",
				types.String(),
				"v={x}\nraw=\\n",
			),
		)

		DescribeTable(
			"ParseString errors",
			func(input string, target types.Type, errSubstring string) {
				Expect(literal.ParseString(input, target)).
					Error().To(MatchError(ContainSubstring(errSubstring)))
			},
			Entry(
				"non-string target type",
				`r"hi"`,
				types.I32(),
				"cannot assign string to",
			),
			Entry(
				"missing leading quote",
				`hi"`,
				types.String(),
				"invalid string literal",
			),
			Entry(
				"missing trailing quote",
				`"hi`,
				types.String(),
				"invalid string literal",
			),
			Entry(
				"single quote too short",
				`"`,
				types.String(),
				"invalid string literal",
			),
			Entry("empty text too short", "", types.String(), "invalid string literal"),
		)

		Describe("Parse AST routing", func() {
			It("Should route raw string literal through ParseString", func() {
				lit := getLiteral(`r"hello"`)
				parsed := MustSucceed(literal.Parse(lit, types.String()))
				Expect(parsed.Value).To(Equal("hello"))
				Expect(parsed.Type).To(Equal(types.String()))
			})

			It("Should preserve real newlines in multi-line literal", func() {
				lit := getLiteral("`a\nb`")
				parsed := MustSucceed(literal.Parse(lit, types.String()))
				Expect(parsed.Value).To(Equal("a\nb"))
			})

			It("Should infer string type when no target type specified", func() {
				lit := getLiteral(`r"hi"`)
				parsed := MustSucceed(literal.Parse(lit, types.Type{}))
				Expect(parsed.Value).To(Equal("hi"))
				Expect(parsed.Type).To(Equal(types.String()))
			})

			It("Should still route regular string literals to ParseString", func() {
				lit := getLiteral(`"hello\n"`)
				parsed := MustSucceed(literal.Parse(lit, types.String()))
				Expect(parsed.Value).To(Equal("hello\n"))
			})
		})
	})

	Describe("Series literals", func() {
		It(
			"Should return error for series literals (not supported for default values)",
			func() {
				lit := getLiteral("[1, 2, 3]")
				Expect(literal.Parse(lit, types.Series(types.I64()))).
					Error().
					To(MatchError(ContainSubstring("series literals not supported for default values")))
			},
		)
	})

	Describe("Negate", func() {
		DescribeTable("should negate numeric values",
			func(text string, target types.Type, expected any) {
				parsed := MustSucceed(
					literal.ParseConst(
						MustSucceed(parser.ParseExpression(text)),
						target,
					),
				)
				Expect(parsed.Value).To(Equal(expected))
			},
			Entry("int8", "-42", types.I8(), int8(-42)),
			Entry("int16", "-1000", types.I16(), int16(-1000)),
			Entry("int32", "-100000", types.I32(), int32(-100000)),
			Entry("int64", "-5000000000", types.I64(), int64(-5000000000)),
			Entry("float32", "-3.14", types.F32(), float32(-3.14)),
			Entry("float64", "-2.718", types.F64(), float64(-2.718)),
			Entry("TimeSpan", "-1s", types.I64(), telem.TimeSpan(-1)),
			Entry("zero int64", "-0", types.I64(), int64(0)),
			Entry("zero float64", "-0.0", types.F64(), float64(0)),
		)

		It("Should return non-numeric types unchanged", func() {
			parsed := MustSucceed(
				literal.ParseConst(
					MustSucceed(parser.ParseExpression(`"hello"`)),
					types.String(),
				),
			)
			Expect(parsed.Value).To(Equal("hello"))
		})
	})

	Describe("IsExactInteger", func() {
		DescribeTable("should correctly identify exact integers",
			func(value float64, expected bool) {
				Expect(literal.IsExactInteger(value)).To(Equal(expected))
			},
			Entry("exact integer 5.0", 5.0, true),
			Entry("exact integer 0.0", 0.0, true),
			Entry("large exact integer", 1000000.0, true),
			Entry("non-integer 5.5", 5.5, false),
			Entry("non-integer 3.14", 3.14, false),
			Entry("near-integer within tolerance", 5.0+1e-12, true),
			Entry("near-zero within tolerance", 1e-12, true),
			Entry("near-zero outside tolerance", 1e-6, false),
		)
	})

	Describe("ParseConst", func() {
		parse := func(text string, target types.Type) (literal.ParsedValue, error) {
			GinkgoHelper()
			return literal.ParseConst(MustSucceed(parser.ParseExpression(text)), target)
		}

		DescribeTable(
			"parses a bare or single-negated literal to a range-checked value",
			func(text string, target types.Type, expected any) {
				parsed := MustSucceed(parse(text, target))
				Expect(parsed.Value).To(Equal(expected))
			},
			Entry("bare positive integer", "5", types.I64(), int64(5)),
			Entry("bare string", `"hi"`, types.String(), "hi"),
			Entry("negated integer", "-5", types.I64(), int64(-5)),
			Entry("i8 type minimum", "-128", types.I8(), int8(-128)),
			Entry("i16 type minimum", "-32768", types.I16(), int16(-32768)),
			Entry("i32 type minimum", "-2147483648", types.I32(), int32(-2147483648)),
			Entry("negated float", "-2.5", types.F64(), float64(-2.5)),
			Entry("negated float narrowed to f32", "-2.5", types.F32(), float32(-2.5)),
			Entry(
				"bare time unit boxes TimeSpan",
				"5s",
				types.I64(),
				telem.TimeSpan(5),
			),
			Entry(
				"negated time unit boxes TimeSpan",
				"-5s",
				types.I64(),
				telem.TimeSpan(-5),
			),
			Entry(
				"negated nanoseconds box TimeSpan",
				"-54ns",
				types.I64(),
				telem.Nanosecond*-54,
			),
		)

		DescribeTable(
			"rejects a non-constant or out-of-range value",
			func(text string, target types.Type, errSubstring string) {
				Expect(
					parse(text, target),
				).Error().
					To(MatchError(ContainSubstring(errSubstring)))
			},
			Entry("identifier", "x", types.I64(), "not a compile-time constant"),
			Entry(
				"multi-term expression",
				"2 + 3",
				types.I64(),
				"not a compile-time constant",
			),
			Entry("function call", "foo()", types.I64(), "not a compile-time constant"),
			Entry(
				"power expression",
				"2 ^ 3",
				types.I64(),
				"not a compile-time constant",
			),
			Entry("negative into unsigned", "-5", types.U8(), "out of range for u8"),
			Entry(
				"negated magnitude overflow",
				"-200",
				types.I8(),
				"out of range for i8",
			),
		)
	})
})

var _ = Describe("Convert", func() {
	var (
		ms  = &types.Unit{Dimensions: types.DimTime, Scale: 1e6, Name: "ms"}
		s   = &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}
		pa  = &types.Unit{Dimensions: types.DimPressure, Scale: 1, Name: "Pa"}
		bar = &types.Unit{Dimensions: types.DimPressure, Scale: 1e5, Name: "bar"}
	)
	in := func(t types.Type, unit *types.Unit) types.Type {
		t.Unit = unit
		return t
	}

	DescribeTable("value in the unit of the target",
		func(
			value float64,
			from *types.Unit,
			target types.Type,
			want any,
			truncated bool,
		) {
			parsed := MustSucceed(literal.Convert(value, from, target))
			Expect(parsed.Value).To(Equal(want))
			Expect(parsed.Truncated).To(Equal(truncated))
		},
		Entry("an integer time to a smaller unit",
			2.0, s, in(types.I64(), ms), telem.TimeSpan(2000), false),
		Entry("an integer time with no remainder",
			2000.0, ms, in(types.I64(), s), telem.TimeSpan(2), false),
		Entry("an integer time with a remainder",
			1500.0, ms, in(types.I64(), s), telem.TimeSpan(1), true),
		Entry("an integer that is not a time",
			2.0, bar, in(types.I64(), pa), int64(200000), false),
		Entry("a float", 1500.0, ms, in(types.F64(), s), 1.5, false),
	)

	It("Should reject a unit of another dimension", func() {
		Expect(literal.Convert(1, s, in(types.I64(), pa))).Error().
			To(MatchError(units.ErrIncompatibleDimensions))
	})
})

var _ = Describe("Boolean literals", func() {
	It("Should parse true with no target type", func() {
		parsed := MustSucceed(literal.Parse(getLiteral("true"), types.Type{}))
		Expect(parsed.Value).To(BeTrue())
		Expect(parsed.Type).To(Equal(types.Bool()))
	})

	It("Should parse false with a bool target type", func() {
		parsed := MustSucceed(literal.Parse(getLiteral("false"), types.Bool()))
		Expect(parsed.Value).To(BeFalse())
		Expect(parsed.Type).To(Equal(types.Bool()))
	})

	It("Should reject a bool literal against a numeric target", func() {
		Expect(literal.Parse(getLiteral("true"), types.I32())).Error().
			To(MatchError(ContainSubstring("cannot assign bool to i32")))
	})
})
