// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// nodeSpec describes a graph node together with its function type and inputs
// parameter values, which now live in the graph's Inputs map rather than on the node.
type nodeSpec struct {
	key string
	typ string
	cfg map[string]any
}

// buildNodes converts node specs into the graph's Nodes slice plus the Inputs map,
// keying each input by node key and storing the function type under "type".
func buildNodes(specs ...nodeSpec) (graph.Nodes, map[string]msgpack.EncodedJSON) {
	nodes := make(graph.Nodes, len(specs))
	inputs := make(map[string]msgpack.EncodedJSON, len(specs))
	for i, s := range specs {
		nodes[i] = graph.Node{Key: s.key}
		cfg := msgpack.EncodedJSON{"type": s.typ}
		maps.Copy(cfg, s.cfg)
		inputs[s.key] = cfg
	}
	return nodes, inputs
}

var _ = Describe("Graph", func() {
	Describe("Parse", func() {
		It("Should correctly parse a single stage", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: "a", Type: types.I64()},
							{Name: "b", Type: types.I64()},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			Expect(g.Functions[0].Body.AST).ToNot(BeNil())
		})

		It("Should correctly parse a single function", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: "a", Type: types.I64()},
							{Name: "b", Type: types.I64()},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			Expect(g.Functions[0].Body.AST).ToNot(BeNil())
		})
	})

	Describe("Analyze", func() {
		It("Should correctly analyze a single function", func(ctx SpecContext) {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: "a", Type: types.I64()},
							{Name: "b", Type: types.I64()},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(1))
			funcScope := MustSucceed(inter.Symbols.Resolve(ctx, "add"))
			Expect(funcScope.Children()).To(HaveLen(4))
			params := funcScope.FilterChildrenByKind(symbol.KindInput)
			Expect(params).To(HaveLen(2))
			Expect(params[0].Name).To(Equal("a"))
			Expect(params[0].Type).To(Equal(types.I64()))
			Expect(params[1].Name).To(Equal("b"))
			Expect(params[1].Type).To(Equal(types.I64()))
		})

		It("Should correctly analyze a complete program", func(ctx SpecContext) {
			nodes, inputs := buildNodes(
				nodeSpec{key: "first", typ: "on", cfg: map[string]any{"channel": 12}},
				nodeSpec{key: "printer", typ: "printer"},
			)
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "on",
						Inputs: types.Params{
							{Name: "channel", Type: types.Chan(types.F32())},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F32()},
						},
					},
					{
						Key: "printer",
						Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.F32()},
						},
					},
				},
				Nodes:  nodes,
				Inputs: inputs,
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: arc.Handle{Node: "first", Param: ir.DefaultOutputParam},
						Target: arc.Handle{
							Node:  "printer",
							Param: ir.DefaultInputParam,
						},
					}},
				},
			}
			root := symbol.NewRoot(nil, stl.NewSymbols())
			root.Parent.AddChild(&symbol.Symbol{
				Name: "ox_pt_1",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
				ID:   12,
			})
			symbol.AutoImportModules(root)
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, root)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(2))
			Expect(inter.Nodes).To(HaveLen(2))
			Expect(inter.Edges).To(HaveLen(1))

			firstNode := inter.Nodes[0]
			Expect(firstNode.Key).To(Equal("first"))
			Expect(firstNode.Type).To(Equal("on"))
			Expect(firstNode.Inputs).To(HaveLen(1))
			Expect(firstNode.Channels.Read).To(HaveLen(1))
		})

		Describe("Polymorphic Stages", func() {
			It(
				"Should correctly infer types for polymorphic stages from F32 inputs",
				func(ctx SpecContext) {
					constraint := types.NumericConstraint()
					nodes, inputs := buildNodes(
						nodeSpec{key: "source1", typ: "f32_source"},
						nodeSpec{key: "source2", typ: "f32_source"},
						nodeSpec{key: "adder", typ: "polymorphic_add"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "polymorphic_add",
								Inputs: types.Params{
									{Name: "a", Type: types.Variable("T", &constraint)},
									{Name: "b", Type: types.Variable("T", &constraint)},
								},
								Outputs: types.Params{
									{
										Name: ir.DefaultOutputParam,
										Type: types.Variable("T", &constraint),
									},
								},
							},
							{
								Key: "f32_source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.F32()},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "source1",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "adder", Param: "a"},
							}},
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "source2",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "adder", Param: "b"},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

					// The fact that analysis succeeded without errors indicates
					// that the type variables were successfully unified with F32
					Expect(inter.Functions).To(HaveLen(2))
					Expect(inter.Nodes).To(HaveLen(3))
					Expect(inter.Edges).To(HaveLen(2))

					// Check that each node instance has concrete resolved types
					// The func definition remains polymorphic, but each node gets concrete types
					adderNode, _ := lo.Find(
						inter.Nodes,
						func(n ir.Node) bool { return n.Key == "adder" },
					)

					// The adder node should have concrete F32 types resolved from edges
					aParam := MustBeOk(adderNode.Inputs.Get("a"))
					Expect(aParam.Type).To(Equal(types.F32()))

					bParam := MustBeOk(adderNode.Inputs.Get("b"))
					Expect(bParam.Type).To(Equal(types.F32()))

					// Return type should also be concrete
					returnParam := MustBeOk(
						adderNode.Outputs.Get(ir.DefaultOutputParam),
					)
					Expect(returnParam.Type).To(Equal(types.F32()))
				},
			)

			It(
				"Should correctly infer types for polymorphic stages from I64 inputs",
				func(ctx SpecContext) {
					constraint := types.NumericConstraint()
					nodes, inputs := buildNodes(
						nodeSpec{key: "int_source1", typ: "i64_source"},
						nodeSpec{key: "int_source2", typ: "i64_source"},
						nodeSpec{key: "multiplier", typ: "polymorphic_multiply"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "polymorphic_multiply",
								Inputs: types.Params{
									{Name: "x", Type: types.Variable("T", &constraint)},
									{Name: "y", Type: types.Variable("T", &constraint)},
								},
								Outputs: types.Params{
									{
										Name: ir.DefaultOutputParam,
										Type: types.Variable("T", &constraint),
									},
								},
							},
							{
								Key: "i64_source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.I64()},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "int_source1",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "multiplier", Param: "x"},
							}},
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "int_source2",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "multiplier", Param: "y"},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

					// Check that the multiplier node instance has concrete resolved types
					multiplierNode, _ := lo.Find(
						inter.Nodes,
						func(n ir.Node) bool { return n.Key == "multiplier" },
					)

					xParam := MustBeOk(multiplierNode.Inputs.Get("x"))
					Expect(xParam.Type).To(Equal(types.I64()))

					yParam := MustBeOk(multiplierNode.Inputs.Get("y"))
					Expect(yParam.Type).To(Equal(types.I64()))

					returnParam := MustBeOk(
						multiplierNode.Outputs.Get(ir.DefaultOutputParam),
					)
					Expect(returnParam.Type).To(Equal(types.I64()))
				},
			)

			It("Should handle chained polymorphic stages", func(ctx SpecContext) {
				constraint := types.NumericConstraint()
				nodes, inputs := buildNodes(
					nodeSpec{key: "src1", typ: "f64_source"},
					nodeSpec{key: "src2", typ: "f64_source"},
					nodeSpec{key: "add1", typ: "poly_add"},
					nodeSpec{key: "scale1", typ: "poly_scale"},
					nodeSpec{key: "scale2", typ: "poly_scale"},
				)
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "poly_add",
							Inputs: types.Params{
								{Name: "a", Type: types.Variable("T", &constraint)},
								{Name: "b", Type: types.Variable("T", &constraint)},
							},
							Outputs: types.Params{
								{
									Name: ir.DefaultOutputParam,
									Type: types.Variable("T", &constraint),
								},
							},
						},
						{
							Key: "poly_scale",
							Inputs: types.Params{
								{
									Name: ir.DefaultInputParam,
									Type: types.Variable("U", &constraint),
								},
							},
							Outputs: types.Params{
								{
									Name: ir.DefaultOutputParam,
									Type: types.Variable("U", &constraint),
								},
							},
						},
						{
							Key: "f64_source",
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.F64()},
							},
						},
					},
					Nodes:  nodes,
					Inputs: inputs,
					Edges: graph.Edges{
						{Edge: ir.Edge{
							Source: ir.Handle{
								Node:  "src1",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{Node: "add1", Param: "a"},
						}},
						{Edge: ir.Edge{
							Source: ir.Handle{
								Node:  "src2",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{Node: "add1", Param: "b"},
						}},
						{Edge: ir.Edge{
							Source: ir.Handle{
								Node:  "add1",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{
								Node:  "scale1",
								Param: ir.DefaultInputParam,
							},
						}},
						{Edge: ir.Edge{
							Source: ir.Handle{
								Node:  "add1",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{
								Node:  "scale2",
								Param: ir.DefaultInputParam,
							},
						}},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Both node instances should have concrete F64 types
				add1Node, _ := lo.Find(
					inter.Nodes,
					func(n ir.Node) bool { return n.Key == "add1" },
				)
				addReturnParam := MustBeOk(add1Node.Outputs.Get(ir.DefaultOutputParam))
				Expect(addReturnParam.Type).To(Equal(types.F64()))

				scale1Node, _ := lo.Find(
					inter.Nodes,
					func(n ir.Node) bool { return n.Key == "scale1" },
				)
				inputParam := MustBeOk(scale1Node.Inputs.Get(ir.DefaultInputParam))
				Expect(inputParam.Type).To(Equal(types.F64()))
			})

			It(
				"Should detect type mismatches in polymorphic edge connections",
				func(ctx SpecContext) {
					constraint := types.NumericConstraint()
					nodes, inputs := buildNodes(
						nodeSpec{key: "float_src", typ: "f32_source"},
						nodeSpec{key: "int_src", typ: "i64_source"},
						nodeSpec{key: "adder", typ: "poly_add"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "f32_source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.F32()},
								},
							},
							{
								Key: "i64_source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.I64()},
								},
							},
							{
								Key: "poly_add",
								Inputs: types.Params{
									{Name: "a", Type: types.Variable("T", &constraint)},
									{Name: "b", Type: types.Variable("T", &constraint)},
								},
								Outputs: types.Params{
									{
										Name: ir.DefaultOutputParam,
										Type: types.Variable("T", &constraint),
									},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "float_src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "adder", Param: "a"},
							}},
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "int_src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "adder", Param: "b"},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					// This should fail because poly_add expects both parameters to be the same type T
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(
						diagnostics.String(),
					).To(ContainSubstring("is not compatible with"))
				},
			)

			It(
				"Should detect non-numeric type mismatches with polymorphic stages",
				func(ctx SpecContext) {
					constraint := types.NumericConstraint()
					nodes, inputs := buildNodes(
						nodeSpec{key: "str_src", typ: "string_source"},
						nodeSpec{key: "numeric_stage", typ: "poly_numeric"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "string_source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.String()},
								},
							},
							{
								Key: "poly_numeric",
								Inputs: types.Params{
									{
										Name: "value",
										Type: types.Variable("T", &constraint),
									},
								},
								Outputs: types.Params{
									{
										Name: ir.DefaultOutputParam,
										Type: types.Variable("T", &constraint),
									},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "str_src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{
									Node:  "numeric_stage",
									Param: "value",
								},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					// This should fail because string doesn't satisfy NumericConstraint
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(
						diagnostics.String(),
					).To(ContainSubstring("is not compatible with"))
				},
			)

			It("Should handle missing edge connections", func(ctx SpecContext) {
				nodes, inputs := buildNodes(
					nodeSpec{key: "src", typ: "source"},
					nodeSpec{key: "snk", typ: "sink"},
				)
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "source",
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.F32()},
							},
						},
						{
							Key: "sink",
							Inputs: types.Params{
								{Name: ir.DefaultInputParam, Type: types.F32()},
							},
						},
					},
					Nodes:  nodes,
					Inputs: inputs,
					Edges: graph.Edges{
						{Edge: ir.Edge{
							Source: ir.Handle{
								Node:  "src",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{
								Node:  "nonexistent",
								Param: ir.DefaultOutputParam,
							}, // Invalid target node
						}},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(
					diagnostics.String(),
				).To(ContainSubstring("edge target node 'nonexistent' not found"))
			})

			It(
				"Should handle invalid parameter references in edges",
				func(ctx SpecContext) {
					nodes, inputs := buildNodes(
						nodeSpec{key: "src", typ: "source"},
						nodeSpec{key: "snk", typ: "sink"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.F32()},
								},
							},
							{
								Key: "sink",
								Inputs: types.Params{
									{Name: ir.DefaultInputParam, Type: types.F32()},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{
									Node:  "snk",
									Param: "invalid_param",
								}, // Invalid parameter
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(
						diagnostics.String(),
					).To(ContainSubstring("missing required input 'input'"))
				},
			)

			It(
				"Should handle concrete type mismatches in edges",
				func(ctx SpecContext) {
					nodes, inputs := buildNodes(
						nodeSpec{key: "str_src", typ: "string_source"},
						nodeSpec{key: "num_snk", typ: "number_sink"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "string_source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.String()},
								},
							},
							{
								Key: "number_sink",
								Inputs: types.Params{
									{Name: "value", Type: types.F32()},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "str_src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "num_snk", Param: "value"},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(diagnostics.String()).To(ContainSubstring("type mismatch"))
				},
			)
		})

		Describe("Integration", func() {
			It(
				"Should parse and analyze a complete alarm system graph",
				func(ctx SpecContext) {
					nodes, inputs := buildNodes(
						nodeSpec{
							key: "on",
							typ: "on",
							cfg: map[string]any{"channel": 12},
						},
						nodeSpec{
							key: "constant",
							typ: "constant",
							cfg: map[string]any{"value": 10},
						},
						nodeSpec{key: "ge", typ: "ge"},
						nodeSpec{
							key: "stable_for",
							typ: "stable_for",
							cfg: map[string]any{
								"duration": int(telem.Millisecond * 1),
							},
						},
						nodeSpec{key: "select", typ: "select"},
						nodeSpec{
							key: "status_success",
							typ: "status.set",
							cfg: map[string]any{
								"key_or_name": "ox_alarm",
								"message":     "OX Pressure Nominal",
								"variant":     "success",
							},
						},
						nodeSpec{
							key: "status_error",
							typ: "status.set",
							cfg: map[string]any{
								"key_or_name": "ox_alarm",
								"message":     "OX Pressure Alarm",
								"variant":     "error",
							},
						},
					)
					g := arc.Graph{
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: arc.Handle{
									Node:  "on",
									Param: ir.DefaultOutputParam,
								},
								Target: arc.Handle{Node: "ge", Param: "a"},
							}},
							{Edge: ir.Edge{
								Source: arc.Handle{
									Node:  "constant",
									Param: ir.DefaultOutputParam,
								},
								Target: arc.Handle{Node: "ge", Param: "b"},
							}},
							{Edge: ir.Edge{
								Source: arc.Handle{
									Node:  "ge",
									Param: ir.DefaultOutputParam,
								},
								Target: arc.Handle{
									Node:  "stable_for",
									Param: ir.DefaultInputParam,
								},
							}},
							{Edge: ir.Edge{
								Source: arc.Handle{
									Node:  "stable_for",
									Param: ir.DefaultOutputParam,
								},
								Target: arc.Handle{
									Node:  "select",
									Param: ir.DefaultInputParam,
								},
							}},
							// status_success/error fulfilled by input; no edges needed.
						},
					}

					// First, define the func signatures that this graph expects
					// Using polymorphic types for constant, ge, and stable_for
					// Each func gets its own type variables

					constraint := types.NumericConstraint()
					functions := []ir.Function{
						{
							Key: "on",
							Inputs: types.Params{
								{Name: "channel", Type: types.U32()},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.F64()},
							}, // Returns sensor reading
						},
						{
							Key: "constant",
							Inputs: types.Params{
								{Name: "value", Type: types.Variable("A", &constraint)},
							},
							Outputs: types.Params{
								{
									Name: ir.DefaultOutputParam,
									Type: types.Variable("A", &constraint),
								},
							},
						},
						{
							Key: "ge",
							Inputs: types.Params{
								{Name: "a", Type: types.Variable("B", &constraint)},
								{Name: "b", Type: types.Variable("B", &constraint)},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.U8()},
							},
						},
						{
							Key: "stable_for",
							Inputs: types.Params{
								{Name: "duration", Type: types.TimeSpan()},
								{
									Name: ir.DefaultInputParam,
									Type: types.Variable("C", nil),
								},
							},
							Outputs: types.Params{
								{
									Name: ir.DefaultOutputParam,
									Type: types.Variable("C", nil),
								},
							},
						},
						{
							Key: "select",
							Inputs: types.Params{
								{Name: ir.DefaultInputParam, Type: types.U8()},
							},
							Outputs: types.Params{
								{Name: "false", Type: types.U8()},
								{Name: "true", Type: types.U8()},
							},
						},
						{
							Key: "status.set",
							Inputs: types.Params{
								{Name: "key_or_name", Type: types.String(), Value: ""},
								{Name: "message", Type: types.String(), Value: ""},
								{Name: "variant", Type: types.String(), Value: ""},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.String()},
							},
						},
					}

					// Convert arc.Graph to graph.Graph
					graphWithFunctions := graph.Graph{
						Functions: functions,
						Nodes:     g.Nodes,
						Inputs:    g.Inputs,
						Edges:     g.Edges,
					}

					// Parse the graph
					parsed := MustSucceed(graph.Parse(graphWithFunctions))

					// The graph should have been parsed successfully
					Expect(parsed.Nodes).To(HaveLen(7))
					Expect(parsed.Edges).To(HaveLen(4))

					// Analyze the graph
					inter, diagnostics := graph.Analyze(ctx, parsed, NewGraphRoot(nil))

					// The analysis should succeed
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

					// Verify the graph structure
					Expect(inter.Nodes).To(HaveLen(7))

					// Verify specific nodes exist and have correct types
					onNode := MustSucceed(inter.Symbols.Resolve(ctx, "on"))
					Expect(onNode.Type.Kind).To(Equal(types.KindFunction))

					// Verify the edges create the correct flow
					// on -> ge.a, constant -> ge.b
					// ge -> stable_for
					// stable_for -> select
					// status_success and status_error are fulfilled by input, not edges.
					Expect(inter.Edges).To(HaveLen(4))

					// Verify inputs were parsed correctly
					Expect(parsed.Inputs["constant"]).To(HaveKeyWithValue("value", 10))
					Expect(
						parsed.Inputs["stable_for"],
					).To(HaveKeyWithValue("duration", int(telem.Millisecond)))

					// Verify polymorphic node instances have concrete resolved types
					// func definitions stay polymorphic, but each node instance gets concrete types

					// The constant node should have concrete F64 type
					// (since it connects to "ge" which receives F64 from "on")
					constantIRNode, _ := lo.Find(
						inter.Nodes,
						func(n ir.Node) bool { return n.Key == "constant" },
					)
					constantReturnType := MustBeOk(
						constantIRNode.Outputs.Get(ir.DefaultOutputParam),
					)
					Expect(constantReturnType.Type).To(Equal(types.F64()))

					// The ge node should have concrete F64 parameters
					// (since it receives F64 inputs from "on" and "constant")
					geIRNode, _ := lo.Find(
						inter.Nodes,
						func(n ir.Node) bool { return n.Key == "ge" },
					)
					aParam := MustBeOk(geIRNode.Inputs.Get("a"))
					Expect(aParam.Type).To(Equal(types.F64()))
					bParam := MustBeOk(geIRNode.Inputs.Get("b"))
					Expect(bParam.Type).To(Equal(types.F64()))

					// The stable_for node should have concrete U8 types
					// (since it receives U8 from "ge" comparison result)
					stableIRNode, _ := lo.Find(
						inter.Nodes,
						func(n ir.Node) bool { return n.Key == "stable_for" },
					)
					inputParam := MustBeOk(
						stableIRNode.Inputs.Get(ir.DefaultInputParam),
					)
					Expect(inputParam.Type).To(Equal(types.U8()))
					stableReturnParam := MustBeOk(
						stableIRNode.Outputs.Get(ir.DefaultOutputParam),
					)
					Expect(stableReturnParam.Type).To(Equal(types.U8()))
				},
			)
		})

		It(
			"Should analyze set_authority with a non-uint8 channel",
			func(ctx SpecContext) {
				nodes, inputs := buildNodes(
					nodeSpec{
						key: "on",
						typ: "on",
						cfg: map[string]any{"channel": 10057},
					},
					nodeSpec{key: "set_auth", typ: "set_authority", cfg: map[string]any{
						"value":   200,
						"channel": 10057,
					}},
				)
				g := arc.Graph{
					Functions: []ir.Function{
						{
							Key: "on",
							Inputs: types.Params{
								{Name: "channel", Type: types.Chan(types.F64())},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.F64()},
							},
						},
					},
					Nodes:  nodes,
					Inputs: inputs,
				}
				resolver := []symbol.Symbol{{
					Name: "f64_sensor",
					Type: types.WriteChan(types.F64()),
					Kind: symbol.KindChannel,
					ID:   10057,
				}}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			},
		)

		It(
			"Should analyze control.set_authority with a non-uint8 channel",
			func(ctx SpecContext) {
				nodes, inputs := buildNodes(
					nodeSpec{
						key: "on",
						typ: "on",
						cfg: map[string]any{"channel": 10057},
					},
					nodeSpec{
						key: "set_auth",
						typ: "control.set_authority",
						cfg: map[string]any{
							"value":   200,
							"channel": 10057,
						},
					},
				)
				g := arc.Graph{
					Functions: []ir.Function{
						{
							Key: "on",
							Inputs: types.Params{
								{Name: "channel", Type: types.Chan(types.F64())},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.F64()},
							},
						},
					},
					Nodes:  nodes,
					Inputs: inputs,
				}
				resolver := []symbol.Symbol{{
					Name: "f64_sensor",
					Type: types.WriteChan(types.F64()),
					Kind: symbol.KindChannel,
					ID:   10057,
				}}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			},
		)

		It("Should reject set_authority with a read channel", func(ctx SpecContext) {
			nodes, inputs := buildNodes(
				nodeSpec{key: "set_auth", typ: "set_authority", cfg: map[string]any{
					"value":   200,
					"channel": 10058,
				}},
			)
			g := arc.Graph{
				Nodes:  nodes,
				Inputs: inputs,
			}
			resolver := []symbol.Symbol{{
				Name: "f64_sensor",
				Type: types.ReadChan(types.F64()),
				Kind: symbol.KindChannel,
				ID:   10058,
			}}
			g = MustSucceed(graph.Parse(g))
			_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeFalse())
		})

		Describe("Edge Validation", func() {
			Describe("Type Matching", func() {
				It("Should validate series type matching", func(ctx SpecContext) {
					nodes, inputs := buildNodes(
						nodeSpec{key: "src", typ: "series_f32_source"},
						nodeSpec{key: "snk_mismatch", typ: "series_i64_sink"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "series_f32_source",
								Outputs: types.Params{
									{
										Name: ir.DefaultOutputParam,
										Type: types.Series(types.F32()),
									},
								},
							},
							{
								Key: "series_i64_sink",
								Inputs: types.Params{
									{
										Name: ir.DefaultInputParam,
										Type: types.Series(types.I64()),
									},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{
									Node:  "snk_mismatch",
									Param: ir.DefaultInputParam,
								},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(diagnostics.String()).To(ContainSubstring("type mismatch"))
				})

				It(
					"Should succeed when all required inputs are connected",
					func(ctx SpecContext) {
						nodes, inputs := buildNodes(
							nodeSpec{key: "src1", typ: "source"},
							nodeSpec{key: "src2", typ: "source"},
							nodeSpec{key: "dual", typ: "dual_input"},
						)
						g := graph.Graph{
							Functions: []ir.Function{
								{
									Key: "source",
									Outputs: types.Params{
										{
											Name: ir.DefaultOutputParam,
											Type: types.F32(),
										},
									},
								},
								{
									Key: "dual_input",
									Inputs: types.Params{
										{Name: "a", Type: types.F32()},
										{Name: "b", Type: types.F32()},
									},
									Outputs: types.Params{
										{
											Name: ir.DefaultOutputParam,
											Type: types.F32(),
										},
									},
								},
							},
							Nodes:  nodes,
							Inputs: inputs,
							Edges: graph.Edges{
								{Edge: ir.Edge{
									Source: ir.Handle{
										Node:  "src1",
										Param: ir.DefaultOutputParam,
									},
									Target: ir.Handle{Node: "dual", Param: "a"},
								}},
								{Edge: ir.Edge{
									Source: ir.Handle{
										Node:  "src2",
										Param: ir.DefaultOutputParam,
									},
									Target: ir.Handle{Node: "dual", Param: "b"},
								}},
							},
						}
						g = MustSucceed(graph.Parse(g))
						inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
						Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
						Expect(inter.Edges).To(HaveLen(2))
					},
				)

				It(
					"Should allow nodes with no inputs to exist without edges",
					func(ctx SpecContext) {
						nodes, inputs := buildNodes(
							nodeSpec{key: "src1", typ: "source_only"},
							nodeSpec{key: "src2", typ: "source_only"},
						)
						g := graph.Graph{
							Functions: []ir.Function{
								{
									Key: "source_only",
									Outputs: types.Params{
										{
											Name: ir.DefaultOutputParam,
											Type: types.F32(),
										},
									},
								},
							},
							Nodes:  nodes,
							Inputs: inputs,
							Edges:  graph.Edges{},
						}
						g = MustSucceed(graph.Parse(g))
						inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
						Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
						Expect(inter.Nodes).To(HaveLen(2))
					},
				)
			})

			Describe("Missing Required Edges", func() {
				It(
					"Should return an error when a graph is missing a required edge",
					func(ctx SpecContext) {
						nodes, inputs := buildNodes(
							nodeSpec{key: "src1", typ: "source"},
							nodeSpec{key: "add1", typ: "add"},
						)
						g := graph.Graph{
							Functions: []ir.Function{
								{
									Key: "source",
									Outputs: types.Params{
										{
											Name: ir.DefaultOutputParam,
											Type: types.F32(),
										},
									},
								},
								{
									Key: "add",
									Inputs: types.Params{
										{Name: ir.LHSInputParam, Type: types.F32()},
										{Name: ir.RHSInputParam, Type: types.F32()},
									},
								},
							},
							Nodes:  nodes,
							Inputs: inputs,
							Edges: graph.Edges{
								{Edge: ir.Edge{
									Source: ir.Handle{
										Node:  "src1",
										Param: ir.DefaultOutputParam,
									},
									Target: ir.Handle{
										Node:  "add1",
										Param: ir.LHSInputParam,
									},
								}},
							},
						}
						g = MustSucceed(graph.Parse(g))
						_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
						Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
						Expect(
							diagnostics,
						).To(MatchError(ContainSubstring("missing required input 'b'")))
					},
				)

				It(
					"Should not return an error when the edge is optional",
					func(ctx SpecContext) {
						nodes, inputs := buildNodes(
							nodeSpec{key: "src1", typ: "source"},
							nodeSpec{key: "add1", typ: "add"},
						)
						g := graph.Graph{
							Functions: []ir.Function{
								{
									Key: "source",
									Outputs: types.Params{
										{
											Name: ir.DefaultOutputParam,
											Type: types.F32(),
										},
									},
								},
								{
									Key: "add",
									Inputs: types.Params{
										{Name: ir.LHSInputParam, Type: types.F32()},
										{
											Name:  ir.RHSInputParam,
											Type:  types.F32(),
											Value: 1,
										},
									},
								},
							},
							Nodes:  nodes,
							Inputs: inputs,
							Edges: graph.Edges{
								{Edge: ir.Edge{
									Source: ir.Handle{
										Node:  "src1",
										Param: ir.DefaultOutputParam,
									},
									Target: ir.Handle{
										Node:  "add1",
										Param: ir.LHSInputParam,
									},
								}},
							},
						}
						g = MustSucceed(graph.Parse(g))
						_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
						Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
					},
				)
			})
		})

		Describe("Duplicate Edge Targets", func() {
			It(
				"Should error when multiple edges target the same input parameter",
				func(ctx SpecContext) {
					nodes, inputs := buildNodes(
						nodeSpec{key: "src1", typ: "source"},
						nodeSpec{key: "src2", typ: "source"},
						nodeSpec{key: "proc", typ: "processor"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.F32()},
								},
							},
							{
								Key: "processor",
								Inputs: types.Params{
									{Name: "input", Type: types.F32()},
								},
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.F32()},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "src1",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "proc", Param: "input"},
							}},
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "src2",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "proc", Param: "input"},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(diagnostics.String()).To(ContainSubstring("multiple edges"))
				},
			)

			It(
				"Should allow multiple edges from the same source parameter",
				func(ctx SpecContext) {
					nodes, inputs := buildNodes(
						nodeSpec{key: "src", typ: "source"},
						nodeSpec{key: "snk1", typ: "sink"},
						nodeSpec{key: "snk2", typ: "sink"},
					)
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source",
								Outputs: types.Params{
									{Name: ir.DefaultOutputParam, Type: types.F32()},
								},
							},
							{
								Key: "sink",
								Inputs: types.Params{
									{Name: "input", Type: types.F32()},
								},
							},
						},
						Nodes:  nodes,
						Inputs: inputs,
						Edges: graph.Edges{
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "snk1", Param: "input"},
							}},
							{Edge: ir.Edge{
								Source: ir.Handle{
									Node:  "src",
									Param: ir.DefaultOutputParam,
								},
								Target: ir.Handle{Node: "snk2", Param: "input"},
							}},
						},
					}
					g = MustSucceed(graph.Parse(g))
					inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
					Expect(inter.Edges).To(HaveLen(2))
				},
			)
		})

		Describe("Malformed Node Input", func() {
			DescribeTable(
				"Should report a clear diagnostic instead of failing to resolve an empty type",
				func(ctx SpecContext, cfg msgpack.EncodedJSON, expected string) {
					g := graph.Graph{
						Nodes:  graph.Nodes{{Key: "n1"}},
						Inputs: map[string]msgpack.EncodedJSON{"n1": cfg},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
					Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
					Expect(diagnostics.String()).To(ContainSubstring(expected))
				},
				Entry("missing type key",
					msgpack.EncodedJSON{"channel": 12},
					"node 'n1' is missing its function type"),
				Entry("nil input entry",
					msgpack.EncodedJSON(nil),
					"node 'n1' is missing its function type"),
				Entry("non-string type",
					msgpack.EncodedJSON{"type": 42},
					"node 'n1' function type must be a string, got int"),
			)
		})
	})

	Describe("Qualified Module Names", func() {
		It("Should analyze bare select", func(ctx SpecContext) {
			nodes, inputs := buildNodes(
				nodeSpec{key: "on", typ: "on", cfg: map[string]any{"channel": 100}},
				nodeSpec{key: "sel", typ: "select"},
			)
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "on",
						Inputs: types.Params{
							{Name: "channel", Type: types.Chan(types.U8())},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
				Nodes:  nodes,
				Inputs: inputs,
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: ir.Handle{Node: "on", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sel", Param: ir.DefaultOutputParam},
					}},
				},
			}
			resolver := []symbol.Symbol{{
				Name: "flag",
				Type: types.Chan(types.U8()),
				Kind: symbol.KindChannel,
				ID:   100,
			}}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Nodes).To(HaveLen(2))
		})

		It("Should analyze stable.for with qualified name", func(ctx SpecContext) {
			nodes, inputs := buildNodes(
				nodeSpec{key: "on", typ: "on", cfg: map[string]any{"channel": 100}},
				nodeSpec{key: "sf", typ: "stable.for", cfg: map[string]any{
					"duration": int(telem.Millisecond),
				}},
			)
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "on",
						Inputs: types.Params{
							{Name: "channel", Type: types.Chan(types.U8())},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
				Nodes:  nodes,
				Inputs: inputs,
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: ir.Handle{Node: "on", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sf", Param: ir.DefaultInputParam},
					}},
				},
			}
			resolver := []symbol.Symbol{{
				Name: "sensor",
				Type: types.Chan(types.U8()),
				Kind: symbol.KindChannel,
				ID:   100,
			}}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Nodes).To(HaveLen(2))
		})
		It("Should analyze status.set with qualified name", func(ctx SpecContext) {
			nodes, inputs := buildNodes(
				nodeSpec{key: "on", typ: "on", cfg: map[string]any{"channel": 100}},
				nodeSpec{key: "ss", typ: "status.set", cfg: map[string]any{
					"key_or_name": "ox_alarm",
					"message":     "Overpressure",
					"variant":     "error",
				}},
			)
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "on",
						Inputs: types.Params{
							{Name: "channel", Type: types.Chan(types.U8())},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
				Nodes:  nodes,
				Inputs: inputs,
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: ir.Handle{Node: "on", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "ss", Param: ir.DefaultOutputParam},
					}},
				},
			}
			statusFnType := types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: "key_or_name", Type: types.String()},
					{Name: "message", Type: types.String()},
					{Name: "variant", Type: types.String()},
				},
			})
			statusModule := &symbol.Symbol{Name: "status", Kind: symbol.KindModule}
			statusModule.AddChild(&symbol.Symbol{
				Name: "set",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecFlow,
				Type: statusFnType,
			})
			channels := []symbol.Symbol{
				{
					Name: "sensor",
					Type: types.Chan(types.U8()),
					Kind: symbol.KindChannel,
					ID:   100,
				},
			}
			root := symbol.NewRoot(nil, stl.NewSymbols())
			for i := range channels {
				s := channels[i]
				root.Parent.AddChild(&s)
			}
			symbol.AutoImportModules(root)
			root.Parent.AddChild(statusModule)
			root.AddChild(&symbol.Symbol{
				Name:   "status",
				Kind:   symbol.KindModuleAlias,
				Target: statusModule,
				Parent: root,
			})
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, root)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Nodes).To(HaveLen(2))
		})
	})
})
