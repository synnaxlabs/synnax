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
	"cmp"
	"context"
	"math"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	stltest "github.com/synnaxlabs/arc/stl/testutil"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// newWriteNode returns a function that writes to key through a write node, the way a
// flow program writes a channel.
func newWriteNode(
	ctx context.Context,
	s *channels.ProgramState,
	key uint32,
	dt telem.DataType,
) (func(data, time telem.Series), error) {
	t := types.FromTelem(dt)
	g := graph.Graph{
		Nodes: []graph.Node{{Key: "upstream"}, {Key: "sink"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"upstream": {"type": "producer"},
			"sink":     {"type": "write"},
		},
		Edges: graph.Edges{{
			Source: ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
		}},
		Functions: []ir.Function{
			{
				Key:     "producer",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: t}},
			},
			{
				Key:     "write",
				Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: t}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			},
		},
	}
	analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
	if !diagnostics.Ok() {
		return nil, diagnostics
	}
	host, err := channels.NewHost(ctx, nil, s, nil)
	if err != nil {
		return nil, err
	}
	state := rnode.New(analyzed)
	sink, err := host.Create(rnode.Config{
		Node: ir.Node{
			Type:   "write",
			Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: key}},
		},
		State: state.Node("sink"),
	})
	if err != nil {
		return nil, err
	}
	upstream := state.Node("upstream")
	nodeCtx := rnode.Context{Context: ctx, MarkChanged: func(int) {}}
	return func(data, time telem.Series) {
		*upstream.Output(0) = data
		*upstream.OutputTime(0) = time
		upstream.MarkFresh(0)
		sink.Next(nodeCtx)
	}, nil
}

// writeChannel writes data and its timestamps to key through a write node.
func writeChannel(s *channels.ProgramState, key uint32, data, time telem.Series) {
	GinkgoHelper()
	write := MustSucceed(newWriteNode(context.Background(), s, key, data.DataType))
	write(data, time)
}

// openHost binds the channel host functions to s and returns a runtime that calls them.
func openHost(s *channels.ProgramState) (*stltest.Runtime, *strings.ProgramState) {
	GinkgoHelper()
	ctx := context.Background()
	rt := stltest.NewRuntime(ctx)
	DeferCleanup(func(ctx context.Context) { Expect(rt.Close(ctx)).To(Succeed()) })
	ss := strings.NewProgramState()
	MustSucceed(channels.NewHost(ctx, rt.Underlying(), s, ss))
	rt.Passthrough(ctx, channels.Name)
	return rt, ss
}

// writeValue writes each sample of value to key through the host functions that a
// function body calls.
func writeValue(s *channels.ProgramState, key uint32, value telem.Series) {
	GinkgoHelper()
	ctx := context.Background()
	rt, ss := openHost(s)
	if value.DataType == telem.StringT {
		for v := range value.Samples() {
			h := stltest.U32(ss.Create(string(v)))
			rt.CallVoid(ctx, channels.Name, "write_str", stltest.U32(key), h)
		}
		return
	}
	for i := range int(value.Len()) {
		var fn string
		var arg uint64
		switch value.DataType {
		case telem.Int32T:
			fn, arg = "write_i32", stltest.I32(value.ValueAt[int32](i))
		case telem.Float32T:
			fn, arg = "write_f32", stltest.F32(value.ValueAt[float32](i))
		case telem.Float64T:
			fn, arg = "write_f64", stltest.F64(value.ValueAt[float64](i))
		default:
			Fail("unsupported data type " + string(value.DataType))
		}
		rt.CallVoid(ctx, channels.Name, fn, stltest.U32(key), arg)
	}
}

// readSeries returns the data and time series that an on node for key emits.
func readSeries(s *channels.ProgramState, key uint32) (data, time []telem.Series) {
	GinkgoHelper()
	ctx := context.Background()
	g := graph.Graph{
		Nodes:  []graph.Node{{Key: "source"}},
		Inputs: map[string]msgpack.EncodedJSON{"source": {"type": "on"}},
		Functions: []ir.Function{{
			Key:     "on",
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F32()}},
		}},
	}
	analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
	Expect(diagnostics.Ok()).To(BeTrue())
	state := rnode.New(analyzed).Node("source")
	source := MustSucceed(MustSucceed(channels.NewHost(ctx, nil, s, nil)).Create(
		rnode.Config{
			Node: ir.Node{
				Type:   "on",
				Inputs: types.Params{{Name: "channel", Type: types.U32(), Value: key}},
			},
			State: state,
		},
	))
	for {
		emitted := false
		source.Next(rnode.Context{
			Context:       ctx,
			ReserveStamps: stltest.ReserveStamps(0),
			MarkChanged:   func(int) { emitted = true },
		})
		if !emitted {
			return data, time
		}
		data = append(data, *state.Output(0))
		time = append(time, *state.OutputTime(0))
	}
}

