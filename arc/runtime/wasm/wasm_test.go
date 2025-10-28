// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Wasm", func() {
	Describe("Next with mismatched input lengths", func() {
		It("Should repeat shorter input values to match longest input", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							Keys:   []string{"lhs", "rhs"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
						Body: ir.Body{Raw: `{
							return lhs + rhs
						}`},
					},
				},
			}
			mod := MustSucceed(arc.CompileGraph(ctx, g))
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "lhs",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key: "rhs",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key:  "add",
						Type: "add",
						Inputs: types.Params{
							Keys:   []string{"lhs", "rhs"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add", Param: "lhs"},
					},
					{
						Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "add", Param: "rhs"},
					},
				},
			}
			s := state.New(cfg)
			lhsNode := s.Node("lhs")
			rhsNode := s.Node("rhs")
			*lhsNode.Output(0) = telem.NewSeriesV[int64](1, 2, 3, 4, 5)
			*lhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)
			*rhsNode.Output(0) = telem.NewSeriesV[int64](10, 20)
			*rhsNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2)
			wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
				Module: mod,
				State:  s,
			}))
			defer func() {
				Expect(wasmMod.Close()).To(Succeed())
			}()
			factory := MustSucceed(wasm.NewFactory(wasmMod))
			n, err := factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[2],
				State:  s.Node("add"),
				Module: mod,
			})
			Expect(err).ToNot(HaveOccurred())
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("add").Output(0)
			Expect(result.Len()).To(Equal(int64(5)))
			vals := telem.UnmarshalSeries[int64](result)
			Expect(vals).To(Equal([]int64{11, 22, 13, 24, 15}))
			resultTime := *s.Node("add").OutputTime(0)
			Expect(resultTime.Len()).To(Equal(int64(5)))
			Expect(resultTime).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
		})
		It("Should handle equal length inputs correctly", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "multiply",
						Inputs: types.Params{
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I32(), types.I32()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
						Body: ir.Body{Raw: `{
							return a * b
						}`},
					},
				},
			}
			mod := MustSucceed(arc.CompileGraph(ctx, g))
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "a",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "b",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key:  "multiply",
						Type: "multiply",
						Inputs: types.Params{
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I32(), types.I32()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "multiply", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "multiply", Param: "b"},
					},
				},
			}
			s := state.New(cfg)
			aNode := s.Node("a")
			bNode := s.Node("b")
			*aNode.Output(0) = telem.NewSeriesV[int32](2, 3, 4)
			*aNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20, 30)
			*bNode.Output(0) = telem.NewSeriesV[int32](5, 6, 7)
			*bNode.OutputTime(0) = telem.NewSeriesSecondsTSV(10, 20, 30)
			wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
				Module: mod,
				State:  s,
			}))
			defer func() {
				Expect(wasmMod.Close()).To(Succeed())
			}()
			factory := MustSucceed(wasm.NewFactory(wasmMod))
			n, err := factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[2],
				State:  s.Node("multiply"),
				Module: mod,
			})
			Expect(err).ToNot(HaveOccurred())
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("multiply").Output(0)
			Expect(result.Len()).To(Equal(int64(3)))
			vals := telem.UnmarshalSeries[int32](result)
			Expect(vals).To(Equal([]int32{10, 18, 28}))
			resultTime := *s.Node("multiply").OutputTime(0)
			Expect(resultTime.Len()).To(Equal(int64(3)))
			Expect(resultTime).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20, 30)))
		})
		It("Should repeat single value input across all iterations", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "subtract",
						Inputs: types.Params{
							Keys:   []string{"x", "y"},
							Values: []types.Type{types.F32(), types.F32()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
						Body: ir.Body{Raw: `{
							return x - y
						}`},
					},
				},
			}
			mod := MustSucceed(arc.CompileGraph(ctx, g))
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "x",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "y",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key:  "subtract",
						Type: "subtract",
						Inputs: types.Params{
							Keys:   []string{"x", "y"},
							Values: []types.Type{types.F32(), types.F32()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "x", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "subtract", Param: "x"},
					},
					{
						Source: ir.Handle{Node: "y", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "subtract", Param: "y"},
					},
				},
			}
			s := state.New(cfg)
			xNode := s.Node("x")
			yNode := s.Node("y")
			*xNode.Output(0) = telem.NewSeriesV[float32](100.0, 200.0, 300.0, 400.0)
			*xNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5, 10, 15, 20)
			*yNode.Output(0) = telem.NewSeriesV[float32](25.0)
			*yNode.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
				Module: mod,
				State:  s,
			}))
			defer func() {
				Expect(wasmMod.Close()).To(Succeed())
			}()
			factory := MustSucceed(wasm.NewFactory(wasmMod))
			n, err := factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[2],
				State:  s.Node("subtract"),
				Module: mod,
			})
			Expect(err).ToNot(HaveOccurred())
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("subtract").Output(0)
			Expect(result.Len()).To(Equal(int64(4)))
			vals := telem.UnmarshalSeries[float32](result)
			Expect(vals).To(Equal([]float32{75.0, 175.0, 275.0, 375.0}))
			resultTime := *s.Node("subtract").OutputTime(0)
			Expect(resultTime.Len()).To(Equal(int64(4)))
			Expect(resultTime).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(5, 10, 15, 20)))
		})
	})
	Describe("Next with multiple outputs", func() {
		It("Should handle functions with multiple outputs and mismatched input lengths", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "math_ops",
						Inputs: types.Params{
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{"sum", "product"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Body: ir.Body{Raw: `{
							sum = a + b
							product = a * b
						}`},
					},
				},
			}
			mod := MustSucceed(arc.CompileGraph(ctx, g))
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "a",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key: "b",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key:  "math_ops",
						Type: "math_ops",
						Inputs: types.Params{
							Keys:   []string{"a", "b"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						Outputs: types.Params{
							Keys:   []string{"sum", "product"},
							Values: []types.Type{types.I64(), types.I64()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "math_ops", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "math_ops", Param: "b"},
					},
				},
			}
			s := state.New(cfg)
			aNode := s.Node("a")
			bNode := s.Node("b")
			*aNode.Output(0) = telem.NewSeriesV[int64](10, 20, 30)
			*aNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			*bNode.Output(0) = telem.NewSeriesV[int64](5)
			*bNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
				Module: mod,
				State:  s,
			}))
			defer func() {
				Expect(wasmMod.Close()).To(Succeed())
			}()
			factory := MustSucceed(wasm.NewFactory(wasmMod))
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[2],
				State:  s.Node("math_ops"),
				Module: mod,
			}))
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains("sum")).To(BeTrue())
			Expect(changed.Contains("product")).To(BeTrue())
			sumResult := *s.Node("math_ops").Output(0)
			Expect(sumResult.Len()).To(Equal(int64(3)))
			sumVals := telem.UnmarshalSeries[int64](sumResult)
			Expect(sumVals).To(Equal([]int64{15, 25, 35}))
			productResult := *s.Node("math_ops").Output(1)
			Expect(productResult.Len()).To(Equal(int64(3)))
			productVals := telem.UnmarshalSeries[int64](productResult)
			Expect(productVals).To(Equal([]int64{50, 100, 150}))
			sumTime := *s.Node("math_ops").OutputTime(0)
			Expect(sumTime.Len()).To(Equal(int64(3)))
			Expect(sumTime).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(1, 2, 3)))
			productTime := *s.Node("math_ops").OutputTime(1)
			Expect(productTime.Len()).To(Equal(int64(3)))
			Expect(productTime).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(1, 2, 3)))
		})
	})

	Describe("Runtime Operations - Channel Read", func() {
		It("Should read from channels using runtime bindings", func() {
			resolver := symbol.MapResolver{
				"sensor": symbol.Symbol{
					Name: "sensor",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.I32()),
					ID:   0,
				},
			}

			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "read_channel",
						Inputs: types.Params{
							Keys:   []string{},
							Values: []types.Type{},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
						Body: ir.Body{Raw: `{
							value i32 := sensor
							return value * 2
						}`},
					},
				},
			}
			mod := MustSucceed(arc.CompileGraph(ctx, g, arc.WithResolver(resolver)))

			cfg := state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 0, DataType: telem.Int32T},
				},
				Nodes: []ir.Node{
					{
						Key:  "read_channel",
						Type: "read_channel",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
			}
			s := state.New(cfg)

			// Ingest test data to channel
			fr := telem.Frame[uint32]{}
			fr = fr.Append(0, telem.NewSeriesV[int32](21))
			s.Ingest(fr)

			wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
				Module: mod,
				State:  s,
			}))
			defer func() {
				Expect(wasmMod.Close()).To(Succeed())
			}()
			factory := MustSucceed(wasm.NewFactory(wasmMod))
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[0],
				State:  s.Node("read_channel"),
				Module: mod,
			}))

			// Trigger execution
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})

			// Verify result
			result := *s.Node("read_channel").Output(0)
			Expect(result.Len()).To(BeNumerically(">", 0))
			vals := telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(42)))
		})
	})

	Describe("Runtime Operations - State Persistence", func() {
		It("Should persist stateful variables across function calls", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "counter",
						Inputs: types.Params{
							Keys:   []string{},
							Values: []types.Type{},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
						Body: ir.Body{Raw: `{
							count i64 $= 0
							count = count + 1
							return count
						}`},
					},
				},
			}
			mod := MustSucceed(arc.CompileGraph(ctx, g))

			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key:  "counter",
						Type: "counter",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
			}
			s := state.New(cfg)
			wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
				Module: mod,
				State:  s,
			}))
			defer func() {
				Expect(wasmMod.Close()).To(Succeed())
			}()
			factory := MustSucceed(wasm.NewFactory(wasmMod))
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[0],
				State:  s.Node("counter"),
				Module: mod,
			}))

			// First call - should return 1
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			result1 := *s.Node("counter").Output(0)
			Expect(result1.Len()).To(Equal(int64(1)))
			vals1 := telem.UnmarshalSeries[int64](result1)
			Expect(vals1[0]).To(Equal(int64(1)))

			// Second call - should return 2 (state persisted)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			result2 := *s.Node("counter").Output(0)
			Expect(result2.Len()).To(Equal(int64(1)))
			vals2 := telem.UnmarshalSeries[int64](result2)
			Expect(vals2[0]).To(Equal(int64(2)))

			// Third call - should return 3 (state persisted)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			result3 := *s.Node("counter").Output(0)
			Expect(result3.Len()).To(Equal(int64(1)))
			vals3 := telem.UnmarshalSeries[int64](result3)
			Expect(vals3[0]).To(Equal(int64(3)))
		})
	})
})
