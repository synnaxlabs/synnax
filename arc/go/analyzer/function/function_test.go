// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package function_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

// analyzeProgram is a helper that parses source code and runs the analyzer,
// returning the context for further assertions.
func analyzeProgram(src string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	prog := MustSucceed(parser.Parse(src))
	ctx := context.CreateRoot(bCtx, prog, resolver)
	analyzer.AnalyzeProgram(ctx)
	return ctx
}

// analyzeExpectSuccess parses and analyzes code, asserting no diagnostics.
func analyzeExpectSuccess(src string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	ctx := analyzeProgram(src, resolver)
	ExpectWithOffset(1, *ctx.Diagnostics).To(BeEmpty(), ctx.Diagnostics.String())
	return ctx
}

// analyzeExpectError parses and analyzes code, asserting a diagnostic error.
func analyzeExpectError(src string, resolver symbol.Resolver, msgMatcher OmegaMatcher) context.Context[parser.IProgramContext] {
	ctx := analyzeProgram(src, resolver)
	ExpectWithOffset(1, *ctx.Diagnostics).To(HaveLen(1))
	ExpectWithOffset(1, (*ctx.Diagnostics)[0].Message).To(msgMatcher)
	ExpectWithOffset(1, (*ctx.Diagnostics)[0].Severity).To(Equal(diagnostics.SeverityError))
	return ctx
}

