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
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// at returns a timestamp series of ts, with the alignment of its first sample.
func at(alignment telem.Alignment, ts ...telem.TimeStamp) telem.Series {
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

var _ = Describe("Join breaker", func() {
	It("Should drop no live stamp after the other input's first frame arrives, when "+
		"each primary frame comes first", func() {
		// Two 1 kHz inputs on two clocks, both streamed in frames of 40 samples (a
		// 25 Hz stream rate). The primary's frame arrives just before the other's. Live
		// judges the limit by arrival (D-378): only frame 0, before the other input has
		// arrived, may drop.
		const frame, frames = 40, 10
		primary := make([]telem.TimeStamp, frame*frames)
		other := make([]telem.TimeStamp, frame*frames)
		pv := make([]float64, len(primary))
		ov := make([]float64, len(other))
		for k := range primary {
			primary[k] = telem.TimeStamp(k) * ms
			other[k] = telem.TimeStamp(k)*ms + ms/2
			pv[k], ov[k] = float64(k), float64(-k)
		}
		history := newJoin(2, 0, 0)
		Expect(history.Push(0, values(pv...), at(0, primary...))).To(Succeed())
		Expect(history.Push(1, values(ov...), at(0, other...))).To(Succeed())
		var want join.Output
		history.Next(telem.TimeStampMax, &want)

		live := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			Live:      true,
		}))
		var (
			got   join.Output
			first int
		)
		for f := range frames {
			lo, hi := f*frame, (f+1)*frame
			Expect(live.Push(
				0,
				values(pv[lo:hi]...),
				at(telem.Alignment(lo), primary[lo:hi]...),
			)).To(Succeed())
			live.Next(primary[hi-1], &got)
			if f == 0 {
				Expect(got.Dropped).To(BeNumerically("<=", frame))
				first = got.Dropped
			}
			Expect(live.Push(1, values(ov[lo:hi]...), at(0, other[lo:hi]...))).
				To(Succeed())
		}
		Expect(want.Dropped).To(Equal(1))
		Expect(got.Dropped).To(Equal(first))
	})

	It("Should hold an input with no steady rate with no limit", func() {
		// A valve state written on change: intervals of 2 s, 13 s, 1 s, and 24 s.
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

	It("Should keep its buffers bounded while a live primary is silent", func() {
		// The primary has one sample, then its device goes quiet while another input
		// streams at 1 kHz in frames of 1000. Each frame, the live read passes the
		// newest primary stamp.
		j := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			Live:      true,
		}))
		Expect(j.Push(0, values(0), at(0, 0))).To(Succeed())
		const frame = 1000
		vs := make([]float64, frame)
		ts := make([]telem.TimeStamp, frame)
		var (
			out  join.Output
			next = ms
		)
		push := func(n int) {
			for range n {
				for k := range ts {
					ts[k], next = next, next+ms
				}
				Expect(j.Push(1, values(vs...), at(0, ts...))).To(Succeed())
				j.Next(0, &out)
				out.Reset()
			}
		}
		var before, after runtime.MemStats
		push(100)
		runtime.GC()
		runtime.ReadMemStats(&before)
		push(1000)
		runtime.GC()
		runtime.ReadMemStats(&after)
		runtime.KeepAlive(j)
		Expect(int64(after.HeapAlloc) - int64(before.HeapAlloc)).
			To(BeNumerically("<", 1<<20))
	})

	It("Should hold a sample within 3 times a median interval past 2^64/3", func() {
		j := newJoin(2, 0, 0)
		Expect(j.Push(1, values(1, 2), at(0, 0, 6_500_000_000_000_000_000))).
			To(Succeed())
		Expect(j.Push(0, values(1), at(0, 8_500_000_000_000_000_000))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		Expect(out.Dropped).To(BeZero())
	})

	It("Should give the run at the last stamp a time range that ends after it starts",
		func() {
			j := newJoin(1, 0, 0)
			Expect(j.Push(0, values(1), at(0, telem.TimeStampMax))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Runs).To(HaveLen(1))
			tr := out.Runs[0].Time.TimeRange
			Expect(tr.End).To(BeNumerically(">=", tr.Start))
		})

	It("Should join three inputs of mixed densities", func() {
		j := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Int8T, telem.Uint64T, telem.TimestampT},
			Primary:   1,
		}))
		Expect(j.Push(0, telem.NewSeriesV[int8](-1, -2), stamps(0, 1, 3))).
			To(Succeed())
		Expect(j.Push(2, telem.NewSeriesV[telem.TimeStamp](7, 8), stamps(0, 0.5, 2.5))).
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
		Expect(run.Values[0].Data).To(Equal(telem.NewSeriesV[int8](-1, -1, -2).Data))
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

	Describe("Seeded live and history", func() {
		// Each seed builds a primary and two other inputs with random rates, gaps,
		// and primary alignment jumps. Live pushes no input late: each other sample
		// arrives before the primary frame that reaches its stamp.
		for seed := range uint64(48) {
			It("Should join live and chunked reads to the history samples", func() {
				r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
				gen := func(n int, step telem.TimeStamp) []telem.TimeStamp {
					ts := make([]telem.TimeStamp, n)
					t := telem.TimeStamp(r.Int64N(int64(step) * 5))
					for i := range ts {
						d := step/2 + telem.TimeStamp(r.Int64N(int64(step)))
						if r.IntN(30) == 0 {
							d *= telem.TimeStamp(2 + r.IntN(20))
						}
						t += d
						ts[i] = t
					}
					return ts
				}
				dts := []telem.DataType{telem.Float64T, telem.Int8T, telem.Uint32T}
				p := r.IntN(3)
				var (
					ts   = make([][]telem.TimeStamp, 3)
					data = make([]telem.Series, 3)
				)
				for i := range ts {
					step := telem.TimeStamp(1+r.IntN(10)) * ms
					ts[i] = gen(200+r.IntN(400), step)
					n := len(ts[i])
					random := map[telem.DataType]func() telem.Series{
						telem.Float64T: func() telem.Series {
							vs := make([]float64, n)
							for k := range vs {
								vs[k] = r.Float64()
							}
							return telem.NewSeriesV(vs...)
						},
						telem.Int8T: func() telem.Series {
							vs := make([]int8, n)
							for k := range vs {
								vs[k] = int8(r.IntN(256) - 128)
							}
							return telem.NewSeriesV(vs...)
						},
						telem.Uint32T: func() telem.Series {
							vs := make([]uint32, n)
							for k := range vs {
								vs[k] = r.Uint32()
							}
							return telem.NewSeriesV(vs...)
						},
					}
					data[i] = random[dts[i]]()
				}
				// Primary alignments: a jump to a new domain now and then.
				align := make([]telem.Alignment, len(ts[p]))
				a := telem.NewAlignment(1, 0)
				for k := range align {
					if k > 0 && r.IntN(50) == 0 {
						a = telem.NewAlignment(a.DomainIndex()+1, 0)
					}
					align[k] = a
					a++
				}
				from := ts[p][r.IntN(len(ts[p]))]
				if r.IntN(4) == 0 {
					from = 0
				}
				newJ := func() *join.Join {
					return MustSucceed(join.New(join.Config{
						DataTypes: dts,
						Primary:   p,
						From:      from,
					}))
				}
				slice := func(i, lo, hi int) (telem.Series, telem.Series) {
					d := int(dts[i].Density())
					s := telem.Series{
						DataType: dts[i],
						Data:     data[i].Data[lo*d : hi*d],
					}
					t := telem.NewSeries(ts[i][lo:hi])
					if i == p {
						t.Alignment = align[lo]
					}
					return s, t
				}
				// pushPrimary pushes primary samples [lo, hi), split where the
				// alignments do not touch.
				pushPrimary := func(j *join.Join, lo, hi int) {
					start := lo
					for k := lo + 1; k <= hi; k++ {
						if k == hi || align[k] != align[k-1]+1 {
							s, t := slice(p, start, k)
							Expect(j.Push(p, s, t)).To(Succeed())
							start = k
						}
					}
				}
				push := func(j *join.Join, i, lo, hi int) {
					if i == p {
						pushPrimary(j, lo, hi)
						return
					}
					s, t := slice(i, lo, hi)
					Expect(j.Push(i, s, t)).To(Succeed())
				}

				history := newJ()
				for i := range ts {
					push(history, i, 0, len(ts[i]))
				}
				var want join.Output
				history.Next(telem.TimeStampMax, &want)
				wantFlat := flatten(want)

				// Chunked history: push a chunk of the input with the earliest
				// newest stamp, then pass the earliest newest stamp.
				chunked := newJ()
				var gotChunked join.Output
				pos := make([]int, 3)
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
					hi := min(len(ts[best]), pos[best]+1+r.IntN(60))
					push(chunked, best, pos[best], hi)
					pos[best] = hi
					until := telem.TimeStampMax
					ready := true
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
						chunked.Next(until, &gotChunked)
					}
				}
				chunked.Next(telem.TimeStampMax, &gotChunked)
				Expect(flatten(gotChunked)).To(Equal(wantFlat))
				Expect(gotChunked.Dropped).To(Equal(want.Dropped))

				// Live: primary frames, each preceded by the other samples up to its
				// last stamp, and a Next of its last stamp.
				live := newJ()
				var gotLive join.Output
				opos := make([]int, 3)
				for lo := 0; lo < len(ts[p]); {
					hi := min(len(ts[p]), lo+1+r.IntN(40))
					end := ts[p][hi-1]
					for i := range ts {
						if i == p {
							continue
						}
						for opos[i] < len(ts[i]) && ts[i][opos[i]] <= end {
							n := opos[i] + 1
							for n < len(ts[i]) && ts[i][n] <= end &&
								n-opos[i] < 1+r.IntN(30) {
								n++
							}
							push(live, i, opos[i], n)
							opos[i] = n
						}
					}
					pushPrimary(live, lo, hi)
					live.Next(end, &gotLive)
					lo = hi
				}
				Expect(flatten(gotLive)).To(Equal(wantFlat))
				Expect(gotLive.Dropped).To(Equal(want.Dropped))
			})
		}
	})
})
