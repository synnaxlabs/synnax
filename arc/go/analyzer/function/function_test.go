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
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

// analyzeProgram is a helper that parses source code and runs the analyzer,
// returning the context for further assertions.
func analyzeProgram(bCtx SpecContext, src string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	prog := MustSucceed(parser.Parse(src))
	ctx := context.CreateRoot(bCtx, prog, resolver)
	analyzer.AnalyzeProgram(ctx)
	return ctx
}

// analyzeExpectSuccess parses and analyzes code, asserting no errors. Warnings
// from passes orthogonal to the function analyzer (e.g., unused-declaration
// warnings on helper functions declared purely as test fixtures) are ignored.
func analyzeExpectSuccess(bCtx SpecContext, src string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	ctx := analyzeProgram(bCtx, src, resolver)
	ExpectWithOffset(1, ctx.Diagnostics.Errors()).To(BeEmpty(), ctx.Diagnostics.String())
	return ctx
}

// analyzeExpectError parses and analyzes code, asserting exactly one error
// whose message matches msgMatcher. Warnings from passes orthogonal to the
// function analyzer are ignored.
func analyzeExpectError(bCtx SpecContext, src string, resolver symbol.Resolver, msgMatcher OmegaMatcher) context.Context[parser.IProgramContext] {
	ctx := analyzeProgram(bCtx, src, resolver)
	errs := ctx.Diagnostics.Errors()
	ExpectWithOffset(1, errs).To(HaveLen(1), ctx.Diagnostics.String())
	ExpectWithOffset(1, errs[0].Message).To(msgMatcher)
	ExpectWithOffset(1, errs[0].Severity).To(Equal(diagnostics.SeverityError))
	return ctx
}