var _ = Describe("Function Analyzer", func() {
	Describe("CollectDeclarations", func() {
		Describe("basic declaration collection", func() {
			It("should handle empty program", func() {
				ctx := analyzeExpectSuccess(``, nil)
				Expect(ctx.Scope.Children).To(BeEmpty())
			})
			It("should collect function with no parameters", func() {
				ctx := analyzeExpectSuccess(`func foo() {}`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(1))
				fn := ctx.Scope.Children[0]
				Expect(fn.Name).To(Equal("foo"))
				Expect(fn.Kind).To(Equal(symbol.KindFunction))
				Expect(fn.Type.Config).To(BeEmpty())
				Expect(fn.Type.Inputs).To(BeEmpty())
				Expect(fn.Type.Outputs).To(BeEmpty())
			})
			It("should collect multiple functions before body analysis", func() {
				ctx := analyzeExpectSuccess(`
					func first() i32 { return second() }
					func second() i32 { return 42 }
				`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(2))
				Expect(ctx.Scope.Children[0].Name).To(Equal("first"))
				Expect(ctx.Scope.Children[1].Name).To(Equal("second"))
			})
		})
		Describe("config parameter collection", func() {
			It("should collect function with only config params", func() {
				ctx := analyzeExpectSuccess(`func foo{x i32}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(1))
				Expect(fn.Type.Config[0]).To(Equal(types.Param{Name: "x", Type: types.I32()}))
				Expect(fn.Type.Inputs).To(BeEmpty())
				Expect(fn.Type.Outputs).To(BeEmpty())
			})
			It("should collect config with channel type", func() {
				ctx := analyzeExpectSuccess(`func foo{sensor chan f64}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(1))
				Expect(fn.Type.Config[0]).To(Equal(types.Param{Name: "sensor", Type: types.Chan(types.F64())}))
			})
			It("should handle empty config block", func() {
				ctx := analyzeExpectSuccess(`func foo{}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(BeEmpty())
			})
			It("should collect config with default value", func() {
				ctx := analyzeExpectSuccess(`func foo{gain f64 = 1.0}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(1))
				Expect(fn.Type.Config[0].Name).To(Equal("gain"))
				Expect(fn.Type.Config[0].Type).To(Equal(types.F64()))
				Expect(fn.Type.Config[0].Value).To(Equal(1.0))
			})
			It("should collect mixed required and optional config params", func() {
				ctx := analyzeExpectSuccess(`func foo{setpoint f64, gain f64 = 1.0}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(2))
				Expect(fn.Type.Config[0].Name).To(Equal("setpoint"))
				Expect(fn.Type.Config[0].Value).To(BeNil())
				Expect(fn.Type.Config[1].Name).To(Equal("gain"))
				Expect(fn.Type.Config[1].Value).To(Equal(1.0))
			})
			It("should reject required config after optional config", func() {
				analyzeExpectError(
					`func foo{gain f64 = 1.0, setpoint f64}() {}`,
					nil,
					ContainSubstring("required config parameter setpoint cannot follow optional config parameters"),
				)
			})
			It("should reject invalid default value for config", func() {
				analyzeExpectError(
					`func foo{x i8 = 128}() {}`,
					nil,
					ContainSubstring("out of range for i8"),
				)
			})
		})
		Describe("input parameter collection", func() {
			It("should collect multiple inputs without defaults", func() {
				ctx := analyzeExpectSuccess(`func foo(a i32, b f64, c u8) {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Inputs).To(HaveLen(3))
				Expect(fn.Type.Inputs[0]).To(Equal(types.Param{Name: "a", Type: types.I32()}))
				Expect(fn.Type.Inputs[1]).To(Equal(types.Param{Name: "b", Type: types.F64()}))
				Expect(fn.Type.Inputs[2]).To(Equal(types.Param{Name: "c", Type: types.U8()}))
			})
			It("should collect all optional inputs", func() {
				ctx := analyzeExpectSuccess(`func foo(x i32 = 1, y i32 = 2) {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Inputs).To(HaveLen(2))
				Expect(fn.Type.Inputs[0].Name).To(Equal("x"))
				Expect(fn.Type.Inputs[0].Value).To(Equal(int32(1)))
				Expect(fn.Type.Inputs[1].Name).To(Equal("y"))
				Expect(fn.Type.Inputs[1].Value).To(Equal(int32(2)))
			})
			It("should preserve order of mixed required and optional", func() {
				ctx := analyzeExpectSuccess(`func foo(a i32, b i32, c i32 = 10) {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Inputs).To(HaveLen(3))
				Expect(fn.Type.Inputs[0].Value).To(BeNil())
				Expect(fn.Type.Inputs[1].Value).To(BeNil())
				Expect(fn.Type.Inputs[2].Value).To(Equal(int32(10)))
			})
		})
		Describe("output parameter collection", func() {
			It("should handle void function", func() {
				ctx := analyzeExpectSuccess(`func foo() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(BeEmpty())
			})
			It("should collect unnamed output with default param name", func() {
				ctx := analyzeExpectSuccess(`func foo() i32 { return 0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(1))
				Expect(fn.Type.Outputs[0].Name).To(Equal(ir.DefaultOutputParam))
				Expect(fn.Type.Outputs[0].Type).To(Equal(types.I32()))
			})
			It("should collect single named output without parens", func() {
				ctx := analyzeExpectSuccess(`func foo() result i32 { result = 0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(1))
				Expect(fn.Type.Outputs[0].Name).To(Equal("result"))
				Expect(fn.Type.Outputs[0].Type).To(Equal(types.I32()))
			})
			It("should collect single named output with parens", func() {
				ctx := analyzeExpectSuccess(`func foo() (result i32) { result = 0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(1))
				Expect(fn.Type.Outputs[0].Name).To(Equal("result"))
				Expect(fn.Type.Outputs[0].Type).To(Equal(types.I32()))
			})
			It("should collect multiple outputs in order", func() {
				ctx := analyzeExpectSuccess(`func foo() (a i32, b f64) { a = 0 b = 0.0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(2))
				Expect(fn.Type.Outputs[0]).To(Equal(types.Param{Name: "a", Type: types.I32()}))
				Expect(fn.Type.Outputs[1]).To(Equal(types.Param{Name: "b", Type: types.F64()}))
			})
		})
		Describe("error conditions", func() {
			It("should fail on duplicate function names", func() {
				ctx := analyzeProgram(`func foo() {} func foo() {}`, nil)
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("conflicts with existing symbol"))
			})
			It("should add functions to root scope", func() {
				ctx := analyzeExpectSuccess(`func outer() { } func inner() { }`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(2))
				for _, child := range ctx.Scope.Children {
					Expect(child.Kind).To(Equal(symbol.KindFunction))
				}
			})
		})
	})

	Describe("Declaration Validation", func() {
		Context("duplicate declarations", func() {
			It("should diagnose duplicate function names", func() {
				ctx := analyzeExpectError(`
					func dog() {}
					func dog() {}
				`, nil, ContainSubstring("name dog conflicts with existing symbol"))
				Expect((*ctx.Diagnostics)[0].Start.Line).To(Equal(3))
			})

			It("should diagnose duplicate parameter names", func() {
				analyzeExpectError(`
					func dog(age i32, age i32) {}
				`, nil, ContainSubstring("name age conflicts with existing symbol"))
			})
		})
	})

	Describe("Parameter Binding", func() {
		Context("basic input and output binding", func() {
			It("should bind input and output types to the function signature", func() {
				ctx := analyzeExpectSuccess(`
					func add(x f64, y f64) f64 {
						return x + y
					}
				`, nil)

				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "add"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("add"))

				output := MustBeOk(funcScope.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.F64()))

				Expect(funcScope.Type.Inputs).To(HaveLen(2))
				Expect(funcScope.Type.Inputs[0]).To(Equal(types.Param{Name: "x", Type: types.F64()}))
				Expect(funcScope.Type.Inputs[1]).To(Equal(types.Param{Name: "y", Type: types.F64()}))

				paramChildren := funcScope.FilterChildrenByKind(symbol.KindInput)
				Expect(paramChildren).To(HaveLen(2))
				Expect(paramChildren[0].Name).To(Equal("x"))
				Expect(paramChildren[0].Type).To(Equal(types.F64()))
				Expect(paramChildren[1].Name).To(Equal("y"))
				Expect(paramChildren[1].Type).To(Equal(types.F64()))
			})
		})

		Context("config, input, and output binding", func() {
			It("should bind config, input, and output types correctly", func() {
				ctx := analyzeExpectSuccess(`
					func controller{
						setpoint f64,
						sensor chan f64,
						actuator chan f64
					} (enable u8) f64 {
						return 1.0
					}
				`, nil)

				fScope := MustSucceed(ctx.Scope.Resolve(ctx, "controller"))
				Expect(fScope.Name).To(Equal("controller"))

				By("binding output type")
				output := MustBeOk(fScope.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.F64()))

				By("binding config parameters")
				Expect(fScope.Type.Config).To(HaveLen(3))
				Expect(fScope.Type.Config[0]).To(Equal(types.Param{Name: "setpoint", Type: types.F64()}))
				Expect(fScope.Type.Config[1]).To(Equal(types.Param{Name: "sensor", Type: types.Chan(types.F64())}))
				Expect(fScope.Type.Config[2]).To(Equal(types.Param{Name: "actuator", Type: types.Chan(types.F64())}))

				By("binding input parameters")
				Expect(fScope.Type.Inputs).To(HaveLen(1))
				Expect(fScope.Type.Inputs[0]).To(Equal(types.Param{Name: "enable", Type: types.U8()}))

				By("creating symbols in scope")
				configSymbols := fScope.FilterChildrenByKind(symbol.KindConfig)
				Expect(configSymbols).To(HaveLen(3))
				Expect(configSymbols[0].Name).To(Equal("setpoint"))
				Expect(configSymbols[1].Name).To(Equal("sensor"))
				Expect(configSymbols[2].Name).To(Equal("actuator"))
			})
		})

		Context("config parameter errors", func() {
			It("should diagnose duplicate config parameter names", func() {
				analyzeExpectError(`
					func controller{
						gain f64,
						gain f64
					} () {}
				`, nil, ContainSubstring("name gain conflicts with existing symbol"))
			})
		})

		Context("complex function analysis", func() {
			It("should analyze a PID controller function", func() {
				resolver := symbol.MapResolver{
					"measurement": {
						Name: "measurement",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   5,
					},
					"measurement_time": {
						Name: "measurement_time",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   6,
					},
				}
				analyzeExpectSuccess(`
					func pid{
						kp f32,
						ki f32,
						kd f32,
						setpoint f32
					} (input u8) f64 {
						error := setpoint - measurement
						p := kp * error
						last_measurement_time $= measurement_time
						dt := measurement_time - last_measurement_time
						integral f32 $= 0
						integral = integral + error * f32(dt)
						i := ki * integral
						last_error $= error
						derivative := (error - last_error) / f32(dt)
						d := kd * derivative
						return p + i + d
					}
				`, resolver)
			})
		})
	})

	Describe("Return Statement Validation", func() {
		DescribeTable("valid return types",
			func(src string) {
				analyzeExpectSuccess(src, nil)
			},
			Entry("integer literal return", `func dog() i64 { return 12 }`),
			Entry("integer literal inferred as i32", `func dog() i32 { return 12 }`),
			Entry("integer expression return", `func dog() i32 { return 1 + 1 }`),
			Entry("integer literal on float return", `func dog() f32 { return 12 }`),
			Entry("void function without return", `func dog() {}`),
		)

		DescribeTable("invalid return types",
			func(src string, msgMatcher OmegaMatcher) {
				analyzeExpectError(src, nil, msgMatcher)
			},
			Entry("non-exact-integer float literal on integer return",
				`func dog() i32 { return 1.5 }`,
				ContainSubstring("is not compatible with")),
			Entry("return value in void function",
				`func dog() { return 5 }`,
				ContainSubstring("cannot return a value from a function with no return type")),
			Entry("missing return in function with return type",
				`func dog() f64 {}`,
				Equal("function 'dog' must return a value of type f64 on all paths")),
			Entry("missing return on else path",
				`func dog() f64 { if (5 > 3) { return 2.3 } }`,
				Equal("function 'dog' must return a value of type f64 on all paths")),
			Entry("missing return in deeply nested branch",
				`func dog() f64 {
					if (5 > 3) { return 2.3 }
					else {
						if (12 > 14) {
							if (5 < 7) { return 7 }
						} else { return 5 }
					}
				}`,
				Equal("function 'dog' must return a value of type f64 on all paths")),
		)

		Context("complete return coverage", func() {
			It("should accept if-else with returns on all paths", func() {
				analyzeExpectSuccess(`
					func dog() f64 {
						if (5 > 3) { return 2.3 }
						else { return 1.0 }
					}
				`, nil)
			})

			It("should accept deeply nested if-else with returns on all paths", func() {
				analyzeExpectSuccess(`
					func dog() f64 {
						if (5 > 3) { return 2.3 }
						else if (12 > 14) { return 7.0 }
						else { return 5.0 }
					}
				`, nil)
			})
		})
	})

	Describe("Channel Binding", func() {
		It("should bind global channels used in function body", func() {
			resolver := symbol.MapResolver{
				"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 12},
				"ox_pt_2": {Name: "ox_pt_2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 13},
			}
			ctx := analyzeExpectSuccess(`
				func add() f32 {
					return (ox_pt_1 + ox_pt_2) / 2
				}
			`, resolver)

			f := MustSucceed(ctx.Scope.Resolve(ctx, "add"))
			Expect(f.Channels.Read).To(HaveLen(2))
			Expect(f.Channels.Write).To(BeEmpty())
			Expect(f.Channels.Read[12]).To(Equal("ox_pt_1"))
			Expect(f.Channels.Read[13]).To(Equal("ox_pt_2"))
		})

		It("should bind channel name when writing to a global channel", func() {
			resolver := symbol.MapResolver{
				"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 12},
				"valve":   {Name: "valve", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
			}
			ctx := analyzeExpectSuccess(`
				func setValve() {
					valve = ox_pt_1 * 2
				}
			`, resolver)
			f := MustSucceed(ctx.Scope.Resolve(ctx, "setValve"))
			Expect(f.Channels.Read).To(HaveLen(1))
			Expect(f.Channels.Read[12]).To(Equal("ox_pt_1"))
			Expect(f.Channels.Write).To(HaveLen(1))
			Expect(f.Channels.Write[20]).To(Equal("valve"))
		})
	})

	Describe("Optional Parameters", func() {
		Context("valid optional parameter usage", func() {
			It("should parse single optional parameter", func() {
				ctx := analyzeExpectSuccess(`func add(x i64, y i64 = 0) i64 { return x + y }`, nil)
				funcScope := ctx.Scope.Children[0]
				Expect(funcScope.Type.Inputs).To(HaveLen(2))
				Expect(funcScope.Type.Inputs[0].Name).To(Equal("x"))
				Expect(funcScope.Type.Inputs[0].Type).To(Equal(types.I64()))
				Expect(funcScope.Type.Inputs[0].Value).To(BeNil())
				Expect(funcScope.Type.Inputs[1].Name).To(Equal("y"))
				Expect(funcScope.Type.Inputs[1].Type).To(Equal(types.I64()))
				Expect(funcScope.Type.Inputs[1].Value).To(Equal(int64(0)))
			})

			It("should parse multiple optional parameters", func() {
				ctx := analyzeExpectSuccess(`func multi(a i32, b f64 = 1.5, c u8 = 10) f64 { return f64(a) + b + f64(c) }`, nil)
				funcScope := ctx.Scope.Children[0]
				Expect(funcScope.Type.Inputs).To(HaveLen(3))
				Expect(funcScope.Type.Inputs[0].Name).To(Equal("a"))
				Expect(funcScope.Type.Inputs[0].Value).To(BeNil())
				Expect(funcScope.Type.Inputs[1].Name).To(Equal("b"))
				Expect(funcScope.Type.Inputs[1].Value).To(Equal(1.5))
				Expect(funcScope.Type.Inputs[2].Name).To(Equal("c"))
				Expect(funcScope.Type.Inputs[2].Value).To(Equal(uint8(10)))
			})

			It("should handle functions with no optional parameters", func() {
				ctx := analyzeExpectSuccess(`func multiply(x i64, y i64) i64 { return x * y }`, nil)
				funcScope := ctx.Scope.Children[0]
				Expect(funcScope.Type.Inputs).To(HaveLen(2))
				for _, p := range funcScope.Type.Inputs {
					Expect(p.Value).To(BeNil())
				}
			})
		})

		Context("invalid optional parameter usage", func() {
			DescribeTable("should reject invalid default values",
				func(src string, msgMatcher OmegaMatcher) {
					analyzeExpectError(src, nil, msgMatcher)
				},
				Entry("required after optional",
					`func add(x i64 = 0, y i64) i64 { return x + y }`,
					ContainSubstring("required parameter y cannot follow optional parameters")),
				Entry("overflow in default value",
					`func foo(x i8 = 128) i8 { return x }`,
					ContainSubstring("out of range for i8")),
				Entry("non-integer float to int",
					`func foo(x i32 = 3.14) i32 { return x }`,
					ContainSubstring("cannot convert non-integer float")),
			)
		})
	})

	Describe("Named Output Parameters", func() {
		Context("single named output", func() {
			It("should bind a single named output using parenthesized syntax", func() {
				ctx := analyzeExpectSuccess(`
					func compute() (result f64) {
						result = 42.0
					}
				`, nil)

				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "compute"))
				Expect(funcScope.Type.Outputs).To(HaveLen(1))
				Expect(funcScope.Type.Outputs[0].Name).To(Equal("result"))
				Expect(funcScope.Type.Outputs[0].Type).To(Equal(types.F64()))
			})

			It("should bind a single named output without parentheses", func() {
				ctx := analyzeExpectSuccess(`
					func compute() result f64 {
						result = 42.0
					}
				`, nil)

				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "compute"))
				Expect(funcScope.Type.Outputs).To(HaveLen(1))
				Expect(funcScope.Type.Outputs[0].Name).To(Equal("result"))
				Expect(funcScope.Type.Outputs[0].Type).To(Equal(types.F64()))
			})

			It("should diagnose duplicate output name without parentheses", func() {
				// This tests error handling when a named output conflicts with an existing symbol
				analyzeExpectError(`
					func compute{result f64}() result f64 {
						result = 42.0
					}
				`, nil, ContainSubstring("name result conflicts with existing symbol"))
			})
		})

		Context("multiple named outputs", func() {
			It("should bind multiple named outputs", func() {
				ctx := analyzeExpectSuccess(`
					func compute() (a f64, b f64) {
						a = 1.0
						b = 2.0
					}
				`, nil)

				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "compute"))
				Expect(funcScope.Type.Outputs).To(HaveLen(2))
				Expect(funcScope.Type.Outputs[0].Name).To(Equal("a"))
				Expect(funcScope.Type.Outputs[0].Type).To(Equal(types.F64()))
				Expect(funcScope.Type.Outputs[1].Name).To(Equal("b"))
				Expect(funcScope.Type.Outputs[1].Type).To(Equal(types.F64()))
			})
		})

		Context("output assignment warnings", func() {
			It("should warn when named output is never assigned", func() {
				ctx := analyzeProgram(`
					func compute() (result f64) {
						x := 42.0
					}
				`, nil)
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Severity).To(Equal(diagnostics.SeverityWarning))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("output 'result' is never assigned"))
			})

			It("should not warn when named output is assigned", func() {
				analyzeExpectSuccess(`
					func compute() (result f64) {
						result = 42.0
					}
				`, nil)
			})

			It("should detect assignment in if statement", func() {
				analyzeExpectSuccess(`
					func compute() (result f64) {
						if (1 > 0) {
							result = 42.0
						}
					}
				`, nil)
			})

			It("should detect assignment in else-if clause", func() {
				analyzeExpectSuccess(`
					func compute() (result f64) {
						if (1 < 0) {
							x := 1
						} else if (1 > 0) {
							result = 42.0
						}
					}
				`, nil)
			})

			It("should detect assignment in else clause", func() {
				analyzeExpectSuccess(`
					func compute() (result f64) {
						if (1 < 0) {
							x := 1
						} else {
							result = 42.0
						}
					}
				`, nil)
			})
		})

		Context("duplicate output names", func() {
			It("should diagnose duplicate output names in multi-output block", func() {
				analyzeExpectError(`
					func compute() (a f64, a f64) {
						a = 1.0
					}
				`, nil, ContainSubstring("name a conflicts with existing symbol"))
			})
		})
	})
})
