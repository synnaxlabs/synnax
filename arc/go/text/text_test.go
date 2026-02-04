// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

// findNodeByKey finds a node by key and asserts it exists
func findNodeByKey(nodes ir.Nodes, key string) ir.Node {
	node, found := nodes.Find(key)
	ExpectWithOffset(1, found).To(BeTrue(), "expected node '%s' to exist", key)
	return node
}

// findNodeByType finds the first node by type and asserts it exists
func findNodeByType(nodes ir.Nodes, nodeType string) ir.Node {
	for _, n := range nodes {
		if n.Type == nodeType {
			return n
		}
	}
	Fail("expected node with type '" + nodeType + "' to exist")
	return ir.Node{}
}

// findEdgeBySourceParam finds an edge by source parameter name
func findEdgeBySourceParam(edges []ir.Edge, param string) ir.Edge {
	for _, e := range edges {
		if e.Source.Param == param {
			return e
		}
	}
	Fail("expected edge with source param '" + param + "' to exist")
	return ir.Edge{}
}

// findEdgeByTarget finds an edge by target node key
func findEdgeByTarget(edges []ir.Edge, targetNode string) ir.Edge {
	for _, e := range edges {
		if e.Target.Node == targetNode {
			return e
		}
	}
	Fail("expected edge with target node '" + targetNode + "' to exist")
	return ir.Edge{}
}

