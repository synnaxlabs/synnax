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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Graph", func() {
	Describe("Parse", func() {
		It("Should correctly parse a single stage", func() {
			g := graph.Graph{
				Stages: []ir.Stage{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			Expect(g.Stages[0].Body.AST).ToNot(BeNil())
		})

		It("Should correctly parse a single function", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
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
		It("Should correctly analyze a single stage", func() {
			g := graph.Graph{
				Stages: []ir.Stage{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Stages).To(HaveLen(1))
			stageScope := MustSucceed(inter.Symbols.Resolve(ctx, "add"))
			Expect(stageScope.Children).To(HaveLen(3))
			params := stageScope.FilterChildrenByKind(ir.KindParam)
			Expect(params).To(HaveLen(2))
			Expect(params[0].Name).To(Equal("a"))
			Expect(params[0].Type).To(Equal(ir.I64{}))
			Expect(params[1].Name).To(Equal("b"))
			Expect(params[1].Type).To(Equal(ir.I64{}))
		})

		It("Should correctly analyze a single function", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
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
			Expect(funcScope.Children).To(HaveLen(3))
			params := funcScope.FilterChildrenByKind(ir.KindParam)
			Expect(params).To(HaveLen(2))
			Expect(params[0].Name).To(Equal("a"))
			Expect(params[0].Type).To(Equal(ir.I64{}))
			Expect(params[1].Name).To(Equal("b"))
			Expect(params[1].Type).To(Equal(ir.I64{}))
		})

		It("Should correctly analyze a complete program", func() {
			g := arc.Graph{
				Stages: []ir.Stage{
					{
						Key: "on",
						Config: ir.NamedTypes{
							Keys:   []string{"channel"},
							Values: []ir.Type{ir.Chan{}},
						},
					},
					{
						Key:    "printer",
						Config: ir.NamedTypes{},
						Params: ir.NamedTypes{
							Keys:   []string{"input"},
							Values: []ir.Type{ir.F32{}},
						},
					},
				},
				Nodes: []graph.Node{
					{Node: arc.Node{
						Key:    "first",
						Type:   "on",
						Config: map[string]any{"channel": 12},
					}},
					{Node: arc.Node{Key: "printer", Type: "printer"}},
				},
				Edges: []arc.Edge{
					{
						Source: arc.Handle{Node: "first", Param: "output"},
						Target: arc.Handle{Node: "printer", Param: "input"},
					},
				},
			}
			resolver := ir.MapResolver{
				"12": ir.Symbol{
					Name: "ox_pt_1",
					Type: ir.Chan{ValueType: ir.F32{}},
					Kind: ir.KindChannel,
					ID:   12,
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, resolver)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Stages).To(HaveLen(2))
			Expect(inter.Nodes).To(HaveLen(2))
			Expect(inter.Edges).To(HaveLen(1))

			firstNode := inter.Nodes[0]
			Expect(firstNode.Key).To(Equal("first"))
			Expect(firstNode.Type).To(Equal("on"))
			Expect(firstNode.Config).To(HaveLen(1))
			Expect(firstNode.Channels.Read).To(HaveLen(1))
		})

		Describe("Polymorphic Stages", func() {
			It("Should correctly infer types for polymorphic stages from F32 inputs", func() {
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key: "polymorphic_add",
							Params: ir.NamedTypes{
								Keys:   []string{"a", "b"},
								Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{}), ir.NewTypeVariable("T", ir.NumericConstraint{})},
							},
							Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
						},
						{
							Key:    "f32_source",
							Return: ir.F32{},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "source1", Type: "f32_source"}},
						{Node: ir.Node{Key: "source2", Type: "f32_source"}},
						{Node: ir.Node{Key: "adder", Type: "polymorphic_add"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "source1", Param: "output"},
							Target: ir.Handle{Node: "adder", Param: "a"},
						},
						{
							Source: ir.Handle{Node: "source2", Param: "output"},
							Target: ir.Handle{Node: "adder", Param: "b"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The fact that analysis succeeded without errors indicates
				// that the type variables were successfully unified with F32
				Expect(inter.Stages).To(HaveLen(2))
				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

				// Now we can check the resolved types through the constraint system
				addStage := MustSucceed(inter.Symbols.Resolve(ctx, "polymorphic_add"))
				stageType := addStage.Type.(ir.Stage)

				// Check that parameters resolve to F32
				aType, _ := stageType.Params.Get("a")
				resolvedA := inter.Constraints.ApplySubstitutions(aType)
				Expect(resolvedA).To(Equal(ir.F32{}))

				bType, _ := stageType.Params.Get("b")
				resolvedB := inter.Constraints.ApplySubstitutions(bType)
				Expect(resolvedB).To(Equal(ir.F32{}))

				// Check that return type resolves to F32
				resolvedReturn := inter.Constraints.ApplySubstitutions(stageType.Return)
				Expect(resolvedReturn).To(Equal(ir.F32{}))
			})

			It("Should correctly infer types for polymorphic stages from I64 inputs", func() {
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key: "polymorphic_multiply",
							Params: ir.NamedTypes{
								Keys:   []string{"x", "y"},
								Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{}), ir.NewTypeVariable("T", ir.NumericConstraint{})},
							},
							Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
						},
						{
							Key:    "i64_source",
							Return: ir.I64{},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "int_source1", Type: "i64_source"}},
						{Node: ir.Node{Key: "int_source2", Type: "i64_source"}},
						{Node: ir.Node{Key: "multiplier", Type: "polymorphic_multiply"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "int_source1", Param: "output"},
							Target: ir.Handle{Node: "multiplier", Param: "x"},
						},
						{
							Source: ir.Handle{Node: "int_source2", Param: "output"},
							Target: ir.Handle{Node: "multiplier", Param: "y"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The polymorphic_multiply stage should have its type variables resolved to I64
				multStage := MustSucceed(inter.Symbols.Resolve(ctx, "polymorphic_multiply"))
				stageType := multStage.Type.(ir.Stage)

				xType, _ := stageType.Params.Get("x")
				resolvedX := inter.Constraints.ApplySubstitutions(xType)
				Expect(resolvedX).To(Equal(ir.I64{}))

				yType, _ := stageType.Params.Get("y")
				resolvedY := inter.Constraints.ApplySubstitutions(yType)
				Expect(resolvedY).To(Equal(ir.I64{}))

				resolvedReturn := inter.Constraints.ApplySubstitutions(stageType.Return)
				Expect(resolvedReturn).To(Equal(ir.I64{}))
			})

			It("Should handle chained polymorphic stages", func() {
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key: "poly_add",
							Params: ir.NamedTypes{
								Keys:   []string{"a", "b"},
								Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{}), ir.NewTypeVariable("T", ir.NumericConstraint{})},
							},
							Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
						},
						{
							Key: "poly_scale",
							Params: ir.NamedTypes{
								Keys:   []string{"input"},
								Values: []ir.Type{ir.NewTypeVariable("U", ir.NumericConstraint{})},
							},
							Return: ir.NewTypeVariable("U", ir.NumericConstraint{}),
						},
						{
							Key:    "f64_source",
							Return: ir.F64{},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "src1", Type: "f64_source"}},
						{Node: ir.Node{Key: "src2", Type: "f64_source"}},
						{Node: ir.Node{Key: "add1", Type: "poly_add"}},
						{Node: ir.Node{Key: "scale1", Type: "poly_scale"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src1", Param: "output"},
							Target: ir.Handle{Node: "add1", Param: "a"},
						},
						{
							Source: ir.Handle{Node: "src2", Param: "output"},
							Target: ir.Handle{Node: "add1", Param: "b"},
						},
						{
							Source: ir.Handle{Node: "add1", Param: "output"},
							Target: ir.Handle{Node: "scale1", Param: "input"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Both stages should resolve to F64
				addStage := MustSucceed(inter.Symbols.Resolve(ctx, "poly_add"))
				addType := addStage.Type.(ir.Stage)
				resolvedAddReturn := inter.Constraints.ApplySubstitutions(addType.Return)
				Expect(resolvedAddReturn).To(Equal(ir.F64{}))

				scaleStage := MustSucceed(inter.Symbols.Resolve(ctx, "poly_scale"))
				scaleType := scaleStage.Type.(ir.Stage)
				inputType, _ := scaleType.Params.Get("input")
				resolvedInput := inter.Constraints.ApplySubstitutions(inputType)
				Expect(resolvedInput).To(Equal(ir.F64{}))
			})

			It("Should detect type mismatches in polymorphic edge connections", func() {
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key:    "f32_source",
							Return: ir.F32{},
						},
						{
							Key:    "i64_source",
							Return: ir.I64{},
						},
						{
							Key: "poly_add",
							Params: ir.NamedTypes{
								Keys:   []string{"a", "b"},
								Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{}), ir.NewTypeVariable("T", ir.NumericConstraint{})},
							},
							Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "float_src", Type: "f32_source"}},
						{Node: ir.Node{Key: "int_src", Type: "i64_source"}},
						{Node: ir.Node{Key: "adder", Type: "poly_add"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "float_src", Param: "output"},
							Target: ir.Handle{Node: "adder", Param: "a"},
						},
						{
							Source: ir.Handle{Node: "int_src", Param: "output"},
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
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key:    "string_source",
							Return: ir.String{},
						},
						{
							Key: "poly_numeric",
							Params: ir.NamedTypes{
								Keys:   []string{"value"},
								Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
							},
							Return: ir.NewTypeVariable("T", ir.NumericConstraint{}),
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "str_src", Type: "string_source"}},
						{Node: ir.Node{Key: "numeric_stage", Type: "poly_numeric"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "str_src", Param: "output"},
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
					Stages: []ir.Stage{
						{
							Key:    "source",
							Return: ir.F32{},
						},
						{
							Key: "sink",
							Params: ir.NamedTypes{
								Keys:   []string{"input"},
								Values: []ir.Type{ir.F32{}},
							},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "src", Type: "source"}},
						{Node: ir.Node{Key: "snk", Type: "sink"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src", Param: "output"},
							Target: ir.Handle{Node: "nonexistent", Param: "output"}, // Invalid target node
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
					Stages: []ir.Stage{
						{
							Key:    "source",
							Return: ir.F32{},
						},
						{
							Key: "sink",
							Params: ir.NamedTypes{
								Keys:   []string{"input"},
								Values: []ir.Type{ir.F32{}},
							},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "src", Type: "source"}},
						{Node: ir.Node{Key: "snk", Type: "sink"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src", Param: "output"},
							Target: ir.Handle{Node: "snk", Param: "invalid_param"}, // Invalid parameter
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("target param 'invalid_param' not found"))
			})

			It("Should handle concrete type mismatches in edges", func() {
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key:    "string_source",
							Return: ir.String{},
						},
						{
							Key: "number_sink",
							Params: ir.NamedTypes{
								Keys:   []string{"value"},
								Values: []ir.Type{ir.F32{}},
							},
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "str_src", Type: "string_source"}},
						{Node: ir.Node{Key: "num_snk", Type: "number_sink"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "str_src", Param: "output"},
							Target: ir.Handle{Node: "num_snk", Param: "value"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				_, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("type mismatch"))
			})

			It("Should allow edges to stages with no parameters (ignored like JS)", func() {
				g := graph.Graph{
					Stages: []ir.Stage{
						{
							Key:    "source",
							Return: ir.F32{},
						},
						{
							Key: "sink_with_no_params",
							// No parameters defined - should ignore incoming edges
						},
					},
					Nodes: []graph.Node{
						{Node: ir.Node{Key: "src", Type: "source"}},
						{Node: ir.Node{Key: "sink", Type: "sink_with_no_params"}},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "src", Param: "output"},
							Target: ir.Handle{Node: "sink", Param: "output"},
						},
					},
				}
				g = MustSucceed(graph.Parse(g))
				inter, diagnostics := graph.Analyze(ctx, g, nil)
				// Should succeed - the sink just ignores the input
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Edges).To(HaveLen(1))
			})
		})

		Describe("Integration", func() {
			It("Should parse and analyze a complete alarm system graph", func() {
				g := arc.Graph{
					Nodes: []graph.Node{
						{Node: arc.Node{
							Key:    "on",
							Type:   "on",
							Config: map[string]any{"channel": 12},
						}},
						{Node: arc.Node{
							Key:    "constant",
							Type:   "constant",
							Config: map[string]any{"value": 10},
						}},
						{Node: arc.Node{
							Key:    "ge",
							Type:   "ge",
							Config: map[string]any{},
						}},
						{Node: arc.Node{
							Key:  "stable_for",
							Type: "stable_for",
							Config: map[string]any{
								"duration": int(telem.Millisecond * 1),
							},
						}},
						{Node: arc.Node{
							Key:  "select",
							Type: "select",
						}},
						{Node: arc.Node{
							Key:  "status_success",
							Type: "set_status",
							Config: map[string]any{
								"key":     "ox_alarm",
								"variant": "success",
								"message": "OX Pressure Nominal",
							},
						}},
						{Node: arc.Node{
							Key:  "status_error",
							Type: "set_status",
							Config: map[string]any{
								"key":     "ox_alarm",
								"variant": "error",
								"message": "OX Pressure Alarm",
							},
						}},
					},
					Edges: []arc.Edge{
						{
							Source: arc.Handle{Node: "on", Param: "output"},
							Target: arc.Handle{Node: "ge", Param: "a"},
						},
						{
							Source: arc.Handle{Node: "constant", Param: "output"},
							Target: arc.Handle{Node: "ge", Param: "b"},
						},
						{
							Source: arc.Handle{Node: "ge", Param: "output"},
							Target: arc.Handle{Node: "stable_for", Param: "output"},
						},
						{
							Source: arc.Handle{Node: "stable_for", Param: "output"},
							Target: arc.Handle{Node: "select", Param: "input"},
						},
						{
							Source: arc.Handle{Node: "select", Param: "false"},
							Target: arc.Handle{Node: "status_success", Param: "input"},
						},
						{
							Source: arc.Handle{Node: "select", Param: "true"},
							Target: arc.Handle{Node: "status_error", Param: "input"},
						},
					},
				}

				// First, define the stage signatures that this graph expects
				// Using polymorphic types for constant, ge, and stable_for
				// Each stage gets its own type variables

				stages := []ir.Stage{
					{
						Key: "on",
						Config: ir.NamedTypes{
							Keys:   []string{"channel"},
							Values: []ir.Type{ir.U32{}},
						},
						Return: ir.F64{}, // Returns sensor reading
					},
					{
						Key: "constant",
						Config: ir.NamedTypes{
							Keys:   []string{"value"},
							Values: []ir.Type{ir.NewTypeVariable("A", ir.NumericConstraint{})},
						},
						Return: ir.NewTypeVariable("A", ir.NumericConstraint{}),
					},
					{
						Key: "ge",
						Params: ir.NamedTypes{
							Keys: []string{"a", "b"},
							Values: []ir.Type{
								ir.NewTypeVariable("B", ir.NumericConstraint{}),
								ir.NewTypeVariable("B", ir.NumericConstraint{}),
							},
						},
						Return: ir.U8{},
					},
					{
						Key: "stable_for",
						Config: ir.NamedTypes{
							Keys:   []string{"duration"},
							Values: []ir.Type{ir.TimeSpan{}},
						},
						Params: ir.NamedTypes{
							Keys:   []string{"input"},
							Values: []ir.Type{ir.NewTypeVariable("C", nil)},
						},
						Return: ir.NewTypeVariable("C", nil),
					},
					{
						Key: "select",
						Params: ir.NamedTypes{
							Keys:   []string{"input", "false", "true"},
							Values: []ir.Type{ir.U8{}, ir.U8{}, ir.U8{}},
						},
						Return: ir.U8{},
					},
					{
						Key: "set_status",
						Config: ir.NamedTypes{
							Keys:   []string{"key", "variant", "message"},
							Values: []ir.Type{ir.String{}, ir.String{}, ir.String{}},
						},
						Params: ir.NamedTypes{
							Keys:   []string{"input"},
							Values: []ir.Type{ir.U8{}},
						},
					},
				}

				// Convert arc.Graph to graph.Graph
				graphWithStages := graph.Graph{
					Stages: stages,
					Nodes:  g.Nodes,
					Edges:  g.Edges,
				}

				// Parse the graph
				parsed := MustSucceed(graph.Parse(graphWithStages))

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
				Expect(onNode.Type).To(BeAssignableToTypeOf(ir.Stage{}))

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

				// Verify polymorphic types were resolved correctly
				// The constant stage should have its type variable T resolved to F64
				// (since it receives config value 10 which gets connected to ge that compares with F64 from "on")
				constantStage := MustSucceed(inter.Symbols.Resolve(ctx, "constant"))
				constantType := constantStage.Type.(ir.Stage)
				// The return type should be resolved to F64 after unification
				// (since "constant" connects to "ge" which receives F64 from "on")
				resolvedConstantReturn := inter.Constraints.ApplySubstitutions(constantType.Return)
				Expect(resolvedConstantReturn).To(Equal(ir.F64{}))

				// The ge stage should have its type variables resolved to F64
				// (since it receives F64 inputs from "on" and "constant")
				geStage := MustSucceed(inter.Symbols.Resolve(ctx, "ge"))
				geType := geStage.Type.(ir.Stage)
				aType, _ := geType.Params.Get("a")
				resolvedA := inter.Constraints.ApplySubstitutions(aType)
				Expect(resolvedA).To(Equal(ir.F64{}))
				bType, _ := geType.Params.Get("b")
				resolvedB := inter.Constraints.ApplySubstitutions(bType)
				Expect(resolvedB).To(Equal(ir.F64{}))

				// The stable_for stage should have its type variable U resolved to U8
				// (since it receives U8 from "ge" comparison result)
				stableStage := MustSucceed(inter.Symbols.Resolve(ctx, "stable_for"))
				stableType := stableStage.Type.(ir.Stage)
				inputType, _ := stableType.Params.Get("input")
				resolvedInput := inter.Constraints.ApplySubstitutions(inputType)
				Expect(resolvedInput).To(Equal(ir.U8{}))
				resolvedStableReturn := inter.Constraints.ApplySubstitutions(stableType.Return)
				Expect(resolvedStableReturn).To(Equal(ir.U8{}))
			})
		})
	})
})
