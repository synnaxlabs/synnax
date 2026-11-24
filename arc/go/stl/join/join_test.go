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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// stamps returns a timestamp series of seconds, with the alignment of its first sample.
func stamps(alignment telem.Alignment, seconds ...float64) telem.Series {
	ts := make([]telem.TimeStamp, len(seconds))
	for i, s := range seconds {
		ts[i] = telem.TimeStamp(s * float64(telem.Second))
	}
	series := telem.NewSeries(ts)
	series.Alignment = alignment
	return series
}

func values(vs ...float64) telem.Series { return telem.NewSeriesV(vs...) }

// floats returns the samples of a float64 series.
func floats(s telem.Series) []float64 {
	out := make([]float64, s.Len())
	for i := range out {
		out[i] = math.Float64frombits(telem.ByteOrder.Uint64(s.Data[i*8:]))
	}
	return out
}

// times returns the samples of a timestamp series.
func times(s telem.Series) []telem.TimeStamp {
	out := make([]telem.TimeStamp, s.Len())
	for i := range out {
		out[i] = telem.TimeStamp(telem.ByteOrder.Uint64(s.Data[i*8:]))
	}
	return out
}

// sample is one joined sample: its stamp in seconds, its alignment, and the value of
// each input.
type sample struct {
	seconds   float64
	alignment telem.Alignment
	values    []float64
}

// samples returns each joined sample of out, and the index of its run.
func samples(out join.Output) ([]sample, []int) {
	var (
		all  []sample
		runs []int
	)
	for r, run := range out.Runs {
		ts := times(run.Time)
		for k, t := range ts {
			vs := make([]float64, len(run.Values))
			for i, v := range run.Values {
				vs[i] = floats(v)[k]
			}
			all = append(all, sample{
				seconds:   float64(t) / float64(telem.Second),
				alignment: run.Time.Alignment + telem.Alignment(k),
				values:    vs,
			})
			runs = append(runs, r)
		}
	}
	return all, runs
}

func newJoin(inputs, primary int, from telem.TimeStamp) *join.Join {
	dts := make([]telem.DataType, inputs)
	for i := range dts {
		dts[i] = telem.Float64T
	}
	return MustSucceed(
		join.New(join.Config{DataTypes: dts, Primary: primary, From: from}),
	)
}

