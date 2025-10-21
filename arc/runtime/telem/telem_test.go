package telem_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	rtelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var bCtx = context.Background()

var _ = Describe("Telem Factory", func() {
	var (
		factory rnode.Factory
		s       *state.State
	)
	BeforeEach(func() {
		factory = rtelem.NewTelemFactory()
		s = state.New(state.Config{
			Nodes: []ir.Node{{Key: "test"}},
		})
	})

	Describe("Source Creation", func() {
		It("Should create source node for on type", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(42),
					},
				},
				State: s.Node("test"),
			}
			node, err := factory.Create(bCtx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(node).ToNot(BeNil())
		})
		It("Should parse channel from config", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": 123,
					},
				},
				State: s.Node("test"),
			}
			node, err := factory.Create(bCtx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(node).ToNot(BeNil())
		})
		It("Should coerce channel to uint32", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": float64(99),
					},
				},
				State: s.Node("test"),
			}
			node, err := factory.Create(bCtx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(node).ToNot(BeNil())
		})
	})

	Describe("Sink Creation", func() {
		It("Should create sink node for write type", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("test"),
			}
			node, err := factory.Create(bCtx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(node).ToNot(BeNil())
		})
	})

	Describe("Error Handling", func() {
		It("Should return query.NotFound for unknown node type", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type: "unknown",
					ConfigValues: map[string]any{
						"channel": uint32(1),
					},
				},
				State: s.Node("test"),
			}
			node, err := factory.Create(bCtx, cfg)
			Expect(err).To(Equal(query.NotFound))
			Expect(node).To(BeNil())
		})
		It("Should return error for invalid config", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"invalid": "field",
					},
				},
				State: s.Node("test"),
			}
			_, err := factory.Create(bCtx, cfg)
			Expect(err).To(HaveOccurred())
		})
		It("Should return error for missing channel", func() {
			cfg := rnode.Config{
				Node: ir.Node{
					Type:         "on",
					ConfigValues: map[string]any{},
				},
				State: s.Node("test"),
			}
			_, err := factory.Create(bCtx, cfg)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Source Node", func() {
	var (
		s       *state.State
		factory rnode.Factory
	)
	BeforeEach(func() {
		s = state.New(state.Config{
			ChannelDigests: []state.ChannelDigest{
				{Key: 10, DataType: telem.Float32T, Index: 11},
				{Key: 20, DataType: telem.Int32T, Index: 0},
			},
			ReactiveDeps: map[uint32][]string{
				10: {"source"},
			},
			Nodes: []ir.Node{
				{
					Key:  "source",
					Type: "on",
					Outputs: types.Params{
						Keys:   []string{ir.DefaultOutputParam},
						Values: []types.Type{types.F32()},
					},
				},
			},
		})
		factory = rtelem.NewTelemFactory()
	})

	Describe("Data Reading", func() {
		It("Should read channel data after ingestion", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			fr := telem.Frame[uint32]{}
			fr = fr.Append(10, telem.NewSeriesV[float32](1.5, 2.5, 3.5))
			fr = fr.Append(11, telem.NewSeriesSecondsTSV(100, 101, 102))
			s.Ingest(fr)
			var outputChanged bool
			source.Next(bCtx, func(string) { outputChanged = true })
			Expect(outputChanged).To(BeTrue())
			Expect(*s.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](1.5, 2.5, 3.5)))
			Expect(*s.Node("source").OutputTime(0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(100, 101, 102)))
		})

		It("Should handle channel without index", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(20),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			fr := telem.UnaryFrame[uint32](20, telem.NewSeriesV[int32](100, 200))
			s.Ingest(fr)
			var outputChanged bool
			source.Next(bCtx, func(string) { outputChanged = true })
			Expect(outputChanged).To(BeTrue())
			Expect(*s.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](100, 200)))
			Expect(s.Node("source").OutputTime(0).DataType).To(Equal(telem.TimeStampT))
		})

		It("Should not trigger on empty channel", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(999),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			var outputChanged bool
			source.Next(bCtx, func(string) { outputChanged = true })
			Expect(outputChanged).To(BeFalse())
		})

		It("Should handle multiple series in MultiSeries", func() {
			nodeState := s.Node("source")
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: nodeState,
			})
			Expect(err).ToNot(HaveOccurred())
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

			source.Next(bCtx, func(string) { outputCount++ })
			Expect(outputCount).To(Equal(1))
			o := nodeState.Output(0)
			Expect(*o).To(telem.MatchSeries(d1))
			ot := nodeState.OutputTime(0)
			Expect(*ot).To(telem.MatchSeries(t1))

			source.Next(bCtx, func(string) { outputCount++ })
			Expect(outputCount).To(Equal(2))
			o = nodeState.Output(0)
			Expect(*o).To(telem.MatchSeries(d2))
			ot = nodeState.OutputTime(0)
			Expect(*ot).To(telem.MatchSeries(t2))
		})
	})

	Describe("High Water Mark", func() {
		It("Should track high water mark to prevent reprocessing", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			fr := telem.Frame[uint32]{}
			fr = fr.Append(10, telem.NewSeriesV[float32](5.0))
			fr = fr.Append(11, telem.NewSeriesSecondsTSV(50))
			s.Ingest(fr)
			firstCallCount := 0
			source.Next(bCtx, func(string) { firstCallCount++ })
			Expect(firstCallCount).To(Equal(1))
			secondCallCount := 0
			source.Next(bCtx, func(string) { secondCallCount++ })
			Expect(secondCallCount).To(Equal(0))
		})

		It("Should process new data after watermark update", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			fr1 := telem.Frame[uint32]{}
			fr1 = fr1.Append(10, telem.NewSeriesV[float32](1.0))
			fr1 = fr1.Append(11, telem.NewSeriesSecondsTSV(10))
			s.Ingest(fr1)
			source.Next(bCtx, func(string) {})
			fr2 := telem.Frame[uint32]{}
			fr2 = fr2.Append(10, telem.NewSeriesV[float32](2.0))
			fr2 = fr2.Append(11, telem.NewSeriesSecondsTSV(20))
			s.Ingest(fr2)
			var newDataProcessed bool
			source.Next(bCtx, func(string) { newDataProcessed = true })
			Expect(newDataProcessed).To(BeTrue())
			Expect(*s.Node("source").Output(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})

		It("Should skip series with timestamps below watermark", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			fr1 := telem.Frame[uint32]{}
			fr1 = fr1.Append(10, telem.NewSeriesV[float32](1.0))
			fr1 = fr1.Append(11, telem.NewSeriesSecondsTSV(100))
			s.Ingest(fr1)
			source.Next(bCtx, func(string) {})
			fr2 := telem.Frame[uint32]{}
			fr2 = fr2.Append(10, telem.NewSeriesV[float32](0.5))
			fr2 = fr2.Append(11, telem.NewSeriesSecondsTSV(50))
			s.Ingest(fr2)
			outputCount := 0
			source.Next(bCtx, func(string) { outputCount++ })
			Expect(outputCount).To(Equal(0))
		})
	})

	Describe("Alignment Validation", func() {
		It("Should skip data when index series count mismatch", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			fr1 := telem.Frame[uint32]{}
			fr1 = fr1.Append(10, telem.NewSeriesV[float32](1.0))
			fr1 = fr1.Append(11, telem.NewSeriesSecondsTSV(10))
			s.Ingest(fr1)
			fr2 := telem.Frame[uint32]{}
			fr2 = fr2.Append(10, telem.NewSeriesV[float32](2.0))
			s.Ingest(fr2)
			callCount := 0
			source.Next(bCtx, func(string) { callCount++ })
			Expect(callCount).To(Equal(1))
		})

		It("Should skip data when alignment mismatch", func() {
			cfg := state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 30, DataType: telem.Float64T, Index: 31},
				},
				ReactiveDeps: map[uint32][]string{
					30: {"misaligned"},
				},
				Nodes: []ir.Node{
					{
						Key:  "misaligned",
						Type: "on",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
					},
				},
			}
			s2 := state.New(cfg)
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(30),
					},
				},
				State: s2.Node("misaligned"),
			})
			Expect(err).ToNot(HaveOccurred())
			dataSeries := telem.NewSeriesV[float64](1.0, 2.0)
			dataSeries.Alignment = 100
			timeSeries := telem.NewSeriesSecondsTSV(10, 20)
			timeSeries.Alignment = 200
			fr := telem.Frame[uint32]{}
			fr = fr.Append(30, dataSeries)
			fr = fr.Append(31, timeSeries)
			s2.Ingest(fr)
			outputCount := 0
			source.Next(bCtx, func(string) { outputCount++ })
			Expect(outputCount).To(Equal(0))
		})
	})
	Describe("Lifecycle", func() {
		It("Should initialize without error", func() {
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(10),
					},
				},
				State: s.Node("source"),
			})
			Expect(err).ToNot(HaveOccurred())
			source.Init(bCtx, func(string) {})
		})
	})
})

