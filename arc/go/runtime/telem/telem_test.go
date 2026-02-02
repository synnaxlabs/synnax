// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	rtelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Telem", func() {
	Describe("Telem Factory", func() {
		var (
			factory rnode.Factory
			s       *state.State
		)
		BeforeEach(func() {
			factory = rtelem.NewTelemFactory()
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "test", Type: "on"}},
				Functions: []graph.Function{{Key: "on"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, rtelem.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})

		Describe("Source Creation", func() {
			It("Should create source node for on type", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(42)}},
					},
					State: s.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
			It("Should parse channel from config", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(123)}},
					},
					State: s.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
			It("Should coerce channel to uint32", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(99)}},
					},
					State: s.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
		})

		Describe("Sink Creation", func() {
			It("Should create sink node for write type", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: s.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
		})

		Describe("Error Handling", func() {
			It("Should return query.ErrNotFound for unknown node type", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "unknown",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(1)}},
					},
					State: s.Node("test"),
				}
				node, err := factory.Create(ctx, cfg)
				Expect(err).To(Equal(query.ErrNotFound))
				Expect(node).To(BeNil())
			})
			It("Should return error for invalid config", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "invalid", Type: types.String(), Value: "field"}},
					},
					State: s.Node("test"),
				}
				_, err := factory.Create(ctx, cfg)
				Expect(err).To(HaveOccurred())
			})
			It("Should return error for missing channel", func() {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{},
					},
					State: s.Node("test"),
				}
				_, err := factory.Create(ctx, cfg)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Source Node", func() {
		var (
			s       *state.State
			factory rnode.Factory
		)
		BeforeEach(func() {
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "source", Type: "on"}},
				Functions: []graph.Function{{
					Key: "on",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.F32()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, rtelem.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{
				IR: analyzed,
				ChannelDigests: []state.ChannelDigest{
					{Key: 10, DataType: telem.Float32T, Index: 11},
					{Key: 20, DataType: telem.Int32T, Index: 0},
				},
			})
			factory = rtelem.NewTelemFactory()
		})

		Describe("Data Reading", func() {
			It("Should read channel data after ingestion", func() {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: s.Node("source"),
				}))
				fr := telem.Frame[uint32]{}
				fr = fr.Append(10, telem.NewSeriesV[float32](1.5, 2.5, 3.5))
				fr = fr.Append(11, telem.NewSeriesSecondsTSV(100, 101, 102))
				s.Ingest(fr)
				var outputChanged bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { outputChanged = true }})
				Expect(outputChanged).To(BeTrue())
				Expect(*s.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](1.5, 2.5, 3.5)))
				Expect(*s.Node("source").OutputTime(0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(100, 101, 102)))
			})

			It("Should handle channel without index", func() {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
					},
					State: s.Node("source"),
				}))
				fr := telem.UnaryFrame[uint32](20, telem.NewSeriesV[int32](100, 200))
				s.Ingest(fr)
				var outputChanged bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { outputChanged = true }})
				Expect(outputChanged).To(BeTrue())
				Expect(*s.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](100, 200)))
				Expect(s.Node("source").OutputTime(0).DataType).To(Equal(telem.TimeStampT))
			})

			It("Should not trigger on empty channel", func() {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(999)}},
					},
					State: s.Node("source"),
				}))
				var outputChanged bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { outputChanged = true }})
				Expect(outputChanged).To(BeFalse())
			})

			It("Should handle multiple series in MultiSeries", func() {
				nodeState := s.Node("source")
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: nodeState,
				}))
				fr1 := telem.Frame[uint32]{}
				d1 := telem.NewSeriesV[float32](1.0)
				d1.Alignment = telem.NewAlignment(1, 0)
				t1 := telem.NewSeriesSecondsTSV(10)
				t1.Alignment = telem.NewAlignment(1, 0)
				fr1 = fr1.Append(10, d1)
				fr1 = fr1.Append(11, t1)
				s.Ingest(fr1)

				fr2 := telem.Frame[uint32]{}
				d2 := telem.NewSeriesV[float32](1.0)
				d2.Alignment = telem.NewAlignment(1, 1)
				t2 := telem.NewSeriesSecondsTSV(10)
				t2.Alignment = telem.NewAlignment(1, 1)
				fr2 = fr2.Append(10, d2)
				fr2 = fr2.Append(11, t2)
				s.Ingest(fr2)

				outputCount := 0

				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { outputCount++ }})
				Expect(outputCount).To(Equal(1))
				o := nodeState.Output(0)
				Expect(*o).To(telem.MatchSeries(d1))
				ot := nodeState.OutputTime(0)
				Expect(*ot).To(telem.MatchSeries(t1))

				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { outputCount++ }})
				Expect(outputCount).To(Equal(2))
				o = nodeState.Output(0)
				Expect(*o).To(telem.MatchSeries(d2))
				ot = nodeState.OutputTime(0)
				Expect(*ot).To(telem.MatchSeries(t2))
			})
		})

		Describe("Alignment Validation", func() {
			It("Should skip data when index series count mismatch", func() {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: s.Node("source"),
				}))
				fr1 := telem.Frame[uint32]{}
				fr1 = fr1.Append(10, telem.NewSeriesV[float32](1.0))
				fr1 = fr1.Append(11, telem.NewSeriesSecondsTSV(10))
				s.Ingest(fr1)
				fr2 := telem.Frame[uint32]{}
				fr2 = fr2.Append(10, telem.NewSeriesV[float32](2.0))
				s.Ingest(fr2)
				callCount := 0
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { callCount++ }})
				Expect(callCount).To(Equal(1))
			})

			It("Should skip data when alignment mismatch", func() {
				g2 := graph.Graph{
					Nodes: []graph.Node{{Key: "misaligned", Type: "on"}},
					Functions: []graph.Function{{
						Key:     "on",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
					}},
				}
				analyzed2, diagnostics2 := graph.Analyze(ctx, g2, rtelem.SymbolResolver)
				Expect(diagnostics2.Ok()).To(BeTrue())
				cfg := state.Config{
					IR: analyzed2,
					ChannelDigests: []state.ChannelDigest{
						{Key: 30, DataType: telem.Float64T, Index: 31},
					},
				}
				s2 := state.New(cfg)
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(30)}},
					},
					State: s2.Node("misaligned"),
				}))
				dataSeries := telem.NewSeriesV[float64](1.0, 2.0)
				dataSeries.Alignment = 100
				timeSeries := telem.NewSeriesSecondsTSV(10, 20)
				timeSeries.Alignment = 200
				fr := telem.Frame[uint32]{}
				fr = fr.Append(30, dataSeries)
				fr = fr.Append(31, timeSeries)
				s2.Ingest(fr)
				outputCount := 0
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) { outputCount++ }})
				Expect(outputCount).To(Equal(0))
			})
		})
	})

	Describe("Sink Node", func() {
		var (
			s       *state.State
			factory rnode.Factory
		)
		BeforeEach(func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "upstream", Type: "producer"},
					{Key: "sink", Type: "write"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key:     "producer",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F32()}},
					},
					{
						Key:    "write",
						Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.F32()}},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, rtelem.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{
				IR: analyzed,
				ChannelDigests: []state.ChannelDigest{
					{Key: 100, DataType: telem.Float32T, Index: 101},
				},
			})
			factory = rtelem.NewTelemFactory()
		})
		Describe("Data Writing", func() {
			It("Should write channel data when input available", func() {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: s.Node("sink"),
				}))
				upstream := s.Node("upstream")
				*upstream.Output(0) = telem.NewSeriesV[float32](7.7, 8.8)
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(500, 501)
				Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				fr, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(100).Series).To(HaveLen(1))
				Expect(fr.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](7.7, 8.8)))
				Expect(fr.Get(101).Series).To(HaveLen(1))
				Expect(fr.Get(101).Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(500, 501)))
			})
			It("Should respect RefreshInputs guard", func() {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: s.Node("sink"),
				}))
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				fr, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeFalse())
				Expect(fr.Get(100).Series).To(BeEmpty())
			})
			It("Should not write when input is empty", func() {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: s.Node("sink"),
				}))
				upstream := s.Node("upstream")
				*upstream.Output(0) = telem.NewSeriesV[float32]()
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV()
				Expect(s.Node("sink").RefreshInputs()).To(BeFalse())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				fr, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeFalse())
				Expect(fr.Get(100).Series).To(BeEmpty())
			})
		})
		Describe("Multiple Writes", func() {
			It("Should handle sequential writes", func() {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: s.Node("sink"),
				}))
				upstream := s.Node("upstream")
				*upstream.Output(0) = telem.NewSeriesV[float32](1.0)
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
				Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				fr1, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr1.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1.0)))
				*upstream.Output(0) = telem.NewSeriesV[float32](2.0)
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
				Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				fr2, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr2.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
			})
		})
	})

	Describe("Integration", func() {
		Describe("Source to Sink Flow", func() {
			It("Should flow data from source through sink", func() {
				g := graph.Graph{
					Nodes: []graph.Node{
						{Key: "read", Type: "on"},
						{Key: "write", Type: "write"},
					},
					Edges: []graph.Edge{
						{
							Source: ir.Handle{Node: "read", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "write", Param: ir.DefaultInputParam},
						},
					},
					Functions: []graph.Function{
						{
							Key:     "on",
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						},
						{
							Key:    "write",
							Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.I32()}},
						},
					},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, rtelem.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				s := state.New(state.Config{
					IR: analyzed,
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, DataType: telem.Int32T, Index: 2},
						{Key: 3, DataType: telem.Int32T, Index: 4},
					},
				})
				factory := rtelem.NewTelemFactory()
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(1)}},
					},
					State: s.Node("read"),
				}))
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(3)}},
					},
					State: s.Node("write"),
				}))
				ingestFr := telem.Frame[uint32]{}
				ingestFr = ingestFr.Append(1, telem.NewSeriesV[int32](42, 99))
				ingestFr = ingestFr.Append(2, telem.NewSeriesSecondsTSV(10, 20))
				s.Ingest(ingestFr)
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				Expect(s.Node("write").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				outputFr, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(outputFr.Get(3).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](42, 99)))
				Expect(outputFr.Get(4).Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)))
			})
		})
		Describe("Multiple Channels", func() {
			It("Should handle multiple independent source-sink pairs", func() {
				g := graph.Graph{
					Nodes: []graph.Node{
						{Key: "read1", Type: "on"},
						{Key: "read2", Type: "on2"},
						{Key: "write1", Type: "write"},
						{Key: "write2", Type: "write2"},
					},
					Edges: []graph.Edge{
						{Source: ir.Handle{Node: "read1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "write1", Param: ir.DefaultInputParam}},
						{Source: ir.Handle{Node: "read2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "write2", Param: ir.DefaultInputParam}},
					},
					Functions: []graph.Function{
						{Key: "on", Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F32()}}},
						{Key: "on2", Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F64()}}},
						{Key: "write", Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.F32()}}},
						{Key: "write2", Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.F64()}}},
					},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, rtelem.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				s := state.New(state.Config{
					IR: analyzed,
					ChannelDigests: []state.ChannelDigest{
						{Key: 10, DataType: telem.Float32T, Index: 11},
						{Key: 20, DataType: telem.Float64T, Index: 21},
						{Key: 30, DataType: telem.Float32T, Index: 31},
						{Key: 40, DataType: telem.Float64T, Index: 41},
					},
				})
				factory := rtelem.NewTelemFactory()
				source1, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "on", Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}}},
					State: s.Node("read1"),
				})
				source2, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "on", Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}}},
					State: s.Node("read2"),
				})
				sink1, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "write", Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(30)}}},
					State: s.Node("write1"),
				})
				sink2, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "write", Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(40)}}},
					State: s.Node("write2"),
				})
				fr := telem.Frame[uint32]{}
				fr = fr.Append(10, telem.NewSeriesV[float32](1.1, 2.2))
				fr = fr.Append(11, telem.NewSeriesSecondsTSV(100, 200))
				fr = fr.Append(20, telem.NewSeriesV[float64](3.3, 4.4))
				fr = fr.Append(21, telem.NewSeriesSecondsTSV(100, 200))
				s.Ingest(fr)
				source1.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				source2.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				Expect(s.Node("write1").RefreshInputs()).To(BeTrue())
				Expect(s.Node("write2").RefreshInputs()).To(BeTrue())
				sink1.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				sink2.Next(rnode.Context{Context: ctx, MarkChanged: func(string) {}})
				outputFr, changed := s.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(outputFr.Get(30).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1.1, 2.2)))
				Expect(outputFr.Get(40).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float64](3.3, 4.4)))
			})
		})
	})
})
