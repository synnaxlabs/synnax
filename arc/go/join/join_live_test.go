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
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Join", func() {
	Describe("Live", func() {
		var now telem.TimeStamp

		newLive := func(from telem.TimeStamp, arrivalTimed bool) *join.Join {
			GinkgoHelper()
			return MustSucceed(join.New(join.Config{
				Inputs: []join.Input{
					{DataType: telem.Float64T},
					{DataType: telem.Float64T, ArrivalTimed: arrivalTimed},
				},
				From: from,
				Live: true,
				Now:  at(&now),
			}))
		}

		It("Should hold a live input that stops for 3 of its frame gaps, and drop "+
			"each stamp of the frames after that", func() {
			// Two 1 kHz inputs in frames of 40 samples. Each primary frame arrives at
			// the end of its 40 ms, and the other's frame 20 ms after it. The other
			// input's last frame arrives at 180 ms, and its frames come 40 ms apart, so
			// it holds for up to 120 ms after that: the primary frames that arrive at
			// 200 ms to 280 ms hold, and those at 320 ms and 360 ms drop. Frame 0 has
			// no other input.
			const frame, frames, stop = 40, 9, 4
			var now telem.TimeStamp
			j := MustSucceed(join.New(join.Config{
				Inputs: floatInputs(2),
				Live:   true,
				Now:    func() telem.TimeStamp { return now },
			}))
			var out join.Output
			dropped := make([]int, 0, frames)
			for f := range frames {
				ps, vs := make([]float64, frame), make([]float64, frame)
				for k := range frame {
					ps[k] = float64(f*frame+k) / 1000
					vs[k] = float64(f*frame + k)
				}
				now = telem.TimeStamp(40*(f+1)) * telem.MillisecondTS
				Expect(j.Push(
					0,
					values(vs...),
					stamps(telem.Alignment(f*frame), ps...),
				)).
					To(Succeed())
				before := out.Dropped
				j.Next(telem.TimeStamp(ps[frame-1]*float64(telem.Second)), &out)
				dropped = append(dropped, out.Dropped-before)
				if f < stop {
					now += 20 * telem.MillisecondTS
					others := make([]float64, frame)
					for k := range frame {
						others[k] = ps[k] + 0.0005
					}
					Expect(j.Push(1, values(vs...), stamps(0, others...))).To(Succeed())
				}
			}
			Expect(dropped).To(Equal([]int{frame, 0, 0, 0, 0, 0, 0, frame, frame}))
		})

		It("Should drop only the primary frame before the other input's first frame "+
			"arrives, when each primary frame comes first", func() {
			// Two 1 kHz inputs on two clocks, both streamed in frames of 40 samples (a
			// 25 Hz stream rate). Each primary frame arrives at the end of its 40 ms,
			// and the other's frame 20 ms after it. Only frame 0, before the other
			// input's first frame, drops: at frame 1 the other input has no frame gap
			// yet, so its window comes from the primary's frame gaps.
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
			Expect(history.Push(0, values(pv...), rawStamps(0, primary...))).
				To(Succeed())
			Expect(history.Push(1, values(ov...), rawStamps(0, other...))).To(Succeed())
			var want join.Output
			history.Next(telem.TimeStampMax, &want)

			var now telem.TimeStamp
			live := MustSucceed(join.New(join.Config{
				Inputs: floatInputs(2),
				Live:   true,
				Now:    func() telem.TimeStamp { return now },
			}))
			var got join.Output
			dropped := make([]int, 0, frames)
			for f := range frames {
				lo, hi := f*frame, (f+1)*frame
				now = telem.TimeStamp(40*(f+1)) * ms
				Expect(live.Push(
					0,
					values(pv[lo:hi]...),
					rawStamps(telem.Alignment(lo), primary[lo:hi]...),
				)).To(Succeed())
				before := got.Dropped
				live.Next(primary[hi-1], &got)
				dropped = append(dropped, got.Dropped-before)
				now += 20 * ms
				Expect(live.Push(
					1,
					values(ov[lo:hi]...),
					rawStamps(0, other[lo:hi]...),
				)).
					To(Succeed())
			}
			Expect(want.Dropped).To(Equal(1))
			Expect(dropped).To(Equal([]int{frame, 0, 0, 0, 0, 0, 0, 0, 0, 0}))
		})

		It("Should keep its buffers bounded while a live primary is silent", func() {
			// The primary has one sample, then its device goes quiet while another
			// input streams at 1 kHz in frames of 1000. Each frame, the live read
			// passes the newest primary stamp.
			var now telem.TimeStamp
			j := MustSucceed(join.New(join.Config{
				Inputs: floatInputs(2),
				Live:   true,
				Now:    at(&now),
			}))
			Expect(j.Push(0, values(0), rawStamps(0, 0))).To(Succeed())
			const frame = 1000
			vs := make([]float64, frame)
			ts := make([]telem.TimeStamp, frame)
			var (
				out  join.Output
				next = ms
			)
			push := func(n int) {
				GinkgoHelper()
				for range n {
					for k := range ts {
						ts[k], next = next, next+ms
					}
					Expect(j.Push(1, values(vs...), rawStamps(0, ts...))).To(Succeed())
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

		DescribeTable(
			"Should hold an input whose stamps lag the primary's while its frames "+
				"arrive, when From is the live open (a 1 s skew stays held)",
			func(lag float64) {
				var now telem.TimeStamp
				j := MustSucceed(join.New(join.Config{
					Inputs: floatInputs(2),
					From:   secs(10),
					Live:   true,
					Now:    func() telem.TimeStamp { return now },
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
					Expect(j.Push(1, values(input...), stamps(0, input...))).
						To(Succeed())
				}
				want := make([]int, n)
				want[0] = 40
				Expect(drops).To(Equal(want))
			},
			Entry("200 ms of latency", 0.2),
			Entry("a clock 1 s behind", 1.0),
		)

		It("Should join an on-change input with no limit live as in history when its "+
			"samples arrive in stamp order", func() {
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
				Inputs: floatInputs(2),
				Live:   true,
				Now:    func() telem.TimeStamp { return now },
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

		Describe("Load", func() {
			It("Should hold a loaded sample for each primary stamp within one hold "+
				"limit of its own stamp", func() {
				j := newLive(secs(18), false)
				stored := make([]float64, 17)
				for i := range stored {
					stored[i] = float64(i)
				}
				Expect(j.Load(1, values(stored...), stamps(0, stored...))).To(Succeed())
				Expect(j.Push(0, values(1, 2, 3), stamps(0, 18, 19, 20))).To(Succeed())
				var out join.Output
				j.Next(secs(20), &out)
				// The limit is 3 s: 18 s and 19 s hold the sample at 16 s, and 20 s
				// drops.
				Expect(out.Dropped).To(Equal(1))
			})

			It("Should count no loaded primary samples as a frame when it settles "+
				"(Join.Load: loaded samples count as no frame)", func() {
				j := newLive(secs(100), false)
				now = 0
				Expect(j.Load(0, values(1, 2, 3), stamps(0, 1, 2, 3))).To(Succeed())
				var out join.Output
				for _, s := range []float64{100, 101, 102} {
					now = secs(s)
					Expect(j.Push(0, values(s), stamps(telem.Alignment(s), s))).
						To(Succeed())
					out.Reset()
					j.Next(secs(s), &out)
				}
				// Frames at 100, 101, and 102 s: 3 times the median gap is 3 s, and
				// only 2 s have passed since the first frame.
				Expect(out.Settled).To(BeFalse())
			})

			It("Should refuse a load of the primary after its first live frame "+
				"(Join.Load: only before the input's first live frame)", func() {
				j := newLive(0, false)
				now = secs(10)
				Expect(j.Push(0, values(1), stamps(0, 10))).To(Succeed())
				Expect(j.Load(0, values(2), stamps(1, 11))).
					To(MatchError(ContainSubstring(
						"input 0 loads after its first live frame",
					)))
			})
		})

		Describe("Arrival", func() {
			It("Should drop an arrival input that stopped when its pushes share one "+
				"arrival (Input.ArrivalTimed: it holds while its last push arrived "+
				"within 3 times the median gap)", func() {
				j := newLive(0, true)
				now = 0
				for range 3 {
					Expect(j.Push(1, values(1), telem.Series{})).To(Succeed())
				}
				var out join.Output
				for s := 1; s <= 100; s++ {
					now = secs(float64(s))
					Expect(j.Push(
						0,
						values(1),
						stamps(telem.Alignment(s), float64(s)),
					)).
						To(Succeed())
					j.Next(secs(float64(s)), &out)
				}
				Expect(out.Dropped).To(BeNumerically(">", 0))
			})
		})
	})
})
