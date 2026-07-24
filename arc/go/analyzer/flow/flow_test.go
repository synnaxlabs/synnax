// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var resolver = []symbol.Symbol{
	{
		Name: "on",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{
					Name: "channel",
					Type: types.String(),
				},
			},
		}),
	},
	{
		Name: "once",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
	},
	{
		Name: "processor",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
	},
	{
		Name: "sensor_chan",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	{
		Name: "output_chan",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	{
		Name: "temp_sensor",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	{
		Name: "valve_cmd",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	{
		Name: "temperature",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	{
		Name: "main",
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
	},
	{
		Name: "initialization",
		Kind: symbol.KindStage,
		Type: types.Stage(),
	},
	{
		Name: "pressurization",
		Kind: symbol.KindStage,
		Type: types.Stage(),
	},
	{
		Name: "abort",
		Kind: symbol.KindStage,
		Type: types.Stage(),
	},
	{
		Name: "sensor",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F32()),
	},
	{
		Name: "pressure",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F32()),
	},
	{
		Name: "start_cmd",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.U8()),
	},
}

var _ = Describe("Flow Statements", func() {
	Describe("Function Without Input Braces", func() {
		It("Should detect when function follows function invocation without input braces", func(bCtx SpecContext) {
			intervalResolver := []symbol.Symbol{
				{
					Name: "tick",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
						Inputs:  types.Params{{Name: "period", Type: types.TimeSpan()}},
					}),
				},
			}
			ast := MustSucceed(parser.Parse(`
func sim_daq() {}

tick{period=50ms} -> sim_daq`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, intervalResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("sim_daq is not a channel"))
			Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Notes[0].Message).To(Equal("use sim_daq{} to instantiate the function"))
		})

		It("Should detect when function follows channel without input braces", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func sim_daq() {}
sensor_chan -> sim_daq`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("sim_daq is not a channel"))
			Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Notes[0].Message).To(Equal("use sim_daq{} to instantiate the function"))
		})

		It("Should detect when function follows expression without input braces", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func sim_daq() {}
sensor_chan > 100 -> sim_daq`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("sim_daq is not a channel"))
			Expect((*ctx.Diagnostics)[0].Notes).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Notes[0].Message).To(Equal("use sim_daq{} to instantiate the function"))
		})
	})

	Describe("Anonymous Input Values", func() {
		It("Should accept multiple anonymous input values", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func transform{scale f64, offset f64} (x f64) f64 {
    return x * scale + offset
}

func sink{} () {}

sensor_chan -> transform{2.5, 0.1} -> sink{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept partial anonymous input when trailing params have defaults", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func controller{setpoint f64, gain f64 = 1.0} () {}

sensor_chan -> controller{100.0}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect type mismatch in anonymous input values", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func filter{threshold f64} (x f64) f64 {
    return x
}

func sink{} () {}

sensor_chan -> filter{"hello"} -> sink{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("threshold"))
		})

		It("Should reject too many anonymous input values", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func filter{threshold f64} (x f64) f64 {
    return x
}

func sink{} () {}

sensor_chan -> filter{5.0, 20, 30} -> sink{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("too many arguments"))
		})

		It("Should reject partial anonymous input missing required params", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func controller{setpoint f64, gain f64} () {}

sensor_chan -> controller{100.0}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("missing required argument for parameter 'gain' of func 'controller'"))
		})

		It("Should accept a single anonymous input value", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func filter{threshold f64} (x f64) f64 {
    return x
}

func sink{} () {}

sensor_chan -> filter{5.0} -> sink{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept a channel identifier as anonymous input value", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func controller{sensor chan f64, setpoint f64} () {
    v := sensor
}

sensor_chan -> controller{sensor_chan, 100.0}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject zero anonymous input values when params are required", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func filter{threshold f64} (x f64) f64 {
    return x
}

func sink{} () {}

