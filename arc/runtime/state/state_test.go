package state_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("State", func() {
	Describe("Frame Ingestion", func() {
		It("Should correctly mark reactive deps as dirty", func() {
			cfg := state.Config{
				ReactiveDeps: map[uint32][]string{
					1: {"cat"},
					2: {"dog"},
				},
			}
			s := state.New(cfg)
			fr := telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](12))
			dirty := make(set.Set[string])
			s.Ingest(fr, func(nodeKey string) { dirty.Add(nodeKey) })
			Expect(dirty.Contains("cat")).To(BeTrue())
			Expect(dirty.Contains("dog")).To(BeFalse())
		})
		It("Should mark multiple nodes dirty for same channel", func() {
			cfg := state.Config{
				ReactiveDeps: map[uint32][]string{
					1: {"cat", "dog", "bird"},
				},
			}
			s := state.New(cfg)
			fr := telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](42))
			dirty := make(set.Set[string])
			s.Ingest(fr, func(nodeKey string) { dirty.Add(nodeKey) })
			Expect(dirty.Contains("cat")).To(BeTrue())
			Expect(dirty.Contains("dog")).To(BeTrue())
			Expect(dirty.Contains("bird")).To(BeTrue())
		})
		It("Should mark deps dirty for index channel", func() {
			cfg := state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 1, Index: 2},
				},
				ReactiveDeps: map[uint32][]string{
					2: {"indexListener"},
				},
			}
			s := state.New(cfg)
			fr := telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](99))
			dirty := make(set.Set[string])
			s.Ingest(fr, func(nodeKey string) { dirty.Add(nodeKey) })
			Expect(dirty.Contains("indexListener")).To(BeTrue())
		})
		It("Should mark deps dirty for both data and index channel", func() {
			cfg := state.Config{
				ChannelDigests: []state.ChannelDigest{
					{Key: 1, Index: 2},
				},
				ReactiveDeps: map[uint32][]string{
					1: {"dataListener"},
					2: {"indexListener"},
				},
			}
			s := state.New(cfg)
			fr := telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](7.5))
			dirty := make(set.Set[string])
			s.Ingest(fr, func(nodeKey string) { dirty.Add(nodeKey) })
			Expect(dirty.Contains("dataListener")).To(BeTrue())
			Expect(dirty.Contains("indexListener")).To(BeTrue())
		})
		It("Should handle frame with multiple channels", func() {
			cfg := state.Config{
				ReactiveDeps: map[uint32][]string{
					1: {"node1"},
					2: {"node2"},
					3: {"node3"},
				},
			}
			s := state.New(cfg)
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](1.1))
			fr = fr.Append(2, telem.NewSeriesV[float32](2.2))
			fr = fr.Append(3, telem.NewSeriesV[float32](3.3))
			dirty := make(set.Set[string])
			s.Ingest(fr, func(nodeKey string) { dirty.Add(nodeKey) })
			Expect(dirty.Contains("node1")).To(BeTrue())
			Expect(dirty.Contains("node2")).To(BeTrue())
			Expect(dirty.Contains("node3")).To(BeTrue())
		})
	})
	Describe("Channel Operations", func() {
		Describe("ReadChan", func() {
			It("Should read channel data after ingestion", func() {
				s := state.New(state.Config{Nodes: []ir.Node{{Key: "test"}}})
				fr := telem.UnaryFrame[uint32](10, telem.NewSeriesV[float32](1, 2, 3))
				s.Ingest(fr, func(string) {})
				n := s.Node("test")
				data, time, ok := n.ReadChan(10)
				Expect(ok).To(BeTrue())
				Expect(data.Series).To(HaveLen(1))
				Expect(data.Series[0]).To(telem.MatchSeries(telem.NewSeriesV[float32](1, 2, 3)))
				Expect(time.Series).To(BeEmpty())
			})
			It("Should read channel with index", func() {
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 5, Index: 6},
					},
					Nodes: []ir.Node{{Key: "test"}},
				}
				s := state.New(cfg)
				fr := telem.Frame[uint32]{}
				fr = fr.Append(5, telem.NewSeriesV[int32](100, 200))
				fr = fr.Append(6, telem.NewSeriesSecondsTSV(10, 20))
				s.Ingest(fr, func(string) {})
				n := s.Node("test")
				data, time, ok := n.ReadChan(5)
				Expect(ok).To(BeTrue())
				Expect(data.Series).To(HaveLen(1))
				Expect(data.Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](100, 200)))
				Expect(time.Series).To(HaveLen(1))
				Expect(time.Series[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)))
			})
			It("Should return false for non-existent channel", func() {
				s := state.New(state.Config{
					Nodes: []ir.Node{{Key: "test"}},
				})
				n := s.Node("test")
				_, _, ok := n.ReadChan(999)
				Expect(ok).To(BeFalse())
			})
			It("Should handle channel without index", func() {
				s := state.New(state.Config{Nodes: []ir.Node{{Key: "test"}}})
				fr := telem.UnaryFrame[uint32](7, telem.NewSeriesV[uint8](5, 10, 15))
				s.Ingest(fr, func(string) {})
				n := s.Node("test")
				_, time, ok := n.ReadChan(7)
				Expect(ok).To(BeTrue())
				Expect(time.Series).To(BeEmpty())
			})
		})
		Describe("WriteChan", func() {
			It("Should write channel data", func() {
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, Index: 2},
					},
					Nodes: []ir.Node{{Key: "writer"}},
				}
				s := state.New(cfg)
				n := s.Node("writer")
				dataToWrite := telem.NewSeriesV[float64](3.14, 2.71)
				timeToWrite := telem.NewSeriesSecondsTSV(100, 200)
				n.WriteChan(1, dataToWrite, timeToWrite)
				fr := telem.Frame[uint32]{}
				fr = s.FlushWrites(fr)
				Expect(len(fr.Get(1).Series)).To(Equal(1))
				Expect(len(fr.Get(2).Series)).To(Equal(1))
				Expect(fr.Get(1).Series[0]).To(telem.MatchSeries(dataToWrite))
				Expect(fr.Get(2).Series[0]).To(telem.MatchSeries(timeToWrite))
			})
			It("Should handle multiple channel writes", func() {
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 10, Index: 11},
						{Key: 20, Index: 21},
					},
					Nodes: []ir.Node{{Key: "writer"}},
				}
				s := state.New(cfg)
				n := s.Node("writer")
				n.WriteChan(10, telem.NewSeriesV[int32](42), telem.NewSeriesSecondsTSV(1))
				n.WriteChan(20, telem.NewSeriesV[int32](99), telem.NewSeriesSecondsTSV(2))
				fr := s.FlushWrites(telem.Frame[uint32]{})
				Expect(len(fr.Get(10).Series)).To(Equal(1))
				Expect(len(fr.Get(11).Series)).To(Equal(1))
				Expect(len(fr.Get(20).Series)).To(Equal(1))
				Expect(len(fr.Get(21).Series)).To(Equal(1))
			})
		})
		Describe("FlushWrites", func() {
			It("Should clear writes after flush", func() {
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, Index: 2},
					},
					Nodes: []ir.Node{{Key: "writer"}},
				}
				s := state.New(cfg)
				n := s.Node("writer")
				n.WriteChan(1, telem.NewSeriesV[float32](1.0), telem.NewSeriesSecondsTSV(1))
				fr1 := s.FlushWrites(telem.Frame[uint32]{})
				Expect(len(fr1.Get(1).Series)).To(Equal(1))
				fr2 := s.FlushWrites(telem.Frame[uint32]{})
				Expect(len(fr2.Get(1).Series)).To(Equal(0))
			})
			It("Should accumulate multiple writes before flush", func() {
				cfg := state.Config{
					ChannelDigests: []state.ChannelDigest{
						{Key: 1, Index: 2},
						{Key: 3, Index: 4},
					},
					Nodes: []ir.Node{{Key: "writer"}},
				}
				s := state.New(cfg)
				n := s.Node("writer")
				n.WriteChan(1, telem.NewSeriesV[float32](1.0), telem.NewSeriesSecondsTSV(1))
				n.WriteChan(3, telem.NewSeriesV[float32](2.0), telem.NewSeriesSecondsTSV(2))
				fr := s.FlushWrites(telem.Frame[uint32]{})
				Expect(len(fr.Get(1).Series)).To(Equal(1))
				Expect(len(fr.Get(3).Series)).To(Equal(1))
			})
		})
	})
	Describe("Input Alignment", func() {
		It("Should correctly align outputs of one node with inputs of another", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "first",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F32()},
						},
					},
					{
						Key: "second",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.F32()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "first", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "second", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			first := s.Node("first")
			second := s.Node("second")
			Expect(first.RefreshInputs()).To(BeFalse())
			Expect(second.RefreshInputs()).To(BeFalse())
			*first.Output(0) = telem.NewSeriesV[float32](1, 2, 3)
			*first.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			Expect(first.RefreshInputs()).To(BeFalse())
			Expect(second.RefreshInputs()).To(BeTrue())
			Expect(second.Input(0)).To(telem.MatchSeries(*first.Output(0)))
			Expect(second.InputTime(0)).To(telem.MatchSeries(*first.OutputTime(0)))
		})
		It("Should not trigger recalculation with empty output", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "src",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{Key: "dest"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "dest", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			src := s.Node("src")
			dest := s.Node("dest")
			*src.Output(0) = telem.NewSeriesV[int32]()
			*src.OutputTime(0) = telem.NewSeriesSecondsTSV()
			Expect(dest.RefreshInputs()).To(BeFalse())
		})
		It("Should track watermark to prevent reprocessing", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "producer",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
					},
					{Key: "consumer"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "producer", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "consumer", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			producer := s.Node("producer")
			consumer := s.Node("consumer")
			*producer.Output(0) = telem.NewSeriesV[float64](1.0)
			*producer.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(consumer.RefreshInputs()).To(BeTrue())
			Expect(consumer.RefreshInputs()).To(BeFalse())
		})
		It("Should handle multiple inputs to single node", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
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
					{Key: "target"},
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
			s := state.New(cfg)
			nodeA := s.Node("a")
			nodeB := s.Node("b")
			target := s.Node("target")
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
			cfg := state.Config{
				Nodes: []ir.Node{
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
					{Key: "target"},
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
			s := state.New(cfg)
			early := s.Node("early")
			late := s.Node("late")
			target := s.Node("target")
			*early.Output(0) = telem.NewSeriesV[int32](10)
			*early.OutputTime(0) = telem.NewSeriesSecondsTSV(100)
			*late.Output(0) = telem.NewSeriesV[int32](20)
			*late.OutputTime(0) = telem.NewSeriesSecondsTSV(200)
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.InputTime(0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(100)))
		})
		It("Should accumulate multiple series before triggering", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{Key: "sink"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			source := s.Node("source")
			sink := s.Node("sink")
			*source.Output(0) = telem.NewSeriesV[int32](1)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(sink.RefreshInputs()).To(BeTrue())
			*source.Output(0) = telem.NewSeriesV[int32](2)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
			Expect(sink.RefreshInputs()).To(BeTrue())
			Expect(sink.Input(0)).To(telem.MatchSeries(telem.NewSeriesV[int32](2)))
		})
		It("Should handle partial input updates", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
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
					{Key: "target"},
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
			s := state.New(cfg)
			nodeA := s.Node("a")
			nodeB := s.Node("b")
			target := s.Node("target")
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
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "src",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{Key: "dst"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "dst", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			src := s.Node("src")
			dst := s.Node("dst")
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
	})
	Describe("Watermark Regression Tests", func() {
		It("Should update all input watermarks on trigger", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
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
					{Key: "op"},
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
			s := state.New(cfg)
			lhs := s.Node("lhs")
			rhs := s.Node("rhs")
			op := s.Node("op")
			*lhs.Output(0) = telem.NewSeriesV[float64](1.5)
			*lhs.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			*rhs.Output(0) = telem.NewSeriesV[float64](2.5)
			*rhs.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			Expect(op.RefreshInputs()).To(BeTrue())
			Expect(op.RefreshInputs()).To(BeFalse())
		})
		It("Should not trigger recalculation when non-trigger input unchanged", func() {
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
					{Key: "compute"},
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
			s := state.New(cfg)
			nodeA := s.Node("a")
			nodeB := s.Node("b")
			compute := s.Node("compute")
			*nodeA.Output(0) = telem.NewSeriesV[int32](100)
			*nodeA.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			*nodeB.Output(0) = telem.NewSeriesV[int32](50)
			*nodeB.OutputTime(0) = telem.NewSeriesSecondsTSV(5)
			Expect(compute.RefreshInputs()).To(BeTrue())
			Expect(compute.RefreshInputs()).To(BeFalse())
			Expect(compute.RefreshInputs()).To(BeFalse())
		})
		It("Should correctly track watermarks with staggered timestamps", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
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
					{Key: "target"},
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
			s := state.New(cfg)
			early := s.Node("early")
			late := s.Node("late")
			target := s.Node("target")
			*early.Output(0) = telem.NewSeriesV[float32](1.0)
			*early.OutputTime(0) = telem.NewSeriesSecondsTSV(10)
			*late.Output(0) = telem.NewSeriesV[float32](2.0)
			*late.OutputTime(0) = telem.NewSeriesSecondsTSV(20)
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.RefreshInputs()).To(BeFalse())
			*early.Output(0) = telem.NewSeriesV[float32](3.0)
			*early.OutputTime(0) = telem.NewSeriesSecondsTSV(30)
			*late.Output(0) = telem.NewSeriesV[float32](4.0)
			*late.OutputTime(0) = telem.NewSeriesSecondsTSV(40)
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.RefreshInputs()).To(BeTrue())
			Expect(target.RefreshInputs()).To(BeFalse())
		})
		It("Should prevent non-trigger input from causing spurious triggers", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
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
					{Key: "processor"},
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
			s := state.New(cfg)
			nodeX := s.Node("x")
			nodeY := s.Node("y")
			processor := s.Node("processor")
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
						Key: "c",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{Key: "combiner"},
				},
				Edges: []ir.Edge{
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
			s := state.New(cfg)
			nodeA := s.Node("a")
			nodeB := s.Node("b")
			nodeC := s.Node("c")
			combiner := s.Node("combiner")
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