var _ = Describe("Function Analyzer", func() {
	Describe("CollectDeclarations", func() {
		Describe("basic declaration collection", func() {
			It("should handle empty program", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, ``, nil)
				Expect(ctx.Scope.Children).To(BeEmpty())
			})
			It("should collect function with no parameters", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo() {}`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(1))
				fn := ctx.Scope.Children[0]
				Expect(fn.Name).To(Equal("foo"))
				Expect(fn.Kind).To(Equal(symbol.KindFunction))
				Expect(fn.Type.Config).To(BeEmpty())
				Expect(fn.Type.Inputs).To(BeEmpty())
				Expect(fn.Type.Outputs).To(BeEmpty())
			})
			It("should collect multiple functions before body analysis", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `
					func first() i32 { return second() }
					func second() i32 { return 42 }
				`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(2))
				Expect(ctx.Scope.Children[0].Name).To(Equal("first"))
				Expect(ctx.Scope.Children[1].Name).To(Equal("second"))
			})
		})
		Describe("config parameter collection", func() {
			It("should collect function with only config params", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo{x i32}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(1))
				Expect(fn.Type.Config[0]).To(Equal(types.Param{Name: "x", Type: types.I32()}))
				Expect(fn.Type.Inputs).To(BeEmpty())
				Expect(fn.Type.Outputs).To(BeEmpty())
			})
			It("should collect config with channel type", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo{sensor chan f64}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(1))
				Expect(fn.Type.Config[0]).To(Equal(types.Param{Name: "sensor", Type: types.Chan(types.F64())}))
			})
			It("should handle empty config block", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo{}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(BeEmpty())
			})
			It("should collect config with default value", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo{gain f64 = 1.0}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(1))
				Expect(fn.Type.Config[0].Name).To(Equal("gain"))
				Expect(fn.Type.Config[0].Type).To(Equal(types.F64()))
				Expect(fn.Type.Config[0].Value).To(Equal(1.0))
			})
			It("should collect mixed required and optional config params", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo{setpoint f64, gain f64 = 1.0}() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Config).To(HaveLen(2))
				Expect(fn.Type.Config[0].Name).To(Equal("setpoint"))
				Expect(fn.Type.Config[0].Value).To(BeNil())
				Expect(fn.Type.Config[1].Name).To(Equal("gain"))
				Expect(fn.Type.Config[1].Value).To(Equal(1.0))
			})
			It("should reject required config after optional config", func(bCtx SpecContext) {
				analyzeExpectError(bCtx,
					`func foo{gain f64 = 1.0, setpoint f64}() {}`,
					nil,
					ContainSubstring("required config parameter setpoint cannot follow optional config parameters"),
				)
			})
			It("should reject invalid default value for config", func(bCtx SpecContext) {
				analyzeExpectError(bCtx,
					`func foo{x i8 = 128}() {}`,
					nil,
					ContainSubstring("out of range for i8"),
				)
			})
		})
		Describe("input parameter collection", func() {
			It("should collect multiple inputs without defaults", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo(a i32, b f64, c u8) {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Inputs).To(HaveLen(3))
				Expect(fn.Type.Inputs[0]).To(Equal(types.Param{Name: "a", Type: types.I32()}))
				Expect(fn.Type.Inputs[1]).To(Equal(types.Param{Name: "b", Type: types.F64()}))
				Expect(fn.Type.Inputs[2]).To(Equal(types.Param{Name: "c", Type: types.U8()}))
			})
			It("should collect all optional inputs", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo(x i32 = 1, y i32 = 2) {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Inputs).To(HaveLen(2))
				Expect(fn.Type.Inputs[0].Name).To(Equal("x"))
				Expect(fn.Type.Inputs[0].Value).To(Equal(int32(1)))
				Expect(fn.Type.Inputs[1].Name).To(Equal("y"))
				Expect(fn.Type.Inputs[1].Value).To(Equal(int32(2)))
			})
			It("should preserve order of mixed required and optional", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo(a i32, b i32, c i32 = 10) {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Inputs).To(HaveLen(3))
				Expect(fn.Type.Inputs[0].Value).To(BeNil())
				Expect(fn.Type.Inputs[1].Value).To(BeNil())
				Expect(fn.Type.Inputs[2].Value).To(Equal(int32(10)))
			})
		})
		Describe("output parameter collection", func() {
			It("should handle void function", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo() {}`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(BeEmpty())
			})
			It("should collect unnamed output with default param name", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo() i32 { return 0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(1))
				Expect(fn.Type.Outputs[0].Name).To(Equal(ir.DefaultOutputParam))
				Expect(fn.Type.Outputs[0].Type).To(Equal(types.I32()))
			})
			It("should collect single named output without parens", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo() result i32 { result = 0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(1))
				Expect(fn.Type.Outputs[0].Name).To(Equal("result"))
				Expect(fn.Type.Outputs[0].Type).To(Equal(types.I32()))
			})
			It("should collect single named output with parens", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo() (result i32) { result = 0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(1))
				Expect(fn.Type.Outputs[0].Name).To(Equal("result"))
				Expect(fn.Type.Outputs[0].Type).To(Equal(types.I32()))
			})
			It("should collect multiple outputs in order", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func foo() (a i32, b f64) { a = 0 b = 0.0 }`, nil)
				fn := ctx.Scope.Children[0]
				Expect(fn.Type.Outputs).To(HaveLen(2))
				Expect(fn.Type.Outputs[0]).To(Equal(types.Param{Name: "a", Type: types.I32()}))
				Expect(fn.Type.Outputs[1]).To(Equal(types.Param{Name: "b", Type: types.F64()}))
			})
		})
		Describe("error conditions", func() {
			It("should fail on duplicate function names", func(bCtx SpecContext) {
				ctx := analyzeProgram(bCtx, `func foo() {} func foo() {}`, nil)
				errs := ctx.Diagnostics.Errors()
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Message).To(ContainSubstring("conflicts with existing symbol"))
			})
			It("should add functions to root scope", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func outer() { } func inner() { }`, nil)
				Expect(ctx.Scope.Children).To(HaveLen(2))
				for _, child := range ctx.Scope.Children {
					Expect(child.Kind).To(Equal(symbol.KindFunction))
				}
			})
		})
	})

	Describe("Declaration Validation", func() {
		Context("duplicate declarations", func() {
			It("should diagnose duplicate function names", func(bCtx SpecContext) {
				ctx := analyzeExpectError(bCtx, `
					func dog() {}
					func dog() {}
				`, nil, ContainSubstring("name dog conflicts with existing symbol"))
				Expect((*ctx.Diagnostics)[0].Start.Line).To(Equal(3))
			})

			It("should diagnose duplicate parameter names", func(bCtx SpecContext) {
				analyzeExpectError(bCtx, `
					func dog(age i32, age i32) {}
				`, nil, ContainSubstring("name age conflicts with existing symbol"))
			})
		})
	})

	Describe("Parameter Binding", func() {
		Context("basic input and output binding", func() {
			It("should bind input and output types to the function signature", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `
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
			It("should bind config, input, and output types correctly", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `
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
			It("should diagnose duplicate config parameter names", func(bCtx SpecContext) {
				analyzeExpectError(bCtx, `
					func controller{
						gain f64,
						gain f64
					} () {}
				`, nil, ContainSubstring("name gain conflicts with existing symbol"))
			})
		})

		Context("complex function analysis", func() {
			It("should analyze a PID controller function", func(bCtx SpecContext) {
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
				analyzeExpectSuccess(bCtx, `
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
						return f64(p + i + d)
					}
				`, resolver)
			})
		})
	})

	Describe("Return Statement Validation", func() {
		DescribeTable("valid return types",
			func(bCtx SpecContext, src string) {
				analyzeExpectSuccess(bCtx, src, nil)
			},
			Entry("integer literal return", `func dog() i64 { return 12 }`),
			Entry("integer literal inferred as i32", `func dog() i32 { return 12 }`),
			Entry("integer expression return", `func dog() i32 { return 1 + 1 }`),
			Entry("integer literal on float return", `func dog() f32 { return 12 }`),
			Entry("void function without return", `func dog() {}`),
		)

		DescribeTable("invalid return types",
			func(bCtx SpecContext, src string, msgMatcher OmegaMatcher) {
				analyzeExpectError(bCtx, src, nil, msgMatcher)
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
			Entry("concrete f64 expression returned from f32 function",
				`func dog(x f64) f32 { return x * 2.0 }`,
				ContainSubstring("cannot return f64 from 'dog': expected f32")),
			Entry("concrete f32 expression returned from f64 function",
				`func dog(x f32) f64 { return x * 2.0 }`,
				ContainSubstring("cannot return f32 from 'dog': expected f64")),
			Entry("concrete i64 expression returned from f32 function",
				`func dog(x i64) f32 { return x * 2 }`,
				ContainSubstring("cannot return i64 from 'dog': expected f32")),
			Entry("concrete f64 expression returned from i32 function",
				`func dog(x f64) i32 { return x * 2.0 }`,
				ContainSubstring("cannot return f64 from 'dog': expected i32")),
			Entry("concrete i32 expression returned from f32 function",
				`func dog(x i32) f32 { return x * 2 }`,
				ContainSubstring("cannot return i32 from 'dog': expected f32")),
		)

		It("Should reject f64 channel multiplied by f32 channel", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch_f64": {Name: "ch_f64", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
				"ch_f32": {Name: "ch_f32", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 2},
			}
			analyzeExpectError(bCtx,
				`func calc() f64 { return ch_f64 * ch_f32 }`,
				resolver,
				ContainSubstring("cannot use f64 and f32 in * operation"),
			)
		})

		It("Should reject f32 channel multiplied by f64 channel", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch_f32": {Name: "ch_f32", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
				"ch_f64": {Name: "ch_f64", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
			}
			ctx := analyzeProgram(bCtx, `func calc() f64 { return ch_f32 * ch_f64 }`, resolver)
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("cannot use f32 and f64 in * operation"))
		})

		It("Should reject f32 return when literals mask f64 channel in denominator", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"input_power":    {Name: "input_power", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1},
				"drive_speed_fb": {Name: "drive_speed_fb", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
			}
			ctx := analyzeProgram(bCtx,
				`func calc() f32 { return f32(input_power*60)/(2*(3.14159)*(drive_speed_fb)) }`,
				resolver,
			)
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("cannot use f32 and f64 in / operation"))
		})

		It("Should reject f32 expression returned from f64 function with channel inputs", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"input_power":    {Name: "input_power", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1},
				"drive_speed_fb": {Name: "drive_speed_fb", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 2},
			}
			analyzeExpectError(bCtx,
				`func calc() f64 { return f32(input_power*60)/(2*(3.14159)*(drive_speed_fb)) }`,
				resolver,
				ContainSubstring("cannot return f32 from 'calc': expected f64"),
			)
		})

		Context("complete return coverage", func() {
			It("should accept if-else with returns on all paths", func(bCtx SpecContext) {
				analyzeExpectSuccess(bCtx, `
					func dog() f64 {
						if (5 > 3) { return 2.3 }
						else { return 1.0 }
					}
				`, nil)
			})

			It("should accept deeply nested if-else with returns on all paths", func(bCtx SpecContext) {
				analyzeExpectSuccess(bCtx, `
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
		It("should bind global channels used in function body", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 12},
				"ox_pt_2": {Name: "ox_pt_2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 13},
			}
			ctx := analyzeExpectSuccess(bCtx, `
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

		It("should bind channel name when writing to a global channel", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 12},
				"valve":   {Name: "valve", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
			}
			ctx := analyzeExpectSuccess(bCtx, `
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

		Context("channel propagation through function calls", func() {
			It("should propagate channel writes from called function to caller", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"virt": {Name: "virt", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 30},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func dog(cat f32) {
						virt = cat
					}
					func abc() {
						dog(-3.1)
					}
				`, resolver)

				dog := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
				Expect(dog.Channels.Write).To(HaveLen(1))
				Expect(dog.Channels.Write[30]).To(Equal("virt"))

				abc := MustSucceed(ctx.Scope.Resolve(ctx, "abc"))
				Expect(abc.Channels.Write).To(HaveLen(1))
				Expect(abc.Channels.Write[30]).To(Equal("virt"))
			})

			It("should propagate channel reads from called function to caller", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 12},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func readSensor() f32 {
						return ox_pt_1
					}
					func process() {
						readSensor()
					}
				`, resolver)

				readSensor := MustSucceed(ctx.Scope.Resolve(ctx, "readSensor"))
				Expect(readSensor.Channels.Read).To(HaveLen(1))
				Expect(readSensor.Channels.Read[12]).To(Equal("ox_pt_1"))

				process := MustSucceed(ctx.Scope.Resolve(ctx, "process"))
				Expect(process.Channels.Read).To(HaveLen(1))
				Expect(process.Channels.Read[12]).To(Equal("ox_pt_1"))
			})

			It("should propagate channels through multi-level call chains", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"virt": {Name: "virt", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 30},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func abc3(val f32) {
						virt = val
					}
					func abc2(val f32) {
						abc3(val)
					}
					func abc1(val f32) {
						abc2(val)
					}
					func abc_entry() {
						abc1(3.3)
					}
				`, resolver)

				abc3 := MustSucceed(ctx.Scope.Resolve(ctx, "abc3"))
				Expect(abc3.Channels.Write).To(HaveLen(1))
				Expect(abc3.Channels.Write[30]).To(Equal("virt"))

				abc2 := MustSucceed(ctx.Scope.Resolve(ctx, "abc2"))
				Expect(abc2.Channels.Write).To(HaveLen(1))
				Expect(abc2.Channels.Write[30]).To(Equal("virt"))

				abc1 := MustSucceed(ctx.Scope.Resolve(ctx, "abc1"))
				Expect(abc1.Channels.Write).To(HaveLen(1))
				Expect(abc1.Channels.Write[30]).To(Equal("virt"))

				abcEntry := MustSucceed(ctx.Scope.Resolve(ctx, "abc_entry"))
				Expect(abcEntry.Channels.Write).To(HaveLen(1))
				Expect(abcEntry.Channels.Write[30]).To(Equal("virt"))
			})

			It("should combine direct and transitive channel accesses", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"virt1": {Name: "virt1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 30},
					"virt2": {Name: "virt2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 31},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func helper() {
						virt2 = 2.0
					}
					func main_fn() {
						virt1 = 1.0
						helper()
					}
				`, resolver)

				mainFn := MustSucceed(ctx.Scope.Resolve(ctx, "main_fn"))
				Expect(mainFn.Channels.Write).To(HaveLen(2))
				Expect(mainFn.Channels.Write[30]).To(Equal("virt1"))
				Expect(mainFn.Channels.Write[31]).To(Equal("virt2"))
			})

			It("should propagate channels from callee declared after caller", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"virt": {Name: "virt", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 30},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func caller() {
						callee()
					}
					func callee() {
						virt = 1.0
					}
				`, resolver)

				caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
				Expect(caller.Channels.Write).To(HaveLen(1))
				Expect(caller.Channels.Write[30]).To(Equal("virt"))
			})

			It("should track correct channel ID for write through chan input param", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 50},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func helper(my_chan chan f32) {
						my_chan = 1.0
					}
					func abc() {
						helper(ox_pt_1)
					}
				`, resolver)

				helper := MustSucceed(ctx.Scope.Resolve(ctx, "helper"))
				Expect(helper.Channels.Write).To(HaveLen(1))

				abc := MustSucceed(ctx.Scope.Resolve(ctx, "abc"))
				Expect(abc.Channels.Write).To(HaveLen(1))
				Expect(abc.Channels.Write[50]).To(Equal("ox_pt_1"))
			})

			It("should propagate caller write channels when callee uses chan input param", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 50},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func helper(my_chan chan f32) {
						my_chan = 1.0
					}
					func abc() {
						helper(ox_pt_1)
					}
				`, resolver)

				abc := MustSucceed(ctx.Scope.Resolve(ctx, "abc"))
				Expect(abc.Channels.Read).To(HaveLen(1))
				Expect(abc.Channels.Read[50]).To(Equal("ox_pt_1"))
				Expect(abc.Channels.Write).To(HaveLen(1))
				Expect(abc.Channels.Write[50]).To(Equal("ox_pt_1"))
			})

			It("should propagate read channels through chan input param", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 50},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func helper(my_chan chan f32) f32 {
						return my_chan
					}
					func abc() f32 {
						return helper(ox_pt_1)
					}
				`, resolver)

				abc := MustSucceed(ctx.Scope.Resolve(ctx, "abc"))
				Expect(abc.Channels.Read).To(HaveLen(1))
				Expect(abc.Channels.Read[50]).To(Equal("ox_pt_1"))
			})

			It("should propagate channel writes through multi-level chan input param chain", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 50},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func leaf(ch chan f32) {
						ch = 1.0
					}
					func middle(ch chan f32) {
						leaf(ch)
					}
					func top() {
						middle(ox_pt_1)
					}
				`, resolver)

				top := MustSucceed(ctx.Scope.Resolve(ctx, "top"))
				Expect(top.Channels.Write).To(HaveLen(1))
				Expect(top.Channels.Write[50]).To(Equal("ox_pt_1"))
			})

			It("should propagate multiple channel params correctly", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"sensor":   {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
					"actuator": {Name: "actuator", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func process(input chan f32, output chan f32) {
						output = input * 2.0
					}
					func abc() {
						process(sensor, actuator)
					}
				`, resolver)

				abc := MustSucceed(ctx.Scope.Resolve(ctx, "abc"))
				Expect(abc.Channels.Read).To(HaveLen(2))
				Expect(abc.Channels.Read[10]).To(Equal("sensor"))
				Expect(abc.Channels.Read[20]).To(Equal("actuator"))
				Expect(abc.Channels.Write).To(HaveLen(1))
				Expect(abc.Channels.Write[20]).To(Equal("actuator"))
			})

			It("should propagate channels when same function called with different channel args", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"valve_a": {Name: "valve_a", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
					"valve_b": {Name: "valve_b", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 200},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func write_to(ch chan f32) {
						ch = 1.0
					}
					func abc() {
						write_to(valve_a)
						write_to(valve_b)
					}
				`, resolver)

				abc := MustSucceed(ctx.Scope.Resolve(ctx, "abc"))
				Expect(abc.Channels.Read).To(HaveLen(2))
				Expect(abc.Channels.Read[100]).To(Equal("valve_a"))
				Expect(abc.Channels.Read[200]).To(Equal("valve_b"))
				Expect(abc.Channels.Write).To(HaveLen(2))
				Expect(abc.Channels.Write[100]).To(Equal("valve_a"))
				Expect(abc.Channels.Write[200]).To(Equal("valve_b"))
			})

			It("should propagate chan param channels when callee is declared after caller", func(bCtx SpecContext) {
				resolver := symbol.MapResolver{
					"ox_pt_1": {Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 50},
				}
				ctx := analyzeExpectSuccess(bCtx, `
					func top() {
						middle(ox_pt_1)
					}
					func middle(ch chan f32) {
						ch = 1.0
					}
				`, resolver)

				top := MustSucceed(ctx.Scope.Resolve(ctx, "top"))
				Expect(top.Channels.Write).To(HaveLen(1))
				Expect(top.Channels.Write[50]).To(Equal("ox_pt_1"))
				Expect(top.Channels.Read).To(HaveLen(1))
				Expect(top.Channels.Read[50]).To(Equal("ox_pt_1"))
			})
		})
	})

	Describe("Optional Parameters", func() {
		Context("valid optional parameter usage", func() {
			It("should parse single optional parameter", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func add(x i64, y i64 = 0) i64 { return x + y }`, nil)
				funcScope := ctx.Scope.Children[0]
				Expect(funcScope.Type.Inputs).To(HaveLen(2))
				Expect(funcScope.Type.Inputs[0].Name).To(Equal("x"))
				Expect(funcScope.Type.Inputs[0].Type).To(Equal(types.I64()))
				Expect(funcScope.Type.Inputs[0].Value).To(BeNil())
				Expect(funcScope.Type.Inputs[1].Name).To(Equal("y"))
				Expect(funcScope.Type.Inputs[1].Type).To(Equal(types.I64()))
				Expect(funcScope.Type.Inputs[1].Value).To(Equal(int64(0)))
			})

			It("should parse multiple optional parameters", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func multi(a i32, b f64 = 1.5, c u8 = 10) f64 { return f64(a) + b + f64(c) }`, nil)
				funcScope := ctx.Scope.Children[0]
				Expect(funcScope.Type.Inputs).To(HaveLen(3))
				Expect(funcScope.Type.Inputs[0].Name).To(Equal("a"))
				Expect(funcScope.Type.Inputs[0].Value).To(BeNil())
				Expect(funcScope.Type.Inputs[1].Name).To(Equal("b"))
				Expect(funcScope.Type.Inputs[1].Value).To(Equal(1.5))
				Expect(funcScope.Type.Inputs[2].Name).To(Equal("c"))
				Expect(funcScope.Type.Inputs[2].Value).To(Equal(uint8(10)))
			})

			It("should handle functions with no optional parameters", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `func multiply(x i64, y i64) i64 { return x * y }`, nil)
				funcScope := ctx.Scope.Children[0]
				Expect(funcScope.Type.Inputs).To(HaveLen(2))
				for _, p := range funcScope.Type.Inputs {
					Expect(p.Value).To(BeNil())
				}
			})
		})

		Context("invalid optional parameter usage", func() {
			DescribeTable("should reject invalid default values",
				func(bCtx SpecContext, src string, msgMatcher OmegaMatcher) {
					analyzeExpectError(bCtx, src, nil, msgMatcher)
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

	Describe("BlockAlwaysReturns", func() {
		parseBlock := func(src string) parser.IBlockContext {
			prog := MustSucceed(parser.Parse(src))
			return prog.TopLevelItem(0).FunctionDeclaration().Block()
		}
		DescribeTable("should return true when all paths return",
			func(src string) {
				Expect(function.BlockAlwaysReturns(parseBlock(src))).To(BeTrue())
			},
			Entry("bare return", `func f() { return }`),
			Entry("return with value", `func f() i64 { return 1 }`),
			Entry("if/else both return", `func f() i64 { if 1 > 0 { return 1 } else { return 2 } }`),
			Entry("if/else-if/else all return",
				`func f() i64 { if 1 > 2 { return 1 } else if 2 > 3 { return 2 } else { return 3 } }`),
			Entry("nested if/else all return",
				`func f() i64 { if 1 > 0 { if 2 > 0 { return 1 } else { return 2 } } else { return 3 } }`),
			Entry("return after non-returning statement",
				`func f() i64 { x := 1 return x }`),
		)
		DescribeTable("should return false when some paths do not return",
			func(src string) {
				Expect(function.BlockAlwaysReturns(parseBlock(src))).To(BeFalse())
			},
			Entry("empty body", `func f() {}`),
			Entry("no return", `func f() { x := 1 }`),
			Entry("if without else", `func f() { if 1 > 0 { return } }`),
			Entry("if/else with one branch missing return",
				`func f() { if 1 > 0 { return } else { x := 1 } }`),
			Entry("if/else-if missing else",
				`func f() { if 1 > 0 { return } else if 2 > 0 { return } }`),
		)
		It("should return false for nil block", func() {
			Expect(function.BlockAlwaysReturns(nil)).To(BeFalse())
		})
	})

	Describe("Named Output Parameters", func() {
		Context("single named output", func() {
			It("should bind a single named output using parenthesized syntax", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `
					func compute() (result f64) {
						result = 42.0
					}
				`, nil)

				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "compute"))
				Expect(funcScope.Type.Outputs).To(HaveLen(1))
				Expect(funcScope.Type.Outputs[0].Name).To(Equal("result"))
				Expect(funcScope.Type.Outputs[0].Type).To(Equal(types.F64()))
			})

			It("should bind a single named output without parentheses", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `
					func compute() result f64 {
						result = 42.0
					}
				`, nil)

				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "compute"))
				Expect(funcScope.Type.Outputs).To(HaveLen(1))
				Expect(funcScope.Type.Outputs[0].Name).To(Equal("result"))
				Expect(funcScope.Type.Outputs[0].Type).To(Equal(types.F64()))
			})

			It("should diagnose duplicate output name without parentheses", func(bCtx SpecContext) {
				// This tests error handling when a named output conflicts with an existing symbol
				analyzeExpectError(bCtx, `
					func compute{result f64}() result f64 {
						result = 42.0
					}
				`, nil, ContainSubstring("name result conflicts with existing symbol"))
			})
		})

		Context("multiple named outputs", func() {
			It("should bind multiple named outputs", func(bCtx SpecContext) {
				ctx := analyzeExpectSuccess(bCtx, `
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
			It("should warn when named output is never assigned", func(bCtx SpecContext) {
				ctx := analyzeProgram(bCtx, `
					func _compute() (result f64) {
					}
				`, nil)
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Severity).To(Equal(diagnostics.SeverityWarning))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("output 'result' is never assigned"))
			})

			It("should not warn when named output is assigned", func(bCtx SpecContext) {
				analyzeExpectSuccess(bCtx, `
					func compute() (result f64) {
						result = 42.0
					}
				`, nil)
			})

			It("should detect assignment in if statement", func(bCtx SpecContext) {
				analyzeExpectSuccess(bCtx, `
					func compute() (result f64) {
						if (1 > 0) {
							result = 42.0
						}
					}
				`, nil)
			})

			It("should detect assignment in else-if clause", func(bCtx SpecContext) {
				analyzeExpectSuccess(bCtx, `
					func compute() (result f64) {
						if (1 < 0) {
						} else if (1 > 0) {
							result = 42.0
						}
					}
				`, nil)
			})

			It("should detect assignment in else clause", func(bCtx SpecContext) {
				analyzeExpectSuccess(bCtx, `
					func compute() (result f64) {
						if (1 < 0) {
						} else {
							result = 42.0
						}
					}
				`, nil)
			})
		})

		Context("duplicate output names", func() {
			It("should diagnose duplicate output names in multi-output block", func(bCtx SpecContext) {
				analyzeExpectError(bCtx, `
					func compute() (a f64, a f64) {
						a = 1.0
					}
				`, nil, ContainSubstring("name a conflicts with existing symbol"))
			})
		})
	})
})
