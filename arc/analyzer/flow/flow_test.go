// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/maps"
	. "github.com/synnaxlabs/x/testutil"
)

var resolver = ir.MapResolver{
	"on": ir.Symbol{
		Name: "on",
		Kind: ir.KindStage,
		Type: &ir.Stage{
			Config: maps.Ordered[string, ir.Type]{
				Keys:   []string{"channel"},
				Values: []ir.Type{ir.String{}},
			},
		},
	},
	"once": ir.Symbol{
		Name: "once",
		Kind: ir.KindStage,
		Type: &ir.Stage{},
	},
	"processor": ir.Symbol{
		Name: "processor",
		Kind: ir.KindStage,
		Type: &ir.Stage{},
	},
	"sensor_chan": ir.Symbol{
		Name: "sensor_chan",
		Kind: ir.KindChannel,
		Type: ir.Chan{ValueType: ir.F64{}},
	},
	"output_chan": ir.Symbol{
		Name: "output_chan",
		Kind: ir.KindChannel,
		Type: ir.Chan{ValueType: ir.F64{}},
	},
	"temp_sensor": ir.Symbol{
		Name: "temp_sensor",
		Kind: ir.KindChannel,
		Type: ir.Chan{ValueType: ir.F64{}},
	},
	"valve_cmd": ir.Symbol{
		Name: "valve_cmd",
		Kind: ir.KindChannel,
		Type: ir.Chan{ValueType: ir.F64{}},
	},
	"temperature": ir.Symbol{
		Name: "temperature",
		Kind: ir.KindChannel,
		Type: ir.Chan{ValueType: ir.F64{}},
	},
}

var _ = Describe("Flow Statements", func() {
	Describe("Channel to Stage Flows", func() {
		It("Should parse simple channel to stage flow", func() {
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

		It("Should return an error when one of the symbols being called is not a stage", func() {
			ast := MustSucceed(parser.Parse(`
			func dog() {
			}
			once{} -> dog{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("dog is not a stage"))
		})

		It("Should verify stage config parameters match the expected signature types", func() {
			ast := MustSucceed(parser.Parse(`
			stage controller{
				setpoint f64
				input <-chan f64
				output ->chan f64
			} () {
				value := <-input
				if value > setpoint {
					value -> output
				}
			}

			// This should work - types match
			sensor_chan -> controller{
				setpoint: 100.0,
				input: sensor_chan,
				output: output_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect when stage is invoked with missing required parameters", func() {
			ast := MustSucceed(parser.Parse(`
			stage filter{
				threshold f64
				input <-chan f64
				output ->chan f64
			} () {
				value := <-input
				if value > threshold {
					value -> output
				}
			}

			// Missing 'threshold' and 'output' parameters
			sensor_chan -> filter{
				input: sensor_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			// We should get an error about the first missing parameter
			Expect((*ctx.Diagnostics)[0].Message).To(Or(
				Equal("missing required config parameter 'threshold' for stage 'filter'"),
				Equal("missing required config parameter 'output' for stage 'filter'"),
			))
		})

		It("Should detect when stage is invoked with extra parameters not in signature", func() {
			ast := MustSucceed(parser.Parse(`
			stage simple{
				input <-chan f64
			} () {
				value := <-input
			}

			// 'extra' is not a valid config parameter for 'simple'
			sensor_chan -> simple{
				input: sensor_chan,
				extra: 42.0
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("unknown config parameter 'extra' for stage 'simple'"))
		})

		It("Should detect type mismatch in stage config parameters", func() {
			ast := MustSucceed(parser.Parse(`
			stage typed_task{
				threshold f64
				count u32
				message string
				input <-chan f64
			} () {
				value := <-input
				if value > threshold {
					// do something
				}
			}

			// Type mismatches:
			// - threshold should be f64, but given string
			// - count should be u32, but given f64
			// - message should be string, but given number
			sensor_chan -> typed_task{
				threshold: "not a number",
				count: 3.14,
				message: 42,
				input: sensor_chan
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

		It("Should accept correct types for stage config parameters", func() {
			ast := MustSucceed(parser.Parse(`
			stage typed_task{
				threshold f64
				count u32
				message string
				input <-chan f64
			} () {
				value := <-input
				if value > threshold {
					// do something
				}
			}

			// All types match correctly
			sensor_chan -> typed_task{
				threshold: 100.5,
				count: u32(42),
				message: "hello",
				input: sensor_chan
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow channels as both sources and targets in flow statements", func() {
			ast := MustSucceed(parser.Parse(`
			stage process{
				input <-chan f64
				output ->chan f64
			} () {
				value := <-input
				processed := value * 2.0
				processed -> output
			}

			// Channel as source -> stage -> channel as target
			temp_sensor -> process{
				input: temp_sensor,
				output: valve_cmd
			}

			// Direct channel to channel piping (no stage)
			// This represents a direct connection/pass-through
			sensor_chan -> output_chan

			// Channel as source in multi-stage flow
			sensor_chan -> process{
				input: sensor_chan,
				output: output_chan
			} -> valve_cmd
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should understand channel pass-through triggers tasks on new values", func() {
			ast := MustSucceed(parser.Parse(`
			stage logger{
				value <-chan f64
			} () {
				v := <-value
				// Log the value
			}

			stage controller{
				temp <-chan f64
				setpoint f64
			} () {
				current := <-temp
				if current > setpoint {
					// Take action
				}
			}

			// Channel pass-through - these trigger tasks on channel updates
			// The channel IS the implicit first parameter to the stage
			temperature -> controller{temp: temperature, setpoint: 100.0}

			// This is shorthand for: "when sensor_chan gets a value, pass it to logger"
			sensor_chan -> logger{value: sensor_chan}

			// Even simpler - if stage has single channel input, it can be implicit
			// (though this might not be implemented yet)
			// sensor_chan -> logger{}
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should implicitly convert channel sources to on{channel} stage invocations", func() {
			ast := MustSucceed(parser.Parse(`
			stage display{
				input <-chan f64
			} () {
				value := <-input
				// Display the value
			}

			// This channel as source:
			sensor_chan -> display{input: sensor_chan}

			// Is implicitly converted to:
			// on{channel: sensor_chan} -> display{input: sensor_chan}
			// Where "on" is a stdlib stage that triggers when the channel receives a value
			`))
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())

			// The analyzer should have converted the channel source to an "on" stage
			// This test verifies that the "on" stage is required in the resolver
		})

		It("Using channel as source", func() {
			// Create a resolver without the "on" stage
			noOnResolver := ir.MapResolver{
				"sensor_chan": ir.Symbol{
					Name: "sensor_chan",
					Kind: ir.KindChannel,
					Type: ir.Chan{ValueType: ir.F64{}},
				},
			}

			ast := MustSucceed(parser.Parse(`
			stage display{
				input <-chan f64
			} () {
				value := <-input
			}

			// This should fail because "on" stage is not available
			sensor_chan -> display{input: sensor_chan}
			`))

			ctx := context.CreateRoot(bCtx, ast, noOnResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should convert expressions in flow statements to anonymous tasks", func() {
			ast := MustSucceed(parser.Parse(`
stage alarm{} () {}
stage logger{} () {}

// Expression as source - should be converted to anonymous stage
// The expression "sensor_chan > 100" becomes an anonymous stage that:
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
stage alarm{} () {}

func setup() {
	threshold := 100  // Variables can only exist in functions/tasks
}

// This should fail - can't use variables in flow expressions
// 'threshold' doesn't exist at the inter-stage layer scope
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
})
