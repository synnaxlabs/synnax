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
				Expect(channelNode.Key).To(Equal("on_0"))
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
				Expect(edge.Source.Node).To(Equal("on_0"))
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

				Expect(srcNode0.Key).To(Equal("on_0"))
				Expect(edge0.Source.Param).To(Equal("output"))
				Expect(srcNode0.Outputs.Has(edge0.Source.Param)).To(BeTrue(),
					"Source param '%s' should exist in node '%s' outputs %v",
					edge0.Source.Param, srcNode0.Key, srcNode0.Outputs)

				Expect(tgtNode0.Key).To(Equal("filter_1"))
				Expect(edge0.Target.Param).To(Equal("data")) // Should match actual input name
				Expect(tgtNode0.Inputs.Has(edge0.Target.Param)).To(BeTrue(),
					"Target param '%s' should exist in node '%s' inputs %v",
					edge0.Target.Param, tgtNode0.Key, tgtNode0.Inputs)

				// Verify Edge 1: filter -> transform
				edge1 := inter.Edges[1]
				srcNode1, _ := inter.Nodes.Find(edge1.Source.Node)
				tgtNode1, _ := inter.Nodes.Find(edge1.Target.Node)

				Expect(srcNode1.Key).To(Equal("filter_1"))
				Expect(edge1.Source.Param).To(Equal("output")) // filter returns i64 (default output name)
				Expect(srcNode1.Outputs.Has(edge1.Source.Param)).To(BeTrue(),
					"Source param '%s' should exist in node '%s' outputs %v",
					edge1.Source.Param, srcNode1.Key, srcNode1.Outputs)

				Expect(tgtNode1.Key).To(Equal("transform_2"))
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
				channelNode, found := inter.Nodes.Find("on_0")
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

				// Find nodes (numbered sequentially: demux_0, alarm_1, logger_2)
				demuxNode, _ := inter.Nodes.Find("demux_0")
				alarmNode, _ := inter.Nodes.Find("alarm_1")
				loggerNode, _ := inter.Nodes.Find("logger_2")

				// Verify demux has both outputs
				Expect(demuxNode.Outputs).To(HaveLen(2))
				Expect(demuxNode.Outputs.Has("high")).To(BeTrue())
				Expect(demuxNode.Outputs.Has("low")).To(BeTrue())

				// Find edges by source parameter
				var highEdge, lowEdge int = -1, -1
				for i := range inter.Edges {
					if inter.Edges[i].Source.Param == "high" {
						highEdge = i
					} else if inter.Edges[i].Source.Param == "low" {
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

				// Verify edge chain (numbered sequentially: demux_0, amplify_1, display_2)
				edge0 := inter.Edges[0]
				Expect(edge0.Source.Node).To(Equal("demux_0"))
				Expect(edge0.Source.Param).To(Equal("high"))
				Expect(edge0.Target.Node).To(Equal("amplify_1"))

				edge1 := inter.Edges[1]
				Expect(edge1.Source.Node).To(Equal("amplify_1"))
				Expect(edge1.Target.Node).To(Equal("display_2"))
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
				Expect(inter.Strata[0]).To(ContainElement("on_0"))

				// Verify filter is in stratum 1 (numbered sequentially)
				Expect(inter.Strata[1]).To(ContainElement("filter_1"))

				// Verify transform is in stratum 2 (numbered sequentially)
				Expect(inter.Strata[2]).To(ContainElement("transform_2"))
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
				// Nodes numbered sequentially: demux_0, alarm_1, logger_2
				Expect(inter.Strata[1]).To(ContainElement("alarm_1"))
				Expect(inter.Strata[1]).To(ContainElement("logger_2"))
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
	})

})
