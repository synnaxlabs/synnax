// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel", func() {
	Describe("WASM Bindings", func() {
		var (
			rt *testutil.Runtime
			cs *channels.ProgramState
			ss *strings.ProgramState
		)

		BeforeEach(func(ctx SpecContext) {
			rt = testutil.NewRuntime(ctx)
			cs = channels.NewProgramState([]channels.Digest{
				{Key: 1, DataType: telem.Float64T},
				{Key: 2, DataType: telem.Int32T},
				{Key: 3, DataType: telem.StringT},
			})
			ss = strings.NewProgramState()
			_, err := channels.NewHost(ctx, rt.Underlying(), cs, ss)
			Expect(err).To(Succeed())
			rt.Passthrough(ctx, "channels")
		})

		AfterEach(func(ctx SpecContext) {
			Expect(rt.Close(ctx)).To(Succeed())
		})

		Describe("i32 types", func() {
			It("Should write and read back u8 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channels", "write_u8", testutil.U32(2), testutil.U32(42))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_u8", testutil.U32(2))
				Expect(testutil.AsU32(result[0])).To(Equal(uint32(42)))
			})

			It("Should write and read back i32 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channels", "write_i32", testutil.U32(2), testutil.U32(100))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_i32", testutil.U32(2))
				Expect(testutil.AsU32(result[0])).To(Equal(uint32(100)))
			})
		})

		Describe("i64 types", func() {
			It("Should write and read back u64 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channels", "write_u64", testutil.U32(1), testutil.U64(12345))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_u64", testutil.U32(1))
				Expect(testutil.AsU64(result[0])).To(Equal(uint64(12345)))
			})

			It("Should write and read back i64 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channels", "write_i64", testutil.U32(1), testutil.U64(99999))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_i64", testutil.U32(1))
				Expect(testutil.AsU64(result[0])).To(Equal(uint64(99999)))
			})
		})

		Describe("float types", func() {
			It("Should write and read back f32 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channels", "write_f32", testutil.U32(1), testutil.F32(3.14))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_f32", testutil.U32(1))
				Expect(testutil.AsF32(result[0])).To(BeNumerically("~", 3.14, 0.001))
			})

			It("Should write and read back f64 values", func(ctx SpecContext) {
				rt.CallVoid(ctx, "channels", "write_f64", testutil.U32(1), testutil.F64(2.718281828))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_f64", testutil.U32(1))
				Expect(testutil.AsF64(result[0])).To(BeNumerically("~", 2.718281828, 0.0001))
			})
		})

		Describe("string type", func() {
			It("Should write and read back string values via handles", func(ctx SpecContext) {
				h := ss.Create("hello world")
				rt.CallVoid(ctx, "channels", "write_str", testutil.U32(3), testutil.U32(h))
				fr := telem.Frame[uint32]{}
				fr, _ = cs.Flush(fr)
				cs.Ingest(fr)
				result := rt.Call(ctx, "channels", "read_str", testutil.U32(3))
				rh := testutil.AsU32(result[0])
				Expect(rh).ToNot(BeZero())
				Expect(MustBeOk(ss.Get(rh))).To(Equal("hello world"))
			})
		})

		Describe("read with no data", func() {
			It("Should return 0 when no data has been ingested", func(ctx SpecContext) {
				result := rt.Call(ctx, "channels", "read_f64", testutil.U32(1))
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
			factory = MustSucceed(channels.NewHost(ctx, nil, nil, nil))
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "test"},
					{Key: "producer"},
					{Key: "writer"},
				},
				Inputs: map[string]msgpack.EncodedJSON{
					"test":     {"type": "on"},
					"producer": {"type": "producer"},
					"writer":   {"type": "write"},
				},
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: ir.Handle{Node: "producer", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "writer", Param: ir.DefaultInputParam},
					}},
				},
				Functions: []graph.Function{
					{Key: "on"},
					{Key: "producer", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F32()}}},
					{
						Key:     "write",
						Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.F32()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			rtState = rnode.New(analyzed)
		})

		Describe("Source Creation", func() {
			It("Should create source node for on type", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(42)}},
					},
					State: rtState.Node("test"),
				}
				node := MustSucceed(factory.Create(ctx, cfg))
				Expect(node).ToNot(BeNil())
			})
			It("Should parse channel from input", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(123)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(99)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
					},
					State: rtState.Node("writer"),
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(1)}},
					},
					State: rtState.Node("test"),
				}
				node, err := factory.Create(ctx, cfg)
				Expect(err).To(Equal(query.ErrNotFound))
				Expect(node).To(BeNil())
			})
			It("Should return error for invalid input", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{{Name: "invalid", Type: types.String(), Value: "field"}},
					},
					State: rtState.Node("test"),
				}
				Expect(factory.Create(ctx, cfg)).Error().To(BeAValidationPathError())
			})
			It("Should return error for missing channel", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{},
					},
					State: rtState.Node("test"),
				}
				Expect(factory.Create(ctx, cfg)).Error().To(BeAValidationPathError())
			})
			It("Should return error for a sink with neither a channel key nor a binding edge", func(ctx SpecContext) {
				cfg := rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Inputs: types.Params{},
					},
					State: rtState.Node("test"),
				}
				Expect(factory.Create(ctx, cfg)).Error().To(BeAValidationPathError())
			})
		})
	})

	Describe("Source Node", func() {
		var (
			progState    *rnode.ProgramState
			channelState *channels.ProgramState
			factory      rnode.Factory
		)
		BeforeEach(func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "source"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": "on"},
				},
				Functions: []graph.Function{{
					Key: "on",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.F32()},
					},
				}},
			}
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			channelState = channels.NewProgramState([]channels.Digest{
				{Key: 10, DataType: telem.Float32T, Index: 11},
				{Key: 20, DataType: telem.Int32T, Index: 0},
			})
			progState = rnode.New(inter)
			factory = MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
		})

		Describe("Data Reading", func() {
			It("Should read channel data after ingestion", func(ctx SpecContext) {
				source := MustSucceed(factory.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(999)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}},
					},
					State: progState.Node("source"),
				}))
				nodeState := progState.Node("source")
				var prevTS telem.TimeStamp
				for i := range 10 {
					d := telem.NewSeriesV(int32(i))
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}},
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
					Nodes: []graph.Node{{Key: "misaligned"}},
					Inputs: map[string]msgpack.EncodedJSON{
						"misaligned": {"type": "on"},
					},
					Functions: []graph.Function{{
						Key:     "on",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
					}},
				}
				mod := MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
				analyzed2, diagnostics2 := graph.Analyze(ctx, g2, NewGraphRoot(nil))
				Expect(diagnostics2.Ok()).To(BeTrue())
				s2 := rnode.New(analyzed2)
				channelState := channels.NewProgramState([]channels.Digest{
					{Key: 30, DataType: telem.Float64T, Index: 31},
				})
				source := MustSucceed(mod.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(30)}},
					},
					State: s2.Node("misaligned"),
				}))
				dataSeries := telem.NewSeriesV(1.0, 2.0)
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

	Describe("Index Pairing", func() {
		// An `on` source must pair a data sample with its co-written index sample by
		// ALIGNMENT, not buffer position: a shared index accumulates more series than
		// any one data channel, so position pairing picks a wrong index and stalls.
		var (
			progState    *rnode.ProgramState
			channelState *channels.ProgramState
			factory      rnode.Factory
		)
		// Channels 10/12/14/16 share index 99; 30 has its own index (31); 40 has
		// none. Node keys s0..s7 let a test bind up to eight `on` sources.
		BeforeEach(func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "s0"}, {Key: "s1"}, {Key: "s2"}, {Key: "s3"},
					{Key: "s4"}, {Key: "s5"}, {Key: "s6"}, {Key: "s7"},
				},
				Inputs: map[string]msgpack.EncodedJSON{
					"s0": {"type": "on"}, "s1": {"type": "on"},
					"s2": {"type": "on"}, "s3": {"type": "on"},
					"s4": {"type": "on"}, "s5": {"type": "on"},
					"s6": {"type": "on"}, "s7": {"type": "on"},
				},
				Functions: []graph.Function{{
					Key:     "on",
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F32()}},
				}},
			}
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			channelState = channels.NewProgramState([]channels.Digest{
				{Key: 10, DataType: telem.Float32T, Index: 99},
				{Key: 12, DataType: telem.Float32T, Index: 99},
				{Key: 14, DataType: telem.Float32T, Index: 99},
				{Key: 16, DataType: telem.Float32T, Index: 99},
				{Key: 30, DataType: telem.Float32T, Index: 31},
				{Key: 40, DataType: telem.Float32T, Index: 0},
			})
			progState = rnode.New(inter)
			factory = MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
		})

		keys := []string{"s0", "s1", "s2", "s3", "s4", "s5", "s6", "s7"}

		newSource := func(ctx SpecContext, nodeKey string, ch uint32) rnode.Node {
			return MustSucceed(factory.Create(ctx, rnode.Config{
				Node: ir.Node{
					Type:   "on",
					Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: ch}},
				},
				State: progState.Node(nodeKey),
			}))
		}
		// writeData co-writes a data sample and its index timestamp at the SAME
		// alignment, as cesium delivers a data channel written with its index.
		writeData := func(dataKey, idxKey uint32, value float32, ts telem.TimeStamp, a telem.Alignment) {
			d := telem.NewSeriesV(value)
			d.Alignment = a
			t := telem.NewSeriesV(ts)
			t.Alignment = a
			fr := telem.Frame[uint32]{}
			fr = fr.Append(idxKey, t)
			fr = fr.Append(dataKey, d)
			channelState.Ingest(fr)
		}
		// writeIndexNoise writes a lone index sample: another channel sharing
		// idxKey being written, which the relay still delivers to arc.
		writeIndexNoise := func(idxKey uint32, ts telem.TimeStamp, a telem.Alignment) {
			t := telem.NewSeriesV(ts)
			t.Alignment = a
			channelState.Ingest(telem.UnaryFrame(idxKey, t))
		}
		// writeDataOnly writes a data sample with no accompanying index sample.
		writeDataOnly := func(dataKey uint32, value float32, a telem.Alignment) {
			d := telem.NewSeriesV(value)
			d.Alignment = a
			channelState.Ingest(telem.UnaryFrame(dataKey, d))
		}
		firesOn := func(ctx SpecContext, src rnode.Node) bool {
			f := false
			src.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { f = true }})
			return f
		}
		emittedValue := func(nodeKey string) float32 {
			return telem.ValueAt[float32](*progState.Node(nodeKey).Output(0), -1)
		}
		emittedTS := func(nodeKey string) telem.TimeStamp {
			return telem.ValueAt[telem.TimeStamp](*progState.Node(nodeKey).OutputTime(0), -1)
		}
		// al builds an alignment in the common shared streaming domain (1).
		al := func(sample uint32) telem.Alignment { return telem.NewAlignment(1, sample) }

		Describe("Dedicated index (baseline)", func() {
			It("Should fire on a co-written data+index sample", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 30)
				writeData(30, 31, 42, 1000, al(0))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(emittedValue("s0")).To(Equal(float32(42)))
				Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(1000)))
			})

			It("Should fire on every consecutive write", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 30)
				for i := range uint32(20) {
					writeData(30, 31, float32(i), telem.TimeStamp(1000+i), al(i))
					Expect(firesOn(ctx, src)).To(BeTrue(), "write %d must fire", i)
					Expect(emittedValue("s0")).To(Equal(float32(i)))
					channelState.ClearReads()
				}
			})

			It("Should not re-fire the same stale sample", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 30)
				writeData(30, 31, 42, 1000, al(5))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(firesOn(ctx, src)).To(BeFalse(), "stale sample must not re-fire")
			})

			DescribeTable("Should fire across alignment domains",
				func(ctx SpecContext, domain uint32) {
					src := newSource(ctx, "s0", 30)
					writeData(30, 31, 7, 2000, telem.NewAlignment(domain, 0))
					Expect(firesOn(ctx, src)).To(BeTrue())
					Expect(emittedValue("s0")).To(Equal(float32(7)))
					Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(2000)))
				},
				Entry("committed low domain", uint32(0)),
				Entry("committed mid domain", uint32(500)),
				Entry("leading (ZeroLeading) domain", uint32(4293967295)),
				Entry("leading domain + offset", uint32(4293967300)),
			)
		})

		Describe("Shared index", func() {
			DescribeTable("Should fire for its channel regardless of shared-index noise placement",
				func(ctx SpecContext, noiseBefore, noiseAfter int) {
					src := newSource(ctx, "s0", 10)
					var s uint32
					for range noiseBefore {
						writeIndexNoise(99, telem.TimeStamp(s), al(s))
						s++
					}
					writeData(10, 99, 42, 7000, al(s))
					s++
					for range noiseAfter {
						writeIndexNoise(99, telem.TimeStamp(s), al(s))
						s++
					}
					Expect(firesOn(ctx, src)).To(BeTrue(),
						"source must fire; its co-written index is present, just not at position i")
					Expect(emittedValue("s0")).To(Equal(float32(42)))
					Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(7000)))
				},
				Entry("no noise", 0, 0),
				Entry("noise before", 3, 0),
				Entry("noise after", 0, 3),
				Entry("noise both sides", 3, 3),
				Entry("heavy noise before", 25, 0),
				Entry("heavy noise after", 0, 25),
				Entry("heavy noise both sides", 25, 25),
				Entry("buried very deep", 60, 60),
			)

			It("Should emit the co-written timestamp, not a neighbouring one", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				writeIndexNoise(99, 111, al(0))
				writeIndexNoise(99, 222, al(1))
				writeData(10, 99, 42, 999, al(2))
				writeIndexNoise(99, 333, al(3))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(999)),
					"must pair with the index sample co-written with the data (alignment 2)")
				Expect(emittedValue("s0")).To(Equal(float32(42)))
			})

			It("Should fire each of two channels sharing one index", func(ctx SpecContext) {
				srcA := newSource(ctx, "s0", 10)
				srcB := newSource(ctx, "s1", 12)
				writeIndexNoise(99, 100, al(0))
				writeData(10, 99, 42, 200, al(1))
				writeIndexNoise(99, 300, al(2))
				writeData(12, 99, 77, 400, al(3))
				writeIndexNoise(99, 500, al(4))
				Expect(firesOn(ctx, srcA)).To(BeTrue())
				Expect(emittedValue("s0")).To(Equal(float32(42)))
				Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(200)))
				Expect(firesOn(ctx, srcB)).To(BeTrue())
				Expect(emittedValue("s1")).To(Equal(float32(77)))
				Expect(emittedTS("s1")).To(Equal(telem.TimeStamp(400)))
			})

			DescribeTable("Should fire every channel sharing one index",
				func(ctx SpecContext, n int) {
					chans := []uint32{10, 12, 14, 16}
					srcs := make([]rnode.Node, n)
					for i := range n {
						srcs[i] = newSource(ctx, keys[i], chans[i])
					}
					var s uint32
					for i := range n {
						writeIndexNoise(99, telem.TimeStamp(s), al(s))
						s++
						writeData(chans[i], 99, float32(100+i), telem.TimeStamp(1000+i), al(s))
						s++
						writeIndexNoise(99, telem.TimeStamp(s), al(s))
						s++
					}
					for i := range n {
						Expect(firesOn(ctx, srcs[i])).To(BeTrue(), "channel %d must fire", i)
						Expect(emittedValue(keys[i])).To(Equal(float32(100 + i)))
					}
				},
				Entry("two channels", 2),
				Entry("three channels", 3),
				Entry("four channels", 4),
			)

			It("Should fire both channels when written in separate frames", func(ctx SpecContext) {
				srcA := newSource(ctx, "s0", 10)
				srcB := newSource(ctx, "s1", 12)
				writeData(10, 99, 42, 100, al(0))
				Expect(firesOn(ctx, srcA)).To(BeTrue())
				Expect(firesOn(ctx, srcB)).To(BeFalse(), "b not written yet")
				channelState.ClearReads()
				writeData(12, 99, 77, 101, al(1))
				Expect(firesOn(ctx, srcB)).To(BeTrue())
				Expect(emittedValue("s1")).To(Equal(float32(77)))
				Expect(emittedTS("s1")).To(Equal(telem.TimeStamp(101)))
			})
		})

		Describe("Single write fires each source once", func() {
			fireCount := func(ctx SpecContext, src rnode.Node) int {
				count := 0
				src.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { count++ }})
				return count
			}

			DescribeTable("Should fire each shared-index channel exactly once per write",
				func(ctx SpecContext, n int) {
					chans := []uint32{10, 12, 14, 16}
					srcs := make([]rnode.Node, n)
					for i := range n {
						srcs[i] = newSource(ctx, keys[i], chans[i])
					}
					for i := range n {
						writeData(chans[i], 99, float32(100+i), telem.TimeStamp(1000+i), al(uint32(i)))
					}
					for i := range n {
						Expect(fireCount(ctx, srcs[i])).To(Equal(1), "channel %d must fire once", i)
						Expect((*progState.Node(keys[i]).Output(0)).Len()).To(Equal(int64(1)),
							"channel %d must emit one sample, not the shared index buffer", i)
						Expect(fireCount(ctx, srcs[i])).To(Equal(0), "channel %d must not re-fire", i)
					}
				},
				Entry("two channels", 2),
				Entry("three channels", 3),
				Entry("four channels", 4),
			)

			It("Should fire once despite a heavily populated shared index", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				var s uint32
				for range 20 {
					writeIndexNoise(99, telem.TimeStamp(s), al(s))
					s++
				}
				writeData(10, 99, 42, 9000, al(s))
				s++
				for range 20 {
					writeIndexNoise(99, telem.TimeStamp(s), al(s))
					s++
				}
				Expect(fireCount(ctx, src)).To(Equal(1), "one data sample must fire once")
				Expect((*progState.Node("s0").Output(0)).Len()).To(Equal(int64(1)))
				Expect(emittedValue("s0")).To(Equal(float32(42)))
				Expect(fireCount(ctx, src)).To(Equal(0), "must not re-fire without new data")
			})
		})

		Describe("Sustained high-rate (permanent-stall regression)", func() {
			DescribeTable("Should keep firing under sustained shared-index writes",
				func(ctx SpecContext, noiseBefore, noiseAfter int) {
					src := newSource(ctx, "s0", 10)
					var s uint32
					fires := 0
					const cycles = 40
					for c := range cycles {
						for range noiseBefore {
							writeIndexNoise(99, telem.TimeStamp(s), al(s))
							s++
						}
						writeData(10, 99, float32(c), telem.TimeStamp(1000+c), al(s))
						s++
						for range noiseAfter {
							writeIndexNoise(99, telem.TimeStamp(s), al(s))
							s++
						}
						if firesOn(ctx, src) {
							fires++
						}
						channelState.ClearReads()
					}
					Expect(fires).To(Equal(cycles),
						"the on-source must fire every cycle it is written")
				},
				Entry("no noise", 0, 0),
				Entry("noise before only", 3, 0),
				Entry("noise after only", 0, 3),
				Entry("noise both sides", 3, 3),
				Entry("heavy noise both sides", 20, 20),
			)

			It("Should fire all four fridge sensors every cycle on a shared 50Hz index", func(ctx SpecContext) {
				chans := []uint32{10, 12, 14, 16}
				srcs := make([]rnode.Node, len(chans))
				for i := range srcs {
					srcs[i] = newSource(ctx, keys[i], chans[i])
				}
				var s uint32
				fires := make([]int, len(chans))
				const cycles = 50
				for c := range cycles {
					for i, ch := range chans {
						writeData(ch, 99, float32(10+i), telem.TimeStamp(1000+c), al(s))
						s++
					}
					for i := range srcs {
						if firesOn(ctx, srcs[i]) {
							fires[i]++
						}
					}
					channelState.ClearReads()
				}
				for i := range fires {
					Expect(fires[i]).To(Equal(cycles), "sensor %d stalled", i)
				}
			})
		})

		Describe("High-water mark and ordering", func() {
			It("Should ignore data below the high-water mark", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				writeData(10, 99, 1, 100, al(5))
				Expect(firesOn(ctx, src)).To(BeTrue())
				channelState.ClearReads()
				writeData(10, 99, 2, 90, al(2))
				Expect(firesOn(ctx, src)).To(BeFalse(), "below-watermark sample must not fire")
			})

			It("Should not fire again without new data", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				writeIndexNoise(99, 10, al(0))
				writeData(10, 99, 5, 20, al(1))
				writeIndexNoise(99, 30, al(2))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(firesOn(ctx, src)).To(BeFalse())
				Expect(firesOn(ctx, src)).To(BeFalse())
			})
		})

		Describe("Late or missing index", func() {
			It("Should not fire until the matching index sample arrives", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				writeIndexNoise(99, 100, al(0))
				writeDataOnly(10, 42, al(5))
				Expect(firesOn(ctx, src)).To(BeFalse(),
					"no index sample matches the data's alignment yet")
				writeIndexNoise(99, 555, al(5))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(emittedValue("s0")).To(Equal(float32(42)))
				Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(555)))
			})

			It("Should not fire when no index matches the data's alignment", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				writeIndexNoise(99, 100, al(0))
				writeIndexNoise(99, 200, al(1))
				writeIndexNoise(99, 300, al(2))
				writeDataOnly(10, 42, al(9))
				Expect(firesOn(ctx, src)).To(BeFalse(),
					"data must not be paired with an unrelated index sample")
			})
		})

		Describe("No index (virtual-style channel)", func() {
			It("Should synthesize timestamps and fire for an index-less channel", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 40)
				writeDataOnly(40, 42, al(0))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(emittedValue("s0")).To(Equal(float32(42)))
				Expect(progState.Node("s0").OutputTime(0).Len()).To(Equal(int64(1)))
			})
		})

		Describe("Reset", func() {
			It("Should ignore buried pre-reset data after reset, then fire on new data", func(ctx SpecContext) {
				src := newSource(ctx, "s0", 10)
				writeIndexNoise(99, 100, al(0))
				writeData(10, 99, 1, 200, al(1))
				writeIndexNoise(99, 300, al(2))
				src.Reset()
				Expect(firesOn(ctx, src)).To(BeFalse(), "pre-reset data must not fire after reset")
				channelState.ClearReads()
				writeData(10, 99, 2, 400, al(5))
				Expect(firesOn(ctx, src)).To(BeTrue())
				Expect(emittedValue("s0")).To(Equal(float32(2)))
				Expect(emittedTS("s0")).To(Equal(telem.TimeStamp(400)))
			})
		})
	})

	Describe("Sink Node", func() {
		var (
			progState    *rnode.ProgramState
			channelState *channels.ProgramState
			factory      rnode.Factory
		)
		BeforeEach(func(ctx SpecContext) {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "upstream"},
					{Key: "sink"},
				},
				Inputs: map[string]msgpack.EncodedJSON{
					"upstream": {"type": "producer"},
					"sink":     {"type": "write"},
				},
				Edges: graph.Edges{
					{Edge: ir.Edge{
						Source: ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					}},
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
			channelState = channels.NewProgramState([]channels.Digest{
				{Key: 100, DataType: telem.Float32T, Index: 101},
			})
			mod := MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
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
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(100)}},
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
						{Key: "read"},
						{Key: "write"},
					},
					Inputs: map[string]msgpack.EncodedJSON{
						"read":  {"type": "on"},
						"write": {"type": "write"},
					},
					Edges: graph.Edges{
						{Edge: ir.Edge{
							Source: ir.Handle{Node: "read", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "write", Param: ir.DefaultInputParam},
						}},
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
				channelState := channels.NewProgramState([]channels.Digest{
					{Key: 1, DataType: telem.Int32T, Index: 2},
					{Key: 3, DataType: telem.Int32T, Index: 4},
				})
				mod := MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue())
				s := rnode.New(analyzed)
				source := MustSucceed(mod.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "on",
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(1)}},
					},
					State: s.Node("read"),
				}))
				sink := MustSucceed(mod.Create(ctx, rnode.Config{
					Node: ir.Node{
						Type:   "write",
						Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(3)}},
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
						{Key: "read1"},
						{Key: "read2"},
						{Key: "write1"},
						{Key: "write2"},
					},
					Inputs: map[string]msgpack.EncodedJSON{
						"read1":  {"type": "on"},
						"read2":  {"type": "on2"},
						"write1": {"type": "write"},
						"write2": {"type": "write2"},
					},
					Edges: graph.Edges{
						{Edge: ir.Edge{Source: ir.Handle{Node: "read1", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "write1", Param: ir.DefaultInputParam}}},
						{Edge: ir.Edge{Source: ir.Handle{Node: "read2", Param: ir.DefaultOutputParam},
							Target: ir.Handle{Node: "write2", Param: ir.DefaultInputParam}}},
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
				channelState := channels.NewProgramState([]channels.Digest{
					{Key: 10, DataType: telem.Float32T, Index: 11},
					{Key: 20, DataType: telem.Float64T, Index: 21},
					{Key: 30, DataType: telem.Float32T, Index: 31},
					{Key: 40, DataType: telem.Float64T, Index: 41},
				})
				mod := MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue())
				s := rnode.New(analyzed)

				factory := mod
				source1, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "on", Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(10)}}},
					State: s.Node("read1"),
				})
				source2, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "on", Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(20)}}},
					State: s.Node("read2"),
				})
				sink1, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "write", Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(30)}}},
					State: s.Node("write1"),
				})
				sink2, _ := factory.Create(ctx, rnode.Config{
					Node:  ir.Node{Type: "write", Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: uint32(40)}}},
					State: s.Node("write2"),
				})
				fr := telem.Frame[uint32]{}
				fr = fr.Append(10, telem.NewSeriesV[float32](1.1, 2.2))
				fr = fr.Append(11, telem.NewSeriesSecondsTSV(100, 200))
				fr = fr.Append(20, telem.NewSeriesV(3.3, 4.4))
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
				Expect(outputFr.Get(40).Series[0]).To(telem.MatchSeries(telem.NewSeriesV(3.3, 4.4)))
			})
		})
	})

	Describe("NewModule nil-safety", func() {
		It("Should not panic when channel state is nil", func(ctx SpecContext) {
			Expect(func() {
				MustSucceed(channels.NewHost(ctx, nil, nil, nil))
			}).ToNot(Panic())
		})
	})
})