// countNodesByType counts nodes of a specific type
func countNodesByType(nodes ir.Nodes, nodeType string) int {
	count := 0
	for _, n := range nodes {
		if n.Type == nodeType {
			count++
		}
	}
	return count
}

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

			n1 := findNodeByKey(inter.Nodes, "adder_0")
			Expect(n1.Type).To(Equal("adder"))
			Expect(n1.Config).To(HaveLen(0))
			Expect(n1.Channels.Read).ToNot(BeNil())
			Expect(n1.Channels.Read).To(BeEmpty())
			Expect(n1.Channels.Write).ToNot(BeNil())
			Expect(n1.Channels.Write).To(BeEmpty())
		})

		Context("Channel Flow Analysis", func() {
			It("Should analyze flow with channel identifier", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 42},
				}
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

				channelNode := findNodeByKey(inter.Nodes, "on_sensor_0")
				Expect(channelNode.Type).To(Equal("on"))
				Expect(channelNode.Config).To(HaveLen(1))
				Expect(channelNode.Config[0].Name).To(Equal("channel"))
				Expect(channelNode.Config[0].Type).To(Equal(types.Chan(types.I32())))
				Expect(channelNode.Channels.Read.Contains(42)).To(BeTrue())

				printNode := findNodeByKey(inter.Nodes, "print_0")
				Expect(printNode.Type).To(Equal("print"))

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
				Expect(diagnostics.String()).To(ContainSubstring("unknown_channel"))
			})
		})

		Context("Expression Flow Analysis", func() {
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

				exprNode := inter.Nodes[0]
				Expect(exprNode.Key).ToNot(BeEmpty())
				Expect(exprNode.Type).ToNot(BeEmpty())

				printNode := findNodeByKey(inter.Nodes, "print_0")
				Expect(printNode.Type).To(Equal("print"))

				edge := inter.Edges[0]
				Expect(edge.Target.Node).To(Equal(printNode.Key))
			})

			DescribeTable("Literal constant generation",
				func(source string, resolver symbol.MapResolver, expectConstant bool, expectedType types.Type) {
					parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
					inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

					constantCount := countNodesByType(inter.Nodes, "constant")
					if expectConstant {
						Expect(constantCount).To(Equal(1), "expected exactly one constant node")
						constantNode := findNodeByType(inter.Nodes, "constant")
						Expect(constantNode.Config).To(HaveLen(1))
						Expect(constantNode.Config[0].Name).To(Equal("value"))
						Expect(constantNode.Config[0].Type).To(Equal(expectedType))
					} else {
						Expect(constantCount).To(Equal(0), "expected no constant nodes for complex expressions")
					}
				},
				Entry("integer literal",
					`1 -> output`,
					symbol.MapResolver{"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1}},
					true, types.F32(),
				),
				Entry("float literal",
					`3.14 -> output`,
					symbol.MapResolver{"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1}},
					true, types.F64(),
				),
				Entry("complex expression (should not generate constant)",
					`1 + 2 -> output`,
					symbol.MapResolver{"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1}},
					false, types.Type{}, // Type ignored when expectConstant is false
				),
			)
		})

		Context("Config Values", func() {
			It("Should extract named config values", func() {
				source := `
				func processor{
					threshold i64,
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
				node := findNodeByKey(inter.Nodes, "processor_0")
				Expect(node.Type).To(Equal("processor"))
				Expect(node.Config).To(HaveLen(2))
				Expect(node.Config[0].Name).To(Equal("threshold"))
				Expect(node.Config[0].Type).To(Equal(types.I64()))
				Expect(node.Config[0].Value).To(Equal(int64(100)))
				Expect(node.Config[1].Name).To(Equal("scale"))
				Expect(node.Config[1].Type).To(Equal(types.F64()))
				Expect(node.Config[1].Value).To(Equal(2.5))
			})

			It("Should handle simple config with multiple values", func() {
				source := `
				func calculator{
					a i64,
					b i64,
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

				node := findNodeByKey(inter.Nodes, "calculator_0")
				Expect(node.Type).To(Equal("calculator"))
				Expect(node.Config).To(HaveLen(3))

				configValues := map[string]int64{
					"a": 10, "b": 20, "c": 30,
				}
				for i, cfg := range node.Config {
					Expect(cfg.Type).To(Equal(types.I64()))
					Expect(cfg.Value).To(Equal(configValues[cfg.Name]), "config[%d] '%s' value mismatch", i, cfg.Name)
				}
			})

			It("Should handle config values using global constants", func() {
				source := `
				A := 10
				B := 20
				C := 30

				func calculator{
					a i64,
					b i64,
					c i64
				} () i64 {
					return a + b + c
				}

				func print{} () {
				}

				calculator{a=A, b=B, c=C} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "calculator_0")
				Expect(node.Type).To(Equal("calculator"))
				Expect(node.Config).To(HaveLen(3))

				configValues := map[string]int64{
					"a": 10, "b": 20, "c": 30,
				}
				for i, cfg := range node.Config {
					Expect(cfg.Type).To(Equal(types.I64()))
					Expect(cfg.Value).To(Equal(configValues[cfg.Name]), "config[%d] '%s' value mismatch", i, cfg.Name)
				}
			})

			It("Should handle f64 global constants in config", func() {
				source := `
				SCALE := 2.5
				OFFSET := 0.1

				func transform{
					scale f64,
					offset f64
				} (x f64) f64 {
					return x * scale + offset
				}

				func sink{} () {
				}

				transform{scale=SCALE, offset=OFFSET} -> sink{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "transform_0")
				Expect(node.Config).To(HaveLen(2))

				configValues := map[string]float64{
					"scale": 2.5, "offset": 0.1,
				}
				for _, cfg := range node.Config {
					Expect(cfg.Type).To(Equal(types.F64()))
					Expect(cfg.Value).To(Equal(configValues[cfg.Name]))
				}
			})

			It("Should handle mixed literal and constant config values", func() {
				source := `
				THRESHOLD := 100

				func filter{
					threshold i64,
					enabled i64
				} (x i64) i64 {
					return x
				}

				func sink{} () {
				}

				filter{threshold=THRESHOLD, enabled=1} -> sink{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "filter_0")
				Expect(node.Config).To(HaveLen(2))

				for _, cfg := range node.Config {
					switch cfg.Name {
					case "threshold":
						Expect(cfg.Value).To(Equal(int64(100)))
					case "enabled":
						Expect(cfg.Value).To(Equal(int64(1)))
					}
				}
			})

			It("Should handle typed global constants in config", func() {
				source := `
				MAX_VALUE i32 := 255

				func clamp{
					max i32
				} (x i32) i32 {
					return x
				}

				func sink{} () {
				}

				clamp{max=MAX_VALUE} -> sink{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "clamp_0")
				Expect(node.Config).To(HaveLen(1))
				Expect(node.Config[0].Name).To(Equal("max"))
				Expect(node.Config[0].Type).To(Equal(types.I32()))
				Expect(node.Config[0].Value).To(Equal(int32(255)))
			})
		})

		Context("Edge Parameter Validation", func() {
			It("Should create edges with parameters that exist in node definitions", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1},
				}
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
				srcNode0 := findNodeByKey(inter.Nodes, edge0.Source.Node)
				tgtNode0 := findNodeByKey(inter.Nodes, edge0.Target.Node)

				Expect(srcNode0.Key).To(Equal("on_sensor_0"))
				Expect(edge0.Source.Param).To(Equal("output"))
				Expect(srcNode0.Outputs.Has(edge0.Source.Param)).To(BeTrue())

				Expect(tgtNode0.Key).To(Equal("filter_0"))
				Expect(edge0.Target.Param).To(Equal("data"))
				Expect(tgtNode0.Inputs.Has(edge0.Target.Param)).To(BeTrue())

				// Verify Edge 1: filter -> transform
				edge1 := inter.Edges[1]
				srcNode1 := findNodeByKey(inter.Nodes, edge1.Source.Node)
				tgtNode1 := findNodeByKey(inter.Nodes, edge1.Target.Node)

				Expect(srcNode1.Key).To(Equal("filter_0"))
				Expect(edge1.Source.Param).To(Equal("output"))
				Expect(srcNode1.Outputs.Has(edge1.Source.Param)).To(BeTrue())

				Expect(tgtNode1.Key).To(Equal("transform_0"))
				Expect(edge1.Target.Param).To(Equal("value"))
				Expect(tgtNode1.Inputs.Has(edge1.Target.Param)).To(BeTrue())
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
				srcNode := findNodeByKey(inter.Nodes, edge.Source.Node)
				tgtNode := findNodeByKey(inter.Nodes, edge.Target.Node)

				Expect(edge.Source.Param).To(Equal("output"))
				Expect(srcNode.Outputs.Has("output")).To(BeTrue())

				Expect(edge.Target.Param).To(Equal("inputValue"))
				Expect(tgtNode.Inputs.Has("inputValue")).To(BeTrue())
			})

			It("Should verify channel node outputs are defined", func() {
				resolver := symbol.MapResolver{
					"temp": {Name: "temp", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 42},
				}
				source := `
				func display{} (value f64) {
				}

				temp -> display{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				channelNode := findNodeByKey(inter.Nodes, "on_temp_0")
				Expect(channelNode.Outputs).To(HaveLen(1))
				Expect(channelNode.Outputs[0].Name).To(Equal("output"))
				Expect(channelNode.Outputs[0].Type).To(Equal(types.F64()))

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
				srcNode := findNodeByKey(inter.Nodes, edge.Source.Node)
				tgtNode := findNodeByKey(inter.Nodes, edge.Target.Node)

				Expect(edge.Source.Param).To(Equal("output"))
				Expect(srcNode.Outputs.Has(edge.Source.Param)).To(BeTrue())

				Expect(edge.Target.Param).To(Equal("value"))
				Expect(tgtNode.Inputs.Has(edge.Target.Param)).To(BeTrue())

				Expect(srcNode.Inputs).To(HaveLen(2))
				Expect(srcNode.Inputs.Has("a")).To(BeTrue())
				Expect(srcNode.Inputs.Has("b")).To(BeTrue())
			})
		})

		Context("Output Routing Tables", func() {
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

				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

				demuxNode := findNodeByKey(inter.Nodes, "demux_0")
				alarmNode := findNodeByKey(inter.Nodes, "alarm_0")
				loggerNode := findNodeByKey(inter.Nodes, "logger_0")

				Expect(demuxNode.Outputs).To(HaveLen(2))
				Expect(demuxNode.Outputs.Has("high")).To(BeTrue())
				Expect(demuxNode.Outputs.Has("low")).To(BeTrue())

				highEdge := findEdgeBySourceParam(inter.Edges, "high")
				Expect(highEdge.Source.Node).To(Equal("demux_0"))
				Expect(highEdge.Target.Node).To(Equal(alarmNode.Key))
				Expect(highEdge.Target.Param).To(Equal("value"))

				lowEdge := findEdgeBySourceParam(inter.Edges, "low")
				Expect(lowEdge.Source.Node).To(Equal("demux_0"))
				Expect(lowEdge.Target.Node).To(Equal(loggerNode.Key))
				Expect(lowEdge.Target.Param).To(Equal("value"))
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

				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

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
				Expect(diagnostics.String()).To(ContainSubstring("nonexistent"))
			})
		})

		Context("Stratification", func() {
			It("Should calculate strata for simple flow chain", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1},
				}
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

				Expect(inter.Strata).ToNot(BeNil())
				Expect(inter.Strata).To(HaveLen(3))

				// Stratum 0: sensor, Stratum 1: filter, Stratum 2: transform
				Expect(inter.Strata[0]).To(ContainElement("on_sensor_0"))
				Expect(inter.Strata[1]).To(ContainElement("filter_0"))
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

				Expect(inter.Strata).ToNot(BeNil())
				Expect(inter.Strata).To(HaveLen(2))

				// Stratum 0: demux, Stratum 1: alarm and logger (parallel)
				Expect(inter.Strata[0]).To(ContainElement("demux_0"))
				Expect(inter.Strata[1]).To(ContainElements("alarm_0", "logger_0"))
			})
		})

		Context("Channel Sink Detection", func() {
			It("Should create write node for channel at end of flow", func() {
				resolver := symbol.MapResolver{
					"input_chan":  {Name: "input_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					"output_chan": {Name: "output_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 2},
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

				inputNode := inter.Nodes[0]
				Expect(inputNode.Type).To(Equal("on"))
				Expect(inputNode.Channels.Read.Contains(uint32(1))).To(BeTrue())
				Expect(inputNode.Outputs).To(HaveLen(1))

				outputNode := inter.Nodes[2]
				Expect(outputNode.Type).To(Equal("write"))
				Expect(outputNode.Channels.Write.Contains(uint32(2))).To(BeTrue())
				Expect(outputNode.Inputs).To(HaveLen(1))
				Expect(outputNode.Inputs[0].Name).To(Equal("input"))
				Expect(outputNode.Outputs).To(BeEmpty())
			})

			It("Should handle channel-to-channel flow", func() {
				resolver := symbol.MapResolver{
					"chan1": {Name: "chan1", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 1},
					"chan2": {Name: "chan2", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 2},
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
					"high_chan": {Name: "high_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
					"low_chan":  {Name: "low_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
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

				Expect(inter.Nodes).To(HaveLen(3))

				writeCount := countNodesByType(inter.Nodes, "write")
				Expect(writeCount).To(Equal(2))

				for _, node := range inter.Nodes {
					if node.Type == "write" {
						Expect(node.Inputs).To(HaveLen(1))
					}
				}
			})
		})

		Context("Single Node Flow Validation", func() {
			DescribeTable("Should reject single-node flows at parse time",
				func(source string) {
					_, diagnostics := text.Parse(text.Text{Raw: source})
					Expect(diagnostics).ToNot(BeNil())
					Expect(diagnostics.Ok()).To(BeFalse())
				},
				Entry("single function node", `
					func print{} () {
					}

					print{}
				`),
				Entry("single channel identifier", `sensor`),
			)
		})

		Context("Sequence Targeting", func() {
			It("Should connect one-shot edge to sequence's first stage entry node", func() {
				resolver := symbol.MapResolver{
					"trigger": {Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
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

				Expect(inter.Nodes).To(HaveLen(2))

				triggerNode := findNodeByKey(inter.Nodes, "on_trigger_0")
				Expect(triggerNode.Type).To(Equal("on"))

				entryNode := findNodeByKey(inter.Nodes, "entry_main_run")
				Expect(entryNode.Type).To(Equal("stage_entry"))
				Expect(entryNode.Inputs).To(HaveLen(1))
				Expect(entryNode.Inputs[0].Name).To(Equal("activate"))

				// Verify no write node was created for sequence
				for _, node := range inter.Nodes {
					Expect(node.Key).ToNot(HavePrefix("write_main"))
				}

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal("on_trigger_0"))
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(edge.Target.Node).To(Equal("entry_main_run"))
				Expect(edge.Target.Param).To(Equal("activate"))
				Expect(edge.Kind).To(Equal(ir.EdgeKindOneShot))
			})

			It("Should handle continuous flow to sequence", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
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

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
				Expect(edge.Target.Node).To(Equal("entry_main_run"))
				Expect(edge.Target.Param).To(Equal("activate"))
			})

			It("Should handle sequence with multiple stages - connects to first stage", func() {
				resolver := symbol.MapResolver{
					"trigger": {Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
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

				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(1))

				edge := inter.Edges[0]
				Expect(edge.Target.Node).To(Equal("entry_main_first"))
				Expect(edge.Target.Param).To(Equal("activate"))
			})

			It("Should error when targeting empty sequence (no stages)", func() {
				resolver := symbol.MapResolver{
					"trigger": {Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
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
					"high_chan": {Name: "high_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
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

				Expect(inter.Nodes).To(HaveLen(3))

				entryNode := findNodeByKey(inter.Nodes, "entry_alarm_active")
				Expect(entryNode.Type).To(Equal("stage_entry"))

				writeNode := findNodeByType(inter.Nodes, "write")
				Expect(writeNode.Channels.Write.Contains(uint32(1))).To(BeTrue())

				Expect(inter.Edges).To(HaveLen(2))

				alarmEdge := findEdgeByTarget(inter.Edges, "entry_alarm_active")
				Expect(alarmEdge.Target.Param).To(Equal("activate"))
			})

		})

		Context("Direct Stage Targeting", func() {
			It("Should allow targeting a stage by name within a sequence", func() {
				resolver := symbol.MapResolver{
					"input": {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
				}
				source := `
				sequence main {
					stage first {
						input > 10 => second
					}
					stage second {
					}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Should have edges connecting to the second stage's entry node
				secondEdge := findEdgeByTarget(inter.Edges, "entry_main_second")
				Expect(secondEdge.Target.Param).To(Equal("activate"))
				Expect(secondEdge.Kind).To(Equal(ir.EdgeKindOneShot))
			})
		})

		Context("next keyword", func() {
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
				resolver := symbol.MapResolver{
					"input":  {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 2},
				}
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diag := text.Analyze(ctx, parsedText, resolver)
				Expect(diag.Ok()).To(BeTrue(), diag.String())

				nextEdge := findEdgeByTarget(inter.Edges, "entry_main_second")
				Expect(nextEdge.Target.Param).To(Equal("activate"))
				Expect(nextEdge.Kind).To(Equal(ir.EdgeKindOneShot))
			})

			DescribeTable("next keyword error cases",
				func(source string, resolver symbol.MapResolver, expectedError string) {
					parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
					_, diag := text.Analyze(ctx, parsedText, resolver)
					Expect(diag).ToNot(BeNil())
					Expect(diag.Ok()).To(BeFalse())
					Expect(diag.String()).To(ContainSubstring(expectedError))
				},
				Entry("next in last stage",
					`
					sequence main {
						stage only {
							input > 10 => next
						}
					}`,
					symbol.MapResolver{
						"input": {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					},
					"no next stage",
				),
				Entry("next outside sequence",
					`input > 10 => next`,
					symbol.MapResolver{
						"input": {Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					},
					"outside of a sequence",
				),
			)
		})

		Context("Implicit Expression Triggers", func() {
			It("Should inject implicit trigger for expression as first flow node", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 42},
				}
				source := `
				func alarm{} (value u8) {
				}

				sensor > 20 => alarm{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))

				triggerNode := findNodeByKey(inter.Nodes, "on_sensor_0")
				Expect(triggerNode.Type).To(Equal("on"))
				Expect(triggerNode.Channels.Read.Contains(uint32(42))).To(BeTrue())

				exprNode := inter.Nodes[1]
				Expect(exprNode.Type).To(HavePrefix("expression_"))
				Expect(exprNode.Channels.Read.Contains(uint32(42))).To(BeTrue())

				Expect(inter.Edges).To(HaveLen(2))

				edge0 := inter.Edges[0]
				Expect(edge0.Source.Node).To(Equal("on_sensor_0"))
				Expect(edge0.Target.Node).To(Equal(exprNode.Key))
				Expect(edge0.Kind).To(Equal(ir.EdgeKindContinuous))

				// Second edge: expression -> alarm (OneShot from =>)
				edge1 := inter.Edges[1]
				Expect(edge1.Source.Node).To(Equal(exprNode.Key))
				Expect(edge1.Target.Node).To(Equal("alarm_0"))
				Expect(edge1.Kind).To(Equal(ir.EdgeKindOneShot))
			})

			It("Should inject multiple triggers for multi-channel expression", func() {
				resolver := symbol.MapResolver{
					"temp":     {Name: "temp", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
					"pressure": {Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 2},
				}
				source := `
				func alarm{} (value u8) {
				}

				temp + pressure > 100 => alarm{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(4))

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(2))

				var exprNode ir.Node
				for _, n := range inter.Nodes {
					if n.Type != "on" && n.Type != "alarm" {
						exprNode = n
						break
					}
				}
				Expect(exprNode.Channels.Read).To(HaveLen(2))
				Expect(exprNode.Channels.Read.Contains(uint32(1))).To(BeTrue())
				Expect(exprNode.Channels.Read.Contains(uint32(2))).To(BeTrue())

				Expect(inter.Edges).To(HaveLen(3))

				exprEdgeCount := 0
				for _, edge := range inter.Edges {
					if edge.Target.Node == exprNode.Key {
						exprEdgeCount++
						Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
					}
				}
				Expect(exprEdgeCount).To(Equal(2))
			})

			It("Should not inject trigger for constant expressions", func() {
				resolver := symbol.MapResolver{
					"output": {Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 1},
				}
				source := `1 + 2 -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(0))
			})

			It("Should not inject trigger when expression is not first node", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 42},
				}
				source := `
				func alarm{} (value u8) {
				}

				sensor -> sensor > 20 => alarm{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(1))

				Expect(inter.Edges).To(HaveLen(2))
			})

			It("Should inject trigger for expression in sequence stage", func() {
				resolver := symbol.MapResolver{
					"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 42},
				}
				source := `
				sequence main {
					stage monitoring {
						sensor > 100 => next
					}
					stage alarm {
					}
				}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(1))

				triggerNode := findNodeByType(inter.Nodes, "on")
				Expect(triggerNode.Channels.Read.Contains(uint32(42))).To(BeTrue())
			})
		})

		Context("Interval One-Shot Edge Generation", func() {
			It("Should generate one-shot edge for interval triggering function", func() {
				resolver := symbol.MapResolver{
					"interval": {
						Name: "interval",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Config:  types.Params{{Name: "period", Type: types.TimeSpan()}},
							Outputs: types.Params{{Name: "output", Type: types.U8()}},
						}),
					},
				}
				source := `
				func press{} () {
				}

				interval{period=50ms} => press{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))

				intervalNode := findNodeByType(inter.Nodes, "interval")
				Expect(intervalNode.Config).To(HaveLen(1))
				Expect(intervalNode.Config[0].Name).To(Equal("period"))

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal(intervalNode.Key))
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(edge.Target.Node).To(Equal("press_0"))
				Expect(edge.Kind).To(Equal(ir.EdgeKindOneShot))
			})

			It("Should generate continuous edge for interval with -> operator", func() {
				resolver := symbol.MapResolver{
					"interval": {
						Name: "interval",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Config:  types.Params{{Name: "period", Type: types.TimeSpan()}},
							Outputs: types.Params{{Name: "output", Type: types.U8()}},
						}),
					},
				}
				source := `
				func handler{} () {
				}

				interval{period=50ms} -> handler{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
			})
		})
	})

	Describe("Unit Dimensional Analysis", func() {
		DescribeTable("dimension compatibility",
			func(source string, expectOk bool, expectedErrorContains string) {
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diag := text.Analyze(ctx, parsedText, nil)
				if expectOk {
					Expect(diag.Ok()).To(BeTrue(), diag.String())
				} else {
					Expect(diag.Ok()).To(BeFalse())
					Expect(diag.String()).To(ContainSubstring(expectedErrorContains))
				}
			},
			Entry("error when adding incompatible dimensions",
				`func bad() f64 { return 5psi + 3s }`,
				false, "incompatible dimensions:",
			),
			Entry("allow adding same dimensions",
				`func good() f64 { return 100psi + 50psi }`,
				true, "",
			),
			Entry("allow multiplying different dimensions",
				`func velocity() f64 { return 100m / 10s }`,
				true, "",
			),
		)
	})

	Describe("Single Invocations in Stages", func() {
		It("Should compile standalone function invocation to IR node", func() {
			source := `
			func setup() {
			}

			sequence main {
				stage start {
					setup{},
				}
			}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			setupNode := findNodeByType(inter.Nodes, "setup")
			Expect(setupNode.Inputs).To(BeEmpty())

			seq := MustBeOk(inter.Sequences.Find("main"))
			Expect(seq.Stages[0].Nodes).To(ContainElement(setupNode.Key))
		})

		It("Should compile standalone expression to IR node", func() {
			source := `
			sequence main {
				stage start {
					1 + 2,
				}
			}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			exprNodes := lo.Filter(inter.Nodes, func(n ir.Node, _ int) bool {
				return strings.HasPrefix(n.Type, "expression_")
			})
			Expect(exprNodes).To(HaveLen(1))
			exprNode := exprNodes[0]
			Expect(exprNode.Outputs).To(HaveLen(1))
			Expect(exprNode.Outputs[0].Type.Kind).To(Equal(types.KindI64))

			seq := MustBeOk(inter.Sequences.Find("main"))
			Expect(seq.Stages[0].Nodes).To(ContainElement(exprNode.Key))
		})

		It("Should place single invocation nodes in stratum 0", func() {
			source := `
			func initialize() u8 {
				return 1
			}

			sequence main {
				stage start {
					initialize{},
				}
			}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			initNode := findNodeByType(inter.Nodes, "initialize")
			Expect(initNode.Inputs).To(BeEmpty())

			seq := MustBeOk(inter.Sequences.Find("main"))
			Expect(seq.Stages[0].Strata).To(HaveLen(1))
			Expect(seq.Stages[0].Strata[0]).To(ContainElement(initNode.Key))
		})
	})

	Describe("Compile", func() {
		It("Should compile a simple arc program to WebAssembly", func() {
			source := `
			func adder{} (a i64, b i64) i64 {
				return a + b
			}

			func print{} () {
			}

			adder{} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			ir, diagnostics := text.Analyze(ctx, parsedText, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			module := MustSucceed(text.Compile(ctx, ir))
			Expect(module.Output.WASM).ToNot(BeEmpty())
		})
	})
})
