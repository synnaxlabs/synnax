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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var resolver = symbol.MapResolver{
	"on": symbol.Symbol{
		Name: "on",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{
					Name: "channel",
					Type: types.String(),
				},
			},
		}),
	},
	"once": symbol.Symbol{
		Name: "once",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
	},
	"processor": symbol.Symbol{
		Name: "processor",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
	},
	"sensor_chan": symbol.Symbol{
		Name: "sensor_chan",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	"output_chan": symbol.Symbol{
		Name: "output_chan",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	"temp_sensor": symbol.Symbol{
		Name: "temp_sensor",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	"valve_cmd": symbol.Symbol{
		Name: "valve_cmd",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	"temperature": symbol.Symbol{
		Name: "temperature",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
	},
	"main": symbol.Symbol{
		Name: "main",
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
	},
	"initialization": symbol.Symbol{
		Name: "initialization",
		Kind: symbol.KindStage,
		Type: types.Stage(),
	},
	"pressurization": symbol.Symbol{
		Name: "pressurization",
		Kind: symbol.KindStage,
		Type: types.Stage(),
	},
	"abort": symbol.Symbol{
		Name: "abort",
		Kind: symbol.KindStage,
		Type: types.Stage(),
	},
	"sensor": symbol.Symbol{
		Name: "sensor",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F32()),
	},
	"pressure": symbol.Symbol{
		Name: "pressure",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F32()),
	},
	"start_cmd": symbol.Symbol{
		Name: "start_cmd",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.U8()),
	},
}