var _ = Describe("Sink Node", func() {
	var (
		s       *state.State
		factory rnode.Factory
	)
	BeforeEach(func() {
		s = state.New(state.Config{
			ChannelDigests: []state.ChannelDigest{
				{Key: 100, DataType: telem.Float32T, Index: 101},
			},
			Nodes: []ir.Node{
				{
					Key:  "upstream",
					Type: "producer",
					Outputs: types.Params{
						Keys:   []string{ir.DefaultOutputParam},
						Values: []types.Type{types.F32()},
					},
				},
				{
					Key:  "sink",
					Type: "write",
					Inputs: types.Params{
						Keys:   []string{ir.DefaultInputParam},
						Values: []types.Type{types.F32()},
					},
				},
			},
			Edges: []ir.Edge{
				{
					Source: ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
				},
			},
		})
		factory = rtelem.NewTelemFactory()
	})
	Describe("Data Writing", func() {
		It("Should write channel data when input available", func() {
			sink, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(100),
					},
				},
				State: s.Node("sink"),
			})
			Expect(err).ToNot(HaveOccurred())
			upstream := s.Node("upstream")
			*upstream.Output(0) = telem.NewSeriesV[float32](7.7, 8.8)
			*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(500, 501)
			Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
			sink.Next(bCtx, func(string) {})
			fr, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series).To(HaveLen(1))
			Expect(fr.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](7.7, 8.8)))
			Expect(fr.Get(101).Series).To(HaveLen(1))
			Expect(fr.Get(101).Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(500, 501)))
		})
		It("Should respect RefreshInputs guard", func() {
			sink, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(100),
					},
				},
				State: s.Node("sink"),
			})
			Expect(err).ToNot(HaveOccurred())
			sink.Next(bCtx, func(string) {})
			fr, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse())
			Expect(fr.Get(100).Series).To(BeEmpty())
		})
		It("Should not write when input is empty", func() {
			sink, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(100),
					},
				},
				State: s.Node("sink"),
			})
			Expect(err).ToNot(HaveOccurred())
			upstream := s.Node("upstream")
			*upstream.Output(0) = telem.NewSeriesV[float32]()
			*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV()
			Expect(s.Node("sink").RefreshInputs()).To(BeFalse())
			sink.Next(bCtx, func(string) {})
			fr, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse())
			Expect(fr.Get(100).Series).To(BeEmpty())
		})
	})
	Describe("Multiple Writes", func() {
		It("Should handle sequential writes", func() {
			sink, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(100),
					},
				},
				State: s.Node("sink"),
			})
			Expect(err).ToNot(HaveOccurred())
			upstream := s.Node("upstream")
			*upstream.Output(0) = telem.NewSeriesV[float32](1.0)
			*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
			sink.Next(bCtx, func(string) {})
			fr1, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr1.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1.0)))
			*upstream.Output(0) = telem.NewSeriesV[float32](2.0)
			*upstream.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
			Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
			sink.Next(bCtx, func(string) {})
			fr2, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr2.Get(100).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})
	})
	Describe("Lifecycle", func() {
		It("Should initialize without error", func() {
			sink, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(100),
					},
				},
				State: s.Node("sink"),
			})
			Expect(err).ToNot(HaveOccurred())
			sink.Init(bCtx, func(string) {})
		})
	})
})

