package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Analyzer", func() {
	Describe("Duplicate Scope", func() {

		It("Should correctly diagnose a duplicate function declaration", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
}

func dog() {
}
		`))
			r := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(r.Diagnostics).To(HaveLen(1))
			diagnostic := r.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 0"))
			Expect(diagnostic.Line).To(Equal(5))
			Expect(diagnostic.Severity).To(Equal(result.Error))
		})

		It("Should correctly diagnose a variable declaration that shadows a function", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
	dog := 1
}
		`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(1))
			diagnostic := result.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 0"))
		})

		It("Should correctly diagnose a function with duplicate parameter names", func() {
			ast := MustSucceed(parser.Parse(`
func dog(age i32, age i32) {
}
		`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(1))
			diagnostic := result.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("duplicate parameter age"))
		})
	})

	Describe("Declaration", func() {
		Describe("Local", func() {
			It("Should return an error diagnostic when a string is declared on an i32", func() {
				ast := MustSucceed(parser.Parse(`
func cat() {
	my_var i32 := "dog"
}
`))
				result := analyzer.Analyze(analyzer.Config{Program: ast})
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("type mismatch: cannot assign string to i32"))
			})

			It("Should allow compatible types in local variable declaration", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42
				}
			`))
				result := analyzer.Analyze(analyzer.Config{Program: ast})
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should infer types from an int literal", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42
				}
				`))
				result := analyzer.Analyze(analyzer.Config{Program: ast})
				Expect(result.Diagnostics).To(HaveLen(0))
				funcScope, idx := MustSucceed2(result.Symbols.GetIndex("testFunc"))
				Expect(idx).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				Expect(idx).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				varScope, idx := MustSucceed2(blockScope.GetIndex("x"))
				Expect(idx).To(Equal(0))
				Expect(varScope.Symbol).ToNot(BeNil())
				Expect(varScope.Symbol.Name).To(Equal("x"))
				Expect(varScope.Symbol.Type).To(Equal(types.I64{}))
			})

			It("Should infer types from a float literal", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42.0
				}
				`))
				result := analyzer.Analyze(analyzer.Config{Program: ast})
				Expect(result.Diagnostics).To(HaveLen(0))
				funcScope, idx := MustSucceed2(result.Symbols.GetIndex("testFunc"))
				Expect(idx).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				Expect(idx).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				varScope, idx := MustSucceed2(blockScope.GetIndex("x"))
				Expect(idx).To(Equal(0))
				Expect(varScope.Symbol).ToNot(BeNil())
				Expect(varScope.Symbol.Name).To(Equal("x"))
				Expect(varScope.Symbol.Type).To(Equal(types.F64{}))
			})

			It("Should automatically cast an int literal to a floating point type", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 42
				}
				`))
				result := analyzer.Analyze(analyzer.Config{Program: ast})
				Expect(result.Diagnostics).To(HaveLen(0))
				funcScope, idx := MustSucceed2(result.Symbols.GetIndex("testFunc"))
				Expect(idx).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				Expect(idx).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				varScope, idx := MustSucceed2(blockScope.GetIndex("x"))
				Expect(idx).To(Equal(0))
				Expect(varScope.Symbol).ToNot(BeNil())
				Expect(varScope.Symbol.Name).To(Equal("x"))
				Expect(varScope.Symbol.Type).To(Equal(types.F32{}))
			})

			It("Should not allow assignment of a float literal to an int type", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42.0
				}
				`))
				result := analyzer.Analyze(analyzer.Config{Program: ast})
				Expect(result.Ok()).To(BeFalse())
				Expect(result.Diagnostics).To(HaveLen(1))
				first := result.Diagnostics[0]
				Expect(first.Message).To(Equal("type mismatch: cannot assign f64 to i32"))
			})
		})
	})

	Describe("Assignment", func() {
		It("Should return an error diagnostic when the variable being assigned to was not declared", func() {
			ast := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat string := "abc"
					bob = cat
				}
		`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
			Expect(first.Line).To(Equal(5))
			Expect(first.Column).To(Equal(5))
		})

		It("Should return an error diagnostic when the variable on the right hand side is not declared", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
	my_var i32 := 1
	cat string := "abc"
	cat = bob
}
		`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
		})

		It("Should return an error when assignment is attempted between incompatible types", func() {
			ast := MustSucceed(parser.Parse(`
				func dog() {
					v1 i32 := 1
					v2 string := "abc"
					v2 = v1
				}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to variable of type string"))
		})
	})

	Describe("Type Signatures", func() {
		It("Should bind function parameter and return types to the function signature", func() {
			ast := MustSucceed(parser.Parse(`
				func add(x f64, y f64) f64 {
					return x + y
				}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(0))
			// The test passes if no errors are generated - the types are properly bound
		})

		It("Should bind task config, runtime params and return types to the task signature", func() {
			ast := MustSucceed(parser.Parse(`
				task controller{
					setpoint f64
					sensor <-chan f64
					actuator ->chan f64
				} (enable u8) f64 {
					return 1.0
				}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(0))
		})
	})

	Describe("Control Flow", func() {
		It("Should return no diagnostics for a valid if statement", func() {
			ast := MustSucceed(parser.Parse(`
				func dog() {
					a f32 := 2.0
					if (a > 5) {
						return 1
					}
					return 2
				}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Ok()).To(BeTrue(), result.String())
		})
	})
})
