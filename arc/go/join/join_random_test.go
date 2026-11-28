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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Join", func() {
	Describe("Random live and history", func() {
		// Each run builds a primary and two other inputs with random rates, gaps, and
		// primary alignment jumps. Live pushes no input late: each other sample arrives
		// before the primary frame that reaches its stamp.
		for run := range uint64(48) {
			It("Should join live and chunked reads to the history samples", func() {
				r := rand.New(rand.NewPCG(run, run^0x9e3779b97f4a7c15))
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
					GinkgoHelper()
					return MustSucceed(join.New(join.Config{
						Inputs:  inputsOf(dts),
						Primary: p,
						From:    from,
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
					GinkgoHelper()
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
					GinkgoHelper()
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

				// Chunked history: push a chunk of the input with the earliest newest
				// stamp, then pass the earliest newest stamp.
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

	Describe("Random live in stamp order", func() {
		// Each run builds a primary and two steady inputs at random rates, each sample
		// in its own frame that arrives at its stamp, with random stops. Live then owes
		// the history output exactly.
		for run := range uint64(48) {
			It("Should join a live read in stamp order to the history samples",
				func() {
					r := rand.New(rand.NewPCG(run, run^0x51ed270b27c1a3))
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
						Inputs: floatInputs(3),
						Live:   true,
						Now:    func() telem.TimeStamp { return now },
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
									rawStamps(0, ts[i][pos[i]]),
								)).To(Succeed())
								pos[i]++
							}
						}
						now = stamp
						Expect(live.Push(
							0,
							values(float64(k)),
							rawStamps(telem.Alignment(k), stamp),
						)).To(Succeed())
						live.Next(stamp, &got)
					}
					Expect(got.Dropped).To(Equal(want.Dropped))
					Expect(flatten(got)).To(Equal(flatten(want)))
				})
		}
	})
})