var _ = Describe("Source Rebind", func() {
	var (
		channelState *channels.ProgramState
		progState    *rnode.ProgramState
		source       rnode.Node
	)
	BeforeEach(func(ctx SpecContext) {
		prog := ir.IR{
			Nodes: ir.Nodes{
				{
					Key:     "bind",
					Type:    "variable",
					Inputs:  types.Params{{Name: "f0", Type: types.U32(), Value: uint32(10)}},
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.Chan(types.F32())}},
				},
				{
					Key:  "source",
					Type: "on",
					Inputs: types.Params{
						{Name: "channel", Type: types.Chan(types.F32()), Value: uint32(10)},
					},
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F32()}},
				},
			},
			Edges: ir.Edges{{
				Source: ir.Handle{Node: "bind", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "source", Param: "channel"},
			}},
		}
		channelState = channels.NewProgramState([]channels.Digest{
			{Key: 10, DataType: telem.Float32T},
			{Key: 20, DataType: telem.Float32T},
		})
		progState = rnode.New(prog)
		factory := MustSucceed(channels.NewHost(ctx, nil, channelState, nil))
		source = MustSucceed(factory.Create(ctx, rnode.Config{
			Node: prog.Nodes[1], State: progState.Node("source"),
		}))
	})

	ingest := func(key, offset uint32, v float32) {
		d := telem.NewSeriesV[float32](v)
		d.Alignment = telem.NewAlignment(1, offset)
		channelState.Ingest(telem.UnaryFrame[uint32](key, d))
	}

	It("Should re-point at the key on the binding edge and skip buffered data", func(ctx SpecContext) {
		ingest(10, 0, 1.5)
		changed := false
		source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { changed = true }})
		Expect(changed).To(BeTrue())

		*progState.Node("bind").Output(0) = telem.NewSeriesV[uint32](20)
		channelState.ClearReads()
		ingest(20, 0, 9.9)
		changed = false
		source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { changed = true }})
		Expect(changed).To(BeFalse(), "data buffered before the rebind must not fire")

		channelState.ClearReads()
		ingest(20, 1, 7.7)
		changed = false
		source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { changed = true }})
		Expect(changed).To(BeTrue())
		out := *progState.Node("source").Output(0)
		Expect(telem.ValueAt[float32](out, -1)).To(Equal(float32(7.7)))
	})

	It("Should rebind on Reset and absorb data buffered on the new channel", func(ctx SpecContext) {
		*progState.Node("bind").Output(0) = telem.NewSeriesV[uint32](20)
		ingest(20, 0, 9.9)
		source.Reset()
		changed := false
		source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { changed = true }})
		Expect(changed).To(BeFalse(), "pre-rebind data must be absorbed by Reset")

		channelState.ClearReads()
		ingest(20, 1, 7.7)
		source.Next(rnode.Context{Context: ctx, MarkChanged: func(int) { changed = true }})
		Expect(changed).To(BeTrue())
	})
})

var _ = Describe("Construction validation", func() {
	It("Should error at construction when the input param is missing", func(ctx SpecContext) {
		prog := ir.IR{Nodes: ir.Nodes{{
			Key:  "write",
			Type: "write",
			Inputs: types.Params{
				{Name: "channel", Type: types.U32(), Value: uint32(1)},
			},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
		}}}
		s := rnode.New(prog)
		factory := MustSucceed(channels.NewHost(ctx, nil, nil, nil))
		cfg := rnode.Config{Node: prog.Nodes[0], State: s.Node("write")}
		Expect(factory.Create(ctx, cfg)).Error().
			To(MatchError(rnode.ErrInputNotFound))
	})
})
