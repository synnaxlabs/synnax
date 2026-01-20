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
	cfg := formatter.DefaultConfig

	Describe("Binary Operators", func() {
		It("should add spaces around :=", func() {
			input := "x:=42"
			expected := "x := 42\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around $=", func() {
			input := "count$=0"
			expected := "count $= 0\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around arithmetic operators", func() {
			input := "x:=a+b*c-d/e%f"
			expected := "x := a + b * c - d / e % f\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around comparison operators", func() {
			input := "x==y"
			expected := "x == y\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around logical operators", func() {
			input := "x and y or z"
			expected := "x and y or z\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around flow operators", func() {
			input := "a->b=>c"
			expected := "a -> b => c\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around compound assignment operators", func() {
			input := "x+=5"
			expected := "x += 5\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Unit Literals", func() {
		It("should not add space between number and unit suffix", func() {
			input := "delay := 100ms"
			expected := "delay := 100ms\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should not add space between float and unit suffix", func() {
			input := "pressure := 14.7psi"
			expected := "pressure := 14.7psi\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Functions", func() {
		It("should format simple function", func() {
			input := "func add(x i32,y i32)i32{return x+y}"
			expected := "func add(x i32, y i32) i32 {\n    return x + y\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format function with config block", func() {
			input := "func threshold{limit f64}(value f64)u8{return u8(0)}"
			expected := "func threshold {\n    limit f64\n} (value f64) u8 {\n    return u8(0)\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format empty function body", func() {
			input := "func noop(){}"
			expected := "func noop() {}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Sequences", func() {
		It("should format sequence with stages", func() {
			input := "sequence main{stage first{}stage second{}}"
			expected := "sequence main {\n    stage first {}\n    stage second {}\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Control Flow", func() {
		It("should format if statement", func() {
			input := "if x>0{return 1}"
			expected := "if x > 0 {\n    return 1\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format if-else statement", func() {
			input := "if x>0{return 1}else{return 0}"
			expected := "if x > 0 {\n    return 1\n} else {\n    return 0\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Comments", func() {
		It("should preserve single-line comments", func() {
			input := "// comment\nx := 1"
			expected := "// comment\nx := 1\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should preserve trailing comments", func() {
			input := "x := 1 // comment"
			expected := "x := 1 // comment\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should preserve multi-line comments", func() {
			input := "/* multi\nline */ x := 1"
			expected := "/* multi\nline */\nx := 1\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Series Literals", func() {
		It("should format series literal", func() {
			input := "[1,2,3]"
			expected := "[1, 2, 3]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format empty series", func() {
			input := "[]"
			expected := "[]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after := before series literal", func() {
			input := "d:=[1,2]"
			expected := "d := [1, 2]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after $= before series literal", func() {
			input := "d$=[1,2]"
			expected := "d $= [1, 2]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Spaces After Binary Operators", func() {
		It("should add space after := before paren", func() {
			input := "x:=(1+2)"
			expected := "x := (1 + 2)\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after $= before paren", func() {
			input := "x$=(a)"
			expected := "x $= (a)\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after = before paren", func() {
			input := "x=(1)"
			expected := "x = (1)\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after := before identifier", func() {
			input := "x:=y"
			expected := "x := y\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after + before paren", func() {
			input := "x:=a+(b)"
			expected := "x := a + (b)\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add space after + before bracket", func() {
			input := "x:=a+[1]"
			expected := "x := a + [1]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Idempotency", func() {
		It("should be idempotent", func() {
			input := "func add(x i32, y i32) i32 {\n    return x + y\n}\n"
			firstPass := formatter.Format(input, cfg)
			secondPass := formatter.Format(firstPass, cfg)
			Expect(secondPass).To(Equal(firstPass))
		})

		It("should produce same output for messy input", func() {
			input := "func   add(x i32,y i32)i32{return   x+y}"
			firstPass := formatter.Format(input, cfg)
			secondPass := formatter.Format(firstPass, cfg)
			Expect(secondPass).To(Equal(firstPass))
		})
	})

	Describe("Edge Cases", func() {
		It("should handle empty input", func() {
			input := ""
			Expect(formatter.Format(input, cfg)).To(Equal(""))
		})

		It("should handle whitespace-only input", func() {
			input := "   \n\n   "
			Expect(formatter.Format(input, cfg)).To(Equal("   \n\n   "))
		})
	})

	Describe("Blank Lines", func() {
		It("should preserve single blank line between functions", func() {
			input := "func first() {}\n\nfunc second() {}"
			expected := "func first() {}\n\nfunc second() {}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should preserve blank lines between statements", func() {
			input := "x := 1\n\ny := 2"
			expected := "x := 1\n\ny := 2\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should limit blank lines to MaxBlankLines", func() {
			input := "x := 1\n\n\n\n\ny := 2"
			expected := "x := 1\n\n\ny := 2\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Multi-line Code", func() {
		It("should preserve newlines between statements", func() {
			input := "x := 1\ny := 2\nz := 3"
			expected := "x := 1\ny := 2\nz := 3\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle already formatted function with newlines", func() {
			input := `func add(x i32, y i32) i32 {
    return x + y
}`
			expected := "func add(x i32, y i32) i32 {\n    return x + y\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle multiple functions", func() {
			input := `func foo() {}
func bar() {}`
			expected := "func foo() {}\nfunc bar() {}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle function with multiple statements", func() {
			input := `func test() {
    x := 1
    y := 2
    return x + y
}`
			expected := "func test() {\n    x := 1\n    y := 2\n    return x + y\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle sequence with stages and content", func() {
			input := `sequence main {
    stage init {
        x := 0
    }
    stage run {
        x := x + 1
    }
}`
			expected := "sequence main {\n    stage init {\n        x := 0\n    }\n    stage run {\n        x := x + 1\n    }\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle nested if statements", func() {
			input := `func test() {
    if x > 0 {
        if y > 0 {
            return 1
        }
    }
}`
			expected := "func test() {\n    if x > 0 {\n        if y > 0 {\n            return 1\n        }\n    }\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Unary Operators", func() {
		It("should handle negation", func() {
			input := "x:=-5"
			expected := "x := -5\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle not operator", func() {
			input := "x:=not y"
			expected := "x := not y\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle negation in expression", func() {
			input := "x:=a+-b"
			expected := "x := a + -b\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Type Casts", func() {
		It("should not add space between type and paren", func() {
			input := "x:=i32(y)"
			expected := "x := i32(y)\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle nested casts", func() {
			input := "x:=f64(i32(y))"
			expected := "x := f64(i32(y))\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("String Literals", func() {
		It("should preserve string content", func() {
			input := `x:="hello world"`
			expected := "x := \"hello world\"\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should preserve strings with spaces", func() {
			input := `msg:="  spaces  "`
			expected := "msg := \"  spaces  \"\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Nested Structures", func() {
		It("should handle nested function calls", func() {
			input := "x:=foo(bar(baz(1)))"
			expected := "x := foo(bar(baz(1)))\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle mixed nesting", func() {
			input := "x:=foo([1,2,3])"
			expected := "x := foo([1, 2, 3])\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Next Statement", func() {
		It("should format next with stage name", func() {
			input := "next done"
			expected := "next done\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format next in stage", func() {
			input := "stage check{if x>0{next success}}"
			expected := "stage check {\n    if x > 0 {\n        next success\n    }\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Configurable Indent", func() {
		It("should respect 2-space indent", func() {
			cfg2 := formatter.Config{IndentWidth: 2, MaxBlankLines: 2}
			input := "func test(){x:=1}"
			expected := "func test() {\n  x := 1\n}\n"
			Expect(formatter.Format(input, cfg2)).To(Equal(expected))
		})

		It("should respect 8-space indent", func() {
			cfg8 := formatter.Config{IndentWidth: 8, MaxBlankLines: 2}
			input := "func test(){x:=1}"
			expected := "func test() {\n        x := 1\n}\n"
			Expect(formatter.Format(input, cfg8)).To(Equal(expected))
		})
	})

	Describe("Comments in Blocks", func() {
		It("should handle comment before closing brace", func() {
			input := "func test() {\n    x := 1\n    // end\n}"
			expected := "func test() {\n    x := 1\n    // end\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should handle comment after opening brace", func() {
			input := "func test() {\n    // start\n    x := 1\n}"
			expected := "func test() {\n    // start\n    x := 1\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Boundary Blank Lines", func() {
		It("should strip leading blank lines", func() {
			input := "\n\nx := 1"
			result := formatter.Format(input, cfg)
			Expect(result).To(Equal("x := 1\n"))
		})

		It("should handle trailing blank lines", func() {
			input := "x := 1\n\n"
			result := formatter.Format(input, cfg)
			Expect(result).To(HaveSuffix("\n"))
		})
	})

	Describe("FormatRange", func() {
		It("should format only the specified range", func() {
			input := "x:=1\ny:=2\nz:=3"
			result := formatter.FormatRange(input, 1, 1, cfg)
			Expect(result).To(ContainSubstring("y := 2"))
		})

		It("should return unchanged for invalid range", func() {
			input := "x := 1"
			result := formatter.FormatRange(input, 5, 10, cfg)
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
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("sequence pressurize {"))
			Expect(result).To(ContainSubstring("    stage start {"))
			Expect(result).To(ContainSubstring("        valve_open := true"))
			Expect(result).To(ContainSubstring("        pump_speed $= 50"))
			Expect(result).To(ContainSubstring("    }"))
		})

		It("should not collapse everything onto one line", func() {
			input := "x := 1\ny := 2"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("\n"))
			Expect(result).ToNot(Equal("x := 1y := 2\n"))
		})
	})

	Describe("Malformed Input", func() {
		It("should handle unclosed brace gracefully", func() {
			input := "func test() {"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("func test()"))
		})

		It("should handle unclosed paren gracefully", func() {
			input := "x := foo(1, 2"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("foo("))
		})

		It("should handle extra closing brace gracefully", func() {
			input := "x := 1}"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("x := 1"))
		})

		It("should handle mismatched delimiters gracefully", func() {
			input := "x := [1, 2)"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("[1, 2)"))
		})

		It("should handle incomplete expression gracefully", func() {
			input := "x := a +"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("x := a +"))
		})
	})

	Describe("Boundary Conditions", func() {
		It("should handle single character", func() {
			input := "x"
			result := formatter.Format(input, cfg)
			Expect(result).To(Equal("x\n"))
		})

		It("should handle very long identifier", func() {
			longName := "x"
			for i := 0; i < 100; i++ {
				longName += "a"
			}
			input := longName + " := 1"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring(longName))
			Expect(result).To(ContainSubstring(":= 1"))
		})

		It("should handle deeply nested structures (10 levels)", func() {
			input := "func a(){func b(){func c(){func d(){func e(){func f(){func g(){func h(){func i(){func j(){}}}}}}}}}}"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("func a()"))
			Expect(result).To(ContainSubstring("func j()"))
		})

		It("should handle file with only comments gracefully", func() {
			input := "// comment 1\n// comment 2\n/* multi\nline */"
			result := formatter.Format(input, cfg)
			// Formatter processes comment-only files without crashing
			Expect(result).ToNot(BeEmpty())
		})

		It("should handle large file with many statements", func() {
			var input string
			for i := 0; i < 100; i++ {
				input += "x := 1\n"
			}
			result := formatter.Format(input, cfg)
			Expect(result).ToNot(BeEmpty())
		})
	})

	Describe("Config Options", func() {
		It("should handle zero indent width", func() {
			cfg0 := formatter.Config{IndentWidth: 0, MaxBlankLines: 2}
			input := "func test(){x:=1}"
			result := formatter.Format(input, cfg0)
			Expect(result).To(ContainSubstring("func test()"))
			Expect(result).To(ContainSubstring("x := 1"))
		})

		It("should handle MaxBlankLines of 0", func() {
			cfg0 := formatter.Config{IndentWidth: 4, MaxBlankLines: 0}
			input := "x := 1\n\n\ny := 2"
			result := formatter.Format(input, cfg0)
			Expect(result).To(Equal("x := 1\ny := 2\n"))
		})

		It("should handle MaxBlankLines of 10", func() {
			cfg10 := formatter.Config{IndentWidth: 4, MaxBlankLines: 10}
			input := "x := 1\n\n\n\n\ny := 2"
			result := formatter.Format(input, cfg10)
			Expect(result).To(ContainSubstring("x := 1"))
			Expect(result).To(ContainSubstring("y := 2"))
		})
	})

	Describe("Comment Edge Cases", func() {
		It("should handle comment on its own line before code", func() {
			input := "/* comment */\nx := 1"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("/* comment */"))
			Expect(result).To(ContainSubstring("x := 1"))
		})

		It("should handle empty single-line comment", func() {
			input := "//\nx := 1"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("//"))
		})

		It("should handle empty multi-line comment", func() {
			input := "/**/\nx := 1"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("/**/"))
		})

		It("should preserve comment with special characters", func() {
			input := "// @#$%^&*()_+\nx := 1"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("// @#$%^&*()_+"))
		})

		It("should handle multiple trailing comments", func() {
			input := "x := 1 // first\ny := 2 // second"
			result := formatter.Format(input, cfg)
			Expect(result).To(ContainSubstring("// first"))
			Expect(result).To(ContainSubstring("// second"))
		})
	})

	Describe("FormatRange Edge Cases", func() {
		It("should handle formatting first line only", func() {
			input := "x:=1\ny := 2\nz := 3"
			result := formatter.FormatRange(input, 0, 0, cfg)
			Expect(result).To(ContainSubstring("x := 1"))
			Expect(result).To(ContainSubstring("y := 2"))
		})

		It("should handle formatting last line only", func() {
			input := "x := 1\ny := 2\nz:=3"
			result := formatter.FormatRange(input, 2, 2, cfg)
			Expect(result).To(ContainSubstring("z := 3"))
		})

		It("should handle negative start line", func() {
			input := "x := 1\ny := 2"
			result := formatter.FormatRange(input, -1, 1, cfg)
			Expect(result).To(Equal(input))
		})

		It("should handle start > end", func() {
			input := "x := 1\ny := 2"
			result := formatter.FormatRange(input, 1, 0, cfg)
			Expect(result).To(Equal(input))
		})

		It("should handle entire file range", func() {
			input := "x:=1\ny:=2\nz:=3"
			result := formatter.FormatRange(input, 0, 2, cfg)
			Expect(result).To(ContainSubstring("x := 1"))
			Expect(result).To(ContainSubstring("y := 2"))
			Expect(result).To(ContainSubstring("z := 3"))
		})
	})

	Describe("Idempotency Stress Tests", func() {
		It("should be idempotent for complex expressions", func() {
			input := "x := (a + b) * (c - d) / (e % f) ^ g"
			first := formatter.Format(input, cfg)
			second := formatter.Format(first, cfg)
			third := formatter.Format(second, cfg)
			Expect(second).To(Equal(first))
			Expect(third).To(Equal(second))
		})

		It("should be idempotent for deeply nested structures", func() {
			input := "func a(){if x>0{if y>0{if z>0{return 1}}}}"
			first := formatter.Format(input, cfg)
			second := formatter.Format(first, cfg)
			third := formatter.Format(second, cfg)
			Expect(second).To(Equal(first))
			Expect(third).To(Equal(second))
		})

		It("should be idempotent for all operator combinations", func() {
			input := "x := a + b - c * d / e % f ^ g == h != i < j > k <= l >= m and n or o"
			first := formatter.Format(input, cfg)
			second := formatter.Format(first, cfg)
			Expect(second).To(Equal(first))
		})

		It("should be idempotent for mixed comments and code", func() {
			input := "// header\nfunc test() {\n    // body\n    x := 1 // inline\n}\n// footer"
			first := formatter.Format(input, cfg)
			second := formatter.Format(first, cfg)
			Expect(second).To(Equal(first))
		})

		It("should be idempotent for sequences with stages", func() {
			input := "sequence s{stage a{x:=1}stage b{y:=2}stage c{z:=3}}"
			first := formatter.Format(input, cfg)
			second := formatter.Format(first, cfg)
			Expect(second).To(Equal(first))
		})
	})
})
