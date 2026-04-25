// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel", func() {
	Describe("WASM Bindings", func() {
		var (
			rt *testutil.Runtime
			cs *channel.ProgramState
			ss *strings.ProgramState
		)

		BeforeEach(func(ctx SpecContext) {
			rt = testutil.NewRuntime(ctx)
			cs = channel.NewProgramState([]channel.Digest{
				{Key: 1, DataType: telem.Float64T},
				{Key: 2, DataType: telem.Int32T},
				{Key: 3, DataType: telem.StringT},
			})
			ss = strings.NewProgramState()
			_, err := channel.NewModule(ctx, cs, ss, rt.Underlying())
			Expect(err).To(Succeed())
			rt.Passthrough(ctx, "channel")
		})

		AfterEach(func(ctx SpecContext) {
			Expect(rt.Close(ctx)).To(Succeed())
		})

		Describe("i32 types", func() {
			It("Should write and read back u8 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channel", "write_u8", testutil.U32(2), testutil.U32(42))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_u8", testutil.U32(2))
				Expect(testutil.AsU32(result[0])).To(Equal(uint32(42)))
			})

			It("Should write and read back i32 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channel", "write_i32", testutil.U32(2), testutil.U32(100))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_i32", testutil.U32(2))
				Expect(testutil.AsU32(result[0])).To(Equal(uint32(100)))
			})
		})

		Describe("i64 types", func() {
			It("Should write and read back u64 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channel", "write_u64", testutil.U32(1), testutil.U64(12345))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_u64", testutil.U32(1))
				Expect(testutil.AsU64(result[0])).To(Equal(uint64(12345)))
			})

			It("Should write and read back i64 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channel", "write_i64", testutil.U32(1), testutil.U64(99999))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_i64", testutil.U32(1))
				Expect(testutil.AsU64(result[0])).To(Equal(uint64(99999)))
			})
		})

		Describe("float types", func() {
			It("Should write and read back f32 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channel", "write_f32", testutil.U32(1), testutil.F32(3.14))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_f32", testutil.U32(1))
				Expect(testutil.AsF32(result[0])).To(BeNumerically("~", 3.14, 0.001))
			})

			It("Should write and read back f64 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channel", "write_f64", testutil.U32(1), testutil.F64(2.718281828))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_f64", testutil.U32(1))
				Expect(testutil.AsF64(result[0])).To(BeNumerically("~", 2.718281828, 0.0001))
			})
		})

		Describe("string type", func() {
			It("Should write and read back string values via handles", func(ctx SpecContext) {
				h := ss.Create("hello world")
				rt.CallVoid(ctx, "channel", "write_str", testutil.U32(3), testutil.U32(h))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channel", "read_str", testutil.U32(3))
				rh := testutil.AsU32(result[0])
				Expect(rh).ToNot(BeZero())
				Expect(MustBeOk(ss.Get(rh))).To(Equal("hello world"))
			})
		})

		Describe("read with no data", func() {
			It("Should return 0 when no data has been ingested", func(ctx SpecContext) {
				result := rt.Call(ctx, "channel", "read_f64", testutil.U32(1))
				Expect(testutil.AsF64(result[0])).To(Equal(float64(0)))
			})
		})
	})

	Describe("Node Factory", func() {
		var (
			factory rnode.Factory
			rtState *rnode.ProgramState
		)
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(channel.NewModule(ctx, nil, nil, nil))
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "test", Type: "on"}},
				Functions: []graph.Function{{Key: "on"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, channel.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			rtState = rnode.New(analyzed)
		})

		Describe("Source Creation", func() {
			It("Should create source node for on type", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(42)}},
					},
					State: rtState.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
			It("Should parse channel from config", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(123)}},
					},
					State: rtState.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
			It("Should coerce channel to uint32", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(99)}},
					},
					State: rtState.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
		})

		Describe("Sink Creation", func() {
			It("Should create sink node for write type", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: rtState.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
		})

		Describe("Error Handling", func() {
			It("Should return query.ErrNotFound for unknown node type", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "unknown",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(1)}},
					},
					State: rtState.Node("test"),
				}
				node, err := factory.Create(ctx, cfg)
				Expect(err).To(Equal(query.ErrNotFound))
				Expect(node).To(BeNil())
			})
			It("Should return error for invalid config", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "invalid", Type: types.String(), Value: "field"}},
					},
					State: rtState.Node("test"),
				}
				_, err := factory.Create(ctx, cfg)
				Expect(err).To(HaveOccurred())
			})
			It("Should return error for missing channel", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{},
					},
					State: rtState.Node("test"),
				}
				_, err := factory.Create(ctx, cfg)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Source Node", func() {
		var (
			progState    *rnode.ProgramState
			channelState *channel.ProgramState
			factory      rnode.Factory
		)
		BeforeEach(func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "source", Type: "on"}},
				Functions: []graph.Function{{
					Key: "on",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.F32()},
					},
				}},
			}
			inter, diagnostics := graph.Analyze(ctx, g, channel.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			channelState = channel.NewProgramState([]channel.Digest{
				{Key: 10, DataType: telem.Float32T, Index: 11},
				{Key: 20, DataType: telem.Int32T, Index: 0},
			})
			progState = rnode.New(inter)
			factory = MustSucceed(channel.NewModule(ctx, channelState, nil, nil))
		})

		Describe("Data Reading", func() {
			It("Should read channel data after ingestion", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: progState.Node("source"),
				}))
				fr := telem.Frame[uint32]{}
				fr = fr.Append(10, telem.NewSeriesV[float32](1.5, 2.5, 3.5))
				fr = fr.Append(11, telem.NewSeriesSecondsTSV(100, 101, 102))
				channelState.Ingest(fr)
				var outputChanged bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { outputChanged = true }})
				Expect(outputChanged).To(BeTrue())
				Expect(*progState.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](1.5, 2.5, 3.5)))
				Expect(*progState.Node("source").OutputTime(0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(100, 101, 102)))
			})

			It("Should handle channel without index", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
					},
					State: progState.Node("source"),
				}))
				fr := telem.UnaryFrame[uint32](20, telem.NewSeriesV[int32](100, 200))
				channelState.Ingest(fr)
				var outputChanged bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { outputChanged = true }})
				Expect(outputChanged).To(BeTrue())
				Expect(*progState.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](100, 200)))
				Expect(progState.Node("source").OutputTime(0).DataType).To(Equal(telem.TimeStampT))
			})

			It("Should not trigger on empty channel", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(999)}},
					},
					State: progState.Node("source"),
				}))
				var outputChanged bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { outputChanged = true }})
				Expect(outputChanged).To(BeFalse())
			})

			It("Should generate a time series matching the current series length for virtual channels with accumulated reads", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
					},
					State: progState.Node("source"),
				}))
				nodeState := progState.Node("source")
				d1 := telem.NewSeriesV[int32](10, 20, 30)
				d1.Alignment = telem.NewAlignment(1, 0)
				channelState.Ingest(telem.UnaryFrame[uint32](20, d1))

				var triggered bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { triggered = true }})
				Expect(triggered).To(BeTrue())
				Expect(nodeState.Output(0).Len()).To(Equal(int64(3)))
				Expect(nodeState.OutputTime(0).Len()).To(Equal(int64(3)))

				channelState.ClearReads()
				triggered = false

				d2 := telem.NewSeriesV[int32](40, 50)
				d2.Alignment = telem.NewAlignment(1, 3)
				channelState.Ingest(telem.UnaryFrame[uint32](20, d2))

				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { triggered = true }})
				Expect(triggered).To(BeTrue())
				Expect(nodeState.Output(0).Len()).To(Equal(int64(2)))
				Expect(nodeState.OutputTime(0).Len()).To(Equal(int64(2)),
					"time series length must match data series length, not total accumulated read buffer length")
			})

			It("Should generate monotonically increasing timestamps across calls for virtual channels", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
					},
					State: progState.Node("source"),
				}))
				nodeState := progState.Node("source")
				var prevTS telem.TimeStamp
				for i := range 10 {
					d := telem.NewSeriesV[int32](int32(i))
					d.Alignment = telem.NewAlignment(1, uint32(i))
					channelState.Ingest(telem.UnaryFrame[uint32](20, d))

					var triggered bool
					source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { triggered = true }})
					Expect(triggered).To(BeTrue())
					ts := telem.ValueAt[telem.TimeStamp](*nodeState.OutputTime(0), 0)
					Expect(ts).To(BeNumerically(">", prevTS),
						"timestamp must strictly increase across consecutive source outputs")
					prevTS = ts
					channelState.ClearReads()
				}
			})

			It("Should handle multiple series in MultiSeries", func(ctx SpecContext) {
				nodeState := progState.Node("source")
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
				channelState.Ingest(fr1)

				fr2 := telem.Frame[uint32]{}
				d2 := telem.NewSeriesV[float32](1.0)
				d2.Alignment = telem.NewAlignment(1, 1)
				t2 := telem.NewSeriesSecondsTSV(10)
				t2.Alignment = telem.NewAlignment(1, 1)
				fr2 = fr2.Append(10, d2)
				fr2 = fr2.Append(11, t2)
				channelState.Ingest(fr2)

				outputCount := 0

				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { outputCount++ }})
				Expect(outputCount).To(Equal(1))
				o := nodeState.Output(0)
				Expect(*o).To(telem.MatchSeries(d1))
				ot := nodeState.OutputTime(0)
				Expect(*ot).To(telem.MatchSeries(t1))

				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { outputCount++ }})
				Expect(outputCount).To(Equal(2))
				o = nodeState.Output(0)
				Expect(*o).To(telem.MatchSeries(d2))
				ot = nodeState.OutputTime(0)
				Expect(*ot).To(telem.MatchSeries(t2))
			})
		})

		Describe("Reset", func() {
			It("Should advance the watermark to prevent stale data from triggering", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: progState.Node("source"),
				}))
				d1 := telem.NewSeriesV[float32](1.0)
				d1.Alignment = telem.NewAlignment(1, 0)
				t1 := telem.NewSeriesSecondsTSV(100)
				t1.Alignment = telem.NewAlignment(1, 0)
				fr1 := telem.Frame[uint32]{}
				fr1 = fr1.Append(10, d1)
				fr1 = fr1.Append(11, t1)
				channelState.Ingest(fr1)

				source.Reset()

				var triggered bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { triggered = true }})
				Expect(triggered).To(BeFalse(), "stale pre-reset data should not trigger the source")

				d2 := telem.NewSeriesV[float32](2.0)
				d2.Alignment = telem.NewAlignment(2, 0)
				t2 := telem.NewSeriesSecondsTSV(200)
				t2.Alignment = telem.NewAlignment(2, 0)
				fr2 := telem.Frame[uint32]{}
				fr2 = fr2.Append(10, d2)
				fr2 = fr2.Append(11, t2)
				channelState.Ingest(fr2)

				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { triggered = true }})
				Expect(triggered).To(BeTrue(), "data written after reset should trigger the source")
			})
			It("Should be a no-op when channel has no data", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: progState.Node("source"),
				}))
				Expect(func() { source.Reset() }).ToNot(Panic())
				var triggered bool
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { triggered = true }})
				Expect(triggered).To(BeFalse())
			})
		})

		Describe("Alignment Validation", func() {
			It("Should skip data when index series count mismatch", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: progState.Node("source"),
				}))
				fr1 := telem.Frame[uint32]{}
				fr1 = fr1.Append(10, telem.NewSeriesV[float32](1.0))
				fr1 = fr1.Append(11, telem.NewSeriesSecondsTSV(10))
				channelState.Ingest(fr1)
				fr2 := telem.Frame[uint32]{}
				fr2 = fr2.Append(10, telem.NewSeriesV[float32](2.0))
				channelState.Ingest(fr2)
				callCount := 0
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { callCount++ }})
				Expect(callCount).To(Equal(1))
			})

			It("Should skip data when alignment mismatch", func(ctx SpecContext) {
				g2 := graph.Graph{
					Nodes: []graph.Node{{Key: "misaligned", Type: "on"}},
					Functions: []graph.Function{{
						Key:     "on",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
					}},
				}
				mod := MustSucceed(channel.NewModule(ctx, channelState, nil, nil))
				analyzed2, diagnostics2 := graph.Analyze(ctx, g2, channel.SymbolResolver)
				Expect(diagnostics2.Ok()).To(BeTrue())
				s2 := rnode.New(analyzed2)
				channelState := channel.NewProgramState([]channel.Digest{
					{Key: 30, DataType: telem.Float64T, Index: 31},
				})
				source := MustSucceed(mod.Create(ctx, rnode.Config{
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
				channelState.Ingest(fr)
				outputCount := 0
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { outputCount++ }})
				Expect(outputCount).To(Equal(0))
			})
		})
	})

	Describe("Sink Node", func() {
		var (
			progState    *rnode.ProgramState
			channelState *channel.ProgramState
			factory      rnode.Factory
		)
		BeforeEach(func(ctx SpecContext) {
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
						Key:     "write",
						Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.F32()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
					},
				},
			}
			channelState = channel.NewProgramState([]channel.Digest{
				{Key: 100, DataType: telem.Float32T, Index: 101},
			})
			mod := MustSucceed(channel.NewModule(ctx, channelState, nil, nil))
			analyzed, diagnostics := graph.Analyze(ctx, g, channel.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			progState = rnode.New(analyzed)
			factory = mod
		})

		Describe("Data Writing", func() {
			It("Should write channel data when input available", func(ctx SpecContext) {
				sinkState := progState.Node("sink")
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: sinkState,
				}))
				upstream := progState.Node("upstream")
				inputData := telem.NewSeriesV[float32](7.7, 8.8)
				inputData.Alignment = 42
				inputData.TimeRange = telem.TimeRange{Start: 500 * telem.SecondTS, End: 501 * telem.SecondTS}
				*upstream.Output(0) = inputData
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(500, 501)
				changed := false
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { changed = true }})
				Expect(changed).To(BeTrue())

				outData := *sinkState.Output(0)
				Expect(outData.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[uint8](outData, 0)).To(Equal(uint8(1)))
				Expect(outData.Alignment).To(Equal(telem.Alignment(42)))
				Expect(outData.TimeRange.Start).To(Equal(500 * telem.SecondTS))

				outTime := *sinkState.OutputTime(0)
				Expect(outTime.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[telem.TimeStamp](outTime, 0)).To(Equal(501 * telem.SecondTS))
				Expect(outTime.Alignment).To(Equal(telem.Alignment(42)))

				fr, flushed := channelState.Flush(telem.Frame[uint32]{})
				Expect(flushed).To(BeTrue())
				Expect(fr.Get(100).Series).To(HaveLen(1))
				Expect(fr.Get(100).Series[0]).To(telem.MatchSeriesDataV[float32](7.7, 8.8))
				Expect(fr.Get(101).Series).To(HaveLen(1))
				Expect(fr.Get(101).Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(500, 501)))
			})
			It("Should respect RefreshInputs guard", func(ctx SpecContext) {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: progState.Node("sink"),
				}))
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				fr, changed := channelState.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeFalse())
				Expect(fr.Get(100).Series).To(BeEmpty())
			})
			It("Should not write when input is empty", func(ctx SpecContext) {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: progState.Node("sink"),
				}))
				upstream := progState.Node("upstream")
				*upstream.Output(0) = telem.NewSeriesV[float32]()
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV()
				Expect(progState.Node("sink").RefreshInputs()).To(BeFalse())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				fr, changed := channelState.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeFalse())
				Expect(fr.Get(100).Series).To(BeEmpty())
			})
		})
		Describe("Multiple Writes", func() {
			It("Should handle sequential writes", func(ctx SpecContext) {
				sink := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
					},
					State: progState.Node("sink"),
				}))
				upstream := progState.Node("upstream")
				*upstream.Output(0) = telem.NewSeriesV[float32](1.0)
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
				Expect(progState.Node("sink").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				fr1, changed := channelState.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr1.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1.0)))
				*upstream.Output(0) = telem.NewSeriesV[float32](2.0)
				*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
				Expect(progState.Node("sink").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				fr2, changed := channelState.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr2.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
			})
		})
	})

	Describe("Integration", func() {
		Describe("Source to Sink Flow", func() {
			It("Should flow data from source through sink", func(ctx SpecContext) {
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
							Key:     "write",
							Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.I32()}},
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
						},
					},
				}
				channelState := channel.NewProgramState([]channel.Digest{
					{Key: 1, DataType: telem.Int32T, Index: 2},
					{Key: 3, DataType: telem.Int32T, Index: 4},
				})
				mod := MustSucceed(channel.NewModule(ctx, channelState, nil, nil))
				analyzed, diagnostics := graph.Analyze(ctx, g, channel.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				s := rnode.New(analyzed)
				source := MustSucceed(mod.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(1)}},
					},
					State: s.Node("read"),
				}))
				sink := MustSucceed(mod.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Config: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(3)}},
					},
					State: s.Node("write"),
				}))
				ingestFr := telem.Frame[uint32]{}
				ingestFr = ingestFr.Append(1, telem.NewSeriesV[int32](42, 99))
				ingestFr = ingestFr.Append(2, telem.NewSeriesSecondsTSV(10, 20))
				channelState.Ingest(ingestFr)
				source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				Expect(s.Node("write").RefreshInputs()).To(BeTrue())
				sink.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				outputFr, changed := channelState.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(outputFr.Get(3).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](42, 99)))
				Expect(outputFr.Get(4).Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)))
			})
		})
		Describe("Multiple Channels", func() {
			It("Should handle multiple independent source-sink pairs", func(ctx SpecContext) {
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
							{Name: ir.DefaultInputParam, Type: types.F32()}},
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
						{Key: "write2", Inputs: types.Params{
							{Name: ir.DefaultInputParam, Type: types.F64()}},
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}}},
					},
				}
				channelState := channel.NewProgramState([]channel.Digest{
					{Key: 10, DataType: telem.Float32T, Index: 11},
					{Key: 20, DataType: telem.Float64T, Index: 21},
					{Key: 30, DataType: telem.Float32T, Index: 31},
					{Key: 40, DataType: telem.Float64T, Index: 41},
				})
				mod := MustSucceed(channel.NewModule(ctx, channelState, nil, nil))
				analyzed, diagnostics := graph.Analyze(ctx, g, channel.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				s := rnode.New(analyzed)

				factory := mod
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
				channelState.Ingest(fr)
				source1.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				source2.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				Expect(s.Node("write1").RefreshInputs()).To(BeTrue())
				Expect(s.Node("write2").RefreshInputs()).To(BeTrue())
				sink1.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				sink2.Next(rnode.Context{Context: ctx, MarkChanged: func(int) {}})
				channelState.ClearReads()
				outputFr, changed := channelState.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(outputFr.Get(30).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1.1, 2.2)))
				Expect(outputFr.Get(40).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float64](3.3, 4.4)))
			})
		})
	})

	Describe("NewModule nil-safety", func() {
		It("Should not panic when channel state is nil", func(ctx SpecContext) {
			Expect(func() {
				MustSucceed(channel.NewModule(ctx, nil, nil, nil))
			}).ToNot(Panic())
		})
	})
})
