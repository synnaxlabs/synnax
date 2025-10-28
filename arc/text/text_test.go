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
			Expect(f.Inputs.Count()).To(Equal(2))
			v, ok := f.Inputs.Get("a")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(types.I64()))
			v, ok = f.Inputs.Get("b")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(types.I64()))

			s := inter.Functions[1]
			Expect(s.Key).To(Equal("adder"))
			Expect(s.Inputs.Count()).To(Equal(2))
			v, ok = s.Inputs.Get("a")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(types.I64()))
			v, ok = s.Inputs.Get("b")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(types.I64()))

			n1 := inter.Nodes[0]
			Expect(n1.Key).To(Equal("adder_0"))
			Expect(n1.Type).To(Equal("adder"))
			Expect(n1.ConfigValues).To(HaveLen(0))
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
				Expect(channelNode.ConfigValues).To(HaveKey("channel"))
				Expect(channelNode.ConfigValues["channel"]).To(Equal(uint32(42)))
				Expect(channelNode.Channels.Read.Contains(42)).To(BeTrue())

				// Second node should be print function
				printNode := inter.Nodes[1]
				Expect(printNode.Type).To(Equal("print"))

				// Edge should connect channel to print
				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal("on_0"))
				Expect(edge.Target.Node).To(Equal(printNode.Key))
			})

			It("Should analyze flow with multiple channels", func() {
				resolver := symbol.MapResolver(map[string]symbol.Symbol{
					"sensor1": {
						Name: "sensor1",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   10,
					},
					"sensor2": {
						Name: "sensor2",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   20,
					},
				})

				source := `
				func process{} () {
				}

				sensor1 -> process{} -> sensor2
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

				// Verify channel nodes
				Expect(inter.Nodes[0].Type).To(Equal("on"))
				Expect(inter.Nodes[0].ConfigValues["channel"]).To(Equal(uint32(10)))
				Expect(inter.Nodes[2].Type).To(Equal("on"))
				Expect(inter.Nodes[2].ConfigValues["channel"]).To(Equal(uint32(20)))
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
				Expect(node.ConfigValues).To(HaveKey("threshold"))
				Expect(node.ConfigValues).To(HaveKey("scale"))
				Expect(node.ConfigValues["threshold"]).To(Equal("100"))
				Expect(node.ConfigValues["scale"]).To(Equal("2.5"))
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
				Expect(node.ConfigValues["a"]).To(Equal("10"))
				Expect(node.ConfigValues["b"]).To(Equal("20"))
				Expect(node.ConfigValues["c"]).To(Equal("30"))
			})
		})

		Describe("Complex Flow Chains", func() {
			It("Should analyze multi-stage flow chains", func() {
				resolver := symbol.MapResolver(map[string]symbol.Symbol{
					"sensor": {
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   1,
					},
					"output": {
						Name: "output",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   2,
					},
				})

				source := `
				func filter{} () {
				}

				func transform{} () {
				}

				sensor -> filter{} -> transform{} -> output
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, resolver)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(4))
				Expect(inter.Edges).To(HaveLen(3))

				// Verify all edges are connected properly
				Expect(inter.Edges[0].Source.Node).To(Equal(inter.Nodes[0].Key))
				Expect(inter.Edges[0].Target.Node).To(Equal(inter.Nodes[1].Key))
				Expect(inter.Edges[1].Source.Node).To(Equal(inter.Nodes[1].Key))
				Expect(inter.Edges[1].Target.Node).To(Equal(inter.Nodes[2].Key))
				Expect(inter.Edges[2].Source.Node).To(Equal(inter.Nodes[2].Key))
				Expect(inter.Edges[2].Target.Node).To(Equal(inter.Nodes[3].Key))
			})
		})
	})

})
