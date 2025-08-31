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
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var resolver = symbol.MapResolver{
	"once": symbol.Symbol{
		Name: "once",
		Kind: symbol.KindTask,
		Type: types.Task{},
	},
	"processor": symbol.Symbol{
		Name: "processor",
		Kind: symbol.KindTask,
		Type: types.Task{},
	},
	"sensor_chan": symbol.Symbol{
		Name: "sensor_chan",
		Kind: symbol.KindChannel,
		Type: types.Chan{ValueType: types.F64{}},
	},
	"output_chan": symbol.Symbol{
		Name: "output_chan",
		Kind: symbol.KindChannel,
		Type: types.Chan{ValueType: types.F64{}},
	},
}

var _ = Describe("Flow Statements", func() {
	Describe("Channel to Task Flows", func() {
		It("Should parse simple channel to task flow", func() {
			ast := MustSucceed(parser.Parse(`
once{} -> processor{}
`))
			result := analyzer.Analyze(analyzer.Config{
				Program:  ast,
				Resolver: resolver,
			})
			Expect(result.Diagnostics).To(HaveLen(0))
		})

		It("Should return an error when once of the tasks being called is not defined", func() {
			ast := MustSucceed(parser.Parse(`
			once{} -> processor{}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast})
			Expect(result.Diagnostics).To(HaveLen(1))
			Expect(result.Diagnostics[0].Message).To(Equal("undefined symbol: once"))
		})

		It("Should return an error when one of the symbols being called is not a task", func() {
			ast := MustSucceed(parser.Parse(`
			func dog() {
			}
			once{} -> dog{}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast, Resolver: resolver})
			Expect(result.Diagnostics).To(HaveLen(1))
			Expect(result.Diagnostics[0].Message).To(Equal("dog is not a task"))
		})

		It("Should verify task config parameters match the expected signature types", func() {
			ast := MustSucceed(parser.Parse(`
			task controller{
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
			result := analyzer.Analyze(analyzer.Config{Program: ast, Resolver: resolver})
			Expect(result.Diagnostics).To(HaveLen(0))
		})

		It("Should detect when task is invoked with missing required parameters", func() {
			ast := MustSucceed(parser.Parse(`
			task filter{
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
			result := analyzer.Analyze(analyzer.Config{Program: ast, Resolver: resolver})
			Expect(result.Diagnostics).To(HaveLen(1))
			// We should get an error about the first missing parameter
			Expect(result.Diagnostics[0].Message).To(Or(
				Equal("missing required config parameter 'threshold' for task 'filter'"),
				Equal("missing required config parameter 'output' for task 'filter'"),
			))
		})

		It("Should detect when task is invoked with extra parameters not in signature", func() {
			ast := MustSucceed(parser.Parse(`
			task simple{
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
			result := analyzer.Analyze(analyzer.Config{Program: ast, Resolver: resolver})
			Expect(result.Diagnostics).To(HaveLen(1))
			Expect(result.Diagnostics[0].Message).To(Equal("unknown config parameter 'extra' for task 'simple'"))
		})

		It("Should detect type mismatch in task config parameters", func() {
			ast := MustSucceed(parser.Parse(`
			task typed_task{
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
			result := analyzer.Analyze(analyzer.Config{Program: ast, Resolver: resolver})
			// Should have at least one type mismatch error
			Expect(result.Diagnostics).ToNot(BeEmpty())
			// Check that at least one error mentions type mismatch
			hasTypeMismatch := false
			for _, diag := range result.Diagnostics {
				if matched, _ := ContainSubstring("type mismatch").Match(diag.Message); matched {
					hasTypeMismatch = true
					break
				}
			}
			Expect(hasTypeMismatch).To(BeTrue(), "Expected at least one type mismatch error")
		})

		It("Should accept correct types for task config parameters", func() {
			ast := MustSucceed(parser.Parse(`
			task typed_task{
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
				count: 42u32,
				message: "hello",
				input: sensor_chan
			}
			`))
			result := analyzer.Analyze(analyzer.Config{Program: ast, Resolver: resolver})
			Expect(result.Diagnostics).To(HaveLen(0))
		})
	})
})
