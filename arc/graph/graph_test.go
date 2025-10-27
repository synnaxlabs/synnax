// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Graph", func() {
	Describe("Parse", func() {
		It("Should correctly parse a single stage", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
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
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
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
		It("Should correctly analyze a single function", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(1))
			funcScope := MustSucceed(inter.Symbols.Resolve(ctx, "add"))
			Expect(funcScope.Children).To(HaveLen(4))
			params := funcScope.FilterChildrenByKind(symbol.KindInput)
			Expect(params).To(HaveLen(2))
			Expect(params[0].Name).To(Equal("a"))
			Expect(params[0].Type).To(Equal(types.I64()))
			Expect(params[1].Name).To(Equal("b"))
			Expect(params[1].Type).To(Equal(types.I64()))
		})

		It("Should correctly analyze a complete program", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "on",
						Config: types.Params{
							Keys:   []string{"channel"},
							Values: []types.Type{types.Chan(types.F32())},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key:    "printer",
						Config: types.Params{},
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.F32()},
						},
					},
				},
				Nodes: []graph.Node{
					{
						Key:    "first",
						Type:   "on",
						Config: map[string]any{"channel": 12},
					},
					{Key: "printer", Type: "printer"},
				},
				Edges: []arc.Edge{
					{
						Source: arc.Handle{Node: "first", Param: ir.DefaultOutputParam},
						Target: arc.Handle{Node: "printer", Param: ir.DefaultInputParam},
					},
				},
			}
			resolver := symbol.MapResolver{
				"12": symbol.Symbol{
					Name: "ox_pt_1",
					Type: types.Chan(types.F32()),
					Kind: symbol.KindChannel,
					ID:   12,
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, resolver)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(2))
			Expect(inter.Nodes).To(HaveLen(2))
			Expect(inter.Edges).To(HaveLen(1))

			firstNode := inter.Nodes[0]
			Expect(firstNode.Key).To(Equal("first"))
			Expect(firstNode.Type).To(Equal("on"))
			Expect(firstNode.ConfigValues).To(HaveLen(1))
			//Expect(firstNode.Channels.Read).To(HaveLen(1))
		})

		Describe("Polymorphic Stages", func() {
			It("Should correctly infer types for polymorphic stages from F32 inputs", func() {
				constraint := types.NumericConstraint()
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "polymorphic_add",
							Inputs: types.Params{
								Keys: []string{"a", "b"},
								Values: []types.Type{
									types.Variable("T", &constraint),
									types.Variable("T", &constraint),
								},
							},
							Outputs: types.Params{
								Keys: []string{ir.DefaultOutputParam},
								Values: []types.Type{
									types.Variable("T", &constraint),
								},
							},
						},
						{
							Key: "f32_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "source1", Type: "f32_source"},
						{Key: "source2", Type: "f32_source"},
						{Key: "adder", Type: "polymorphic_add"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "source1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "adder", Param: "a"},
						},
						{
							Source: ir.Handle{Node: "source2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "adder", Param: "b"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The fact that analysis succeeded without errors indicates
				// that the type variables were successfully unified with F32
				Expect(inter.Functions).To(HaveLen(2))
				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

				// Check that each node instance has concrete resolved types
				// The func definition remains polymorphic, but each node gets concrete types
				adderNode, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "adder" })

				// The adder node should have concrete F32 types resolved from edges
				aType, _ := adderNode.Inputs.Get("a")
				Expect(aType).To(Equal(types.F32()))

				bType, _ := adderNode.Inputs.Get("b")
				Expect(bType).To(Equal(types.F32()))

				// Return type should also be concrete
				returnType, _ := adderNode.Outputs.Get(ir.DefaultOutputParam)
				Expect(returnType).To(Equal(types.F32()))
			})

			It("Should correctly infer types for polymorphic stages from I64 inputs", func() {
				constraint := types.NumericConstraint()
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "polymorphic_multiply",
							Inputs: types.Params{
								Keys: []string{"x", "y"},
								Values: []types.Type{
									types.Variable("T", &constraint),
									types.Variable("T", &constraint),
								},
							},
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.Variable("T", &constraint)},
							},
						},
						{
							Key: "i64_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.I64()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "int_source1", Type: "i64_source"},
						{Key: "int_source2", Type: "i64_source"},
						{Key: "multiplier", Type: "polymorphic_multiply"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "int_source1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "multiplier", Param: "x"},
						},
						{
							Source: ir.Handle{Node: "int_source2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "multiplier", Param: "y"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Check that the multiplier node instance has concrete resolved types
				multiplierNode, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "multiplier" })

				xType, _ := multiplierNode.Inputs.Get("x")
				Expect(xType).To(Equal(types.I64()))

				yType, _ := multiplierNode.Inputs.Get("y")
				Expect(yType).To(Equal(types.I64()))

				returnType, _ := multiplierNode.Outputs.Get(ir.DefaultOutputParam)
				Expect(returnType).To(Equal(types.I64()))
			})

			It("Should handle chained polymorphic stages", func() {
				constraint := types.NumericConstraint()
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "poly_add",
							Inputs: types.Params{
								Keys: []string{"a", "b"},
								Values: []types.Type{
									types.Variable("T", &constraint),
									types.Variable("T", &constraint),
								},
							},
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.Variable("T", &constraint)},
							},
						},
						{
							Key: "poly_scale",
							Inputs: types.Params{
								Keys:   []string{ir.DefaultInputParam},
								Values: []types.Type{types.Variable("U", &constraint)},
							},
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.Variable("U", &constraint)},
							},
						},
						{
							Key: "f64_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F64()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "src1", Type: "f64_source"},
						{Key: "src2", Type: "f64_source"},
						{Key: "add1", Type: "poly_add"},
						{Key: "scale1", Type: "poly_scale"},
						{Key: "scale2", Type: "poly_scale"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "add1", Param: "a"},
						},
						{
							Source: ir.Handle{Node: "src2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "add1", Param: "b"},
						},
						{
							Source: ir.Handle{Node: "add1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "scale1", Param: ir.DefaultInputParam},
						},
						{
							Source: ir.Handle{Node: "add1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "scale2", Param: ir.DefaultInputParam},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Both node instances should have concrete F64 types
				add1Node, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "add1" })
				addReturnType, _ := add1Node.Outputs.Get(ir.DefaultOutputParam)
				Expect(addReturnType).To(Equal(types.F64()))

				scale1Node, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "scale1" })
				inputType, _ := scale1Node.Inputs.Get(ir.DefaultInputParam)
				Expect(inputType).To(Equal(types.F64()))
			})

			It("Should detect type mismatches in polymorphic edge connections", func() {
				constraint := types.NumericConstraint()
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "f32_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "i64_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.I64()},
							},
						},
						{
							Key: "poly_add",
							Inputs: types.Params{
								Keys: []string{"a", "b"},
								Values: []types.Type{
									types.Variable("T", &constraint),
									types.Variable("T", &constraint),
								},
							},
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.Variable("T", &constraint)},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "float_src", Type: "f32_source"},
						{Key: "int_src", Type: "i64_source"},
						{Key: "adder", Type: "poly_add"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "float_src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "adder", Param: "a"},
						},
						{
							Source: ir.Handle{Node: "int_src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "adder", Param: "b"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				// This should fail because poly_add expects both parameters to be the same type T
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("failed to unify"))
			})

			It("Should detect non-numeric type mismatches with polymorphic stages", func() {
				constraint := types.NumericConstraint()
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "string_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.String()},
							},
						},
						{
							Key: "poly_numeric",
							Inputs: types.Params{
								Keys:   []string{"value"},
								Values: []types.Type{types.Variable("T", &constraint)},
							},
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.Variable("T", &constraint)},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "str_src", Type: "string_source"},
						{Key: "numeric_stage", Type: "poly_numeric"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "str_src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "numeric_stage", Param: "value"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				// This should fail because string doesn't satisfy NumericConstraint
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("does not satisfy constraint"))
			})

			It("Should handle missing edge connections", func() {
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "sink",
							Inputs: types.Params{
								Keys:   []string{ir.DefaultInputParam},
								Values: []types.Type{types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "src", Type: "source"},
						{Key: "snk", Type: "sink"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "nonexistent", Param: ir.DefaultOutputParam}, // Invalid target node
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("edge target node 'nonexistent' not found"))
			})

			It("Should handle invalid parameter references in edges", func() {
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "sink",
							Inputs: types.Params{
								Keys:   []string{ir.DefaultInputParam},
								Values: []types.Type{types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "src", Type: "source"},
						{Key: "snk", Type: "sink"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "snk", Param: "invalid_param"}, // Invalid parameter
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("'invalid_param' not found"))
			})

			It("Should handle concrete type mismatches in edges", func() {
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "string_source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.String()},
							},
						},
						{
							Key: "number_sink",
							Inputs: types.Params{
								Keys:   []string{"value"},
								Values: []types.Type{types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "str_src", Type: "string_source"},
						{Key: "num_snk", Type: "number_sink"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "str_src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "num_snk", Param: "value"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("type mismatch"))
			})

			//It("Should allow edges to stages with no parameters (ignored like JS)", func() {
			//	g := graph.Graph{
			//		Functions: []ir.Function{
			//			{
			//				Key: "source",
			//				Outputs: types.Params{
			//					Keys:   []string{ir.DefaultOutputParam},
			//					Values: []types.Type{types.F32()},
			//				},
			//			},
			//			{
			//				Key: "sink_with_no_params",
			//				// No parameters defined - should ignore incoming edges
			//			},
			//		},
			//		Nodes: []graph.Node{
			//			{Key: "src", Type: "source"},
			//			{Key: "sink", Type: "sink_with_no_params"},
			//		},
			//		Edges: []ir.Edge{
			//			{
			//				Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
			//				Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
			//			},
			//		},
			//	}
			//	g = MustSucceed(graph.Parse(g))
			//	inter, diagnostics := graph.Analyze(ctx, g, nil)
			//	// Should succeed - the sink just ignores the input
			//	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			//	Expect(inter.Edges).To(HaveLen(1))
			//})
		})

		Describe("Integration", func() {
			It("Should parse and analyze a complete alarm system graph", func() {
				g := arc.Graph{
					Nodes: []graph.Node{
						{
							Key:    "on",
							Type:   "on",
							Config: map[string]any{"channel": 12},
						},
						{
							Key:    "constant",
							Type:   "constant",
							Config: map[string]any{"value": 10},
						},
						{
							Key:    "ge",
							Type:   "ge",
							Config: map[string]any{},
						},
						{
							Key:  "stable_for",
							Type: "stable_for",
							Config: map[string]any{
								"duration": int(telem.Millisecond * 1),
							},
						},
						{
							Key:  "select",
							Type: "select",
						},
						{
							Key:  "status_success",
							Type: "set_status",
							Config: map[string]any{
								"key":     "ox_alarm",
								"variant": "success",
								"message": "OX Pressure Nominal",
							},
						},
						{
							Key:  "status_error",
							Type: "set_status",
							Config: map[string]any{
								"key":     "ox_alarm",
								"variant": "error",
								"message": "OX Pressure Alarm",
							},
						},
					},
					Edges: []arc.Edge{
						{
							Source: arc.Handle{Node: "on", Param: ir.DefaultOutputParam},
							Target: arc.Handle{Node: "ge", Param: "a"},
						},
						{
							Source: arc.Handle{Node: "constant", Param: ir.DefaultOutputParam},
							Target: arc.Handle{Node: "ge", Param: "b"},
						},
						{
							Source: arc.Handle{Node: "ge", Param: ir.DefaultOutputParam},
							Target: arc.Handle{Node: "stable_for", Param: ir.DefaultInputParam},
						},
						{
							Source: arc.Handle{Node: "stable_for", Param: ir.DefaultOutputParam},
							Target: arc.Handle{Node: "select", Param: ir.DefaultInputParam},
						},
						{
							Source: arc.Handle{Node: "select", Param: "false"},
							Target: arc.Handle{Node: "status_success", Param: ir.DefaultInputParam},
						},
						{
							Source: arc.Handle{Node: "select", Param: "true"},
							Target: arc.Handle{Node: "status_error", Param: ir.DefaultInputParam},
						},
					},
				}

				// First, define the func signatures that this graph expects
				// Using polymorphic types for constant, ge, and stable_for
				// Each func gets its own type variables

				constraint := types.NumericConstraint()
				functions := []ir.Function{
					{
						Key: "on",
						Config: types.Params{
							Keys:   []string{"channel"},
							Values: []types.Type{types.U32()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						}, // Returns sensor reading
					},
					{
						Key: "constant",
						Config: types.Params{
							Keys:   []string{"value"},
							Values: []types.Type{types.Variable("A", &constraint)},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.Variable("A", &constraint)},
						},
					},
					{
						Key: "ge",
						Inputs: types.Params{
							Keys: []string{"a", "b"},
							Values: []types.Type{
								types.Variable("B", &constraint),
								types.Variable("B", &constraint),
							},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key: "stable_for",
						Config: types.Params{
							Keys:   []string{"duration"},
							Values: []types.Type{types.TimeSpan()},
						},
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.Variable("C", nil)},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.Variable("C", nil)},
						},
					},
					{
						Key: "select",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.U8()},
						},
						Outputs: types.Params{
							Keys:   []string{"false", "true"},
							Values: []types.Type{types.U8(), types.U8()},
						},
					},
					{
						Key: "set_status",
						Config: types.Params{
							Keys:   []string{"key", "variant", "message"},
							Values: []types.Type{types.String(), types.String(), types.String()},
						},
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.U8()},
						},
					},
				}

				// Convert arc.Graph to graph.Graph
				graphWithFunctions := graph.Graph{
					Functions: functions,
					Nodes:     g.Nodes,
					Edges:     g.Edges,
				}

				// Parse the graph
				parsed := MustSucceed(graph.Parse(graphWithFunctions))

				// The graph should have been parsed successfully
				Expect(parsed.Nodes).To(HaveLen(7))
				Expect(parsed.Edges).To(HaveLen(6))

				// Analyze the graph
				inter, diagnostics := graph.Analyze(ctx, parsed, nil)

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
				// select.false -> status_success
				// select.true -> status_error
				Expect(inter.Edges).To(HaveLen(6))

				// Verify configuration was parsed correctly
				constantNode := lo.Filter(parsed.Nodes, func(n graph.Node, _ int) bool {
					return n.Key == "constant"
				})[0]
				Expect(constantNode.Config).To(HaveKeyWithValue("value", 10))

				stableForNode := lo.Filter(parsed.Nodes, func(n graph.Node, _ int) bool {
					return n.Key == "stable_for"
				})[0]
				Expect(stableForNode.Config).To(HaveKeyWithValue("duration", int(telem.Millisecond)))

				// Verify polymorphic node instances have concrete resolved types
				// func definitions stay polymorphic, but each node instance gets concrete types

				// The constant node should have concrete F64 type
				// (since it connects to "ge" which receives F64 from "on")
				constantIRNode, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "constant" })
				constantReturnType, _ := constantIRNode.Outputs.Get(ir.DefaultOutputParam)
				Expect(constantReturnType).To(Equal(types.F64()))

				// The ge node should have concrete F64 parameters
				// (since it receives F64 inputs from "on" and "constant")
				geIRNode, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "ge" })
				aType, _ := geIRNode.Inputs.Get("a")
				Expect(aType).To(Equal(types.F64()))
				bType, _ := geIRNode.Inputs.Get("b")
				Expect(bType).To(Equal(types.F64()))

				// The stable_for node should have concrete U8 types
				// (since it receives U8 from "ge" comparison result)
				stableIRNode, _ := lo.Find(inter.Nodes, func(n ir.Node) bool { return n.Key == "stable_for" })
				inputType, _ := stableIRNode.Inputs.Get(ir.DefaultInputParam)
				Expect(inputType).To(Equal(types.U8()))
				stableReturnType, _ := stableIRNode.Outputs.Get(ir.DefaultOutputParam)
				Expect(stableReturnType).To(Equal(types.U8()))
			})
		})

		Describe("Edge Validation", func() {
			Describe("Type Matching", func() {
				It("Should validate series type matching", func() {
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "series_f32_source",
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.Series(types.F32())},
								},
							},
							{
								Key: "series_i64_sink",
								Inputs: types.Params{
									Keys:   []string{ir.DefaultInputParam},
									Values: []types.Type{types.Series(types.I64())},
								},
							},
						},
						Nodes: []graph.Node{
							{Key: "src", Type: "series_f32_source"},
							{Key: "snk_mismatch", Type: "series_i64_sink"},
						},
						Edges: []ir.Edge{
							{
								Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
								Target: ir.Handle{Node: "snk_mismatch", Param: ir.DefaultInputParam},
							},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, nil)
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(diagnostics.String()).To(ContainSubstring("type mismatch"))
				})

				It("Should error when a node is missing some required inputs", func() {
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source",
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
							{
								Key: "dual_input",
								Inputs: types.Params{
									Keys:   []string{"a", "b"},
									Values: []types.Type{types.F32(), types.F32()},
								},
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
						},
						Nodes: []graph.Node{
							{Key: "src", Type: "source"},
							{Key: "dual", Type: "dual_input"},
						},
						Edges: []ir.Edge{
							{
								Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
								Target: ir.Handle{Node: "dual", Param: "a"},
							},
						},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, nil)
					Expect(diagnostics.Ok()).To(BeFalse())
					Expect(diagnostics.String()).To(ContainSubstring("missing required input"))
				})
				It("Should succeed when all required inputs are connected", func() {
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source",
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
							{
								Key: "dual_input",
								Inputs: types.Params{
									Keys:   []string{"a", "b"},
									Values: []types.Type{types.F32(), types.F32()},
								},
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
						},
						Nodes: []graph.Node{
							{Key: "src1", Type: "source"},
							{Key: "src2", Type: "source"},
							{Key: "dual", Type: "dual_input"},
						},
						Edges: []ir.Edge{
							{
								Source: ir.Handle{Node: "src1", Param: ir.DefaultOutputParam},
								Target: ir.Handle{Node: "dual", Param: "a"},
							},
							{
								Source: ir.Handle{Node: "src2", Param: ir.DefaultOutputParam},
								Target: ir.Handle{Node: "dual", Param: "b"},
							},
						},
					}
					g = MustSucceed(graph.Parse(g))
					inter, diagnostics := graph.Analyze(ctx, g, nil)
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
					Expect(inter.Edges).To(HaveLen(2))
				})
				It("Should allow nodes with no inputs to exist without edges", func() {
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source_only",
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
						},
						Nodes: []graph.Node{
							{Key: "src1", Type: "source_only"},
							{Key: "src2", Type: "source_only"},
						},
						Edges: []ir.Edge{},
					}
					g = MustSucceed(graph.Parse(g))
					inter, diagnostics := graph.Analyze(ctx, g, nil)
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
					Expect(inter.Nodes).To(HaveLen(2))
				})
				It("Should error when multiple nodes have missing inputs", func() {
					g := graph.Graph{
						Functions: []ir.Function{
							{
								Key: "source",
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
							{
								Key: "processor",
								Inputs: types.Params{
									Keys:   []string{ir.DefaultInputParam},
									Values: []types.Type{types.F32()},
								},
								Outputs: types.Params{
									Keys:   []string{ir.DefaultOutputParam},
									Values: []types.Type{types.F32()},
								},
							},
						},
						Nodes: []graph.Node{
							{Key: "src", Type: "source"},
							{Key: "proc1", Type: "processor"},
							{Key: "proc2", Type: "processor"},
						},
						Edges: []ir.Edge{},
					}
					g = MustSucceed(graph.Parse(g))
					_, diagnostics := graph.Analyze(ctx, g, nil)
					Expect(diagnostics.Ok()).To(BeFalse())
					diag := diagnostics.String()
					Expect(diag).To(ContainSubstring("proc1"))
					Expect(diag).To(ContainSubstring("proc2"))
				})
			})
		})
		Describe("Duplicate Edge Targets", func() {
			It("Should error when multiple edges target the same input parameter", func() {
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "processor",
							Inputs: types.Params{
								Keys:   []string{"input"},
								Values: []types.Type{types.F32()},
							},
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "src1", Type: "source"},
						{Key: "src2", Type: "source"},
						{Key: "proc", Type: "processor"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "proc", Param: "input"},
						},
						{
							Source: ir.Handle{Node: "src2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "proc", Param: "input"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("multiple edges"))
			})
			It("Should allow multiple edges from the same source parameter", func() {
				g := graph.Graph{
					Functions: []ir.Function{
						{
							Key: "source",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "sink",
							Inputs: types.Params{
								Keys:   []string{"input"},
								Values: []types.Type{types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "src", Type: "source"},
						{Key: "snk1", Type: "sink"},
						{Key: "snk2", Type: "sink"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "snk1", Param: "input"},
						},
						{
							Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "snk2", Param: "input"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Edges).To(HaveLen(2))
			})
		})

	})
})
