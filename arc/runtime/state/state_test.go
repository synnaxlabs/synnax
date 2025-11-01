// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("State", func() {
	Describe("Channel Operations", func() {
		Describe("ReadChan", func() {
			It("Should read channel data after ingestion", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "test", Type: "test"}},
					Functions: []graph.Function{{Key: "test"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				s := state.New(state.Config{IR: ir})
				fr := telem.UnaryFrame[uint32](10, telem.NewSeriesV[float32](1, 2, 3))
				s.Ingest(fr)
				n := s.Node(ctx, "test")
				data, time, ok := n.ReadChan(10)
				Expect(ok).To(BeTrue())
				Expect(data.Series).To(HaveLen(1))
				Expect(data.Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1, 2, 3)))
				Expect(time.Series).To(BeEmpty())
			})

			It("Should read channel with index", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "test", Type: "test"}},
					Functions: []graph.Function{{Key: "test"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 5, Index: 6},
					},
					IR: ir,
				}
				s := state.New(cfg)
				fr := telem.Frame[uint32]{}
				fr = fr.Append(5, telem.NewSeriesV[int32](100, 200))
				fr = fr.Append(6, telem.NewSeriesSecondsTSV(10, 20))
				s.Ingest(fr)
				n := s.Node(ctx, "test")
				data, time, ok := n.ReadChan(5)
				Expect(ok).To(BeTrue())
				Expect(data.Series).To(HaveLen(1))
				Expect(data.Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](100, 200)))
				Expect(time.Series).To(HaveLen(1))
				Expect(time.Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)))
			})

			It("Should return false for non-existent channel", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "test", Type: "test"}},
					Functions: []graph.Function{{Key: "test"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				s := state.New(state.Config{IR: ir})
				n := s.Node(ctx, "test")
				_, _, ok := n.ReadChan(999)
				Expect(ok).To(BeFalse())
			})
		})

		Describe("WriteChan", func() {
			It("Should write channel data", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "writer", Type: "writer"}},
					Functions: []graph.Function{{Key: "writer"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, Index: 2},
					},
					IR: ir,
				}
				s := state.New(cfg)
				n := s.Node(ctx, "writer")
				dataToWrite := telem.NewSeriesV[float64](3.14, 2.71)
				timeToWrite := telem.NewSeriesSecondsTSV(100, 200)
				n.WriteChan(1, dataToWrite, timeToWrite)
				fr := telem.Frame[uint32]{}
				fr, changed := s.FlushWrites(fr)
				Expect(changed).To(BeTrue())
				Expect(len(fr.Get(1).Series)).To(Equal(1))
				Expect(len(fr.Get(2).Series)).To(Equal(1))
				Expect(fr.Get(1).Series[0]).To(telem.MatchSeries(dataToWrite))
				Expect(fr.Get(2).Series[0]).To(telem.MatchSeries(timeToWrite))
			})

			It("Should handle multiple channel writes", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "writer", Type: "writer"}},
					Functions: []graph.Function{{Key: "writer"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 10, Index: 11},
						{Key: 20, Index: 21},
					},
					IR: ir,
				}
				s := state.New(cfg)
				n := s.Node(ctx, "writer")
				n.WriteChan(10, telem.NewSeriesV[int32](42), telem.NewSeriesSecondsTSV(1))
				n.WriteChan(20, telem.NewSeriesV[int32](99), telem.NewSeriesSecondsTSV(2))
				fr, changed := s.FlushWrites(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(len(fr.Get(10).Series)).To(Equal(1))
				Expect(len(fr.Get(11).Series)).To(Equal(1))
				Expect(len(fr.Get(20).Series)).To(Equal(1))
				Expect(len(fr.Get(21).Series)).To(Equal(1))
			})
		})

		Describe("FlushWrites", func() {
			It("Should clear writes after flush", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "writer", Type: "writer"}},
					Functions: []graph.Function{{Key: "writer"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, Index: 2},
					},
					IR: ir,
				}
				s := state.New(cfg)
				n := s.Node(ctx, "writer")
				n.WriteChan(1, telem.NewSeriesV[float32](1.0), telem.NewSeriesSecondsTSV(1))
				fr1, changed := s.FlushWrites(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(len(fr1.Get(1).Series)).To(Equal(1))
				fr2, changed := s.FlushWrites(telem.Frame[uint32]{})
				Expect(changed).To(BeFalse())
				Expect(len(fr2.Get(1).Series)).To(Equal(0))
			})

			It("Should accumulate multiple writes before flush", func() {
				g := graph.Graph{
					Nodes:     []graph.Node{{Key: "writer", Type: "writer"}},
					Functions: []graph.Function{{Key: "writer"}},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue())
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, Index: 2},
						{Key: 3, Index: 4},
					},
					IR: ir,
				}
				s := state.New(cfg)
				n := s.Node(ctx, "writer")
				n.WriteChan(1, telem.NewSeriesV[float32](1.0), telem.NewSeriesSecondsTSV(1))
				n.WriteChan(3, telem.NewSeriesV[float32](2.0), telem.NewSeriesSecondsTSV(2))
				fr, changed := s.FlushWrites(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(len(fr.Get(1).Series)).To(Equal(1))
				Expect(len(fr.Get(3).Series)).To(Equal(1))
			})
		})
	})

	Describe("Input Alignment", func() {
		It("Should correctly order the inputs regardless of edge order", func() {
			g := graph.Graph{
				Nodes: graph.Nodes{
					{Key: "in1", Type: "in1"},
					{Key: "in2", Type: "in2"},
					{Key: "in3", Type: "in3"},
					{Key: "target", Type: "target"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "in2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: "in2"},
					},
					{
						Source: ir.Handle{Node: "in1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: "in1"},
					},
					{
						Source: ir.Handle{Node: "in3", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: "in3"},
					},
				},
				Functions: []ir.Function{
					{
						Key: "in1",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "in2",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "in3",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key: "target",
						Inputs: types.Params{
							Keys:   []string{"in1", "in2", "in3"},
							Values: []types.Type{types.I32(), types.F32(), types.U8()},
						},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			var (
				s      = state.New(state.Config{IR: ir})
				in1    = s.Node(ctx, "in1")
				in2    = s.Node(ctx, "in2")
				in3    = s.Node(ctx, "in3")
				target = s.Node(ctx, "target")
			)
			*in1.Output(0) = telem.NewSeriesV[int32](1)
			*in1.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			*in2.Output(0) = telem.NewSeriesV[float32](2)
			*in2.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			*in3.Output(0) = telem.NewSeriesV[uint8](3)
			*in3.OutputTime(0) = telem.NewSeriesSecondsTSV(3)
			target.RefreshInputs()
			target1In1 := target.Input(0)
			Expect(target1In1).To(telem.MatchSeriesDataV[int32](1))
			target1In2 := target.Input(1)
			Expect(target1In2).To(telem.MatchSeriesDataV[float32](2))
			target1In3 := target.Input(2)
			Expect(target1In3).To(telem.MatchSeriesDataV[uint8](3))
		})

		It("Should correctly align outputs of one node with inputs of another", func() {
			g := graph.Graph{
				Nodes: graph.Nodes{{Key: "first", Type: "first"}, {Key: "second", Type: "second"}},
				Functions: []graph.Function{
					{
						Key: "first",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "second",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.F32()},
						},
					},
				},
				Edges: []graph.Edge{{
					Source: graph.Handle{Node: "first", Param: ir.DefaultOutputParam},
					Target: graph.Handle{Node: "second", Param: ir.DefaultInputParam},
				}},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			cfg := state.Config{IR: ir}
			s := state.New(cfg)
			first := s.Node(ctx, "first")
			second := s.Node(ctx, "second")
			Expect(first.RefreshInputs()).To(BeTrue())
			Expect(second.RefreshInputs()).To(BeFalse())
			*first.Output(0) = telem.NewSeriesV[float32](1, 2, 3)
			*first.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			Expect(first.RefreshInputs()).To(BeTrue())
			Expect(second.RefreshInputs()).To(BeTrue())
			Expect(second.Input(0)).To(telem.MatchSeries(*first.Output(0)))
			Expect(second.InputTime(0)).To(telem.MatchSeries(*first.OutputTime(0)))
		})

		It("Should not trigger recalculation with empty output", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "src",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "dest",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "src", Type: "src"},
					{Key: "dest", Type: "dest"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "dest", Param: ir.DefaultInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			cfg := state.Config{IR: ir}
			s := state.New(cfg)
			src := s.Node(ctx, "src")
			dest := s.Node(ctx, "dest")
			*src.Output(0) = telem.NewSeriesV[int32]()
			*src.OutputTime(0) = telem.NewSeriesSecondsTSV()
			Expect(dest.RefreshInputs()).To(BeFalse())
		})

		It("Should track watermark to prevent reprocessing", func() {
			g := graph.Graph{
				Functions: ir.Functions{
					{
						Key: "producer",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
					},
					{
						Key: "consumer",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.F64()},
						},
					},
				},
				Nodes: graph.Nodes{
					{Key: "producer", Type: "producer"},
					{Key: "consumer", Type: "consumer"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "producer", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "consumer", Param: ir.DefaultInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			producer := s.Node(ctx, "producer")
			consumer := s.Node(ctx, "consumer")
			*producer.Output(0) = telem.NewSeriesV[float64](1.0)
			*producer.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(consumer.RefreshInputs()).To(BeTrue())
			Expect(consumer.RefreshInputs()).To(BeFalse())
		})

		It("Should handle multiple inputs to single node", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "a",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "b",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "target",
						Inputs: types.Params{
							Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
							Values: []types.Type{types.F32(), types.F32()},
						},
					},
				},
				Nodes: graph.Nodes{
					{Key: "a", Type: "a"},
					{Key: "b", Type: "b"},
					{Key: "target", Type: "target"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: ir.RHSInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			cfg := state.Config{IR: ir}
			s := state.New(cfg)
			nodeA := s.Node(ctx, "a")
			nodeB := s.Node(ctx, "b")
			target := s.Node(ctx, "target")
			*nodeA.Output(0) = telem.NewSeriesV[float32](1.0)
			*nodeA.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			Expect(target.RefreshInputs()).To(BeFalse())
			*nodeB.Output(0) = telem.NewSeriesV[float32](2.0)
			*nodeB.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](1.0)))
			Expect(target.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})

		It("Should select earliest timestamp as trigger", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "early",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "late",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "target",
						Inputs: types.Params{
							Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
							Values: []types.Type{types.I32(), types.I32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "early", Type: "early"},
					{Key: "late", Type: "late"},
					{Key: "target", Type: "target"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "early", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "late", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: ir.RHSInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			early := s.Node(ctx, "early")
			late := s.Node(ctx, "late")
			target := s.Node(ctx, "target")
			*early.Output(0) = telem.NewSeriesV[int32](10)
			*early.OutputTime(0) = telem.NewSeriesSecondsTSV(100)
			*late.Output(0) = telem.NewSeriesV[int32](20)
			*late.OutputTime(0) = telem.NewSeriesSecondsTSV(200)
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.InputTime(0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(100)))
		})

		It("Should accumulate multiple series before triggering", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "sink",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "sink", Type: "sink"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			source := s.Node(ctx, "source")
			sink := s.Node(ctx, "sink")
			*source.Output(0) = telem.NewSeriesV[int32](1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(sink.RefreshInputs()).To(BeTrue())
			*source.Output(0) = telem.NewSeriesV[int32](2)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
			Expect(sink.RefreshInputs()).To(BeTrue())
			Expect(sink.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](2)))
		})

		It("Should handle partial input updates", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "a",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "b",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "target",
						Inputs: types.Params{
							Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
							Values: []types.Type{types.F32(), types.F32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "a", Type: "a"},
					{Key: "b", Type: "b"},
					{Key: "target", Type: "target"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: ir.LHSInputParam},
					},
					{
						Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "target", Param: ir.RHSInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			cfg := state.Config{IR: ir}
			s := state.New(cfg)
			nodeA := s.Node(ctx, "a")
			nodeB := s.Node(ctx, "b")
			target := s.Node(ctx, "target")
			*nodeA.Output(0) = telem.NewSeriesV[float32](1.0)
			*nodeA.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			*nodeB.Output(0) = telem.NewSeriesV[float32](2.0)
			*nodeB.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(target.RefreshInputs()).To(BeTrue())
			*nodeA.Output(0) = telem.NewSeriesV[float32](3.0)
			*nodeA.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](3.0)))
			Expect(target.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})

		It("Should prune old series after watermark update", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "src",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key: "dst",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "src", Type: "src"},
					{Key: "dst", Type: "dst"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "dst", Param: ir.DefaultInputParam},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			cfg := state.Config{IR: ir}
			s := state.New(cfg)
			src := s.Node(ctx, "src")
			dst := s.Node(ctx, "dst")
			*src.Output(0) = telem.NewSeriesV[int64](10)
			*src.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			Expect(dst.RefreshInputs()).To(BeTrue())
			*src.Output(0) = telem.NewSeriesV[int64](20)
			*src.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(dst.RefreshInputs()).To(BeTrue())
			*src.Output(0) = telem.NewSeriesV[int64](30)
			*src.OutputTime(0) = telem.NewSeriesSecondsTSV(15)
			Expect(dst.RefreshInputs()).To(BeTrue())
			Expect(dst.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int64](30)))
		})

		Describe("Watermark Regression Tests", func() {
			It("Should update all input watermarks on trigger", func() {
				g := graph.Graph{
					Functions: []graph.Function{
						{
							Key: "lhs",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F64()},
							},
						},
						{
							Key: "rhs",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F64()},
							},
						},
						{
							Key: "op",
							Inputs: types.Params{
								Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
								Values: []types.Type{types.F64(), types.F64()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "lhs", Type: "lhs"},
						{Key: "rhs", Type: "rhs"},
						{Key: "op", Type: "op"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "lhs", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
						},
						{
							Source: ir.Handle{Node: "rhs", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
						},
					},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				cfg := state.Config{IR: ir}
				s := state.New(cfg)
				lhs := s.Node(ctx, "lhs")
				rhs := s.Node(ctx, "rhs")
				op := s.Node(ctx, "op")
				*lhs.Output(0) = telem.NewSeriesV[float64](1.5)
				*lhs.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
				*rhs.Output(0) = telem.NewSeriesV[float64](2.5)
				*rhs.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
				Expect(op.RefreshInputs()).To(BeTrue())
				Expect(op.RefreshInputs()).To(BeFalse())
			})

			It("Should not trigger recalculation when non-trigger input unchanged", func() {
				g := graph.Graph{
					Functions: []graph.Function{
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
							Key: "compute",
							Inputs: types.Params{
								Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
								Values: []types.Type{types.I32(), types.I32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "a", Type: "a"},
						{Key: "b", Type: "b"},
						{Key: "compute", Type: "compute"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "compute", Param: ir.LHSInputParam},
						},
						{
							Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "compute", Param: ir.RHSInputParam},
						},
					},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				cfg := state.Config{IR: ir}
				s := state.New(cfg)
				nodeA := s.Node(ctx, "a")
				nodeB := s.Node(ctx, "b")
				compute := s.Node(ctx, "compute")
				*nodeA.Output(0) = telem.NewSeriesV[int32](100)
				*nodeA.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
				*nodeB.Output(0) = telem.NewSeriesV[int32](50)
				*nodeB.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
				Expect(compute.RefreshInputs()).To(BeTrue())
				Expect(compute.RefreshInputs()).To(BeFalse())
				Expect(compute.RefreshInputs()).To(BeFalse())
			})

			It("Should correctly track watermarks with staggered timestamps", func() {
				g := graph.Graph{
					Functions: []graph.Function{
						{
							Key: "early",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "late",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.F32()},
							},
						},
						{
							Key: "target",
							Inputs: types.Params{
								Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
								Values: []types.Type{types.F32(), types.F32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "early", Type: "early"},
						{Key: "late", Type: "late"},
						{Key: "target", Type: "target"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "early", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "target", Param: ir.LHSInputParam},
						},
						{
							Source: ir.Handle{Node: "late", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "target", Param: ir.RHSInputParam},
						},
					},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				cfg := state.Config{IR: ir}
				s := state.New(cfg)
				early := s.Node(ctx, "early")
				late := s.Node(ctx, "late")
				target := s.Node(ctx, "target")
				*early.Output(0) = telem.NewSeriesV[float32](1.0)
				*early.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
				*late.Output(0) = telem.NewSeriesV[float32](2.0)
				*late.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
				Expect(target.RefreshInputs()).To(BeTrue())
				Expect(target.RefreshInputs()).To(BeFalse())
				*early.Output(0) = telem.NewSeriesV[float32](3.0)
				*early.OutputTime(0) = telem.NewSeriesSecondsTSV(30)
				*late.Output(0) = telem.NewSeriesV[float32](4.0)
				*late.OutputTime(0) = telem.NewSeriesSecondsTSV(40)
				Expect(target.RefreshInputs()).To(BeTrue())
				Expect(target.RefreshInputs()).To(BeFalse())
			})

			It("Should prevent non-trigger input from causing spurious triggers", func() {
				g := graph.Graph{
					Functions: []graph.Function{
						{
							Key: "x",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.U32()},
							},
						},
						{
							Key: "y",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.U32()},
							},
						},
						{
							Key: "processor",
							Inputs: types.Params{
								Keys:   []string{ir.LHSInputParam, ir.RHSInputParam},
								Values: []types.Type{types.U32(), types.U32()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "x", Type: "x"},
						{Key: "y", Type: "y"},
						{Key: "processor", Type: "processor"},
					},
					Edges: []ir.Edge{
						{
							Source: ir.Handle{Node: "x", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "processor", Param: ir.LHSInputParam},
						},
						{
							Source: ir.Handle{Node: "y", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "processor", Param: ir.RHSInputParam},
						},
					},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				cfg := state.Config{IR: ir}
				s := state.New(cfg)
				nodeX := s.Node(ctx, "x")
				nodeY := s.Node(ctx, "y")
				processor := s.Node(ctx, "processor")
				*nodeX.Output(0) = telem.NewSeriesV[uint32](10)
				*nodeX.OutputTime(0) = telem.NewSeriesSecondsTSV(100)
				*nodeY.Output(0) = telem.NewSeriesV[uint32](20)
				*nodeY.OutputTime(0) = telem.NewSeriesSecondsTSV(100)
				firstRefresh := processor.RefreshInputs()
				Expect(firstRefresh).To(BeTrue())
				secondRefresh := processor.RefreshInputs()
				Expect(secondRefresh).To(BeFalse())
				thirdRefresh := processor.RefreshInputs()
				Expect(thirdRefresh).To(BeFalse())
			})

			It("Should handle three inputs with same timestamp", func() {
				g := graph.Graph{
					Functions: []graph.Function{
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
							Key: "c",
							Outputs: types.Params{
								Keys:   []string{ir.DefaultOutputParam},
								Values: []types.Type{types.I64()},
							},
						},
						{
							Key: "combiner",
							Inputs: types.Params{
								Keys:   []string{"in0", "in1", "in2"},
								Values: []types.Type{types.I64(), types.I64(), types.I64()},
							},
						},
					},
					Nodes: []graph.Node{
						{Key: "a", Type: "a"},
						{Key: "b", Type: "b"},
						{Key: "c", Type: "c"},
						{Key: "combiner", Type: "combiner"},
					},
					Edges: []graph.Edge{
						{
							Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "combiner", Param: "in0"},
						},
						{
							Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "combiner", Param: "in1"},
						},
						{
							Source: ir.Handle{Node: "c", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "combiner", Param: "in2"},
						},
					},
				}
				ir, diagnostics := graph.Analyze(ctx, g, nil)
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				cfg := state.Config{IR: ir}
				s := state.New(cfg)
				nodeA := s.Node(ctx, "a")
				nodeB := s.Node(ctx, "b")
				nodeC := s.Node(ctx, "c")
				combiner := s.Node(ctx, "combiner")
				*nodeA.Output(0) = telem.NewSeriesV[int64](1)
				*nodeA.OutputTime(0) = telem.NewSeriesSecondsTSV(50)
				*nodeB.Output(0) = telem.NewSeriesV[int64](2)
				*nodeB.OutputTime(0) = telem.NewSeriesSecondsTSV(50)
				*nodeC.Output(0) = telem.NewSeriesV[int64](3)
				*nodeC.OutputTime(0) = telem.NewSeriesSecondsTSV(50)
				Expect(combiner.RefreshInputs()).To(BeTrue())
				Expect(combiner.RefreshInputs()).To(BeFalse())
			})
		})

	})

	Describe("Optional Input Parameters", func() {
		It("Should use default value for unconnected optional input", func() {
			g := graph.Graph{
				Functions: []graph.Function{
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
							Keys:   []string{"data", "multiplier"},
							Values: []types.Type{types.F32(), types.F32()},
						},
						InputDefaults: map[string]any{
							"multiplier": float32(2.0),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "processor", Type: "processor"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "processor", Param: "data"},
					},
					// Note: "multiplier" input is not connected, should use default
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			source := s.Node(ctx, "source")
			processor := s.Node(ctx, "processor")
			*source.Output(0) = telem.NewSeriesV[float32](5.0)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(processor.RefreshInputs()).To(BeTrue())
			Expect(processor.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](5.0)))
			Expect(processor.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})

		It("Should override default value when input is connected", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "data_source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "multiplier_source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "processor",
						Inputs: types.Params{
							Keys:   []string{"value", "factor"},
							Values: []types.Type{types.I32(), types.I32()},
						},
						InputDefaults: map[string]any{
							"factor": int32(10),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "data_source", Type: "data_source"},
					{Key: "multiplier_source", Type: "multiplier_source"},
					{Key: "processor", Type: "processor"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "data_source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "processor", Param: "value"},
					},
					{
						Source: ir.Handle{Node: "multiplier_source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "processor", Param: "factor"},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			dataSource := s.Node(ctx, "data_source")
			multiplierSource := s.Node(ctx, "multiplier_source")
			processor := s.Node(ctx, "processor")
			*dataSource.Output(0) = telem.NewSeriesV[int32](100)
			*dataSource.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			*multiplierSource.Output(0) = telem.NewSeriesV[int32](3)
			*multiplierSource.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(processor.RefreshInputs()).To(BeTrue())
			Expect(processor.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](100)))
			Expect(processor.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[int32](3)))
		})

		It("Should handle multiple optional parameters with defaults", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
					},
					{
						Key: "calculator",
						Inputs: types.Params{
							Keys:   []string{"x", "offset", "scale"},
							Values: []types.Type{types.F64(), types.F64(), types.F64()},
						},
						InputDefaults: map[string]any{
							"offset": float64(5.0),
							"scale":  float64(2.5),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "calculator", Type: "calculator"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "calculator", Param: "x"},
					},
					// "offset" and "scale" are unconnected, should use defaults
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			input := s.Node(ctx, "input")
			calculator := s.Node(ctx, "calculator")
			*input.Output(0) = telem.NewSeriesV[float64](10.0)
			*input.OutputTime(0) = telem.NewSeriesSecondsTSV(15)
			Expect(calculator.RefreshInputs()).To(BeTrue())
			Expect(calculator.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[float64](10.0)))
			Expect(calculator.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[float64](5.0)))
			Expect(calculator.Input(2)).To(telem.MatchSeries(telem.NewSeriesV[float64](2.5)))
		})

		It("Should handle mix of connected and unconnected optional inputs", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "src1",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key: "src2",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key: "combiner",
						Inputs: types.Params{
							Keys:   []string{"a", "b", "c"},
							Values: []types.Type{types.I64(), types.I64(), types.I64()},
						},
						InputDefaults: map[string]any{
							"b": int64(20),
							"c": int64(30),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "src1", Type: "src1"},
					{Key: "src2", Type: "src2"},
					{Key: "combiner", Type: "combiner"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "src1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combiner", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "src2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combiner", Param: "c"},
					},
					// "b" is unconnected, should use default value 20
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			src1 := s.Node(ctx, "src1")
			src2 := s.Node(ctx, "src2")
			combiner := s.Node(ctx, "combiner")
			*src1.Output(0) = telem.NewSeriesV[int64](100)
			*src1.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			*src2.Output(0) = telem.NewSeriesV[int64](300)
			*src2.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			Expect(combiner.RefreshInputs()).To(BeTrue())
			Expect(combiner.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int64](100)))
			Expect(combiner.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[int64](20)))
			Expect(combiner.Input(2)).To(telem.MatchSeries(telem.NewSeriesV[int64](300)))
		})

		It("Should allow default values with different types", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "data",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "processor",
						Inputs: types.Params{
							Keys:   []string{"data", "enabled", "threshold"},
							Values: []types.Type{types.F32(), types.U8(), types.I32()},
						},
						InputDefaults: map[string]any{
							"enabled":   uint8(1),
							"threshold": int32(50),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "data", Type: "data"},
					{Key: "processor", Type: "processor"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "data", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "processor", Param: "data"},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			data := s.Node(ctx, "data")
			processor := s.Node(ctx, "processor")
			*data.Output(0) = telem.NewSeriesV[float32](42.5)
			*data.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(processor.RefreshInputs()).To(BeTrue())
			Expect(processor.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](42.5)))
			Expect(processor.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
			Expect(processor.Input(2)).To(telem.MatchSeries(telem.NewSeriesV[int32](50)))
		})

		It("Should persist default values across multiple executions", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U32()},
						},
					},
					{
						Key: "processor",
						Inputs: types.Params{
							Keys:   []string{"value", "offset"},
							Values: []types.Type{types.U32(), types.U32()},
						},
						InputDefaults: map[string]any{
							"offset": uint32(100),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "source", Type: "source"},
					{Key: "processor", Type: "processor"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "processor", Param: "value"},
					},
					// "offset" is unconnected, will use default
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			source := s.Node(ctx, "source")
			processor := s.Node(ctx, "processor")
			// First execution with data at t=5
			*source.Output(0) = telem.NewSeriesV[uint32](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			Expect(processor.RefreshInputs()).To(BeTrue())
			Expect(processor.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[uint32](10)))
			Expect(processor.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[uint32](100)))
			// Second execution with new data at t=10 - default should persist
			*source.Output(0) = telem.NewSeriesV[uint32](20)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(processor.RefreshInputs()).To(BeTrue())
			Expect(processor.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[uint32](20)))
			Expect(processor.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[uint32](100)))
			// Third execution with new data at t=15 - default should still persist
			*source.Output(0) = telem.NewSeriesV[uint32](30)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(15)
			Expect(processor.RefreshInputs()).To(BeTrue())
			Expect(processor.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[uint32](30)))
			Expect(processor.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[uint32](100)))
			// No new data - should not trigger
			Expect(processor.RefreshInputs()).To(BeFalse())
		})

		It("Should handle optional input with zero value as default", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key: "adder",
						Inputs: types.Params{
							Keys:   []string{"base", "offset"},
							Values: []types.Type{types.I32(), types.I32()},
						},
						InputDefaults: map[string]any{
							"offset": int32(0),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "adder", Type: "adder"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "adder", Param: "base"},
					},
				},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			input := s.Node(ctx, "input")
			adder := s.Node(ctx, "adder")
			*input.Output(0) = telem.NewSeriesV[int32](50)
			*input.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(adder.RefreshInputs()).To(BeTrue())
			Expect(adder.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](50)))
			Expect(adder.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[int32](0)))
		})

		It("Should handle node with only optional inputs", func() {
			g := graph.Graph{
				Functions: []graph.Function{
					{
						Key: "generator",
						Inputs: types.Params{
							Keys:   []string{"seed", "multiplier"},
							Values: []types.Type{types.I64(), types.I64()},
						},
						InputDefaults: map[string]any{
							"seed":       int64(42),
							"multiplier": int64(7),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "generator", Type: "generator"},
				},
				Edges: []graph.Edge{},
			}
			ir, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: ir})
			generator := s.Node(ctx, "generator")
			Expect(generator.RefreshInputs()).To(BeTrue())
			Expect(generator.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int64](42)))
			Expect(generator.Input(1)).To(telem.MatchSeries(telem.NewSeriesV[int64](7)))
		})
	})
})
