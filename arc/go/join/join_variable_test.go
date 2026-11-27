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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/join"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// texts returns a variable-density series of dt that holds samples.
func texts(dt telem.DataType, samples ...string) telem.Series {
	s := telem.Series{DataType: dt}
	for _, sample := range samples {
		s.Data = append(s.Data, telem.MarshalVariableSample([]byte(sample))...)
	}
	return s
}

// textsOf returns the samples of a variable-density series.
func textsOf(s telem.Series) []string {
	var out []string
	for sample := range s.Samples() {
		out = append(out, string(sample))
	}
	return out
}

// joinedTexts returns the stamp in seconds of each joined sample of out, and the
// samples of input i.
func joinedTexts(out join.Output, i int) ([]float64, []string) {
	n := 0
	for _, run := range out.Runs {
		n += int(run.Time.Len())
	}
	seconds, samples := make([]float64, 0, n), make([]string, 0, n)
	for _, run := range out.Runs {
		for _, t := range times(run.Time) {
			seconds = append(seconds, float64(t)/float64(telem.Second))
		}
		samples = append(samples, textsOf(run.Values[i])...)
	}
	return seconds, samples
}

var _ = Describe("Variable-density inputs", func() {
	var now telem.TimeStamp
	BeforeEach(func() { now = 0 })
	newTextJoin := func(live bool, inputs ...join.Input) *join.Join {
		GinkgoHelper()
		return MustSucceed(
			join.New(join.Config{Inputs: inputs, Live: live, Now: at(&now)}),
		)
	}
	numberAndText := func(dt telem.DataType) []join.Input {
		return []join.Input{{DataType: telem.Float64T}, {DataType: dt}}
	}

	DescribeTable("Should hold the last sample at or before each stamp",
		func(dt telem.DataType, first, second string) {
			j := newTextJoin(false, numberAndText(dt)...)
			Expect(j.Push(1, texts(dt, first, second), stamps(0, 1, 2))).To(Succeed())
			Expect(j.Push(0, values(1, 2, 3, 4), stamps(0, 1, 1.5, 2, 2.5))).
				To(Succeed())
			var out join.Output
			j.Next(telem.TimeStampMax, &out)
			seconds, samples := joinedTexts(out, 1)
			Expect(seconds).To(Equal([]float64{1, 1.5, 2, 2.5}))
			Expect(samples).To(Equal([]string{first, first, second, second}))
			Expect(out.Runs[0].Values[1].DataType).To(Equal(dt))
		},
		Entry("string", telem.StringT, "a", "bb"),
		Entry("JSON", telem.JSONT, `{"a":1}`, `[2]`),
		Entry("bytes", telem.BytesT, "\x00\x01", ""),
	)

	It("Should keep samples of every length apart across compactions", func() {
		// The input pushes all of its samples first, so each compaction keeps samples
		// that Next has not passed.
		j := newTextJoin(false, numberAndText(telem.StringT)...)
		want := make([]string, 200)
		ss := make([]float64, 200)
		for i := range want {
			want[i], ss[i] = strings.Repeat("x", i%7), float64(i+1)
		}
		Expect(j.Push(1, texts(telem.StringT, want...), stamps(0, ss...))).To(Succeed())
		var out join.Output
		for i, stamp := range ss {
			Expect(j.Push(0, values(stamp), stamps(telem.Alignment(i), stamp))).
				To(Succeed())
			j.Next(telem.TimeStamp(stamp*float64(telem.Second)), &out)
		}
		_, samples := joinedTexts(out, 1)
		Expect(samples).To(Equal(want))
	})

	It("Should take the stamps and samples of a variable-density primary", func() {
		j := newTextJoin(false, join.Input{DataType: telem.StringT}, join.Input{
			DataType: telem.Float64T,
		})
		Expect(j.Push(0, texts(telem.StringT, "on", "off"), stamps(4, 1, 2))).
			To(Succeed())
		Expect(j.Push(1, values(10, 20), stamps(0, 1, 2))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		seconds, samples := joinedTexts(out, 0)
		Expect(seconds).To(Equal([]float64{1, 2}))
		Expect(samples).To(Equal([]string{"on", "off"}))
		Expect(floats(out.Runs[0].Values[1])).To(Equal([]float64{10, 20}))
		Expect(out.Runs[0].Time.Alignment).To(Equal(telem.Alignment(4)))
	})

	It("Should skip the live samples whose stamps do not follow the last one", func() {
		j := newTextJoin(true, numberAndText(telem.StringT)...)
		Expect(j.Push(1, texts(telem.StringT, "a", "b", "c"), stamps(0, 1, 2, 3))).
			To(Succeed())
		Expect(j.Push(1, texts(telem.StringT, "bb", "cc", "dd"), stamps(0, 2, 3, 4))).
			To(Succeed())
		Expect(j.Push(0, values(1, 2), stamps(0, 3, 4))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStamp(4*telem.Second), &out)
		_, samples := joinedTexts(out, 1)
		Expect(samples).To(Equal([]string{"c", "dd"}))
	})

	It("Should refuse a history sample that does not follow, and keep the input as it "+
		"was", func() {
		j := newTextJoin(false, numberAndText(telem.StringT)...)
		Expect(
			j.Push(1, texts(telem.StringT, "a", "bb"), stamps(0, 1, 2)),
		).To(Succeed())
		Expect(
			j.Push(1, texts(telem.StringT, "ccc", "d"), stamps(0, 3, 2)),
		).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("does not follow")),
		))
		Expect(j.Push(1, texts(telem.StringT, "e"), stamps(0, 3))).To(Succeed())
		Expect(j.Push(0, values(1, 2), stamps(0, 2, 3))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStampMax, &out)
		_, samples := joinedTexts(out, 1)
		Expect(samples).To(Equal([]string{"bb", "e"}))
	})

	It("Should hold the newest sample of an arrival-timed input", func() {
		j := newTextJoin(true, join.Input{DataType: telem.Float64T}, join.Input{
			DataType:     telem.StringT,
			ArrivalTimed: true,
		})
		Expect(
			j.Push(1, texts(telem.StringT, "a", "bbb"), telem.Series{}),
		).To(Succeed())
		Expect(j.Push(0, values(1), stamps(0, 10))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStamp(10*telem.Second), &out)
		Expect(j.Push(1, texts(telem.StringT, "c"), telem.Series{})).To(Succeed())
		Expect(j.Push(0, values(2), stamps(1, 11))).To(Succeed())
		j.Next(telem.TimeStamp(11*telem.Second), &out)
		_, samples := joinedTexts(out, 1)
		Expect(samples).To(Equal([]string{"bbb", "c"}))
	})

	It("Should skip the pushed samples that the loaded ones already hold", func() {
		j := newTextJoin(true, numberAndText(telem.StringT)...)
		loaded := make([]string, 17)
		loadedStamps := make([]float64, 17)
		for i := range loaded {
			loaded[i], loadedStamps[i] = strings.Repeat("l", i), float64(i)
		}
		Expect(j.Load(1, texts(telem.StringT, loaded...), stamps(0, loadedStamps...))).
			To(Succeed())
		Expect(j.Push(
			1,
			texts(telem.StringT, "x", "y", "new"),
			stamps(0, 15, 16, 17),
		)).To(Succeed())
		Expect(j.Push(0, values(1, 2), stamps(0, 16, 17))).To(Succeed())
		var out join.Output
		j.Next(telem.TimeStamp(17*telem.Second), &out)
		_, samples := joinedTexts(out, 1)
		Expect(samples).To(Equal([]string{loaded[16], "new"}))
	})

	It("Should refuse a different count of samples and stamps", func() {
		j := newTextJoin(false, numberAndText(telem.StringT)...)
		Expect(j.Push(1, texts(telem.StringT, "a", "b"), stamps(0, 1))).
			To(MatchError(ContainSubstring("input 1 has 2 samples and 1 stamps")))
	})
})