// aligned returns s at alignment a, so that reads tell apart ingested series.
func aligned(s telem.Series, a telem.Alignment) telem.Series {
	s.Alignment = a
	return s
}

var _ = Describe("ProgramState", func() {
	var s *channels.ProgramState

	BeforeEach(func() {
		s = channels.NewProgramState([]channels.Digest{
			{Key: 1, DataType: telem.Float32T, Index: 2},
			{Key: 3, DataType: telem.Int32T},
			{Key: 5, DataType: telem.Float64T, Index: 6},
		})
	})

	Describe("NewProgramState", func() {
		It("Should initialize index mappings from digests", func() {
			cs := channels.NewProgramState([]channels.Digest{
				{Key: 10, Index: 11},
				{Key: 20, Index: 21},
			})
			writeChannel(
				cs,
				10,
				telem.NewSeriesV[float32](1.0),
				telem.NewSeriesSecondsTSV(100),
			)
			fr, _, changed := cs.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(11).Series).To(HaveLen(1))
		})

		It("Should handle nil digests", func() {
			cs := channels.NewProgramState(nil)
			Expect(cs).ToNot(BeNil())
			_, ok := cs.ReadValue(1)
			Expect(ok).To(BeFalse())
		})

		It("Should handle empty digests", func() {
			cs := channels.NewProgramState([]channels.Digest{})
			Expect(cs).ToNot(BeNil())
			_, ok := cs.ReadValue(1)
			Expect(ok).To(BeFalse())
		})

		It("Should ignore zero-value index in digests", func() {
			cs := channels.NewProgramState([]channels.Digest{{Key: 10, Index: 0}})
			writeValue(cs, 10, telem.NewSeriesV[int32](42))
			fr, _, changed := cs.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(10).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("Ingest", func() {
		It("Should buffer ingested frame data for later reads", func() {
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.5, 2.5)))
			ser := MustBeOk(s.ReadValue(1))
			Expect(ser).To(telem.MatchSeries(telem.NewSeriesV[float32](1.5, 2.5)))
		})

		It("Should accumulate multiple ingestions into MultiSeries", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(
				telem.UnaryFrame[uint32](3, aligned(telem.NewSeriesV[int32](2), 1)),
			)
			data, _ := readSeries(s, 3)
			Expect(data).To(HaveLen(2))
		})

		It("Should handle multiple channels in a single frame", func() {
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](10.0))
			fr = fr.Append(3, telem.NewSeriesV[int32](42))
			s.Ingest(fr)
			Expect(MustBeOk(s.ReadValue(1))).To(
				telem.MatchSeries(telem.NewSeriesV[float32](10.0)),
			)
			Expect(MustBeOk(s.ReadValue(3))).To(
				telem.MatchSeries(telem.NewSeriesV[int32](42)),
			)
		})

		It("Should handle ingestion of channels not in digests", func() {
			s.Ingest(telem.UnaryFrame[uint32](999, telem.NewSeriesV(1.0)))
			ser := MustBeOk(s.ReadValue(999))
			Expect(ser).To(telem.MatchSeries(telem.NewSeriesV(1.0)))
		})

		It("Should handle an empty frame without panicking", func() {
			Expect(func() { s.Ingest(telem.Frame[uint32]{}) }).ToNot(Panic())
		})

		It("Should handle series with boundary float values", func() {
			s.Ingest(telem.UnaryFrame[uint32](
				5,
				telem.NewSeriesV(
					math.MaxFloat64,
					math.SmallestNonzeroFloat64,
					math.Inf(1),
					math.Inf(-1),
				),
			))
			ser := MustBeOk(s.ReadValue(5))
			Expect(ser.Len()).To(Equal(int64(4)))
		})

		It("Should handle series with NaN values", func() {
			s.Ingest(telem.UnaryFrame[uint32](5, telem.NewSeriesV(math.NaN())))
			ser := MustBeOk(s.ReadValue(5))
			Expect(ser.Len()).To(Equal(int64(1)))
			Expect(math.IsNaN(ser.ValueAt[float64](0))).To(BeTrue())
		})

		It("Should handle series with max/min integer values", func() {
			s.Ingest(telem.UnaryFrame[uint32](
				3, telem.NewSeriesV[int32](math.MaxInt32, math.MinInt32, 0),
			))
			ser := MustBeOk(s.ReadValue(3))
			Expect(ser.ValueAt[int32](0)).To(Equal(int32(math.MaxInt32)))
			Expect(ser.ValueAt[int32](1)).To(Equal(int32(math.MinInt32)))
			Expect(ser.ValueAt[int32](2)).To(Equal(int32(0)))
		})

		It("Should handle empty series", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32]()))
			ser := MustBeOk(s.ReadValue(3))
			Expect(ser.Len()).To(Equal(int64(0)))
		})
	})

	Describe("ReadValue", func() {
		It("Should return an empty series ingested for a channel", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.Series{}))
			Expect(MustBeOk(s.ReadValue(3)).Len()).To(BeZero())
		})

		It("Should return the last series from the read buffer", func() {
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.0)))
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](2.0)))
			ser := MustBeOk(s.ReadValue(1))
			Expect(ser).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})

		It("Should return false for an unknown channel", func() {
			_, ok := s.ReadValue(999)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for a channel with no data", func() {
			s = channels.NewProgramState([]channels.Digest{{Key: 10}})
			_, ok := s.ReadValue(10)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for channel key zero", func() {
			_, ok := s.ReadValue(0)
			Expect(ok).To(BeFalse())
		})

		It("Should return the latest series after many ingestions", func() {
			for i := range 100 {
				s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV(int32(i))))
			}
			ser := MustBeOk(s.ReadValue(3))
			Expect(ser.ValueAt[int32](0)).To(Equal(int32(99)))
		})
	})

	Describe("Writes from a function body", func() {
		It("Should buffer the write for later flushing", func() {
			writeValue(s, 3, telem.NewSeriesV[int32](100))
			fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[int32](100)),
			)
		})

		It("Should auto-write index channel for indexed channels", func() {
			writeValue(s, 1, telem.NewSeriesV[float32](5.0))
			fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series[0].DataType).To(Equal(telem.TimestampT))
		})

		It("Should not write to an index for non-indexed channels", func() {
			writeValue(s, 3, telem.NewSeriesV[int32](50))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(3).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})

		It("Should accumulate multiple writes in same cycle", func() {
			writeValue(s, 3, telem.NewSeriesV[int32](1))
			writeValue(s, 3, telem.NewSeriesV[int32](2))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[int32](1, 2)),
			)
		})

		It("Should handle writes to channels not in digests", func() {
			writeValue(s, 888, telem.NewSeriesV(3.14))
			fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(888).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV(3.14)),
			)
		})
	})

	Describe("Index stamping", func() {
		It("Should stamp one sample per data sample, 1ns apart", func() {
			writeValue(s, 1, telem.NewSeriesV[float32](1, 2, 3))
			fr, highest, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(2).Series[0]).To(telem.MatchSeries(telem.NewSeriesV(
				telem.TimeStamp(flushNow),
				telem.TimeStamp(flushNow+1),
				telem.TimeStamp(flushNow+2),
			)))
			Expect(highest).To(Equal(telem.TimeStamp(flushNow + 2)))
		})

		It("Should start every member of a group at the same stamp", func() {
			grouped := channels.NewProgramState([]channels.Digest{
				{Key: 1, DataType: telem.Float32T, Index: 2},
				{Key: 3, DataType: telem.Float32T, Index: 2},
			})
			writeValue(grouped, 1, telem.NewSeriesV[float32](1, 2))
			writeValue(grouped, 3, telem.NewSeriesV[float32](3, 4))
			fr, _, _ := grouped.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(2).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series[0]).To(telem.MatchSeries(telem.NewSeriesV(
				telem.TimeStamp(flushNow),
				telem.TimeStamp(flushNow+1),
			)))
		})

		It("Should keep the timestamps a write supplied", func() {
			writeChannel(
				s,
				1,
				telem.NewSeriesV[float32](1, 2),
				telem.NewSeriesSecondsTSV(7, 8),
			)
			fr, highest, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(2).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesSecondsTSV(7, 8)),
			)
			Expect(highest).To(Equal(telem.TimeStamp(0)))
		})

		It("Should not stamp a channel with no index", func() {
			writeValue(s, 3, telem.NewSeriesV[int32](1))
			fr, highest, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(0).Series).To(BeEmpty())
			Expect(highest).To(Equal(telem.TimeStamp(0)))
		})
	})

	Describe("Mixed writes", func() {
		supplied := flushNow - 5*telem.SecondTS

		DescribeTable(
			"Should give a body write its own stamp and order values by time",
			func(bodyFirst bool) {
				write := func() {
					writeChannel(
						s,
						1,
						telem.NewSeriesV[float32](1),
						telem.NewSeriesV(supplied),
					)
				}
				if !bodyFirst {
					write()
				}
				writeValue(s, 1, telem.NewSeriesV[float32](2))
				if bodyFirst {
					write()
				}
				fr, highest, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
				Expect(fr.Get(1).Series[0]).To(
					telem.MatchSeries(telem.NewSeriesV[float32](1, 2)),
				)
				Expect(fr.Get(2).Series[0]).To(
					telem.MatchSeries(telem.NewSeriesV(supplied, flushNow)),
				)
				Expect(highest).To(Equal(flushNow))
			},
			Entry("flow write first", false),
			Entry("body write first", true),
		)

		It("Should put a body write before a supplied stamp later than now", func() {
			later := flushNow + 5*telem.SecondTS
			writeChannel(s, 1, telem.NewSeriesV[float32](1), telem.NewSeriesV(later))
			writeValue(s, 1, telem.NewSeriesV[float32](2))
			fr, highest, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(1).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[float32](2, 1)),
			)
			Expect(fr.Get(2).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV(flushNow, later)),
			)
			Expect(highest).To(Equal(flushNow))
		})

		DescribeTable(
			"Should keep the write order of a body write and a supplied write at now",
			func(bodyFirst bool, values []float32) {
				write := func() {
					writeChannel(
						s,
						1,
						telem.NewSeriesV[float32](0),
						telem.NewSeriesV(flushNow),
					)
				}
				if !bodyFirst {
					write()
				}
				writeValue(s, 1, telem.NewSeriesV[float32](0.7))
				if bodyFirst {
					write()
				}
				fr, highest, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
				Expect(fr.Get(1).Series[0]).To(
					telem.MatchSeries(telem.NewSeriesV(values...)),
				)
				Expect(fr.Get(2).Series[0]).To(
					telem.MatchSeries(telem.NewSeriesV(flushNow, flushNow+1)),
				)
				Expect(highest).To(Equal(flushNow + 1))
			},
			Entry("body write first", true, []float32{0.7, 0}),
			Entry("supplied write first", false, []float32{0, 0.7}),
		)

		It("Should move equal supplied stamps 1ns apart in write order", func() {
			writeChannel(s, 1, telem.NewSeriesV[float32](1), telem.NewSeriesV(flushNow))
			writeChannel(s, 1, telem.NewSeriesV[float32](2), telem.NewSeriesV(flushNow))
			fr, highest, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(1).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[float32](1, 2)),
			)
			Expect(fr.Get(2).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV(flushNow, flushNow+1)),
			)
			Expect(highest).To(Equal(flushNow + 1))
		})

		It("Should order supplied stamps from separate writes by time", func() {
			writeChannel(
				s,
				1,
				telem.NewSeriesV[float32](1),
				telem.NewSeriesSecondsTSV(20),
			)
			writeChannel(
				s,
				1,
				telem.NewSeriesV[float32](2),
				telem.NewSeriesSecondsTSV(10),
			)
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(1).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[float32](2, 1)),
			)
			Expect(fr.Get(2).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)),
			)
		})

		It("Should stamp a body write after the previous frame's last stamp", func() {
			later := flushNow + 5*telem.SecondTS
			writeChannel(s, 1, telem.NewSeriesV[float32](1), telem.NewSeriesV(later))
			s.Flush(telem.Frame[uint32]{}, flushNow)
			writeValue(s, 1, telem.NewSeriesV[float32](2))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(2).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV(later + 1)),
			)
		})

		It("Should order a string channel by time, then write order", func() {
			str := channels.NewProgramState([]channels.Digest{
				{Key: 7, DataType: telem.StringT, Index: 8},
			})
			writeChannel(
				str,
				7,
				telem.NewSeriesV("late"),
				telem.NewSeriesSecondsTSV(20),
			)
			writeChannel(
				str,
				7,
				telem.NewSeriesV("early"),
				telem.NewSeriesSecondsTSV(10),
			)
			writeValue(str, 7, telem.NewSeriesV("body"))
			writeChannel(str, 7, telem.NewSeriesV("tie"), telem.NewSeriesV(flushNow))
			fr, _, _ := str.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(7).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV("early", "late", "body", "tie")),
			)
			Expect(fr.Get(8).Series[0]).To(telem.MatchSeries(telem.NewSeriesV(
				10*telem.SecondTS,
				20*telem.SecondTS,
				flushNow,
				flushNow+1,
			)))
		})
	})

	Describe("Stamp invariants", func() {
		// Each order mixes body writes (b) with supplied writes stamped behind now (l),
		// at now (n), and ahead of now (a). A second cycle, 1ns later, makes one body
		// write.
		DescribeTable(
			"Should order samples by time, then by write order",
			func(order string) {
				type sample struct {
					stamp telem.TimeStamp
					value float64
				}
				const now telem.TimeStamp = 1000
				var (
					written       []sample
					synthesized   []bool
					behind, ahead = now - 100, now + 100
				)
				for i, op := range order {
					value := float64(i + 1)
					stamp := now
					switch op {
					case 'b':
						writeValue(s, 5, telem.NewSeriesV(value))
					case 'l':
						stamp, behind = behind, behind+1
					case 'a':
						stamp, ahead = ahead, ahead+1
					}
					if op != 'b' {
						writeChannel(
							s,
							5,
							telem.NewSeriesV(value),
							telem.NewSeriesV(stamp),
						)
					}
					written = append(written, sample{stamp, value})
					synthesized = append(synthesized, op == 'b')
				}
				positions := make([]int, len(written))
				for i := range positions {
					positions[i] = i
				}
				slices.SortStableFunc(positions, func(a, b int) int {
					return cmp.Compare(written[a].stamp, written[b].stamp)
				})
				var (
					expected []sample
					highest  telem.TimeStamp
				)
				for i, p := range positions {
					want := written[p]
					if i > 0 && want.stamp <= expected[i-1].stamp {
						want.stamp = expected[i-1].stamp + 1
						synthesized[p] = true
					}
					if synthesized[p] {
						highest = want.stamp
					}
					expected = append(expected, want)
				}
				fr, actualHighest, _ := s.Flush(telem.Frame[uint32]{}, now)
				values, index := fr.Get(5).Series[0], fr.Get(6).Series[0]
				Expect(index.Len()).To(BeEquivalentTo(len(order)))
				Expect(values.Len()).To(Equal(index.Len()))
				for i, want := range expected {
					Expect(sample{
						index.ValueAt[telem.TimeStamp](i),
						values.ValueAt[float64](i),
					}).To(Equal(want))
				}
				Expect(actualHighest).To(Equal(highest))
				last := expected[len(expected)-1].stamp
				writeValue(s, 5, telem.NewSeriesV(0.0))
				fr, _, _ = s.Flush(telem.Frame[uint32]{}, now+1)
				Expect(fr.Get(6).Series[0]).
					To(telem.MatchSeries(telem.NewSeriesV(max(now+1, last+1))))
			},
			writeOrders(),
		)
	})

	Describe("Flush", func() {
		It("Should return false when no writes are buffered", func() {
			_, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeFalse())
		})

		It("Should extract buffered writes and clear the write buffer", func() {
			writeValue(s, 1, telem.NewSeriesV[float32](9.9))
			fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			_, _, changed = s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeFalse())
		})

		It("Should keep flushed frames apart from later writes", func() {
			writeValue(s, 1, telem.NewSeriesV[float32](1))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			writeValue(s, 1, telem.NewSeriesV[float32](2))
			s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(1).Series[0]).To(telem.MatchSeriesDataV[float32](1))
		})

		It("Should append to an existing frame", func() {
			existing := telem.UnaryFrame[uint32](100, telem.NewSeriesV[int32](1))
			writeValue(s, 3, telem.NewSeriesV[int32](2))
			fr, _, changed := s.Flush(existing, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series).To(HaveLen(1))
			Expect(fr.Get(3).Series).To(HaveLen(1))
		})

		It("Should flush writes for multiple channels", func() {
			writeValue(s, 1, telem.NewSeriesV[float32](1.0))
			writeValue(s, 3, telem.NewSeriesV[int32](2))
			fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(fr.Get(3).Series).To(HaveLen(1))
		})
	})

	Describe("Reads through an on node", func() {
		It("Should read a channel without an index", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1, 2)))
			data, _ := readSeries(s, 3)
			Expect(data).To(HaveLen(1))
			Expect(data[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](1, 2)))
		})

		It("Should read the data and time of an indexed channel", func() {
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](1.0, 2.0))
			fr = fr.Append(2, telem.NewSeriesSecondsTSV(10, 20))
			s.Ingest(fr)
			data, time := readSeries(s, 1)
			Expect(data).To(HaveLen(1))
			Expect(time).To(HaveLen(1))
			Expect(time[0]).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(10, 20)))
		})

		It("Should read nothing when index data is missing", func() {
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.0)))
			Expect(readSeries(s, 1)).To(BeEmpty())
		})

		It("Should read nothing for an unknown channel", func() {
			Expect(readSeries(s, 999)).To(BeEmpty())
		})

		It("Should read data accumulated across multiple ingestions", func() {
			fr1 := telem.Frame[uint32]{}
			fr1 = fr1.Append(1, telem.NewSeriesV[float32](1.0))
			fr1 = fr1.Append(2, telem.NewSeriesSecondsTSV(10))
			s.Ingest(fr1)
			fr2 := telem.Frame[uint32]{}
			fr2 = fr2.Append(1, aligned(telem.NewSeriesV[float32](2.0), 1))
			fr2 = fr2.Append(2, aligned(telem.NewSeriesSecondsTSV(20), 1))
			s.Ingest(fr2)
			data, time := readSeries(s, 1)
			Expect(data).To(HaveLen(2))
			Expect(time).To(HaveLen(2))
		})
	})

	Describe("Writes through a write node", func() {
		It("Should buffer both data and time series", func() {
			data := telem.NewSeriesV[float32](1.0, 2.0)
			time := telem.NewSeriesSecondsTSV(100, 200)
			writeChannel(s, 1, data, time)
			fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series[0]).To(telem.MatchSeries(data))
			Expect(fr.Get(2).Series[0]).To(telem.MatchSeries(time))
		})

		It("Should write time to index channel", func() {
			data := telem.NewSeriesV(9.9)
			time := telem.NewSeriesSecondsTSV(500)
			writeChannel(s, 5, data, time)
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(6).Series).To(HaveLen(1))
		})

		It("Should not write time to index for non-indexed channels", func() {
			data := telem.NewSeriesV[int32](42)
			time := telem.NewSeriesSecondsTSV(100)
			writeChannel(s, 3, data, time)
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(3).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("ClearReads", func() {
		It("Should preserve the latest series for each channel", func() {
			for i := range 3 {
				s.Ingest(telem.UnaryFrame[uint32](3, aligned(
					telem.NewSeriesV(int32(i+1)),
					telem.Alignment(i),
				)))
			}
			s.ClearReads()
			data, _ := readSeries(s, 3)
			Expect(data).To(HaveLen(1))
			Expect(data[0]).To(telem.MatchSeriesDataV[int32](3))
		})

		It("Should be a no-op for channels with a single series", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.ClearReads()
			data, _ := readSeries(s, 3)
			Expect(data).To(HaveLen(1))
		})

		It("Should be a no-op for empty state", func() {
			Expect(func() { s.ClearReads() }).ToNot(Panic())
		})

		It("Should handle multiple channels independently", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(
				telem.UnaryFrame[uint32](3, aligned(telem.NewSeriesV[int32](2), 1)),
			)
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](10.0))
			fr = fr.Append(2, telem.NewSeriesSecondsTSV(100))
			s.Ingest(fr)
			s.ClearReads()
			d3, _ := readSeries(s, 3)
			Expect(d3).To(HaveLen(1))
			Expect(d3[0]).To(telem.MatchSeriesDataV[int32](2))
			d1, _ := readSeries(s, 1)
			Expect(d1).To(HaveLen(1))
		})

		It("Should allow new ingestions after clear", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(
				telem.UnaryFrame[uint32](3, aligned(telem.NewSeriesV[int32](2), 1)),
			)
			s.ClearReads()
			s.Ingest(
				telem.UnaryFrame[uint32](3, aligned(telem.NewSeriesV[int32](3), 2)),
			)
			data, _ := readSeries(s, 3)
			Expect(data).To(HaveLen(2))
		})

		It("Should use fresh allocation when capacity exceeds threshold", func() {
			for i := range 100 {
				s.Ingest(telem.UnaryFrame[uint32](
					3,
					aligned(telem.NewSeriesV[int32](1), telem.Alignment(i)),
				))
			}
			s.ClearReads()
			data, _ := readSeries(s, 3)
			Expect(data).To(HaveLen(1))
		})
	})

	Describe("Fixed host writes", func() {
		DescribeTable(
			"Should write each fixed type and auto-index timestamps",
			func(fn string, arg uint64, expected telem.Series) {
				rt, _ := openHost(s)
				rt.CallVoid(
					context.Background(),
					channels.Name,
					fn,
					stltest.U32(1),
					arg,
				)
				fr, _, changed := s.Flush(telem.Frame[uint32]{}, flushNow)
				Expect(changed).To(BeTrue())
				Expect(fr.Get(1).Series).To(HaveLen(1))
				Expect(fr.Get(1).Series[0]).To(telem.MatchSeriesData(expected))
				Expect(fr.Get(2).Series).To(HaveLen(1))
				Expect(fr.Get(2).Series[0].DataType).To(Equal(telem.TimestampT))
			},
			Entry("u8", "write_u8", stltest.U32(42), telem.NewSeriesV[uint8](42)),
			Entry("u16", "write_u16", stltest.U32(42), telem.NewSeriesV[uint16](42)),
			Entry("u32", "write_u32", stltest.U32(42), telem.NewSeriesV[uint32](42)),
			Entry("u64", "write_u64", stltest.U64(42), telem.NewSeriesV[uint64](42)),
			Entry("i8", "write_i8", stltest.I32(-4), telem.NewSeriesV[int8](-4)),
			Entry("i16", "write_i16", stltest.I32(-4), telem.NewSeriesV[int16](-4)),
			Entry("i32", "write_i32", stltest.I32(-4), telem.NewSeriesV[int32](-4)),
			Entry("i64", "write_i64", stltest.I64(-4), telem.NewSeriesV[int64](-4)),
			Entry("f32", "write_f32", stltest.F32(1.5), telem.NewSeriesV[float32](1.5)),
			Entry("f64", "write_f64", stltest.F64(1.5), telem.NewSeriesV(1.5)),
		)

		It("Should not write index for channels without one", func() {
			writeValue(s, 3, telem.NewSeriesV[int32](10))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(3).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("Write accumulation metadata", func() {
		It("Should merge time ranges across multiple writes", func() {
			ser1 := telem.NewSeriesV[int32](1)
			ser1.TimeRange = telem.TimeRange{Start: 100, End: 200}
			ser2 := telem.NewSeriesV[int32](2)
			ser2.TimeRange = telem.TimeRange{Start: 50, End: 300}
			writeChannel(s, 3, ser1, telem.NewSeriesSecondsTSV(1))
			writeChannel(s, 3, ser2, telem.NewSeriesSecondsTSV(2))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(3).Series[0].TimeRange.Start).To(Equal(telem.TimeStamp(50)))
			Expect(fr.Get(3).Series[0].TimeRange.End).To(Equal(telem.TimeStamp(300)))
		})
	})

	Describe("Ingest then Flush roundtrip", func() {
		It("Should not cross-contaminate reads and writes", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](10)))
			writeValue(s, 3, telem.NewSeriesV[int32](20))
			readSer := MustBeOk(s.ReadValue(3))
			Expect(readSer.ValueAt[int32](0)).To(Equal(int32(10)))
			fr, _, _ := s.Flush(telem.Frame[uint32]{}, flushNow)
			Expect(fr.Get(3).Series[0].ValueAt[int32](0)).To(Equal(int32(20)))
		})
	})
})

// writeOrders returns an entry for every order of one to four writes. Each write is a
// body write (b) or a supplied write stamped behind now (l), at now (n), or ahead (a).
func writeOrders() []TableEntry {
	var (
		entries []TableEntry
		orders  = []string{""}
	)
	for range 4 {
		var next []string
		for _, order := range orders {
			for _, op := range "blna" {
				next = append(next, order+string(op))
			}
		}
		orders = next
		for _, order := range orders {
			entries = append(entries, Entry(order, order))
		}
	}
	return entries
}