var _ = Describe("Join", func() {
	Describe("New", func() {
		It("Should refuse a join with no inputs", func() {
			Expect(join.New(join.Config{})).Error().
				To(MatchError(ContainSubstring("join needs at least one input")))
		})
		It("Should refuse a primary that is not an input", func() {
			Expect(join.New(join.Config{
				DataTypes: []telem.DataType{telem.Float64T},
				Primary:   1,
			})).Error().To(MatchError(validate.ErrValidation))
		})
		It("Should refuse an input with no fixed density", func() {
			Expect(join.New(join.Config{
				DataTypes: []telem.DataType{telem.Float64T, telem.StringT},
			})).Error().To(MatchError(ContainSubstring("input 1 has data type string")))
		})
		DescribeTable(
			"Should refuse arrival inputs that the join cannot take",
			func(arrival []bool, live bool, message string) {
				Expect(join.New(join.Config{
					DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
					Live:      live,
					Arrival:   arrival,
				})).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry(
				"a mark for each input",
				[]bool{true},
				true,
				"arrival marks 1 inputs of 2",
			),
			Entry(
				"an arrival primary",
				[]bool{true, false},
				true,
				"primary input 0 cannot pair by arrival",
			),
			Entry(
				"a history join",
				[]bool{false, true},
				false,
				"only a live join pairs by arrival",
			),
		)
	})

	Describe("Push", func() {
		It("Should refuse samples of another data type", func() {
			j := newJoin(1, 0, 0)
			Expect(j.Push(0, telem.NewSeriesV[int32](1), stamps(0, 1))).
				To(MatchError(ContainSubstring("input 0 has data type float64, not int32")))
		})
		It("Should refuse a different count of samples and stamps", func() {
			j := newJoin(1, 0, 0)
			Expect(j.Push(0, values(1, 2), stamps(0, 1))).
				To(MatchError(ContainSubstring("input 0 has 2 samples and 1 stamps")))
		})
		It("Should refuse a stamp that does not follow the last one pushed", func() {
			j := newJoin(1, 0, 0)
			Expect(j.Push(0, values(1), stamps(0, 2))).To(Succeed())
			Expect(j.Push(0, values(2), stamps(1, 2))).
				To(MatchError(ContainSubstring("does not follow")))
		})
	})

	Describe("Next", func() {
		It("Should pass the samples of one input with their alignments", func() {
			j := newJoin(1, 0, 0)
			Expect(j.Push(0, values(1, 2, 3), stamps(10, 1, 2, 3))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, runs := samples(out)
			Expect(got).To(Equal([]sample{
				{1, 10, []float64{1}},
				{2, 11, []float64{2}},
				{3, 12, []float64{3}},
			}))
			Expect(runs).To(Equal([]int{0, 0, 0}))
		})

		It("Should take the stamps of a primary that is not the first input", func() {
			j := newJoin(2, 1, 0)
			Expect(j.Push(0, values(10, 20), stamps(0, 1, 2))).To(Succeed())
			Expect(j.Push(1, values(1, 2, 3), stamps(7, 1, 1.5, 2))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(Equal([]sample{
				{1, 7, []float64{10, 1}},
				{1.5, 8, []float64{10, 2}},
				{2, 9, []float64{20, 3}},
			}))
		})

		It("Should pair inputs that share the primary's stamps", func() {
			j := newJoin(2, 0, 0)
			Expect(j.Push(0, values(1, 2), stamps(5, 1, 2))).To(Succeed())
			Expect(j.Push(1, values(10, 20), stamps(5, 1, 2))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(Equal([]sample{
				{1, 5, []float64{1, 10}},
				{2, 6, []float64{2, 20}},
			}))
		})

		It("Should hold the last value of an input at or before each stamp", func() {
			j := newJoin(2, 0, 0)
			Expect(
				j.Push(0, values(1, 2, 3, 4), stamps(0, 1, 1.5, 2, 2.5)),
			).To(Succeed())
			Expect(j.Push(1, values(10, 20), stamps(0, 1, 2))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(Equal([]sample{
				{1, 0, []float64{1, 10}},
				{1.5, 1, []float64{2, 10}},
				{2, 2, []float64{3, 20}},
				{2.5, 3, []float64{4, 20}},
			}))
		})

		It("Should give no output before an input has a sample", func() {
			j := newJoin(2, 0, 0)
			Expect(j.Push(0, values(1, 2), stamps(0, 1, 2))).To(Succeed())
			Expect(j.Push(1, values(20), stamps(0, 2))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(Equal([]sample{{2, 1, []float64{2, 20}}}))
			Expect(out.Dropped).To(Equal(1))
		})

		It(
			"Should join only the stamps up to until, and the rest on a later call",
			func() {
				j := newJoin(2, 0, 0)
				Expect(j.Push(0, values(1, 2, 3), stamps(0, 1, 2, 3))).To(Succeed())
				Expect(j.Push(1, values(10), stamps(0, 1))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStamp(2*telem.Second), &out)
				got, _ := samples(out)
				Expect(got).To(HaveLen(2))
				Expect(j.Push(1, values(30), stamps(1, 3))).To(Succeed())
				out.Reset()
				j.Next(telem.TimeStampMax, &out)
				got, _ = samples(out)
				Expect(got).To(Equal([]sample{{3, 2, []float64{3, 30}}}))
			},
		)

		Describe("Hold limit", func() {
			// steady returns n values of one input, one per second from second 1.
			steady := func(n int) (telem.Series, telem.Series) {
				vs, ss := make([]float64, n), make([]float64, n)
				for i := range n {
					vs[i], ss[i] = float64(i+1), float64(i+1)
				}
				return values(vs...), stamps(0, ss...)
			}

			It(
				"Should drop the stamps past 3 median intervals, and start a new run after",
				func() {
					j := newJoin(2, 0, 0)
					Expect(
						j.Push(0, values(1, 2, 3, 4), stamps(0, 3, 5, 6.5, 8)),
					).To(Succeed())
					sv, ss := steady(3)
					Expect(j.Push(1, sv, ss)).To(Succeed())
					Expect(j.Push(1, values(80), stamps(3, 8))).To(Succeed())
					var out join.Output
					j.Next(telem.TimeStampMax, &out)
					got, runs := samples(out)
					// The interval is 1 s, so the limit is 3 s: 3 holds 3 and 5 holds 3
					// for
					// 2 s, 6.5 holds 3 for 3.5 s and drops, and 8 has its own sample.
					Expect(got).To(Equal([]sample{
						{3, 0, []float64{1, 3}},
						{5, 1, []float64{2, 3}},
						{8, 3, []float64{4, 80}},
					}))
					Expect(runs).To(Equal([]int{0, 0, 1}))
					Expect(out.Dropped).To(Equal(1))
				},
			)

			It("Should hold an input with one sample with no limit", func() {
				j := newJoin(2, 0, 0)
				Expect(j.Push(0, values(1, 2), stamps(0, 1, 1000))).To(Succeed())
				Expect(j.Push(1, values(10), stamps(0, 1))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				got, _ := samples(out)
				Expect(got).To(HaveLen(2))
			})

			It("Should take the median of the last 16 intervals", func() {
				j := newJoin(2, 0, 0)
				// 16 intervals of 1 s, then 2 of 2 s: the median of the last 16 is
				// still 1 s, so a stamp 4 s after the last sample drops.
				vs, ss := make([]float64, 0, 19), make([]float64, 0, 19)
				for i := range 17 {
					vs, ss = append(vs, 0), append(ss, float64(i))
				}
				vs, ss = append(vs, 0, 0), append(ss, 18, 20)
				Expect(j.Push(1, values(vs...), stamps(0, ss...))).To(Succeed())
				Expect(j.Push(0, values(1, 2), stamps(0, 20, 24))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				got, _ := samples(out)
				Expect(got).To(HaveLen(1))
				Expect(out.Dropped).To(Equal(1))
			})

			It("Should average the two middle intervals of an even count", func() {
				j := newJoin(2, 0, 0)
				// Intervals of 1 s and 3 s: the median is 2 s, so the limit is 6 s.
				Expect(j.Push(1, values(0, 0, 0), stamps(0, 0, 1, 4))).To(Succeed())
				Expect(j.Push(0, values(1, 2), stamps(0, 10, 10.5))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				got, _ := samples(out)
				Expect(got).To(Equal([]sample{{10, 0, []float64{1, 0}}}))
				Expect(out.Dropped).To(Equal(1))
			})
		})

		It(
			"Should fill the hold limits from samples before From, with no output for them",
			func() {
				j := newJoin(2, 0, telem.TimeStamp(10*telem.Second))
				Expect(j.Push(0, values(1, 2, 3), stamps(0, 8, 9, 10))).To(Succeed())
				Expect(j.Push(1, values(80, 90), stamps(0, 8, 9))).To(Succeed())
				var out join.Output
				j.Next(telem.TimeStampMax, &out)
				got, _ := samples(out)
				Expect(got).To(Equal([]sample{{10, 2, []float64{3, 90}}}))
				Expect(out.Dropped).To(BeZero())
			},
		)

		It(
			"Should join live and historical reads to the same samples, past the limit too",
			func() {
				// other's interval is 1 s, so its limit is 3 s: 7.5 and 8 hold its
				// sample
				// at 4 past the limit.
				primary := stamps(
					0,
					0,
					0.5,
					1,
					1.5,
					2,
					2.5,
					3,
					3.5,
					4,
					4.5,
					5,
					7.5,
					8,
					9,
					9.5,
				)
				primaryValues := values(
					1,
					2,
					3,
					4,
					5,
					6,
					7,
					8,
					9,
					10,
					11,
					12,
					13,
					14,
					15,
				)
				other := stamps(0, 0, 1, 2, 3, 4, 9)
				otherValues := values(10, 20, 30, 40, 50, 90)

				history := newJoin(2, 0, 0)
				Expect(history.Push(0, primaryValues, primary)).To(Succeed())
				Expect(history.Push(1, otherValues, other)).To(Succeed())
				var want join.Output
				history.Next(telem.TimeStampMax, &want)

				live := newJoin(2, 0, 0)
				var got join.Output
				// Each input's samples arrive at their stamps, and each primary sample
				// is
				// joined at its arrival.
				for k := range primary.Len() {
					stamp := telem.TimeStamp(telem.ByteOrder.Uint64(primary.Data[k*8:]))
					for o := range other.Len() {
						if telem.TimeStamp(
							telem.ByteOrder.Uint64(other.Data[o*8:]),
						) == stamp {
							Expect(live.Push(
								1,
								telem.NewSeriesV(floats(otherValues)[o]),
								stamps(
									telem.Alignment(o),
									float64(stamp)/float64(telem.Second),
								),
							)).To(Succeed())
						}
					}
					Expect(live.Push(
						0,
						telem.NewSeriesV(floats(primaryValues)[k]),
						stamps(
							telem.Alignment(k),
							float64(stamp)/float64(telem.Second),
						),
					)).To(Succeed())
					live.Next(stamp, &got)
				}
				wantSamples, wantRuns := samples(want)
				gotSamples, gotRuns := samples(got)
				Expect(gotSamples).To(Equal(wantSamples))
				Expect(gotRuns).To(Equal(wantRuns))
				Expect(got.Dropped).To(Equal(want.Dropped))
				Expect(want.Dropped).To(BeNumerically(">", 0))
			},
		)
	})

	It("Should drop a live primary stamp before the samples that the cap passed",
		func() {
			j := MustSucceed(join.New(join.Config{
				DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
				Live:      true,
			}))
			n := 1<<16 + 10
			vs, ss := make([]float64, n), make([]float64, n)
			for i := range ss {
				vs[i], ss[i] = float64(i), 1+float64(i)/1000
			}
			Expect(j.Push(1, values(vs...), stamps(0, ss...))).To(Succeed())
			var out join.Output
			j.Next(0, &out)
			Expect(j.Push(0, values(1, 2), stamps(0, 1.001, 1.5))).To(Succeed())
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(Equal([]sample{{1.5, 1, []float64{2, 500}}}))
			Expect(out.Dropped).To(Equal(1))
		})

	It("Should refuse a history push past the cap, and keep the input as it was",
		func() {
			j := newJoin(2, 0, 0)
			n := 1 << 16
			vs, ss := make([]float64, n), make([]float64, n)
			for i := range ss {
				vs[i], ss[i] = float64(i), float64(i)/1000
			}
			Expect(j.Push(1, values(vs...), stamps(0, ss...))).To(Succeed())
			Expect(j.Push(1, values(1), stamps(0, 66))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"input 1 would hold more than 65536 samples that the join has not "+
						"passed",
				)),
			))
			Expect(j.Push(0, values(1), stamps(0, 65.535))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			got, _ := samples(out)
			Expect(got).To(Equal([]sample{{65.535, 0, []float64{1, 65535}}}))
		})

	DescribeTable(
		"Should count an interval at a factor of 2 of the median as near it",
		func(intervals ...float64) {
			ss := make([]float64, 1, len(intervals)+1)
			for _, iv := range intervals {
				ss = append(ss, ss[len(ss)-1]+iv)
			}
			j := newJoin(2, 0, 0)
			Expect(j.Push(1, values(make([]float64, len(ss))...), stamps(0, ss...))).
				To(Succeed())
			Expect(j.Push(0, values(1), stamps(0, ss[len(ss)-1]+10))).To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			Expect(out.Dropped).To(Equal(1), "steady, so 10 s past the limit of 6 s")
		},
		Entry("twice the median", 2.0, 2.0, 2.0, 2.0, 2.0, 4.0, 4.0, 4.0),
		Entry("half the median", 1.0, 1.0, 1.0, 2.0, 2.0, 2.0, 2.0, 2.0),
	)

	Describe("Live", func() {
		var now telem.TimeStamp
		BeforeEach(func() { now = 0 })
		newLive := func(from telem.TimeStamp) *join.Join {
			return MustSucceed(join.New(join.Config{
				DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
				From:      from,
				Live:      true,
				Now:       func() telem.TimeStamp { return now },
			}))
		}
		seconds := func(s float64) telem.TimeStamp {
			return telem.TimeStamp(s * float64(telem.Second))
		}
		// frames pushes n frames of 40 ms: a primary frame of 40 samples at 1 kHz from
		// 10 s, joined at once, and then, for the first stop frames, an input frame of
		// 20 samples at 500 Hz whose stamps lag the primary's by lag seconds. Each
		// frame arrives at its end. It returns the drops of each frame.
		frames := func(n, stop int, lag float64) []int {
			GinkgoHelper()
			j := newLive(0)
			drops := make([]int, n)
			var out join.Output
			for f := range n {
				now = seconds(0.04 * float64(f+1))
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
				j.Next(seconds(primary[39]), &out)
				drops[f] = out.Dropped - before
				if f < stop {
					Expect(j.Push(1, values(input...), stamps(0, input...))).
						To(Succeed())
				}
			}
			return drops
		}

		// Frame 0 has no input. At frame 1 the input has no frame gap yet, so it takes
		// the primary's.
		DescribeTable(
			"Should hold an input whose stamps lag the primary's while its frames arrive",
			func(lag float64) {
				Expect(frames(10, 10, lag)).
					To(Equal([]int{40, 0, 0, 0, 0, 0, 0, 0, 0, 0}))
			},
			Entry("200 ms of latency", 0.2),
			Entry("a clock 1 s behind", 1.0),
		)

		It(
			"Should drop each primary frame that arrives past 3 times the input's frame "+
				"gap after its last frame",
			func() {
				Expect(
					frames(10, 5, 0),
				).To(Equal([]int{40, 0, 0, 0, 0, 0, 0, 0, 40, 40}))
			},
		)

		It("Should count no seed before From as a frame", func() {
			// A seed at 1 Hz up to 16 s, then frames that arrive at 100 s and 101 s:
			// the frame gap is 1 s, so a primary frame that arrives 9 s later drops.
			j := newLive(seconds(100))
			seed := make([]float64, 17)
			for i := range seed {
				seed[i] = float64(i)
			}
			Expect(j.Push(1, values(seed...), stamps(0, seed...))).To(Succeed())
			now = seconds(100)
			Expect(j.Push(1, values(100), stamps(0, 100))).To(Succeed())
			now = seconds(101)
			Expect(j.Push(1, values(101), stamps(0, 101))).To(Succeed())
			now = seconds(110)
			Expect(j.Push(0, values(1), stamps(0, 110))).To(Succeed())
			var out join.Output
			j.Next(seconds(110), &out)
			Expect(out.Dropped).To(Equal(1))
		})

		DescribeTable(
			"Should hold a seed only within its hold limit of a primary frame's first "+
				"stamp",
			func(from float64, dropped int) {
				// A seed at 1 Hz up to 16 s holds for 3 s.
				j := newLive(seconds(from))
				seed := make([]float64, 17)
				for i := range seed {
					seed[i] = float64(i)
				}
				Expect(j.Push(1, values(seed...), stamps(0, seed...))).To(Succeed())
				Expect(j.Push(0, values(1), stamps(0, from))).To(Succeed())
				var out join.Output
				j.Next(seconds(from), &out)
				Expect(out.Dropped).To(Equal(dropped))
			},
			Entry("1 s after the seed", 17.0, 0),
			Entry("4 s after the seed", 20.0, 1),
		)

		// primaryAt pushes a primary frame of one sample at stamp seconds that arrives
		// at stamp, joins it, and returns its drops.
		primaryAt := func(j *join.Join, stamp float64, out *join.Output) int {
			GinkgoHelper()
			now = seconds(stamp)
			Expect(j.Push(0, values(stamp), stamps(telem.Alignment(stamp), stamp))).
				To(Succeed())
			before := out.Dropped
			j.Next(seconds(stamp), out)
			return out.Dropped - before
		}

		It(
			"Should judge a framed input with no hold limit by the "+
				"primary's frame gaps",
			func() {
				// One sample gives the input no interval, so no hold limit. The
				// primary's frame gaps of 1 s give a window of 3 s, and the input's
				// frame came 5 s before the primary's last.
				j := newLive(0)
				now = seconds(1)
				Expect(j.Push(1, values(7), stamps(0, 0.5))).To(Succeed())
				var out join.Output
				drops := make([]int, 0, 4)
				for _, stamp := range []float64{1, 2, 3, 6} {
					drops = append(drops, primaryAt(j, stamp, &out))
				}
				Expect(drops).To(Equal([]int{0, 0, 0, 1}))
			},
		)

		Describe("Arrival", func() {
			newArrival := func() *join.Join {
				return MustSucceed(join.New(join.Config{
					DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
					Live:      true,
					Arrival:   []bool{false, true},
					Now:       func() telem.TimeStamp { return now },
				}))
			}

			It(
				"Should pair each primary stamp with the newest pushed sample, "+
					"whatever its stamps",
				func() {
					j := newArrival()
					Expect(j.Push(1, values(5, 6), stamps(0, 100, 101))).To(Succeed())
					Expect(j.Push(0, values(1, 2), stamps(0, 10, 11))).To(Succeed())
					var out join.Output
					j.Next(seconds(11), &out)
					Expect(j.Push(1, values(7), telem.Series{})).To(Succeed())
					Expect(j.Push(0, values(3), stamps(2, 12))).To(Succeed())
					j.Next(seconds(12), &out)
					all, _ := samples(out)
					Expect(all).To(Equal([]sample{
						{seconds: 10, alignment: 0, values: []float64{1, 6}},
						{seconds: 11, alignment: 1, values: []float64{2, 6}},
						{seconds: 12, alignment: 2, values: []float64{3, 7}},
					}))
					Expect(out.Dropped).To(BeZero())
				},
			)

			It("Should give no output before an arrival input has a sample", func() {
				var out join.Output
				Expect(primaryAt(newArrival(), 1, &out)).To(Equal(1))
			})

			It(
				"Should drop primary stamps once its pushes stop past 3 times "+
					"their gap",
				func() {
					j := newArrival()
					for _, at := range []float64{1, 2} {
						now = seconds(at)
						Expect(j.Push(1, values(at), telem.Series{})).To(Succeed())
					}
					var out join.Output
					drops := make([]int, 0, 3)
					for _, stamp := range []float64{3, 5, 6} {
						drops = append(drops, primaryAt(j, stamp, &out))
					}
					Expect(drops).To(Equal([]int{0, 0, 1}))
				},
			)

			It(
				"Should take the primary's frame gaps until its pushes have a gap",
				func() {
					// With no gap on either side, the input holds at 1 s. Then the
					// primary's gaps of 1 s give a window of 3 s, and its frame at 6 s
					// comes 6 s after the input's push.
					j := newArrival()
					now = 0
					Expect(j.Push(1, values(9), telem.Series{})).To(Succeed())
					var out join.Output
					drops := make([]int, 0, 4)
					for _, stamp := range []float64{1, 2, 3, 6} {
						drops = append(drops, primaryAt(j, stamp, &out))
					}
					Expect(drops).To(Equal([]int{0, 0, 0, 1}))
				},
			)
		})
	})
})