sensor_chan -> filter{} -> sink{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(
				Equal("missing required argument for parameter 'threshold' of func 'filter'"),
			)
		})

		It("Should accept empty input when all params have defaults", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func controller{gain f64 = 1.0, offset f64 = 0.0} () {}

sensor_chan -> controller{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect type mismatch in second anonymous input value", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func transform{scale f64, offset f64} (x f64) f64 {
    return x * scale + offset
}

func sink{} () {}

sensor_chan -> transform{2.5, "bad"} -> sink{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("offset"))
		})
	})

	Describe("Positional arguments with a trigger-first param", func() {
		// Mirrors stdlib funcs (control.set_authority, channels.write) whose
		// trigger is the first input param, so positional binding accounts for it.
		triggerFirstResolver := []symbol.Symbol{
			{
				Name: "set_auth",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecFlow,
				Type: types.Function(types.FunctionProperties{
					Inputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8(), Value: uint8(0)},
						{Name: "value", Type: types.U8()},
						{Name: "channel", Type: types.U8(), Value: uint8(0)},
					},
				}),
				Trigger: symbol.TriggerInput(ir.DefaultOutputParam),
			},
			{
				Name: "start_high_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
			},
		}

		It("Should bind a positional value to the first non-trigger param on a standalone node", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
sequence main {
    stage done {
        set_auth{0}
    }
}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, triggerFirstResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should bind a positional value while the upstream feeds the trigger", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`start_high_cmd -> set_auth{200}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, triggerFirstResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should count only non-trigger params when rejecting too many positional values", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
sequence main {
    stage done {
        set_auth{0, 1, 2}
    }
}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, triggerFirstResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("too many arguments"))
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("expected at most 2"))
		})
	})

	Describe("WASM functions used as flow nodes", func() {
		wasmResolver := []symbol.Symbol{
			{
				Name: "compute",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecWASM,
				Type: types.Function(types.FunctionProperties{
					Inputs:  types.Params{{Name: "input", Type: types.F64()}},
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
				}),
				Trigger: symbol.TriggerOnly,
			},
			{
				Name: "src_chan",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
			},
		}

		It("Should reject a WASM function wired as a flow node", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`src_chan -> compute{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, wasmResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("cannot be used as a flow statement"))
		})

		It("Should reject a WASM function invoked as a standalone stage node", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
sequence main {
    stage done {
        compute{}
    }
}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, wasmResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("cannot be used as a flow statement"))
		})
	})

	Describe("Channel to func Flows", func() {
		Context("function to function connections", func() {
			It("Should detect when func with no output connects to func expecting input", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func source() {}
func sink(v u8) {}
source{} -> sink{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(Equal("func 'source' has no return value but 'sink' expects an input parameter"))
			})

			It("Should detect type mismatch between func output and next func input", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func producer() u8 {
    return 1
}
func consumer(v f64) {}
producer{} -> consumer{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("trigger parameter"))
			})

			It("Should detect unit mismatch between func output and next func input", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func producer() f32 psi {
    return 5psi
}
func consumer(v f32 bar) {}
producer{} -> consumer{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("trigger parameter"))
			})

			It("Should detect when func with multiple params is used without routing table", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func source() u8 {
    return 1
}
func multi(a u8, b u8) {}
source{} -> multi{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(Equal("missing required argument for parameter 'b' of func 'multi'"))
			})

			It("Should detect when func with named outputs is chained without routing table", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func splitter() (high f64, low f64) {
    high = 1.0
    low = 0.0
}
func consumer(v f64) {}
splitter{} -> consumer{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("has named outputs and requires a routing table"))
			})

			It("Should allow valid func-to-func connection with matching types", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func producer() f64 {
    return 1.0
}
func consumer(v f64) {}
producer{} -> consumer{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})
		})

		Context("channel to function type checking", func() {
			It("Should detect when non-channel identifier is used as flow source", func(bCtx SpecContext) {
				localResolver := []symbol.Symbol{
					{Name: "my_func", Kind: symbol.KindFunction, Type: types.Function(types.FunctionProperties{})},
				}
				ast := MustSucceed(parser.Parse(`
func consumer(v f64) {}
my_func -> consumer{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, localResolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(Equal("my_func is not a channel"))
			})

			It("Should detect channel value type mismatch with func parameter", func(bCtx SpecContext) {
				localResolver := []symbol.Symbol{
					{Name: "int_chan", Kind: symbol.KindChannel, Type: types.Chan(types.I32())},
				}
				ast := MustSucceed(parser.Parse(`
func consumer(v f64) {}
int_chan -> consumer{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, localResolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(Equal(
					"upstream value type i32 does not match func 'consumer' trigger parameter 'v' type f64"))
			})
		})

		It("Should parse simple channel to func flow", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			once{} -> processor{}
			`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should return an error when once of the tasks being called is not defined", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			once{} -> processor{}
			`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			// Complete analysis reports all undefined symbols
			Expect(*ctx.Diagnostics).To(HaveLen(2))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: once"))
			Expect((*ctx.Diagnostics)[1].Message).To(Equal("undefined symbol: processor"))
		})

		It("Should verify func input parameters match the expected signature types", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func controller{setpoint f64, input chan f64, output chan f64} () {
			    value := input
			    if value > setpoint {
			        output = value
			    }
			}
			// This should work - types match
			sensor_chan -> controller{setpoint=100.0, input=sensor_chan, output=output_chan}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect when func is invoked with missing required parameters", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func filter{threshold f64, input chan f64, output chan f64} () {
			    value := input
			    if value > threshold {
			        output = value
			    }
			}
			// Missing 'threshold' and 'output' parameters
			sensor_chan -> filter{input=sensor_chan}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			// Complete analysis reports all missing required parameters
			Expect(*ctx.Diagnostics).To(HaveLen(2))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("missing required argument for parameter 'threshold' of func 'filter'"))
			Expect((*ctx.Diagnostics)[1].Message).To(Equal("missing required argument for parameter 'output' of func 'filter'"))
		})

		It("Should allow omitting input param with default value", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func controller{setpoint f64, gain f64 = 1.0} () {}

			sensor_chan -> controller{setpoint=100.0}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow overriding input param default value", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func controller{setpoint f64, gain f64 = 1.0} () {}

			sensor_chan -> controller{setpoint=100.0, gain=2.5}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should still require input params without defaults", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func controller{setpoint f64, gain f64 = 1.0} () {}

			sensor_chan -> controller{gain=2.5}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("missing required argument for parameter 'setpoint' of func 'controller'"))
		})

		It("Should detect when func is invoked with extra parameters not in signature", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func simple{input chan f64} () {
			    value := input
			}
			// 'extra' is not a valid input parameter for 'simple'
			sensor_chan -> simple{input=sensor_chan, extra=42.0}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("unknown parameter 'extra' for func 'simple'"))
		})

		It("Should detect type mismatch in func input parameters", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func typed_task{threshold f64, count u32, message str, input chan f64} () {
			    value := input
			    if value > threshold {
			    // do something
			}
			}
			// Type mismatches:
			// - threshold should be f64, but given str
			// - count should be u32, but given f64
			// - message should be str, but given number
			sensor_chan -> typed_task{
			    threshold = "not a number",
			    count = 3.14,
			    message = 42,
			    input = sensor_chan
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			// Complete analysis reports all type mismatches
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		It("Should accept correct types for func input parameters", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func typed_task{threshold f64, count u32, message str, input chan f64} () {
			    value := input
			    if value > threshold {
			    // do something
			}
			}
			// All types match correctly
			sensor_chan -> typed_task{
			    threshold = 100.5,
			    count = u32(42),
			    message = "hello",
			    input = sensor_chan
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow channels as both sources and targets in flow statements", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func process{input chan f64, output chan f64} () {
			    value := input
			    processed := value * 2.0
			    output = processed
			}
			// Channel as source -> func -> channel as target
			temp_sensor -> process{input=temp_sensor, output=valve_cmd}
			// Direct channel to channel piping (no fn)
			// This represents a direct connection/pass-through
			sensor_chan -> output_chan
			// Channel as source in multi-func flow
			sensor_chan -> process{input=sensor_chan, output=output_chan} -> valve_cmd`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should understand channel pass-through triggers tasks on new values", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func logger{value chan f64} () {
			    v := value
			    // Log the value
			}

			func controller{temp chan f64, setpoint f64} () {
			    current := temp
			    if current > setpoint {
			    // Take action
			}
			}
			// Channel pass-through - these trigger tasks on channel updates
			// The channel IS the implicit first parameter to the fn
			temperature -> controller{temp=temperature, setpoint=100.0}
			// This is shorthand for: "when sensor_chan gets a value, pass it to logger"
			sensor_chan -> logger{value=sensor_chan}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should implicitly convert channel sources to on{channel} func invocations", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func display{input chan f64} () {
			    value := input
			    // Display the value
			}
			// This channel as source:
			sensor_chan -> display{input=sensor_chan}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			// The analyzer should have converted the channel source to an "on" fn
			// This test verifies that the "on" func is required in the resolver
		})

		It("Using channel as source", func(bCtx SpecContext) {
			// Create a resolver without the "on" fn
			noOnResolver := []symbol.Symbol{
				{
					Name: "sensor_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
				},
			}

			ast := MustSucceed(parser.Parse(`
			func display{input chan f64} () {
			    value := input
			}
			// This should fail because "on" func is not available
			sensor_chan -> display{input=sensor_chan}`))

			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, noOnResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should convert expressions in flow statements to anonymous tasks", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func alarm{} () {}
func logger{} () {}
// Expression as source - should be converted to anonymous fn
// The expression "sensor_chan > 100" becomes an anonymous func that:
// 1. Reads from sensor_chan
// 2. Evaluates the comparison
// 3. Outputs u8 (boolean) result
sensor_chan > 100 -> alarm{}
// Arithmetic expression
 (sensor_chan * 1.8) + 32.0 -> logger{}`))

			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			// The expressions should be validated successfully
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should validate that expressions in flows only reference channels and literals", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
func alarm{} () {}

func setup() {
    threshold := 100 // Variables can only exist in functions/tasks
}
// This should fail - can't use variables in flow expressions
// 'threshold' doesn't exist at the inter-func layer scope
sensor_chan > threshold -> alarm{}`))

			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: threshold"))
		})
	})

	Describe("Multi-Output fns and Routing Tables", func() {
		Context("routing table structure validation", func() {
			It("Should detect output routing table not following a func", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func target(v f64) {}
sensor_chan -> {
    high: target{}
}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("output routing table must follow a func invocation"))
			})

			It("Should detect when routing target is not a channel or sequence", func(bCtx SpecContext) {
				localResolver := []symbol.Symbol{
					{Name: "sensor_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
					{Name: "some_var", Kind: symbol.KindVariable, Type: types.F64()},
				}
				ast := MustSucceed(parser.Parse(`
func demux(value f64) (high f64, low f64) {
    high = value
    low = value
}
sensor_chan -> demux{} -> {
    high: some_var
}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, localResolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(Equal("some_var is not a channel, sequence, or stage"))
			})

			It("Should accept a stage name as a routing table target", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
func router{} (value f64) (high u8, low u8) {
    if (value > 50) {
        high = 1
    } else {
        low = 1
    }
}

sequence main {
    stage first {
        sensor_chan -> router{} -> {
            high: opt_1,
        }
    }
    stage opt_1 {}
}`))
				localResolver := []symbol.Symbol{
					{Name: "sensor_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				}
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, localResolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})
		})

		It("Should analyze func with multiple named outputs", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{threshold f64} (value f32) (high f32, low f32) {
			    if (value > f32(threshold)) {
			        high = value
			    } else {
			        low = value
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			demuxSymbol := MustSucceed(ctx.Scope.Resolve(ctx, "demux"))
			// Verify this has named outputs (not a default output)
			_, hasDefaultOutput := demuxSymbol.Type.Outputs.Get(ir.DefaultOutputParam)
			hasNamedOutputs := len(demuxSymbol.Type.Outputs) > 1 || (len(demuxSymbol.Type.Outputs) == 1 && !hasDefaultOutput)
			Expect(hasNamedOutputs).To(BeTrue())
			Expect(demuxSymbol.Type.Outputs).To(HaveLen(2))

			highParam := MustBeOk(demuxSymbol.Type.Outputs.Get("high"))
			Expect(highParam.Type).To(Equal(types.F32()))

			lowParam := MustBeOk(demuxSymbol.Type.Outputs.Get("low"))
			Expect(lowParam.Type).To(Equal(types.F32()))
		})

		It("Should analyze routing table with named outputs", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{threshold f64} (value f64) (high f64, low f64) {
			    if (value > threshold) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func alarm{} (value f64) {}
			func logger{} (value f64) {}

			sensor_chan -> demux{threshold=100.0} -> {
			    high: alarm{},
			    low: logger{}
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect when routing table is used with func without named outputs", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func simple{} (value f64) f64 {
			    return value * 2.0
			}

			func target{} (value f64) {}

			sensor_chan -> simple{} -> {
			    output: target{}
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have named outputs"))
		})

		It("Should detect when routing to non-existent output", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func target{} (value f64) {}

			sensor_chan -> demux{} -> {
			    high: target{},
			    medium: target{}
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have output 'medium'"))
		})

		It("Should type-check routing table targets", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func u32_target{} (value u32) {}

			sensor_chan -> demux{} -> {
			    high: u32_target{}
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should analyze routing table with chained nodes", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func multiplier{} (value f64) f64 {
			    return value * 2.0
			}

			func alarm{} (value f64) {}
			func logger{} (value f64) {}

			sensor_chan -> demux{} -> {
			    high: multiplier{} -> alarm{},
			    low: logger{}
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should type-check multi-node case body via chain, not select output", func(bCtx SpecContext) {
			customResolver := StaticResolver{
				{Name: "sensor_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			sensor_chan -> demux{} -> {
			    high: "above" -> log_str,
			    low: 1 -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject type mismatch in a chained case body node", func(bCtx SpecContext) {
			customResolver := StaticResolver{
				{Name: "sensor_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			sensor_chan -> demux{} -> {
			    high: 123 -> log_str,
			    low: "below" -> log_str
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		It("Should reject a func with named outputs in a non-terminal chain position", func(bCtx SpecContext) {
			customResolver := StaticResolver{
				{Name: "sensor_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			sensor_chan -> demux{} -> {
			    high: demux{} -> log_num,
			    low: 1 -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("named outputs and requires a routing table"))
		})

		It("Should accept select{} with boolean discriminator and string-literal chain bodies", func(bCtx SpecContext) {
			customResolver := StaticResolver{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> select{} -> {
			    true: "high" -> log_str,
			    false: "low" -> log_str
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should infer channel value type for a channel in a non-terminal chain position", func(bCtx SpecContext) {
			customResolver := StaticResolver{
				{Name: "source_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "feed_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func pass{} (value f64) f64 {
			    return value
			}

			source_chan -> demux{} -> {
			    high: feed_chan -> pass{} -> log_num,
			    low: 1 -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should report undefined symbol for an unresolved func in a chain", func(bCtx SpecContext) {
			customResolver := StaticResolver{
				{Name: "source_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			source_chan -> demux{} -> {
			    high: unknown_fn{} -> log_num,
			    low: 1 -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unknown_fn"))
		})

		It("Should accept inline stage with multiple parallel flow statements per case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        flag -> gate{} -> {
			            yes: stage {
			                "yes_branch_a" -> log_str
			                "yes_branch_b" -> log_str
			                1 -> log_num
			            },
			            no: stage {
			                "no_branch_a" -> log_str
			                "no_branch_b" -> log_str
			                0 -> log_num
			            }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept inline sequence with multiple sequential steps per case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        flag -> gate{} -> {
			            yes: sequence {
			                "yes_step_1" -> log_str
			                "yes_step_2" -> log_str
			                1 -> log_num
			            },
			            no: sequence {
			                "no_step_1" -> log_str
			                "no_step_2" -> log_str
			                0 -> log_num
			            }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should register inline stage case bodies under the lexically enclosing scope", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			stage main {
			    flag -> gate{} -> {
			        yes: stage { "yes_branch" -> log_str },
			        no: stage { "no_branch" -> log_str }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			mainScope := ctx.Scope.FindChild("main")
			Expect(mainScope).ToNot(BeNil())
			var inlines []*symbol.Symbol
			for _, child := range mainScope.Children() {
				if strings.HasPrefix(child.Name, ir.InlinePrefix) {
					inlines = append(inlines, child)
				}
			}
			Expect(inlines).To(HaveLen(2),
				"both inline stages must be registered under main with the synth prefix")
		})

		It("Should register inline sequence case bodies under the lexically enclosing scope", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			stage main {
			    flag -> gate{} -> {
			        yes: sequence {
			            "yes_a" -> log_str
			            "yes_b" -> log_str
			        },
			        no: sequence {
			            "no_a" -> log_str
			            "no_b" -> log_str
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			mainScope := ctx.Scope.FindChild("main")
			Expect(mainScope).ToNot(BeNil())
			var inlines []*symbol.Symbol
			for _, child := range mainScope.Children() {
				if strings.HasPrefix(child.Name, ir.InlinePrefix) {
					inlines = append(inlines, child)
				}
			}
			Expect(inlines).To(HaveLen(2),
				"both inline sequences must be registered under main with the synth prefix")
		})

		It("Should reject a named inline stage case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage owner {
			        flag -> gate{} -> {
			            yes: stage my_step { "yes_branch" -> log_str },
			            no: stage { "no_branch" -> log_str }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			var msgs []string
			for _, d := range *ctx.Diagnostics {
				msgs = append(msgs, d.Message)
			}
			Expect(msgs).To(ContainElement(SatisfyAll(
				ContainSubstring("inline routing case body stages must be anonymous"),
				ContainSubstring(`"my_step"`),
			)), "diagnostics: %v", msgs)
		})

		It("Should reject a named inline sequence case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage owner {
			        flag -> gate{} -> {
			            yes: sequence my_seq { "yes_branch" -> log_str },
			            no: sequence { "no_branch" -> log_str }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			var msgs []string
			for _, d := range *ctx.Diagnostics {
				msgs = append(msgs, d.Message)
			}
			Expect(msgs).To(ContainElement(SatisfyAll(
				ContainSubstring("inline routing case body sequences must be anonymous"),
				ContainSubstring(`"my_seq"`),
			)), "diagnostics: %v", msgs)
		})

		It("Should reject `=> name` from a peer stage targeting a named inline", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage owner {
			        flag -> gate{} -> {
			            yes: stage child { "took_yes" -> log_str },
			            no: stage { "took_no" -> log_str }
			        }
			    }
			    stage peer {
			        trigger => child
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse(),
				"expected peer to be unable to reference inline by name; got: %s",
				ctx.Diagnostics.String())
		})

		It("Should accept a routing table mixing one inline case body and one chain case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        flag -> gate{} -> {
			            yes: stage {
			                "yes_inline" -> log_str
			                1 -> log_num
			            },
			            no: "no_chain" -> log_str
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject a type mismatch inside an inline stage case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        flag -> gate{} -> {
			            yes: stage {
			                "wrong_type" -> log_num
			            }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should reject a type mismatch inside an inline sequence case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        flag -> gate{} -> {
			            yes: sequence {
			                "wrong_type" -> log_num
			            }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should report an undefined symbol referenced inside an inline stage case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        flag -> gate{} -> {
			            yes: stage {
			                "hello" -> mystery_chan
			            }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("mystery_chan"))
		})

		It("Should accept an inline case body routed off a custom multi-output func", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "vent_log", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "press_log", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "abort_log", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func decide{low f64, high f64} (value f64) (vent u8, press u8, abort u8) {
			    if (value < low) {
			        vent = 1
			    } else if (value <= high) {
			        press = 1
			    } else {
			        abort = 1
			    }
			}

			sequence main {
			    stage hold {
			        pressure -> decide{low=30.0, high=80.0} -> {
			            vent: stage { "vent" -> vent_log },
			            press: sequence {
			                "press_step_1" -> press_log
			                "press_step_2" -> press_log
			            },
			            abort: "abort" -> abort_log
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept nested routing tables inside an inline stage case body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "outer_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "inner_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			func gate{} (value u8) (yes u8, no u8) {
			    if (value > 0) {
			        yes = 1
			    } else {
			        no = 1
			    }
			}

			sequence main {
			    stage hold {
			        outer_flag -> gate{} -> {
			            yes: stage {
			                inner_flag -> gate{} -> {
			                    yes: stage { "inner_yes" -> log_str },
			                    no: "inner_no" -> log_str
			                }
			            }
			        }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept a top-level inline stage with multiple parallel flow statements", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> stage {
			    "branch_a" -> log_str
			    "branch_b" -> log_str
			    1 -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept a top-level inline sequence with multiple sequential steps", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> sequence {
			    "step_1" -> log_str
			    "step_2" -> log_str
			    1 -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should register top-level inline stage flow targets under the root scope", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "other", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> stage { "first_branch" -> log_str }
			other -> stage { "second_branch" -> log_str }`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			var inlines []*symbol.Symbol
			for _, child := range ctx.Scope.Children() {
				if strings.HasPrefix(child.Name, ir.InlinePrefix) {
					inlines = append(inlines, child)
				}
			}
			Expect(inlines).To(HaveLen(2),
				"both top-level inline stage flow targets must be registered under the root scope")
		})

		It("Should register top-level inline sequence flow targets under the root scope", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "other", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> sequence {
			    "first_a" -> log_str
			    "first_b" -> log_str
			}
			other -> sequence {
			    "second_a" -> log_str
			    "second_b" -> log_str
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())

			var inlines []*symbol.Symbol
			for _, child := range ctx.Scope.Children() {
				if strings.HasPrefix(child.Name, ir.InlinePrefix) {
					inlines = append(inlines, child)
				}
			}
			Expect(inlines).To(HaveLen(2),
				"both top-level inline sequence flow targets must be registered under the root scope")
		})

		It("Should reject a named top-level inline stage flow target", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> stage my_step { "branch" -> log_str }`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			var msgs []string
			for _, d := range *ctx.Diagnostics {
				msgs = append(msgs, d.Message)
			}
			Expect(msgs).To(ContainElement(SatisfyAll(
				ContainSubstring("inline routing case body stages must be anonymous"),
				ContainSubstring(`"my_step"`),
			)), "diagnostics: %v", msgs)
		})

		It("Should reject a named top-level inline sequence flow target", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> sequence my_seq { "branch" -> log_str }`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			var msgs []string
			for _, d := range *ctx.Diagnostics {
				msgs = append(msgs, d.Message)
			}
			Expect(msgs).To(ContainElement(SatisfyAll(
				ContainSubstring("inline routing case body sequences must be anonymous"),
				ContainSubstring(`"my_seq"`),
			)), "diagnostics: %v", msgs)
		})

		It("Should reject `=> name` from a peer stage targeting a named inline flow target", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			sequence main {
			    stage owner {
			        flag -> stage child { "took" -> log_str }
			    }
			    stage peer {
			        trigger => child
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse(),
				"expected peer to be unable to reference inline by name; got: %s",
				ctx.Diagnostics.String())
		})

		It("Should accept a top-level scope mixing one inline flow target and one plain flow chain", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> stage {
			    "inline_branch" -> log_str
			    1 -> log_num
			}
			flag -> log_num`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject a type mismatch inside a top-level inline stage flow target", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> stage {
			    "wrong_type" -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should reject a type mismatch inside a top-level inline sequence flow target", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_num", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> sequence {
			    "wrong_type" -> log_num
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should report an undefined symbol referenced inside a top-level inline stage flow target", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			flag -> stage {
			    "hello" -> mystery_chan
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("mystery_chan"))
		})

		It("Should accept nested top-level inline flow targets", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "outer_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "inner_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			outer_flag -> stage {
			    inner_flag -> stage { "inner" -> log_str }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept a top-level inline flow body containing a routing table with its own inline body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "outer_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "inner_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			outer_flag -> stage {
			    inner_flag -> select{} -> {
			        true: stage { "inner" -> log_str }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept a top-level routing case inline body containing an inline flow body", func(bCtx SpecContext) {
			customResolver := []symbol.Symbol{
				{Name: "outer_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "inner_flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
				{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			}
			ast := MustSucceed(parser.Parse(`
			outer_flag -> select{} -> {
			    true: stage {
			        inner_flag -> stage { "inner" -> log_str }
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, customResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should route to channels in routing table", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			sensor_chan -> demux{} -> {
			    high: output_chan,
			    low: temp_sensor
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should warn about unassigned outputs", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func incomplete{} (value f32) (high f32, low f32) {
			    if (value > 100.0) {
			        high = value
			    }
			    // 'low' is never assigned
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			// Should have warning about unassigned output
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Severity).To(Equal(protocol.DiagnosticSeverityWarning))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("never assigned"))
		})

		It("Should validate input parameters in routing table targets", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func configurable{threshold f64} (value f64) {}

			sensor_chan -> demux{} -> {
			    high: configurable{threshold=50.0},
			    low: configurable{}
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			// Should fail because 'low' route is missing a required argument
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("missing required argument"))
		})

		It("Should analyze routing table with parameter mapping", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func combiner{} (a f64, b f64) f64 {
			    return a + b
			}

			sensor_chan -> demux{} -> {
			    high: output_chan: a,
			    low: temp_sensor: b
			} -> combiner{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect invalid parameter name in routing table", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func doubler{} (a f64) f64 {
			    return a * 2.0
			}

			sensor_chan -> demux{} -> {
			    high: output_chan: invalid_param
			} -> doubler{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have parameter 'invalid_param'"))
		})

		It("Should type-check parameter mapping", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func multiplier{} (value f64) f64 {
			    return value * 2.0
			}

			func converter{} (a f32) f64 {
			    return f64(a) + 1.0
			}

			sensor_chan -> demux{} -> {
			    high: multiplier{}: a
			} -> converter{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		It("Should analyze routing table with chained processing and parameter mapping", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			func filter{} (value f64) f64 {
			    return value
			}

			func amplifier{} (value f64) f64 {
			    return value * 10.0
			}

			func scaler{} (input f64, scale f64) f64 {
			    return input * scale
			}

			sensor_chan -> demux{} -> {
			    high: filter{} -> amplifier{}: input,
			    low: output_chan: scale
			} -> scaler{}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should require next func when using parameter mapping", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
			    if (value > 100.0) {
			        high = value
			    } else {
			        low = value
			    }
			}

			sensor_chan -> demux{} -> {
			    high: output_chan: a
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("parameter mapping requires a func after the routing table"))
		})

		Context("Input routing tables", func() {
			It("Should analyze input routing table mapping sources to parameters", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				func combiner{} (a f64, b f64) f64 {
				    return a + b
				}

				 {
				    sensor_chan: a,
				    output_chan: b
				} -> combiner{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("Should detect when input routing table has invalid parameter name", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				func combiner{} (a f64, b f64) f64 {
				    return a + b
				}

				 {
				    sensor_chan: invalid_param
				} -> combiner{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have parameter 'invalid_param'"))
			})

			It("Should detect when input routing table is not followed by a func", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				{ sensor_chan: a } -> output_chan
				`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("input routing table must precede a func invocation"))
			})

			It("Should detect when input routing entry does not end with identifier", func(bCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				func combiner{} (a f64, b f64) f64 {
				    return a + b
				}

				func processor{} () f64 {
				    return 1.0
				}

				 {
				    sensor_chan: processor{}
				} -> combiner{}`))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("last element in input routing entry must be a parameter name"))
			})
		})
	})

	Describe("Literal Type Inference", func() {
		DescribeTable("Should infer literal types from target channel",
			func(bCtx SpecContext, source string, chanType types.Type) {
				literalResolver := []symbol.Symbol{
					{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(chanType)},
				}
				ast := MustSucceed(parser.Parse(source))
				ctx := context.NewRoot(bCtx, ast, NewRoot(nil, literalResolver...))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			},
			Entry("integer literal to f32 channel", `1 -> output`, types.F32()),
			Entry("integer literal to i32 channel", `1 -> output`, types.I32()),
			Entry("float literal to f64 channel", `1.5 -> output`, types.F64()),
		)
	})

	Describe("Single Invocations in Stages", func() {
		It("Should allow standalone function invocation with no required inputs", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func setup{} () {
			// Do something
			}

			sequence main {
			    stage start {
			        setup{}
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect when standalone function has required inputs without defaults", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func process{} (value f64) {
			// value is required but has no source
			}

			sequence main {
			    stage start {
			        process{}
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("missing required argument for parameter 'value'"))
		})

		It("Should allow standalone function with inputs that have default values", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func greet{} (count u8 = 1) {
			// count has a default value, so this is valid
			}

			sequence main {
			    stage start {
			        greet{}
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow standalone expression in stage", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			sequence main {
			    stage start {
			        1 + 1
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow mixing flow statements and single invocations", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func setup{} () {}
			func process{} (value f32) u8 {
			    return 1
			}

			sequence main {
			    stage start {
			        setup{}
			        sensor -> process{} => next
			    }

			    stage running {
			        sensor -> process{} => next
			    }
			}`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Sequence Stages and Flow Operators", func() {
		It("Should compile sequences with stage targets and mixed flow operators", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
			func threshold{} (val f32) u8 {
			    return val > 100
			}

			func prepare{} () u8 {
			    return 1
			}

			func recover{} () u8 {
			    return 1
			}

			sequence main {
			    stage initialization {
			        sensor -> prepare{} => next
			    }

			    stage pressurization {
			        sensor -> threshold{} => next
			        pressure -> threshold{} => abort
			    }

			    stage abort {
			        recover{} => initialization
			    }
			}

			start_cmd => main`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})
	})
})

var _ = Describe("TriggerOnly upstream activation", func() {
	It("Should accept a void func feeding a TriggerOnly func", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func void_src() {}
		func sink{value f64} () {}
		void_src{} -> sink{value=5.0}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})

	It("Should not type-check a channel upstream against a TriggerOnly func", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func sink{value f64} () {}
		sensor_chan -> sink{value=5.0}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})

	It("Should still reject a void func feeding a func with a wire-fed input", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func void_src() {}
		func consumer(v f64) {}
		void_src{} -> consumer{}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeFalse())
		Expect(ctx.Diagnostics.String()).To(ContainSubstring("has no return value"))
	})
})

var _ = Describe("Trigger call-site conflict", func() {
	It("Should flag a param bound by both a named call-site argument and an upstream wire", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func consumer(v f64) {}
		sensor_chan -> consumer{v=5.0}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeFalse())
		Expect(ctx.Diagnostics.String()).To(ContainSubstring(
			"parameter 'v' of func 'consumer' is bound by both a call-site argument and an upstream wire"))
	})

	It("Should reject a positional argument when the only param is the trigger", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func consumer(v f64) {}
		sensor_chan -> consumer{5.0}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeFalse())
		Expect(ctx.Diagnostics.String()).To(ContainSubstring(
			"too many arguments for func 'consumer': expected at most 0"))
	})
})

var _ = Describe("Duplicate call-site argument", func() {
	It("Should flag a parameter supplied twice by name", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func sink{v i32}() {}
		sensor_chan -> sink{v=1, v=2}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeFalse())
		Expect(ctx.Diagnostics.String()).To(ContainSubstring(
			"duplicate argument for parameter 'v' of func 'sink'"))
	})
})

var _ = Describe("Trigger in select routing branches", func() {
	It("Should not type-check a select branch routing into a TriggerOnly func", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func setter{key_or_name str, message str, variant str} () {}
		start_cmd -> select{} -> {
			true: setter{key_or_name="a", message="up", variant="info"},
			false: setter{key_or_name="b", message="down", variant="info"}
		}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})

	It("Should type-check a select branch routing into a func with a wire-fed input", func(bCtx SpecContext) {
		ast := MustSucceed(parser.Parse(`
		func sink(v str) {}
		start_cmd -> select{} -> {
			true: sink{},
			false: sink{}
		}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeFalse())
		Expect(ctx.Diagnostics.String()).To(ContainSubstring("does not match"))
	})

	It("Should accept a select branch chaining time.now into an i64 channel", func(bCtx SpecContext) {
		customResolver := StaticResolver{
			{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			{Name: "i64_ch", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
		}
		ast := MustSucceed(parser.Parse(`
		import time
		flag -> select{} -> {
			true: time.now{} -> i64_ch,
			false: time.now{} -> i64_ch
		}`))
		ctx := context.NewRoot(bCtx, ast, NewRoot(customResolver))
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})
})

var _ = Describe("Flow Sink Type Compatibility", func() {
	sinkResolver := []symbol.Symbol{
		{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
		{Name: "num_f64", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
		{Name: "sink_f64", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
		{Name: "num_i64", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
	}

	type mismatchCase struct {
		source     string
		line       uint32
		substrings []string
	}

	DescribeTable("Should report a located error when a source does not match the channel sink",
		func(bCtx SpecContext, tc mismatchCase) {
			ast := MustSucceed(parser.Parse(tc.source))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, sinkResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse(), ctx.Diagnostics.String())
			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Range.Start.Line).To(Equal(tc.line))
			for _, s := range tc.substrings {
				Expect(diag.Message).To(ContainSubstring(s))
			}
		},
		Entry("value variable source", mismatchCase{
			source:     "z := 3\n\nz -> log_str",
			line:       2,
			substrings: []string{"z value type", "does not match channel log_str value type str"},
		}),
		Entry("channel source", mismatchCase{
			source:     "num_f64 -> log_str",
			line:       0,
			substrings: []string{"num_f64 value type f64", "does not match channel log_str value type str"},
		}),
		Entry("expression source", mismatchCase{
			source:     "num_f64 + 1.0 -> log_str",
			line:       0,
			substrings: []string{"expression type", "does not match channel log_str value type str"},
		}),
	)

	DescribeTable("Should accept a source whose type matches the channel sink",
		func(bCtx SpecContext, source string) {
			ast := MustSucceed(parser.Parse(source))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, sinkResolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		},
		Entry("value variable to matching channel", "z := 3\n\nz -> num_i64"),
		Entry("channel to matching channel", "num_f64 -> sink_f64"),
	)

	Describe("Writing to a channelRead variable", func() {
		It("Should reject a flow that writes into a channelRead variable", func(bCtx SpecContext) {
			ast := MustSucceed(parser.Parse(`
r := sensor_chan + 1
sensor_chan -> r`))
			ctx := context.NewRoot(bCtx, ast, NewRoot(nil, resolver...))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("read-only"))
		})
	})
})
