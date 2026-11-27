// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join_test

import (
	"math/rand/v2"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// rawStamps returns a timestamp series of ts, with the alignment of its first sample.
func rawStamps(alignment telem.Alignment, ts ...telem.TimeStamp) telem.Series {
	s := telem.NewSeries(ts)
	s.Alignment = alignment
	return s
}

// flat is every joined sample of an output, with its run index.
type flat struct {
	stamps     []telem.TimeStamp
	alignments []telem.Alignment
	values     [][]byte
	runs       []int
}

func flatten(out join.Output) flat {
	var f flat
	for r, run := range out.Runs {
		for k, t := range times(run.Time) {
			f.stamps = append(f.stamps, t)
			f.alignments = append(f.alignments, run.Time.Alignment+telem.Alignment(k))
			var b []byte
			for _, v := range run.Values {
				d := int(v.DataType.Density())
				b = append(b, v.Data[k*d:(k+1)*d]...)
			}
			f.values = append(f.values, b)
			f.runs = append(f.runs, r)
		}
	}
	return f
}

const ms = telem.TimeStamp(telem.Millisecond)

// inputsOf returns an input of each data type of dts.
func inputsOf(dts []telem.DataType) []join.Input {
	inputs := make([]join.Input, len(dts))
	for i, dt := range dts {
		inputs[i] = join.Input{DataType: dt}
	}
	return inputs
}

func secs(s float64) telem.TimeStamp {
	return telem.TimeStamp(s * float64(telem.Second))
}

// pace reads ts in history the way a reader paces its inputs: it pushes a chunk of the
// input with the earliest newest stamp, then passes the earliest newest stamp of the
// inputs that have more. It returns the first failed push, or nil.
func pace(
	j *join.Join,
	ts [][]telem.TimeStamp,
	chunk int,
	out *join.Output,
) error {
	pos := make([]int, len(ts))
	for {
		best := -1
		for i := range ts {
			switch {
			case pos[i] == len(ts[i]):
			case pos[i] == 0:
				best = i
			case best == -1 || pos[best] > 0 &&
				ts[i][pos[i]-1] < ts[best][pos[best]-1]:
				best = i
			}
		}
		if best == -1 {
			break
		}
		hi := min(len(ts[best]), pos[best]+chunk)
		t := telem.NewSeries(ts[best][pos[best]:hi])
		t.Alignment = telem.Alignment(pos[best])
		if err := j.Push(
			best,
			telem.NewSeries(make([]float64, hi-pos[best])),
			t,
		); err != nil {
			return err
		}
		pos[best] = hi
		until, ready := telem.TimeStampMax, true
		for i := range ts {
			if pos[i] == 0 {
				ready = false
				break
			}
			if pos[i] < len(ts[i]) {
				until = min(until, ts[i][pos[i]-1])
			}
		}
		if ready {
			j.Next(until, out)
		}
	}
	j.Next(telem.TimeStampMax, out)
	return nil
}

var _ = Describe("Join", func() {
	Describe("Next", func() {
		It("Should start a new run when the alignments of the primary jump", func() {
			j := newJoin(2, 0, 0)
			Expect(j.Push(1, values(10, 20, 30), stamps(0, 1, 2, 3))).To(Succeed())
			Expect(j.Push(0, values(1, 2), stamps(0, 1, 2))).To(Succeed())
			Expect(j.Push(0, values(3), stamps(100, 3))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, runs := samples(out)
			Expect(got).To(Equal([]sample{
				{1, 0, []float64{1, 10}},
				{2, 1, []float64{2, 20}},
				{3, 100, []float64{3, 30}},
			}))
			Expect(runs).To(Equal([]int{0, 0, 1}))
		})

		It("Should give the run at the last stamp a time range that ends after it "+
			"starts",
			func() {
				j := newJoin(1, 0, 0)
				Expect(j.Push(0, values(1), rawStamps(0, telem.TimeStampMax))).
					To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				Expect(out.Runs).To(HaveLen(1))
				tr := out.Runs[0].Time.TimeRange
				Expect(tr.End).To(BeNumerically(">=", tr.Start))
			})

		It("Should join three inputs of mixed densities", func() {
			j := MustSucceed(join.New(join.Config{
				Inputs: inputsOf(
					[]telem.DataType{telem.Int8T, telem.Uint64T, telem.TimestampT},
				),
				Primary: 1,
			}))
			Expect(j.Push(0, telem.NewSeriesV[int8](-1, -2), stamps(0, 1, 3))).
				To(Succeed())
			Expect(j.Push(
				2,
				telem.NewSeriesV[telem.TimeStamp](7, 8),
				stamps(0, 0.5, 2.5),
			)).
				To(Succeed())
			Expect(j.Push(
				1,
				telem.NewSeriesV[uint64](1<<63, 1<<63+1, 1<<63+2),
				stamps(4, 1, 2, 3),
			)).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Runs).To(HaveLen(1))
			run := out.Runs[0]
			Expect(run.Values[0].Data).
				To(Equal(telem.NewSeriesV[int8](-1, -1, -2).Data))
			Expect(run.Values[1].Data).
				To(Equal(telem.NewSeriesV[uint64](1<<63, 1<<63+1, 1<<63+2).Data))
			Expect(run.Values[2].Data).
				To(Equal(telem.NewSeriesV[telem.TimeStamp](7, 7, 8).Data))
			Expect(run.Time.Alignment).To(Equal(telem.Alignment(4)))
		})

		It("Should start a new run after a reset and continue it without one", func() {
			j := newJoin(1, 0, 0)
			var out join.Output
			Expect(j.Push(0, values(1), stamps(0, 1))).To(Succeed())
			j.Next(telem.TimeStampMax, &out)
			Expect(j.Push(0, values(2), stamps(1, 2))).To(Succeed())
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Runs).To(HaveLen(1))
			out.Reset()
			Expect(j.Push(0, values(3), stamps(2, 3))).To(Succeed())
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Runs).To(HaveLen(1))
			Expect(out.Runs[0].Time.Alignment).To(Equal(telem.Alignment(2)))
			Expect(times(out.Runs[0].Time)).To(HaveLen(1))
		})

		It("Should treat From edge stamps as the design says", func() {
			j := newJoin(2, 0, telem.TimeStamp(5*telem.Second))
			Expect(j.Push(1, values(1, 2, 3), stamps(0, 1, 2, 3))).To(Succeed())
			Expect(j.Push(0, values(1, 2, 3), stamps(0, 1, 5, 7))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, runs := samples(out)
			Expect(got).To(Equal([]sample{{5, 1, []float64{2, 3}}}))
			Expect(runs).To(Equal([]int{0}))
			Expect(out.Dropped).To(Equal(1))
		})

		It("Should never reach the cap in a paced history read when the primary has a "+
			"gap beside a 1 kHz input", func() {
			var primary []telem.TimeStamp
			for s := range 11 {
				primary = append(primary, secs(float64(s)))
			}
			for s := 120; s <= 130; s++ {
				primary = append(primary, secs(float64(s)))
			}
			input := make([]telem.TimeStamp, 130_001)
			for k := range input {
				input[k] = telem.TimeStamp(k) * ms
			}
			j := newJoin(2, 0, 0)
			var out join.Output
			Expect(pace(j, [][]telem.TimeStamp{primary, input}, 4096, &out)).
				To(Succeed())
			Expect(out.Dropped).To(BeZero())
			got, _ := samples(out)
			Expect(got).To(HaveLen(len(primary)))
		})

		Describe("Hold limit", func() {
			It("Should drop the oldest intervals from the ring", func() {
				// 16 intervals of 1 s, then 9 of 10 s: the median of the last 16 is 10
				// s, so the limit is 30 s and a stamp 24 s after the last sample holds
				// it. The median of the first 16, or of all 25, is 1 s.
				ss := make([]float64, 0, 26)
				for i := range 17 {
					ss = append(ss, float64(i))
				}
				for i := range 9 {
					ss = append(ss, float64(26+10*i))
				}
				j := newJoin(2, 0, 0)
				Expect(j.Push(
					1,
					values(make([]float64, len(ss))...),
					stamps(0, ss...),
				)).
					To(Succeed())
				Expect(j.Push(0, values(1), stamps(0, 130))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				got, _ := samples(out)
				Expect(got).To(Equal([]sample{{130, 0, []float64{1, 0}}}))
			})

			DescribeTable("Should keep the hold limit while 3 in 4 of the last "+
				"intervals lie within a factor of 2 of their median, and drop it "+
				"below that",
				func(spans []float64, held bool) {
					ss := make([]float64, 0, len(spans)+1)
					ss = append(ss, 0)
					for _, span := range spans {
						ss = append(ss, ss[len(ss)-1]+span)
					}
					j := newJoin(2, 0, 0)
					Expect(j.Push(
						1,
						values(make([]float64, len(ss))...),
						stamps(0, ss...),
					)).
						To(Succeed())
					last := ss[len(ss)-1]
					Expect(j.Push(0, values(1), stamps(0, last+100))).To(Succeed())
					var out join.Output
					j.Next(telem.TimeStampMax, &out)
					got, _ := samples(out)
					Expect(got).To(HaveLen(map[bool]int{true: 1, false: 0}[held]))
				},
				Entry("12 of 16 within: steady, so 100 s past drops",
					[]float64{1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 1, 1}, false),
				Entry("11 of 16 within: no limit, so 100 s past holds",
					[]float64{1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 3, 1, 1}, true),
				Entry("3 of 4 within: steady", []float64{1, 1, 1, 5}, false),
				Entry("2 of 4 within: no limit", []float64{1, 1, 5, 5}, true),
			)

			It(
				"Should drop stamps long after an input stops that had one gap before "+
					"it stopped",
				func() {
					// A 1 Hz input with one 3 s gap among its last 16 intervals stops
					// at 18 s.
					ss := []float64{
						0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 14, 15, 16, 17, 18,
					}
					j := newJoin(2, 0, 0)
					Expect(j.Push(
						1,
						values(make([]float64, len(ss))...),
						stamps(0, ss...),
					)).
						To(Succeed())
					Expect(j.Push(0, values(1, 2), stamps(0, 19, 1000))).To(Succeed())
					var out join.Output
					j.Next(telem.TimeStampMax, &out)
					Expect(out.Dropped).To(Equal(1))
				},
			)

			DescribeTable("Should hold an input of Poisson stamps with no limit",
				func(run uint64) {
					r := rand.New(rand.NewPCG(run, 1))
					ss := make([]float64, 0, 33)
					ss = append(ss, 0)
					for range 32 {
						ss = append(ss, ss[len(ss)-1]+r.ExpFloat64())
					}
					j := newJoin(2, 0, 0)
					Expect(j.Push(
						1,
						values(make([]float64, len(ss))...),
						stamps(0, ss...),
					)).
						To(Succeed())
					Expect(j.Push(0, values(1), stamps(0, ss[len(ss)-1]+1000))).
						To(Succeed())
					var out join.Output
					j.Next(telem.TimeStampMax, &out)
					Expect(out.Dropped).To(BeZero())
				},
				Entry("run 0", uint64(0)),
				Entry("run 1", uint64(1)),
				Entry("run 2", uint64(2)),
				Entry("run 3", uint64(3)),
				Entry("run 4", uint64(4)),
				Entry("run 5", uint64(5)),
				Entry("run 6", uint64(6)),
				Entry("run 7", uint64(7)),
			)

			It("Should hold an input with no steady rate with no limit", func() {
				// A valve state written on change: intervals of 2 s, 13 s, 1 s, and 24
				// s.
				j := newJoin(2, 0, 0)
				Expect(j.Push(1, values(0, 1, 0, 1, 0), stamps(0, 0, 2, 15, 16, 40))).
					To(Succeed())
				Expect(j.Push(0, values(1, 2), stamps(0, 41, 70))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				got, _ := samples(out)
				Expect(got).To(Equal([]sample{
					{41, 0, []float64{1, 0}},
					{70, 1, []float64{2, 0}},
				}))
				Expect(out.Dropped).To(BeZero())
			})

			It("Should hold a sample within 3 times a median interval past "+
				"2^64/3", func() {
				j := newJoin(2, 0, 0)
				Expect(j.Push(
					1,
					values(1, 2),
					rawStamps(0, 0, 6_500_000_000_000_000_000),
				)).
					To(Succeed())
				Expect(j.Push(0, values(1), rawStamps(0, 8_500_000_000_000_000_000))).
					To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				Expect(out.Dropped).To(BeZero())
			})

			It("Should take 3 times the exact median of an even count of intervals",
				func() {
					// Intervals of 1 ns and 2 ns: the median is 1.5 ns, so the limit is
					// 4.5 ns, and a primary stamp 4 ns past the held sample holds.
					j := newJoin(2, 0, 0)
					Expect(j.Push(1, values(1, 2, 3), rawStamps(0, 0, 1, 3))).
						To(Succeed())
					Expect(j.Push(0, values(9), rawStamps(0, 7))).To(Succeed())
					var out join.Output
					j.Next(telem.TimeStampMax, &out)
					Expect(out.Dropped).To(BeZero())
				})

			It("Should drop a primary stamp far past an input's hold limit when the "+
				"stamps lie more than 2^63 ns apart (hold limit: 3 times the median "+
				"interval)", func() {
				// The input's intervals are 1 s, so its limit is 3 s. The primary stamp
				// lies about 9.4e18 ns after the input's last sample.
				const base = telem.TimeStamp(-4_700_000_000_000_000_000)
				sec := telem.TimeStamp(telem.Second)
				j := newJoin(2, 0, base)
				Expect(j.Push(
					1,
					values(1, 2, 3),
					rawStamps(0, base, base+sec, base+2*sec),
				)).
					To(Succeed())
				Expect(j.Push(0, values(9), rawStamps(0, 4_700_000_000_000_000_000))).
					To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				Expect(out.Dropped).To(Equal(1))
			})
		})
	})
})

// TestJoinAllocs checks the steady push loop allocates nothing per sample after its
// warm-up, live and in history.
func TestJoinAllocs(t *testing.T) {
	for _, live := range []bool{false, true} {
		const frames, size = 400, 40
		var now telem.TimeStamp
		j, err := join.New(join.Config{
			Inputs: []join.Input{{DataType: telem.Float64T}, {DataType: telem.Int16T}},
			Live:   live,
			Now:    func() telem.TimeStamp { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		type frame struct{ p, pt, o, ot telem.Series }
		fs := make([]frame, frames)
		for f := range fs {
			pt, ot := make([]telem.TimeStamp, size), make([]telem.TimeStamp, size/2)
			for k := range pt {
				pt[k] = telem.TimeStamp(f*size+k) * ms
			}
			for k := range ot {
				ot[k] = telem.TimeStamp(f*size+2*k) * ms
			}
			fs[f] = frame{
				p:  telem.NewSeries(make([]float64, size)),
				pt: telem.NewSeries(pt),
				o:  telem.NewSeries(make([]int16, size/2)),
				ot: telem.NewSeries(ot),
			}
			fs[f].pt.Alignment = telem.Alignment(f * size)
		}
		var out join.Output
		f := 0
		step := func() {
			fr := fs[f]
			f++
			now = telem.TimeStamp(f) * 40 * ms
			if err := j.Push(1, fr.o, fr.ot); err != nil {
				t.Fatal(err)
			}
			if err := j.Push(0, fr.p, fr.pt); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			j.Next(telem.TimeStamp(f*size-1)*ms, &out)
		}
		for range 50 {
			step()
		}
		if allocs := testing.AllocsPerRun(300, step); allocs != 0 {
			t.Errorf("live %v: %v allocations per frame", live, allocs)
		}
		if out.Dropped != 0 {
			t.Errorf("live %v: %d dropped", live, out.Dropped)
		}
	}
}