var _ = Describe("Integration", func() {
	Describe("Source to Sink Flow", func() {
		It("Should flow data from source through sink", func() {
			s := state.New(state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 1, DataType: telem.Int32T, Index: 2},
					{Key: 3, DataType: telem.Int32T, Index: 4},
				},
				ReactiveDeps: map[uint32][]string{
					1: {"read"},
				},
				Nodes: []ir.Node{
					{
						Key:  "read",
						Type: "on",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key:  "write",
						Type: "write",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.I32()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "read", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "write", Param: ir.DefaultInputParam},
					},
				},
			})
			factory := rtelem.NewTelemFactory()
			source, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "on",
					ConfigValues: map[string]any{
						"channel": uint32(1),
					},
				},
				State: s.Node("read"),
			})
			Expect(err).ToNot(HaveOccurred())
			sink, err := factory.Create(bCtx, rnode.Config{
				Node: ir.Node{
					Type: "write",
					ConfigValues: map[string]any{
						"channel": uint32(3),
					},
				},
				State: s.Node("write"),
			})
			Expect(err).ToNot(HaveOccurred())
			ingestFr := telem.Frame[uint32]{}
			ingestFr = ingestFr.Append(1, telem.NewSeriesV[int32](42, 99))
			ingestFr = ingestFr.Append(2, telem.NewSeriesSecondsTSV(10, 20))
			s.Ingest(ingestFr)
			source.Next(bCtx, func(string) {})
			Expect(s.Node("write").RefreshInputs()).To(BeTrue())
			sink.Next(bCtx, func(string) {})
			outputFr, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outputFr.Get(3).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](42, 99)))
			Expect(outputFr.Get(4).Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)))
		})
	})
	Describe("Multiple Channels", func() {
		It("Should handle multiple independent source-sink pairs", func() {
			s := state.New(state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 10, DataType: telem.Float32T, Index: 11},
					{Key: 20, DataType: telem.Float64T, Index: 21},
					{Key: 30, DataType: telem.Float32T, Index: 31},
					{Key: 40, DataType: telem.Float64T, Index: 41},
				},
				ReactiveDeps: map[uint32][]string{
					10: {"read1"},
					20: {"read2"},
				},
				Nodes: []ir.Node{
					{Key: "read1", Type: "on", Outputs: types.Params{
						Keys: []string{ir.DefaultOutputParam}, Values: []types.Type{types.F32()}}},
					{Key: "read2", Type: "on", Outputs: types.Params{
						Keys: []string{ir.DefaultOutputParam}, Values: []types.Type{types.F64()}}},
					{Key: "write1", Type: "write", Inputs: types.Params{
						Keys: []string{ir.DefaultInputParam}, Values: []types.Type{types.F32()}}},
					{Key: "write2", Type: "write", Inputs: types.Params{
						Keys: []string{ir.DefaultInputParam}, Values: []types.Type{types.F64()}}},
				},
				Edges: []ir.Edge{
					{Source: ir.Handle{Node: "read1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "write1", Param: ir.DefaultInputParam}},
					{Source: ir.Handle{Node: "read2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "write2", Param: ir.DefaultInputParam}},
				},
			})
			factory := rtelem.NewTelemFactory()
			source1, _ := factory.Create(bCtx, rnode.Config{
				Node:  ir.Node{Type: "on", ConfigValues: map[string]any{"channel": uint32(10)}},
				State: s.Node("read1"),
			})
			source2, _ := factory.Create(bCtx, rnode.Config{
				Node:  ir.Node{Type: "on", ConfigValues: map[string]any{"channel": uint32(20)}},
				State: s.Node("read2"),
			})
			sink1, _ := factory.Create(bCtx, rnode.Config{
				Node:  ir.Node{Type: "write", ConfigValues: map[string]any{"channel": uint32(30)}},
				State: s.Node("write1"),
			})
			sink2, _ := factory.Create(bCtx, rnode.Config{
				Node:  ir.Node{Type: "write", ConfigValues: map[string]any{"channel": uint32(40)}},
				State: s.Node("write2"),
			})
			fr := telem.Frame[uint32]{}
			fr = fr.Append(10, telem.NewSeriesV[float32](1.1, 2.2))
			fr = fr.Append(11, telem.NewSeriesSecondsTSV(100, 200))
			fr = fr.Append(20, telem.NewSeriesV[float64](3.3, 4.4))
			fr = fr.Append(21, telem.NewSeriesSecondsTSV(100, 200))
			s.Ingest(fr)
			source1.Next(bCtx, func(string) {})
			source2.Next(bCtx, func(string) {})
			Expect(s.Node("write1").RefreshInputs()).To(BeTrue())
			Expect(s.Node("write2").RefreshInputs()).To(BeTrue())
			sink1.Next(bCtx, func(string) {})
			sink2.Next(bCtx, func(string) {})
			outputFr, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outputFr.Get(30).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1.1, 2.2)))
			Expect(outputFr.Get(40).Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float64](3.3, 4.4)))
		})
	})
	Describe("Watermark Behavior", func() {
		It("Should prevent reprocessing same data through pipeline", func() {
			s := state.New(state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 50, DataType: telem.Uint32T, Index: 51},
					{Key: 60, DataType: telem.Uint32T, Index: 61},
				},
				ReactiveDeps: map[uint32][]string{
					50: {"source"},
				},
				Nodes: []ir.Node{
					{Key: "source", Type: "on", Outputs: types.Params{
						Keys: []string{ir.DefaultOutputParam}, Values: []types.Type{types.U32()}}},
					{Key: "sink", Type: "write", Inputs: types.Params{
						Keys: []string{ir.DefaultInputParam}, Values: []types.Type{types.U32()}}},
				},
				Edges: []ir.Edge{
					{Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam}},
				},
			})
			factory := rtelem.NewTelemFactory()
			source, _ := factory.Create(bCtx, rnode.Config{
				Node:  ir.Node{Type: "on", ConfigValues: map[string]any{"channel": uint32(50)}},
				State: s.Node("source"),
			})
			sink, _ := factory.Create(bCtx, rnode.Config{
				Node:  ir.Node{Type: "write", ConfigValues: map[string]any{"channel": uint32(60)}},
				State: s.Node("sink"),
			})
			fr := telem.Frame[uint32]{}
			fr = fr.Append(50, telem.NewSeriesV[uint32](777))
			fr = fr.Append(51, telem.NewSeriesSecondsTSV(999))
			s.Ingest(fr)
			source.Next(bCtx, func(string) {})
			Expect(s.Node("sink").RefreshInputs()).To(BeTrue())
			sink.Next(bCtx, func(string) {})
			firstWrite, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(firstWrite.Get(60).Series).To(HaveLen(1))
			var secondOutputChange bool
			source.Next(bCtx, func(string) { secondOutputChange = true })
			Expect(secondOutputChange).To(BeFalse())
			Expect(s.Node("sink").RefreshInputs()).To(BeFalse())
			sink.Next(bCtx, func(string) {})
			secondWrite, changed := s.FlushWrites(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse())
			Expect(secondWrite.Get(60).Series).To(BeEmpty())
		})
	})
})