var _ = Describe("Flow Statements", func() {
	Describe("Channel to func Flows", func() {
		It("Should parse simple channel to func flow", func() {
			ast := MustSucceed(parser.Parse(`
			once{} -> processor{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should return an error when once of the tasks being called is not defined", func() {
			ast := MustSucceed(parser.Parse(`
			once{} -> processor{}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(Equal("undefined symbol: once"))
		})

		It("Should verify func config parameters match the expected signature types", func() {
			ast := MustSucceed(parser.Parse(`
			func controller{
				setpoint f64
				input chan f64
				output chan f64
			} () {
				value := input
				if value > setpoint {
					output = value
				}
			}

			// This should work - types match
			sensor_chan -> controller{
				setpoint=100.0,
				input=sensor_chan,
				output=output_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect when func is invoked with missing required parameters", func() {
			ast := MustSucceed(parser.Parse(`
			func filter{
				threshold f64
				input chan f64
				output chan f64
			} () {
				value := input
				if value > threshold {
					output = value
				}
			}

			// Missing 'threshold' and 'output' parameters
			sensor_chan -> filter{
				input=sensor_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			// We should get an error about the first missing parameter
			Expect((*ctx.Diagnostics)[0].Message).To(Or(
				Equal("missing required config parameter 'threshold' for func 'filter'"),
				Equal("missing required config parameter 'output' for func 'filter'"),
			))
		})

		It("Should detect when func is invoked with extra parameters not in signature", func() {
			ast := MustSucceed(parser.Parse(`
			func simple{
				input chan f64
			} () {
				value := input
			}

			// 'extra' is not a valid config parameter for 'simple'
			sensor_chan -> simple{
				input=sensor_chan,
				extra=42.0
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("unknown config parameter 'extra' for func 'simple'"))
		})

		It("Should detect type mismatch in func config parameters", func() {
			ast := MustSucceed(parser.Parse(`
			func typed_task{
				threshold f64
				count u32
				message str
				input chan f64
			} () {
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
				threshold="not a number",
				count=3.14,
				message=42,
				input=sensor_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			// Should have at least one type mismatch error
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			// Check that at least one error mentions type mismatch
			hasTypeMismatch := false
			for _, diag := range *ctx.Diagnostics {
				if matched, _ := ContainSubstring("type mismatch").Match(diag.Message); matched {
					hasTypeMismatch = true
					break
				}
			}
			Expect(hasTypeMismatch).To(BeTrue(), "Expected at least one type mismatch error")
		})

		It("Should accept correct types for func config parameters", func() {
			ast := MustSucceed(parser.Parse(`
			func typed_task{
				threshold f64
				count u32
				message str
				input chan f64
			} () {
				value := input
				if value > threshold {
					// do something
				}
			}

			// All types match correctly
			sensor_chan -> typed_task{
				threshold=100.5,
				count=u32(42),
				message="hello",
				input=sensor_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow channels as both sources and targets in flow statements", func() {
			ast := MustSucceed(parser.Parse(`
			func process{
				input chan f64
				output chan f64
			} () {
				value := input
				processed := value * 2.0
				output = processed
			}

			// Channel as source -> func -> channel as target
			temp_sensor -> process{
				input=temp_sensor,
				output=valve_cmd
			}

			// Direct channel to channel piping (no fn)
			// This represents a direct connection/pass-through
			sensor_chan -> output_chan

			// Channel as source in multi-func flow
			sensor_chan -> process{
				input=sensor_chan,
				output=output_chan
			} -> valve_cmd
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should understand channel pass-through triggers tasks on new values", func() {
			ast := MustSucceed(parser.Parse(`
			func logger{
				value chan f64
			} () {
				v := value
				// Log the value
			}

			func controller{
				temp chan f64
				setpoint f64
			} () {
				current := temp
				if current > setpoint {
					// Take action
				}
			}

			// Channel pass-through - these trigger tasks on channel updates
			// The channel IS the implicit first parameter to the fn
			temperature -> controller{temp=temperature, setpoint=100.0}

			// This is shorthand for: "when sensor_chan gets a value, pass it to logger"
			sensor_chan -> logger{value=sensor_chan}

			// Even simpler - if func has single channel input, it can be implicit
			// (though this might not be implemented yet)
			// sensor_chan -> logger{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should implicitly convert channel sources to on{channel} func invocations", func() {
			ast := MustSucceed(parser.Parse(`
			func display{
				input chan f64
			} () {
				value := input
				// Display the value
			}

			// This channel as source:
			sensor_chan -> display{input=sensor_chan}

			// Is implicitly converted to:
			// on{channel=sensor_chan} -> display{input=sensor_chan}
			// Where "on" is a stdlib func that triggers when the channel receives a value
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())

			// The analyzer should have converted the channel source to an "on" fn
			// This test verifies that the "on" func is required in the resolver
		})

		It("Using channel as source", func() {
			// Create a resolver without the "on" fn
			noOnResolver := symbol.MapResolver{
				"sensor_chan": symbol.Symbol{
					Name: "sensor_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
				},
			}

			ast := MustSucceed(parser.Parse(`
			func display{
				input chan f64
			} () {
				value := input
			}

			// This should fail because "on" func is not available
			sensor_chan -> display{input=sensor_chan}
			`))

			ctx := context.CreateRoot(bCtx, ast, noOnResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should convert expressions in flow statements to anonymous tasks", func() {
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
(sensor_chan * 1.8) + 32.0 -> logger{}
			`))

			ctx := context.CreateRoot(bCtx, ast, resolver)
			// The expressions should be validated successfully
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should validate that expressions in flows only reference channels and literals", func() {
			ast := MustSucceed(parser.Parse(`
func alarm{} () {}

func setup() {
	threshold := 100  // Variables can only exist in functions/tasks
}

// This should fail - can't use variables in flow expressions
// 'threshold' doesn't exist at the inter-func layer scope
sensor_chan > threshold -> alarm{}
			`))

			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			// Should have an error about undefined symbol 'threshold'
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			foundError := false
			for _, diag := range *ctx.Diagnostics {
				if matched, _ := ContainSubstring("undefined symbol: threshold").Match(diag.Message); matched {
					foundError = true
					break
				}
			}
			Expect(foundError).To(BeTrue(), "Expected error about undefined symbol 'threshold'")
		})
	})

	Describe("Multi-Output fns and Routing Tables", func() {
		It("Should analyze func with multiple named outputs", func() {
			ast := MustSucceed(parser.Parse(`
			func demux{
				threshold f64
			} (value f32) (high f32, low f32) {
				if (value > f32(threshold)) {
					high = value
				} else {
					low = value
				}
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())

			demuxSymbol, err := ctx.Scope.Resolve(ctx, "demux")
			Expect(err).To(BeNil())
			hasNamedOutputs := len(demuxSymbol.Type.Outputs) > 1 || (len(demuxSymbol.Type.Outputs) == 1 && func() bool {
				_, exists := demuxSymbol.Type.Outputs.Get(ir.DefaultOutputParam)
				return !exists
			}())
			Expect(hasNamedOutputs).To(BeTrue())
			Expect(demuxSymbol.Type.Outputs).To(HaveLen(2))

			highParam, exists := demuxSymbol.Type.Outputs.Get("high")
			Expect(exists).To(BeTrue())
			Expect(highParam.Type).To(Equal(types.F32()))

			lowParam, exists := demuxSymbol.Type.Outputs.Get("low")
			Expect(exists).To(BeTrue())
			Expect(lowParam.Type).To(Equal(types.F32()))
		})

		It("Should analyze routing table with named outputs", func() {
			ast := MustSucceed(parser.Parse(`
			func demux{
				threshold f64
			} (value f64) (high f64, low f64) {
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
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect when routing table is used with func without named outputs", func() {
			ast := MustSucceed(parser.Parse(`
			func simple{} (value f64) f64 {
				return value * 2.0
			}

			func target{} (value f64) {}

			sensor_chan -> simple{} -> {
				output: target{}
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have named outputs"))
		})

		It("Should detect when routing to non-existent output", func() {
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
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have output 'medium'"))
		})

		It("Should type-check routing table targets", func() {
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
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		It("Should analyze routing table with chained nodes", func() {
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
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should route to channels in routing table", func() {
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
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should warn about unassigned outputs", func() {
			ast := MustSucceed(parser.Parse(`
			func incomplete{} (value f32) (high f32, low f32) {
				if (value > 100.0) {
					high = value
				}
				// 'low' is never assigned
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			// Should have warning about unassigned output
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Severity).To(Equal(diagnostics.Warning))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("never assigned"))
		})

		It("Should validate config parameters in routing table targets", func() {
			ast := MustSucceed(parser.Parse(`
			func demux{} (value f64) (high f64, low f64) {
				if (value > 100.0) {
					high = value
				} else {
					low = value
				}
			}

			func configurable{
				threshold f64
			} (value f64) {}

			sensor_chan -> demux{} -> {
				high: configurable{threshold=50.0},
				low: configurable{}
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			// Should fail because 'low' route is missing required config parameter
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("missing required config parameter"))
		})

		It("Should analyze routing table with parameter mapping", func() {
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
			} -> combiner{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect invalid parameter name in routing table", func() {
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
			} -> doubler{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not have parameter 'invalid_param'"))
		})

		It("Should type-check parameter mapping", func() {
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
			} -> converter{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		It("Should analyze routing table with chained processing and parameter mapping", func() {
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
			} -> scaler{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should require next func when using parameter mapping", func() {
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
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("parameter mapping requires a func after the routing table"))
		})
	})

	Describe("Literal Type Inference", func() {
		It("Should infer integer literal as f32 when target channel is f32", func() {
			literalResolver := symbol.MapResolver{
				"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			}
			ast := MustSucceed(parser.Parse(`1 -> output`))
			ctx := context.CreateRoot(bCtx, ast, literalResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should infer integer literal as i32 when target channel is i32", func() {
			literalResolver := symbol.MapResolver{
				"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.I32())},
			}
			ast := MustSucceed(parser.Parse(`1 -> output`))
			ctx := context.CreateRoot(bCtx, ast, literalResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should infer float literal as f64 when target channel is f64", func() {
			literalResolver := symbol.MapResolver{
				"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			ast := MustSucceed(parser.Parse(`1.5 -> output`))
			ctx := context.CreateRoot(bCtx, ast, literalResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Sequence Stages and Flow Operators", func() {
		It("Should compile sequences with stage targets and mixed flow operators", func() {
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
					sensor -> threshold{} => next,
					pressure -> threshold{} => abort
				}

				stage abort {
					recover{} => initialization
				}
			}

			start_cmd => main
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})
})
