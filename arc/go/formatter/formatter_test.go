// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/formatter"
)

var _ = Describe("Formatter", func() {
	DescribeTable("Binary Operators",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("should add spaces around :=", "x:=42", "x := 42\n"),
		Entry("should add spaces around $=", "count$=0", "count $= 0\n"),
		Entry("should add spaces around arithmetic operators", "x:=a+b*c-d/e%f", "x := a + b * c - d / e % f\n"),
		Entry("should add spaces around comparison operators", "x==y", "x == y\n"),
		Entry("should add spaces around logical operators", "x and y or z", "x and y or z\n"),
		Entry("should add spaces around flow operators", "a->b=>c", "a -> b => c\n"),
		Entry("should add spaces around compound assignment operators", "x+=5", "x += 5\n"),
	)

	DescribeTable("Unit Literals",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("should not add space between number and unit suffix", "delay := 100ms", "delay := 100ms\n"),
		Entry("should not add space between float and unit suffix", "pressure := 14.7psi", "pressure := 14.7psi\n"),
	)

	DescribeTable("Functions",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("simple function", "func add(x i32,y i32)i32{return x+y}", "func add(x i32, y i32) i32 {\n    return x + y\n}\n"),
		Entry("function with config block", "func threshold{limit f64}(value f64)u8{return u8(0)}", "func threshold {\n    limit f64\n} (value f64) u8 {\n    return u8(0)\n}\n"),
		Entry("empty function body", "func noop(){}", "func noop() {}\n"),
	)

	DescribeTable("Sequences",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("sequence with stages", "sequence main{stage first{}stage second{}}", "sequence main {\n    stage first {}\n    stage second {}\n}\n"),
	)

	DescribeTable("Control Flow",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("if statement", "if x>0{return 1}", "if x > 0 {\n    return 1\n}\n"),
		Entry("if-else statement", "if x>0{return 1}else{return 0}", "if x > 0 {\n    return 1\n} else {\n    return 0\n}\n"),
	)

	DescribeTable("Comments",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("preserve single-line comments", "// comment\nx := 1", "// comment\nx := 1\n"),
		Entry("preserve trailing comments", "x := 1 // comment", "x := 1 // comment\n"),
		Entry("preserve multi-line comments", "/* multi\nline */ x := 1", "/* multi\nline */\nx := 1\n"),
	)

	DescribeTable("Series Literals",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("format series literal", "[1,2,3]", "[1, 2, 3]\n"),
		Entry("format empty series", "[]", "[]\n"),
		Entry("add space after := before series literal", "d:=[1,2]", "d := [1, 2]\n"),
		Entry("add space after $= before series literal", "d$=[1,2]", "d $= [1, 2]\n"),
	)

	DescribeTable("Spaces After Binary Operators",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("should add space after := before paren", "x:=(1+2)", "x := (1 + 2)\n"),
		Entry("should add space after $= before paren", "x$=(a)", "x $= (a)\n"),
		Entry("should add space after = before paren", "x=(1)", "x = (1)\n"),
		Entry("should add space after := before identifier", "x:=y", "x := y\n"),
		Entry("should add space after + before paren", "x:=a+(b)", "x := a + (b)\n"),
		Entry("should add space after + before bracket", "x:=a+[1]", "x := a + [1]\n"),
	)

	DescribeTable("Idempotency",
		func(input string) {
			first := formatter.Format(input)
			second := formatter.Format(first)
			third := formatter.Format(second)
			Expect(second).To(Equal(first))
			Expect(third).To(Equal(second))
		},
		Entry("formatted function", "func add(x i32, y i32) i32 {\n    return x + y\n}\n"),
		Entry("messy input", "func   add(x i32,y i32)i32{return   x+y}"),
		Entry("complex expressions", "x := (a + b) * (c - d) / (e % f) ^ g"),
		Entry("deeply nested structures", "func a(){if x>0{if y>0{if z>0{return 1}}}}"),
		Entry("all operator combinations", "x := a + b - c * d / e % f ^ g == h != i < j > k <= l >= m and n or o"),
		Entry("mixed comments and code", "// header\nfunc test() {\n    // body\n    x := 1 // inline\n}\n// footer"),
		Entry("sequences with stages", "sequence s{stage a{x:=1}stage b{y:=2}stage c{z:=3}}"),
		Entry("config values", "wait{duration=2ms, retries=3}"),
		Entry("config values in flow", "sensor -> filter{threshold=10} -> output"),
	)

	DescribeTable("Edge Cases",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("empty input", "", ""),
		Entry("whitespace-only input", "   \n\n   ", "   \n\n   "),
	)

	DescribeTable("Blank Lines",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("preserve single blank line between functions", "func first() {}\n\nfunc second() {}", "func first() {}\n\nfunc second() {}\n"),
		Entry("preserve blank lines between statements", "x := 1\n\ny := 2", "x := 1\n\ny := 2\n"),
		Entry("limit blank lines to MaxBlankLines", "x := 1\n\n\n\n\ny := 2", "x := 1\n\n\ny := 2\n"),
	)

	DescribeTable("Multi-line Code",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("preserve newlines between statements", "x := 1\ny := 2\nz := 3", "x := 1\ny := 2\nz := 3\n"),
		Entry("already formatted function with newlines", "func add(x i32, y i32) i32 {\n    return x + y\n}", "func add(x i32, y i32) i32 {\n    return x + y\n}\n"),
		Entry("multiple functions", "func foo() {}\nfunc bar() {}", "func foo() {}\nfunc bar() {}\n"),
		Entry("function with multiple statements", "func test() {\n    x := 1\n    y := 2\n    return x + y\n}", "func test() {\n    x := 1\n    y := 2\n    return x + y\n}\n"),
		Entry("sequence with stages and content", "sequence main {\n    stage init {\n        x := 0\n    }\n    stage run {\n        x := x + 1\n    }\n}", "sequence main {\n    stage init {\n        x := 0\n    }\n    stage run {\n        x := x + 1\n    }\n}\n"),
		Entry("nested if statements", "func test() {\n    if x > 0 {\n        if y > 0 {\n            return 1\n        }\n    }\n}", "func test() {\n    if x > 0 {\n        if y > 0 {\n            return 1\n        }\n    }\n}\n"),
	)

	DescribeTable("Unary Operators",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("negation", "x:=-5", "x := -5\n"),
		Entry("not operator", "x:=not y", "x := not y\n"),
		Entry("negation in expression", "x:=a+-b", "x := a + -b\n"),
	)

	DescribeTable("Type Casts",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("no space between type and paren", "x:=i32(y)", "x := i32(y)\n"),
		Entry("nested casts", "x:=f64(i32(y))", "x := f64(i32(y))\n"),
	)

	DescribeTable("String Literals",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("preserve string content", `x:="hello world"`, "x := \"hello world\"\n"),
		Entry("preserve strings with spaces", `msg:="  spaces  "`, "msg := \"  spaces  \"\n"),
	)

	DescribeTable("Nested Structures",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("nested function calls", "x:=foo(bar(baz(1)))", "x := foo(bar(baz(1)))\n"),
		Entry("mixed nesting", "x:=foo([1,2,3])", "x := foo([1, 2, 3])\n"),
	)

	DescribeTable("Next Statement",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("next with stage name", "next done", "next done\n"),
		Entry("next in stage", "stage check{if x>0{next success}}", "stage check {\n    if x > 0 {\n        next success\n    }\n}\n"),
	)

	DescribeTable("Comments in Blocks",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("comment before closing brace", "func test() {\n    x := 1\n    // end\n}", "func test() {\n    x := 1\n    // end\n}\n"),
		Entry("comment after opening brace", "func test() {\n    // start\n    x := 1\n}", "func test() {\n    // start\n    x := 1\n}\n"),
	)

	Describe("Boundary Blank Lines", func() {
		It("should strip leading blank lines", func() {
			input := "\n\nx := 1"
			result := formatter.Format(input)
			Expect(result).To(Equal("x := 1\n"))
		})

		It("should handle trailing blank lines", func() {
			input := "x := 1\n\n"
			result := formatter.Format(input)
			Expect(result).To(HaveSuffix("\n"))
		})
	})

	Describe("FormatRange", func() {
		It("should format only the specified range", func() {
			input := "x:=1\ny:=2\nz:=3"
			result := formatter.FormatRange(input, 1, 1)
			Expect(result).To(ContainSubstring("y := 2"))
		})

		It("should return unchanged for invalid range", func() {
			input := "x := 1"
			result := formatter.FormatRange(input, 5, 10)
			Expect(result).To(Equal(input))
		})
	})

	Describe("Real World Code", func() {
		It("should format a complete sequence correctly", func() {
			input := `sequence pressurize {
    stage start {
        valve_open := true
        pump_speed $= 50
    }
    stage monitor {
        if pressure > 100psi {
            pump_speed $= 0
            next done
        }
    }
    stage done {
        valve_open := false
    }
}`
			result := formatter.Format(input)
			Expect(result).To(ContainSubstring("sequence pressurize {"))
			Expect(result).To(ContainSubstring("    stage start {"))
			Expect(result).To(ContainSubstring("        valve_open := true"))
			Expect(result).To(ContainSubstring("        pump_speed $= 50"))
			Expect(result).To(ContainSubstring("    }"))
		})

		It("should not collapse everything onto one line", func() {
			input := "x := 1\ny := 2"
			result := formatter.Format(input)
			Expect(result).To(ContainSubstring("\n"))
			Expect(result).ToNot(Equal("x := 1y := 2\n"))
		})
	})

	DescribeTable("Malformed Input",
		func(input, shouldContain string) {
			result := formatter.Format(input)
			Expect(result).To(ContainSubstring(shouldContain))
		},
		Entry("unclosed brace", "func test() {", "func test()"),
		Entry("unclosed paren", "x := foo(1, 2", "foo("),
		Entry("extra closing brace", "x := 1}", "x := 1"),
		Entry("mismatched delimiters", "x := [1, 2)", "[1, 2)"),
		Entry("incomplete expression", "x := a +", "x := a +"),
	)

	DescribeTable("Config Values (Function Instantiation)",
		func(input, expected string) {
			Expect(formatter.Format(input)).To(Equal(expected))
		},
		Entry("short config values inline without spaces around =", "wait{duration=2ms}", "wait{duration=2ms}\n"),
		Entry("multiple config values inline", "wait{duration=2ms,retries=3}", "wait{duration=2ms, retries=3}\n"),
		Entry("empty config values inline", "wait{}", "wait{}\n"),
		Entry("config values in flow statements", "sensor -> filter{threshold=10} -> output", "sensor -> filter{threshold=10} -> output\n"),
		Entry("function declaration config block multi-line", "func threshold{limit f64}(value f64)u8{return u8(0)}", "func threshold {\n    limit f64\n} (value f64) u8 {\n    return u8(0)\n}\n"),
		Entry("nested config values", "x := foo{a=1} + bar{b=2}", "x := foo{a=1} + bar{b=2}\n"),
	)

	Describe("Boundary Conditions", func() {
		It("should handle single character", func() {
			input := "x"
			result := formatter.Format(input)
			Expect(result).To(Equal("x\n"))
		})

		It("should handle very long identifier", func() {
			longName := "x"
			for i := 0; i < 100; i++ {
				longName += "a"
			}
			input := longName + " := 1"
			result := formatter.Format(input)
			Expect(result).To(ContainSubstring(longName))
			Expect(result).To(ContainSubstring(":= 1"))
		})

		It("should handle deeply nested structures (10 levels)", func() {
			input := "func a(){func b(){func c(){func d(){func e(){func f(){func g(){func h(){func i(){func j(){}}}}}}}}}}"
			result := formatter.Format(input)
			Expect(result).To(ContainSubstring("func a()"))
			Expect(result).To(ContainSubstring("func j()"))
		})

		It("should handle file with only comments gracefully", func() {
			input := "// comment 1\n// comment 2\n/* multi\nline */"
			result := formatter.Format(input)
			Expect(result).ToNot(BeEmpty())
		})

		It("should handle large file with many statements", func() {
			var input string
			for i := 0; i < 100; i++ {
				input += "x := 1\n"
			}
			result := formatter.Format(input)
			Expect(result).ToNot(BeEmpty())
		})
	})

	DescribeTable("Comment Edge Cases",
		func(input string, shouldContain string) {
			result := formatter.Format(input)
			Expect(result).To(ContainSubstring(shouldContain))
		},
		Entry("comment on its own line before code", "/* comment */\nx := 1", "/* comment */"),
		Entry("empty single-line comment", "//\nx := 1", "//"),
		Entry("empty multi-line comment", "/**/\nx := 1", "/**/"),
		Entry("comment with special characters", "// @#$%^&*()_+\nx := 1", "// @#$%^&*()_+"),
		Entry("multiple trailing comments - first", "x := 1 // first\ny := 2 // second", "// first"),
		Entry("multiple trailing comments - second", "x := 1 // first\ny := 2 // second", "// second"),
	)

	DescribeTable("FormatRange Edge Cases",
		func(input string, startLine, endLine int, check func(string)) {
			result := formatter.FormatRange(input, startLine, endLine)
			check(result)
		},
		Entry("formatting first line only", "x:=1\ny := 2\nz := 3", 0, 0, func(r string) {
			Expect(r).To(ContainSubstring("x := 1"))
			Expect(r).To(ContainSubstring("y := 2"))
		}),
		Entry("formatting last line only", "x := 1\ny := 2\nz:=3", 2, 2, func(r string) {
			Expect(r).To(ContainSubstring("z := 3"))
		}),
		Entry("negative start line", "x := 1\ny := 2", -1, 1, func(r string) {
			Expect(r).To(Equal("x := 1\ny := 2"))
		}),
		Entry("start > end", "x := 1\ny := 2", 1, 0, func(r string) {
			Expect(r).To(Equal("x := 1\ny := 2"))
		}),
		Entry("entire file range", "x:=1\ny:=2\nz:=3", 0, 2, func(r string) {
			Expect(r).To(ContainSubstring("x := 1"))
			Expect(r).To(ContainSubstring("y := 2"))
			Expect(r).To(ContainSubstring("z := 3"))
		}),
	)
})
