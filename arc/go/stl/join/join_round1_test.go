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
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func secs(s float64) telem.TimeStamp {
	return telem.TimeStamp(s * float64(telem.Second))
}

// pace reads ts in history the way D-378 (c) says a reader paces its inputs: it pushes
// a chunk of the input with the earliest newest stamp, then passes the earliest newest
// stamp of the inputs that have more. It returns the first failed push, or nil.
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

var _ = Describe("Join break", func() {
	DescribeTable(
		"Should hold an input whose stamps lag the primary's while its frames arrive, "+
			"when From is the live open (D-396 amended: a 1 s skew stays held)",
		func(lag float64) {
			var now telem.TimeStamp
			j := MustSucceed(join.New(join.Config{
				DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
				From:      secs(10),
				Live:      true,
				Now:       func() telem.TimeStamp { return now },
			}))
			const n = 40
			drops := make([]int, n)
			var out join.Output
			for f := range n {
				now = secs(10 + 0.04*float64(f+1))
				primary, input := make([]float64, 40), make([]float64, 20)
				for k := range primary {
					primary[k] = 10 + 0.001*float64(40*f+k)
				}
				for k := range input {
					input[k] = 10 - lag + 0.002*float64(20*f+k)
				}
				Expect(j.Push(
					0,
					values(primary...),
					stamps(telem.Alignment(40*f), primary...),
				)).To(Succeed())
				before := out.Dropped
				j.Next(secs(primary[39]), &out)
				drops[f] = out.Dropped - before
				Expect(j.Push(1, values(input...), stamps(0, input...))).To(Succeed())
			}
			want := make([]int, n)
			want[0] = 40
			Expect(drops).To(Equal(want))
		},
		Entry("200 ms of latency", 0.2),
		Entry("a clock 1 s behind", 1.0),
	)

	It("Should never reach the cap in a paced history read when the primary has a "+
		"gap beside a 1 kHz input (D-378 (c): history at the largest chunk size "+
		"never reaches the cap)", func() {
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

	It("Should join an on-change input with no limit live as in history when its "+
		"samples arrive in stamp order (D-384: it holds with no limit; D-378 (3): "+
		"outputs are equal for in-order arrival)", func() {
		other := []float64{0, 2, 15, 16, 40}
		history := newJoin(2, 0, 0)
		primary := make([]float64, 46)
		for s := range primary {
			primary[s] = float64(s)
		}
		Expect(history.Push(0, values(primary...), stamps(0, primary...))).
			To(Succeed())
		Expect(history.Push(1, values(1, 2, 3, 4, 5), stamps(0, other...))).
			To(Succeed())
		var want join.Output
		history.Next(telem.TimeStampMax, &want)

		var now telem.TimeStamp
		live := MustSucceed(join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			Live:      true,
			Now:       func() telem.TimeStamp { return now },
		}))
		var got join.Output
		o := 0
		for _, s := range primary {
			now = secs(s)
			if o < len(other) && other[o] == s {
				Expect(live.Push(1, values(float64(o+1)), stamps(0, s))).
					To(Succeed())
				o++
			}
			Expect(live.Push(0, values(s), stamps(telem.Alignment(s), s))).
				To(Succeed())
			live.Next(secs(s), &got)
		}
		wantSamples, _ := samples(want)
		gotSamples, _ := samples(got)
		Expect(got.Dropped).To(Equal(want.Dropped))
		Expect(gotSamples).To(Equal(wantSamples))
	})

	It("Should take 3 times the exact median of an even count of intervals "+
		"(D-372: the limit is 3 times the median)", func() {
		// Intervals of 1 ns and 2 ns: the median is 1.5 ns, so the limit is 4.5 ns,
		// and a primary stamp 4 ns past the held sample holds.
		j := newJoin(2, 0, 0)
		Expect(j.Push(1, values(1, 2, 3), at(0, 0, 1, 3))).To(Succeed())
		Expect(j.Push(0, values(9), at(0, 7))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		Expect(out.Dropped).To(BeZero())
	})

	Describe("Seeded live in stamp order", func() {
		// Each seed builds a primary and two steady inputs at random rates, each
		// sample in its own frame that arrives at its stamp, with random stops. Live
		// then owes the history output exactly.
		for seed := range uint64(48) {
			It("Should join a live read in stamp order to the history samples",
				func() {
					r := rand.New(rand.NewPCG(seed, seed^0x51ed270b27c1a3))
					gen := func(start, step telem.TimeStamp, n int) []telem.TimeStamp {
						ts := make([]telem.TimeStamp, n)
						t := start
						for i := range ts {
							d := step*3/4 + telem.TimeStamp(r.Int64N(int64(step)/2))
							if r.IntN(80) == 0 {
								d *= telem.TimeStamp(4 + r.IntN(30))
							}
							t += d
							ts[i] = t
						}
						return ts
					}
					ts := make([][]telem.TimeStamp, 3)
					for i := range ts {
						step := telem.TimeStamp(1+r.IntN(10)) * ms
						start := telem.TimeStamp(0)
						if i == 0 {
							start = 200 * ms
						}
						ts[i] = gen(start, step, 300+r.IntN(300))
					}
					history := newJoin(3, 0, 0)
					for i := range ts {
						vs := make([]float64, len(ts[i]))
						for k := range vs {
							vs[k] = float64(i*100000 + k)
						}
						s := telem.NewSeries(ts[i])
						Expect(history.Push(i, values(vs...), s)).To(Succeed())
					}
					var want join.Output
					history.Next(telem.TimeStampMax, &want)

					var now telem.TimeStamp
					live := MustSucceed(join.New(join.Config{
						DataTypes: []telem.DataType{
							telem.Float64T, telem.Float64T, telem.Float64T,
						},
						Live: true,
						Now:  func() telem.TimeStamp { return now },
					}))
					var got join.Output
					pos := make([]int, 3)
					for k, stamp := range ts[0] {
						for i := 1; i < 3; i++ {
							for pos[i] < len(ts[i]) && ts[i][pos[i]] <= stamp {
								now = ts[i][pos[i]]
								Expect(live.Push(
									i,
									values(float64(i*100000+pos[i])),
									at(0, ts[i][pos[i]]),
								)).To(Succeed())
								pos[i]++
							}
						}
						now = stamp
						Expect(live.Push(
							0,
							values(float64(k)),
							at(telem.Alignment(k), stamp),
						)).To(Succeed())
						live.Next(stamp, &got)
					}
					Expect(got.Dropped).To(Equal(want.Dropped))
					Expect(flatten(got)).To(Equal(flatten(want)))
				})
		}
	})
})

// TestJoinAllocs checks the steady push loop allocates nothing per sample after its
// warm-up, live and in history.
func TestJoinAllocs(t *testing.T) {
	for _, live := range []bool{false, true} {
		const frames, size = 400, 40
		var now telem.TimeStamp
		j, err := join.New(join.Config{
			DataTypes: []telem.DataType{telem.Float64T, telem.Int16T},
			Live:      live,
			Now:       func() telem.TimeStamp { return now },
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
