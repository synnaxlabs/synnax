// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Text", func() {
	Describe("Parse", func() {
		It("Should correctly parse a text-based arc program", func() {
			source := `
			func add(a i64, b i64) i64 {
				return a + b
			}

			func adder{} (a i64, b i64) i64 {
				return add(a, b)
			}

			func print{} () {
			}

			adder{} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			Expect(parsedText.AST).ToNot(BeNil())
		})
	})

	Describe("Analyze", func() {
		It("Should correctly analyze a text-based arc program", func() {
			source := `
			func add(a i64, b i64) i64 {
				return a + b
			}

			func adder{} (a i64, b i64) i64 {
				return a + b
			}

			func print{} () {
			}

			adder{} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			Expect(parsedText.AST).ToNot(BeNil())
			inter, diagnostics := text.Analyze(ctx, parsedText, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(3))
			Expect(inter.Nodes).To(HaveLen(2))
			Expect(inter.Edges).To(HaveLen(1))

			f := inter.Functions[0]
			Expect(f.Key).To(Equal("add"))
			Expect(f.Inputs).To(HaveLen(2))
			Expect(f.Inputs[0].Type).To(Equal(types.I64()))
			Expect(f.Inputs[1].Type).To(Equal(types.I64()))

			s := inter.Functions[1]
			Expect(s.Key).To(Equal("adder"))
			Expect(s.Inputs).To(HaveLen(2))
			Expect(s.Inputs[0].Type).To(Equal(types.I64()))
			Expect(s.Inputs[1].Type).To(Equal(types.I64()))

			n1 := inter.Nodes[0]
			Expect(n1.Key).To(Equal("adder_0"))
			Expect(n1.Type).To(Equal("adder"))
			Expect(n1.Config).To(HaveLen(0))
			Expect(n1.Channels.Read).ToNot(BeNil())
			Expect(n1.Channels.Read).To(BeEmpty())
			Expect(n1.Channels.Write).ToNot(BeNil())
			Expect(n1.Channels.Write).To(BeEmpty())
		})

		Describe("Channel Flow Analysis", func() {
			It("Should analyze flow with channel identifier", func() {
				resolver := symbol.MapResolver(map[string]symbol.Symbol{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   42,
					},
				})

				source := `
				func print{} () {
				}

				sensor -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				Expect(inter.Edges).To(HaveLen(1))

				// First node should be "on" node for channel
				channelNode := inter.Nodes[0]
				Expect(channelNode.Key).To(Equal("on_sensor_0"))
				Expect(channelNode.Type).To(Equal("on"))
				Expect(channelNode.Config).To(HaveLen(1))
				Expect(channelNode.Config[0].Name).To(Equal("channel"))
				Expect(channelNode.Config[0].Type).To(Equal(types.Chan(types.I32())))
				Expect(channelNode.Channels.Read.Contains(42)).To(BeTrue())

				// Second node should be print function
				printNode := inter.Nodes[1]
				Expect(printNode.Type).To(Equal("print"))

				// Edge should connect channel to print
				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal("on_sensor_0"))
				Expect(edge.Target.Node).To(Equal(printNode.Key))
			})

			It("Should report error for unresolved channel", func() {
				source := `
				func print{} () {
				}

				unknown_channel -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
			})
		})

		Describe("Expression Flow Analysis", func() {
			It("Should analyze flow with expression nodes", func() {
				source := `
				func add(a i64, b i64) i64 {
					return a + b
				}

				func print{} () {
				}

				add(1, 2) -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				Expect(inter.Edges).To(HaveLen(1))

				// First node should be the expression
				exprNode := inter.Nodes[0]
				Expect(exprNode.Key).ToNot(BeEmpty())
				Expect(exprNode.Type).ToNot(BeEmpty())

				// Second node should be print
				printNode := inter.Nodes[1]
				Expect(printNode.Type).To(Equal("print"))

				// Edge should connect expression to print
				edge := inter.Edges[0]
				Expect(edge.Target.Node).To(Equal(printNode.Key))
			})

			It("Should generate constant node for integer literal", func() {
				resolver := symbol.MapResolver{
					"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
				}
				source := `1 -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Find the constant node
				var constantNode *ir.Node
				for i := range inter.Nodes {
					if inter.Nodes[i].Type == "constant" {
						constantNode = &inter.Nodes[i]
						break
					}
				}
				Expect(constantNode).ToNot(BeNil(), "expected constant node")
				Expect(constantNode.Type).To(Equal("constant"))
				Expect(constantNode.Config).To(HaveLen(1))
				Expect(constantNode.Config[0].Name).To(Equal("value"))
				Expect(constantNode.Config[0].Type).To(Equal(types.F32()))
			})

			It("Should generate constant node for float literal", func() {
				resolver := symbol.MapResolver{
					"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
				}
				source := `3.14 -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				var constantNode *ir.Node
				for i := range inter.Nodes {
					if inter.Nodes[i].Type == "constant" {
						constantNode = &inter.Nodes[i]
						break
					}
				}
				Expect(constantNode).ToNot(BeNil())
				Expect(constantNode.Config[0].Type).To(Equal(types.F64()))
			})

			It("Should generate expr node for complex expression, not constant", func() {
				resolver := symbol.MapResolver{
					"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1},
				}
				source := `1 + 2 -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should NOT have a constant node
				for _, node := range inter.Nodes {
					Expect(node.Type).ToNot(Equal("constant"), "complex expressions should use expression_ not constant")
				}
			})
		})

		Describe("Config Values", func() {
			It("Should extract named config values", func() {
				source := `
				func processor{
					threshold i64
					scale f64
				} () i64 {
					return threshold
				}

				func print{} () {
				}

				processor{threshold=100, scale=2.5} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				node := inter.Nodes[0]
				Expect(node.Type).To(Equal("processor"))
				Expect(len(node.Config)).To(Equal(2))
				Expect(node.Config[0].Name).To(Equal("threshold"))
				Expect(node.Config[0].Type).To(Equal(types.I64()))
				Expect(node.Config[0].Value).To(Equal("100"))
				Expect(node.Config[1].Name).To(Equal("scale"))
				Expect(node.Config[1].Type).To(Equal(types.F64()))
				Expect(node.Config[1].Value).To(Equal("2.5"))
			})

			It("Should handle simple config with multiple values", func() {
				source := `
				func calculator{
					a i64
					b i64
					c i64
				} () i64 {
					return a + b + c
				}

				func print{} () {
				}

				calculator{a=10,b=20,c=30} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := inter.Nodes[0]
				Expect(node.Type).To(Equal("calculator"))
				Expect(len(node.Config)).To(Equal(3))
				Expect(node.Config[0].Name).To(Equal("a"))
				Expect(node.Config[0].Type).To(Equal(types.I64()))
				Expect(node.Config[0].Value).To(Equal("10"))
				Expect(node.Config[1].Name).To(Equal("b"))
				Expect(node.Config[1].Type).To(Equal(types.I64()))
				Expect(node.Config[1].Value).To(Equal("20"))
				Expect(node.Config[2].Name).To(Equal("c"))
				Expect(node.Config[2].Type).To(Equal(types.I64()))
				Expect(node.Config[2].Value).To(Equal("30"))
			})
		})

		Describe("Edge Parameter Validation", func() {
			It("Should create edges with parameters that exist in node definitions", func() {
				resolver := symbol.MapResolver(map[string]symbol.Symbol{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   1,
					},
				})

				source := `
				func filter{} (data i64) i64 {
					return data
				}

				func transform{} (value i64) i64 {
					return value * 2
				}

				sensor -> filter{} -> transform{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

				// Verify Edge 0: sensor -> filter
				edge0 := inter.Edges[0]
				srcNode0, _ := inter.Nodes.Find(edge0.Source.Node)
				tgtNode0, _ := inter.Nodes.Find(edge0.Target.Node)

				Expect(srcNode0.Key).To(Equal("on_sensor_0"))
				Expect(edge0.Source.Param).To(Equal("output"))
				Expect(srcNode0.Outputs.Has(edge0.Source.Param)).To(BeTrue(),
					"Source param '%s' should exist in node '%s' outputs %v",
					edge0.Source.Param, srcNode0.Key, srcNode0.Outputs)

				Expect(tgtNode0.Key).To(Equal("filter_0"))
				Expect(edge0.Target.Param).To(Equal("data")) // Should match actual input name
				Expect(tgtNode0.Inputs.Has(edge0.Target.Param)).To(BeTrue(),
					"Target param '%s' should exist in node '%s' inputs %v",
					edge0.Target.Param, tgtNode0.Key, tgtNode0.Inputs)

				// Verify Edge 1: filter -> transform
				edge1 := inter.Edges[1]
				srcNode1, _ := inter.Nodes.Find(edge1.Source.Node)
				tgtNode1, _ := inter.Nodes.Find(edge1.Target.Node)

				Expect(srcNode1.Key).To(Equal("filter_0"))
				Expect(edge1.Source.Param).To(Equal("output")) // filter returns i64 (default output name)
				Expect(srcNode1.Outputs.Has(edge1.Source.Param)).To(BeTrue(),
					"Source param '%s' should exist in node '%s' outputs %v",
					edge1.Source.Param, srcNode1.Key, srcNode1.Outputs)

				Expect(tgtNode1.Key).To(Equal("transform_0"))
				Expect(edge1.Target.Param).To(Equal("value")) // Should match actual input name
				Expect(tgtNode1.Inputs.Has(edge1.Target.Param)).To(BeTrue(),
					"Target param '%s' should exist in node '%s' inputs %v",
					edge1.Target.Param, tgtNode1.Key, tgtNode1.Inputs)
			})

			It("Should handle functions with custom input parameter names", func() {
				source := `
				func generator{} () i64 {
					return 42
				}

				func processor{} (inputValue i64) i64 {
					return inputValue * 2
				}

				generator{} -> processor{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				srcNode, _ := inter.Nodes.Find(edge.Source.Node)
				tgtNode, _ := inter.Nodes.Find(edge.Target.Node)

				// Source should use default output "output"
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(srcNode.Outputs.Has("output")).To(BeTrue())

				// Target should reference the actual input name "inputValue"
				Expect(edge.Target.Param).To(Equal("inputValue"))
				Expect(tgtNode.Inputs.Has("inputValue")).To(BeTrue())
			})

			It("Should verify channel node outputs are defined", func() {
				resolver := symbol.MapResolver(map[string]symbol.Symbol{
					"temp": {
						Name: "temp",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   42,
					},
				})

				source := `
				func display{} (value f64) {
				}

				temp -> display{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Find channel node
				channelNode, found := inter.Nodes.Find("on_temp_0")
				Expect(found).To(BeTrue())

				// Verify channel node has outputs defined
				Expect(channelNode.Outputs).To(HaveLen(1))
				Expect(channelNode.Outputs[0].Name).To(Equal("output"))
				Expect(channelNode.Outputs[0].Type).To(Equal(types.F64()))

				// Verify edge uses the defined output parameter
				edge := inter.Edges[0]
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(channelNode.Outputs.Has(edge.Source.Param)).To(BeTrue())
			})

			It("Should handle binary operator parameter names", func() {
				source := `
				func add{} (a i64, b i64) i64 {
					return a + b
				}

				func print{} (value i64) {
				}

				add{} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				edge := inter.Edges[0]
				srcNode, _ := inter.Nodes.Find(edge.Source.Node)
				tgtNode, _ := inter.Nodes.Find(edge.Target.Node)

				// Source should use default output (add returns i64)
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(srcNode.Outputs.Has(edge.Source.Param)).To(BeTrue())

				// Target should use first input name "value"
				Expect(edge.Target.Param).To(Equal("value"))
				Expect(tgtNode.Inputs.Has(edge.Target.Param)).To(BeTrue())

				// Verify add node has both inputs defined
				Expect(srcNode.Inputs).To(HaveLen(2))
				Expect(srcNode.Inputs.Has("a")).To(BeTrue())
				Expect(srcNode.Inputs.Has("b")).To(BeTrue())
			})
		})

		Describe("Output Routing Tables", func() {
			It("Should analyze simple output routing with multiple targets", func() {
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
					if (value > threshold) {
						high = value
					} else {
						low = value
					}
				}

				func alarm{} (value f64) {
				}

				func logger{} (value f64) {
				}

				demux{threshold=100.0} -> {
					high: alarm{},
					low: logger{}
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have: demux, alarm, logger
				Expect(inter.Nodes).To(HaveLen(3))

				// Should have 2 edges: demux.high -> alarm, demux.low -> logger
				Expect(inter.Edges).To(HaveLen(2))

				// Find nodes (each function type has its own counter)
				demuxNode, _ := inter.Nodes.Find("demux_0")
				alarmNode, _ := inter.Nodes.Find("alarm_0")
				loggerNode, _ := inter.Nodes.Find("logger_0")

				// Verify demux has both outputs
				Expect(demuxNode.Outputs).To(HaveLen(2))
				Expect(demuxNode.Outputs.Has("high")).To(BeTrue())
				Expect(demuxNode.Outputs.Has("low")).To(BeTrue())

				// Find edges by source parameter
				var highEdge, lowEdge = -1, -1
				for i := range inter.Edges {
					switch inter.Edges[i].Source.Param {
					case "high":
						highEdge = i
					case "low":
						lowEdge = i
					}
				}

				Expect(highEdge).ToNot(Equal(-1))
				Expect(lowEdge).ToNot(Equal(-1))

				// Verify high edge
				Expect(inter.Edges[highEdge].Source.Node).To(Equal("demux_0"))
				Expect(inter.Edges[highEdge].Source.Param).To(Equal("high"))
				Expect(inter.Edges[highEdge].Target.Node).To(Equal(alarmNode.Key))
				Expect(inter.Edges[highEdge].Target.Param).To(Equal("value")) // alarm's input parameter

				// Verify low edge
				Expect(inter.Edges[lowEdge].Source.Node).To(Equal("demux_0"))
				Expect(inter.Edges[lowEdge].Source.Param).To(Equal("low"))
				Expect(inter.Edges[lowEdge].Target.Node).To(Equal(loggerNode.Key))
				Expect(inter.Edges[lowEdge].Target.Param).To(Equal("value")) // logger's input parameter
			})

			It("Should handle routing with chained processing", func() {
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
					if (value > threshold) {
						high = value
					} else {
						low = value
					}
				}

				func amplify{} (signal f64) f64 {
					return signal * 2
				}

				func display{} (value f64) {
				}

				demux{threshold=100.0} -> {
					high: amplify{} -> display{}
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have: demux, amplify, display
				Expect(inter.Nodes).To(HaveLen(3))

				// Should have 2 edges: demux.high -> amplify, amplify -> display
				Expect(inter.Edges).To(HaveLen(2))

				// Verify edge chain (each function type has its own counter)
				edge0 := inter.Edges[0]
				Expect(edge0.Source.Node).To(Equal("demux_0"))
				Expect(edge0.Source.Param).To(Equal("high"))
				Expect(edge0.Target.Node).To(Equal("amplify_0"))

				edge1 := inter.Edges[1]
				Expect(edge1.Source.Node).To(Equal("amplify_0"))
				Expect(edge1.Target.Node).To(Equal("display_0"))
			})

			It("Should report error for non-existent output parameter", func() {
				source := `
				func simple{} () (bob i64) {
					bob = 42
				}

				func display{} (value i64) {
				}

				simple{} -> {
					nonexistent: display{}
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				// Error message comes from text compiler validation
				Expect(diagnostics.String()).To(ContainSubstring("nonexistent"))
			})
		})

		Describe("Stratification", func() {
			It("Should calculate strata for simple flow chain", func() {
				resolver := symbol.MapResolver(map[string]symbol.Symbol{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   1,
					},
				})

				source := `
				func filter{} (data i64) i64 {
					return data
				}

				func transform{} (value i64) i64 {
					return value * 2
				}

				sensor -> filter{} -> transform{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Verify strata are calculated
				Expect(inter.Strata).ToNot(BeNil())
				Expect(len(inter.Strata)).To(BeNumerically(">=", 1))

				// Strata should have nodes in topological order
				// Stratum 0: sensor (no dependencies)
				// Stratum 1: filter (depends on sensor)
				// Stratum 2: transform (depends on filter)
				Expect(len(inter.Strata)).To(Equal(3))

				// Verify sensor is in stratum 0
				Expect(inter.Strata[0]).To(ContainElement("on_sensor_0"))

				// Verify filter is in stratum 1 (each function type has its own counter)
				Expect(inter.Strata[1]).To(ContainElement("filter_0"))

				// Verify transform is in stratum 2 (each function type has its own counter)
				Expect(inter.Strata[2]).To(ContainElement("transform_0"))
			})

			It("Should calculate strata for output routing tables", func() {
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
					if (value > threshold) {
						high = value
					} else {
						low = value
					}
				}

				func alarm{} (value f64) {
				}

				func logger{} (value f64) {
				}

				demux{threshold=100.0} -> {
					high: alarm{},
					low: logger{}
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Verify strata exist
				Expect(inter.Strata).ToNot(BeNil())
				Expect(len(inter.Strata)).To(Equal(2))

				// Stratum 0: demux (source)
				Expect(inter.Strata[0]).To(ContainElement("demux_0"))

				// Stratum 1: alarm and logger (both depend on demux, can execute in parallel)
				// Each function type has its own counter
				Expect(inter.Strata[1]).To(ContainElement("alarm_0"))
				Expect(inter.Strata[1]).To(ContainElement("logger_0"))
			})
		})

		Describe("Channel Sink Detection", func() {
			It("Should create write node for channel at end of flow", func() {
				resolver := symbol.MapResolver{
					"input_chan": {
						Name: "input_chan",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   1,
					},
					"output_chan": {
						Name: "output_chan",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   2,
					},
				}
				source := `
				func double{} (x f32) f32 {
					return x * 2
				}

				input_chan -> double{} -> output_chan
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))

				// First node: source channel
				inputNode := inter.Nodes[0]
				Expect(inputNode.Type).To(Equal("on"))
				Expect(inputNode.Channels.Read.Contains(uint32(1))).To(BeTrue())
				Expect(inputNode.Outputs).To(HaveLen(1))

				// Last node: sink channel
				outputNode := inter.Nodes[2]
				Expect(outputNode.Type).To(Equal("write"))
				Expect(outputNode.Channels.Write.Contains(uint32(2))).To(BeTrue())
				Expect(outputNode.Inputs).To(HaveLen(1))
				Expect(outputNode.Inputs[0].Name).To(Equal("input"))
				Expect(outputNode.Outputs).To(BeEmpty())
			})

			It("Should handle channel-to-channel flow", func() {
				resolver := symbol.MapResolver{
					"chan1": {
						Name: "chan1",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   1,
					},
					"chan2": {
						Name: "chan2",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   2,
					},
				}
				source := `chan1 -> chan2`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				Expect(inter.Nodes[0].Type).To(Equal("on"))
				Expect(inter.Nodes[0].Channels.Read.Contains(uint32(1))).To(BeTrue())
				Expect(inter.Nodes[1].Type).To(Equal("write"))
				Expect(inter.Nodes[1].Channels.Write.Contains(uint32(2))).To(BeTrue())
			})

			It("Should handle channel sinks in routing tables", func() {
				resolver := symbol.MapResolver{
					"high_chan": {
						Name: "high_chan",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   1,
					},
					"low_chan": {
						Name: "low_chan",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   2,
					},
				}
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
					if (value > threshold) {
						high = value
					} else {
						low = value
					}
				}

				demux{threshold=100.0} -> {
					high: high_chan,
					low: low_chan
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have: demux, high_chan (write), low_chan (write)
				Expect(inter.Nodes).To(HaveLen(3))

				// Find channel nodes by type
				var writeNodes []string
				for _, node := range inter.Nodes {
					if node.Type == "write" {
						writeNodes = append(writeNodes, node.Key)
						Expect(node.Inputs).To(HaveLen(1))
					}
				}
				Expect(writeNodes).To(HaveLen(2))
			})
		})

		Describe("Single Node Flow Validation", func() {
			It("Should error on single-node flow at parse time", func() {
				source := `
				func print{} () {
				}

				print{}
				`
				// Single-node flows are rejected at parse time by the grammar
				_, diagnostics := text.Parse(text.Text{Raw: source})
				Expect(diagnostics).ToNot(BeNil())
				Expect(diagnostics.Ok()).To(BeFalse())
			})

			It("Should error on single-channel flow at parse time", func() {
				source := `sensor`
				// Single-node flows are rejected at parse time by the grammar
				_, diagnostics := text.Parse(text.Text{Raw: source})
				Expect(diagnostics).ToNot(BeNil())
				Expect(diagnostics.Ok()).To(BeFalse())
			})
		})

		Describe("Sequence Targeting", func() {
			It("Should connect one-shot edge to sequence's first stage entry node", func() {
				resolver := symbol.MapResolver{
					"trigger": {
						Name: "trigger",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   1,
					},
				}
				source := `
				sequence main {
					stage run {
					}
				}

				trigger => main
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have:
				// - on_trigger_0 (channel source)
				// - entry_main_run (stage entry from sequence)
				// No write_main_0 should exist!
				Expect(inter.Nodes).To(HaveLen(2))

				// Verify nodes
				triggerNode, found := inter.Nodes.Find("on_trigger_0")
				Expect(found).To(BeTrue())
				Expect(triggerNode.Type).To(Equal("on"))

				entryNode, found := inter.Nodes.Find("entry_main_run")
				Expect(found).To(BeTrue())
				Expect(entryNode.Type).To(Equal("stage_entry"))
				Expect(entryNode.Inputs).To(HaveLen(1))
				Expect(entryNode.Inputs[0].Name).To(Equal("activate"))

				// Verify no write node was created for sequence
				for _, node := range inter.Nodes {
					Expect(node.Key).ToNot(HavePrefix("write_main"))
				}

				// Should have exactly 1 edge
				Expect(inter.Edges).To(HaveLen(1))

				// Verify the edge connects trigger to entry node's activate input
				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal("on_trigger_0"))
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(edge.Target.Node).To(Equal("entry_main_run"))
				Expect(edge.Target.Param).To(Equal("activate"))
				Expect(edge.Kind).To(Equal(ir.OneShot))
			})

			It("Should handle continuous flow to sequence", func() {
				resolver := symbol.MapResolver{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   1,
					},
				}
				source := `
				sequence main {
					stage run {
					}
				}

				sensor -> main
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Verify edge kind is Continuous for -> operator
				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Kind).To(Equal(ir.Continuous))
				Expect(edge.Target.Node).To(Equal("entry_main_run"))
				Expect(edge.Target.Param).To(Equal("activate"))
			})

			It("Should handle sequence with multiple stages - connects to first stage", func() {
				resolver := symbol.MapResolver{
					"trigger": {
						Name: "trigger",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   1,
					},
				}
				source := `
				sequence main {
					stage first {
					}
					stage second {
					}
				}

				trigger => main
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have: on_trigger_0, entry_main_first, entry_main_second
				Expect(inter.Nodes).To(HaveLen(3))

				// Verify edge targets the first stage
				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Target.Node).To(Equal("entry_main_first"))
				Expect(edge.Target.Param).To(Equal("activate"))
			})

			It("Should error when targeting empty sequence (no stages)", func() {
				resolver := symbol.MapResolver{
					"trigger": {
						Name: "trigger",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   1,
					},
				}
				source := `
				sequence empty {
				}

				trigger => empty
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("no stages"))
			})

			It("Should handle sequence in routing table as sink", func() {
				resolver := symbol.MapResolver{
					"high_chan": {
						Name: "high_chan",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   1,
					},
				}
				source := `
				sequence alarm {
					stage active {
					}
				}

				func demux{threshold f64} (value f64) (high f64, low f64) {
					if (value > threshold) {
						high = value
					} else {
						low = value
					}
				}

				demux{threshold=100.0} -> {
					high: alarm,
					low: high_chan
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have: demux_0, entry_alarm_active, write_high_chan_0
				Expect(inter.Nodes).To(HaveLen(3))

				// Verify alarm is an entry node, not a write node
				entryNode, found := inter.Nodes.Find("entry_alarm_active")
				Expect(found).To(BeTrue())
				Expect(entryNode.Type).To(Equal("stage_entry"))

				// Verify high_chan is a write node
				var writeNode *ir.Node
				for i := range inter.Nodes {
					if inter.Nodes[i].Type == "write" {
						writeNode = &inter.Nodes[i]
						break
					}
				}
				Expect(writeNode).ToNot(BeNil())
				Expect(writeNode.Channels.Write.Contains(uint32(1))).To(BeTrue())

				// Should have 2 edges
				Expect(inter.Edges).To(HaveLen(2))

				// Find edge to alarm
				var alarmEdge *ir.Edge
				for i := range inter.Edges {
					if inter.Edges[i].Target.Node == "entry_alarm_active" {
						alarmEdge = &inter.Edges[i]
						break
					}
				}
				Expect(alarmEdge).ToNot(BeNil())
				Expect(alarmEdge.Target.Param).To(Equal("activate"))
			})
		})

		Describe("next keyword", func() {
			It("Should wire next to the following stage's entry node", func() {
				source := `
				sequence main {
					stage first {
						1 -> output,
						input > 10 => next
					}
					stage second {
						0 -> output
					}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diag := text.Analyze(
					ctx,
					parsedText,
					symbol.MapResolver{
						"input":  {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
						"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 2},
					},
				)
				Expect(diag.Ok()).To(BeTrue(), diag.String())

				var nextEdge *ir.Edge
				for i := range inter.Edges {
					if inter.Edges[i].Target.Node == "entry_main_second" {
						nextEdge = &inter.Edges[i]
						break
					}
				}
				Expect(nextEdge).ToNot(BeNil())
				Expect(nextEdge.Target.Param).To(Equal("activate"))
				Expect(nextEdge.Kind).To(Equal(ir.OneShot))
			})

			It("Should error when next is used in the last stage", func() {
				source := `
				sequence main {
					stage only {
						input > 10 => next
					}
				}`
				_, diag := text.Analyze(
					ctx,
					MustSucceed(text.Parse(text.Text{Raw: source})),
					symbol.MapResolver{
						"input": {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					},
				)
				Expect(diag).ToNot(BeNil())
				Expect(diag.String()).To(ContainSubstring("no next stage"))
			})

			It("Should error when next is used outside a sequence", func() {
				source := `input > 10 => next`
				_, diag := text.Analyze(
					ctx,
					MustSucceed(text.Parse(text.Text{Raw: source})),
					symbol.MapResolver{
						"input": {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					},
				)
				Expect(diag).ToNot(BeNil())
				Expect(diag.String()).To(ContainSubstring("outside of a sequence"))
			})
		})
	})

	Describe("Unit Dimensional Analysis", func() {
		It("Should error when adding incompatible dimensions", func() {
			source := `
			func bad() f64 {
				return 5psi + 3s
			}
			`
			_, diag := text.Analyze(
				ctx,
				MustSucceed(text.Parse(text.Text{Raw: source})),
				nil,
			)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("incompatible dimensions"))
		})

		It("Should allow adding same dimensions", func() {
			source := `
			func good() f64 {
				return 100psi + 50psi
			}
			`
			_, diag := text.Analyze(
				ctx,
				MustSucceed(text.Parse(text.Text{Raw: source})),
				nil,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should allow multiplying different dimensions", func() {
			source := `
			func velocity() f64 {
				return 100m / 10s
			}
			`
			_, diag := text.Analyze(
				ctx,
				MustSucceed(text.Parse(text.Text{Raw: source})),
				nil,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})
	})
})
