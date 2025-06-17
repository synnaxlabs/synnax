// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculator", func() {

	Context("Single Channel Calculation", func() {
		It("Should correctly calculate the output value", func() {
			out := channel.Channel{
				Leaseholder: 1,
				LocalKey:    1,
				Name:        "out",
				DataType:    telem.Float32T,
				Expression:  "return in_ch * 2",
			}
			in := channel.Channel{
				Leaseholder: 1,
				LocalKey:    2,
				Name:        "in_ch",
				DataType:    telem.Float32T,
			}
			calc := MustSucceed(calculation.OpenCalculator(
				out,
				[]channel.Channel{in},
			))
			outSeries := MustSucceed(calc.Next(core.UnaryFrame(in.Key(), telem.NewSeriesV[float32](1, 2, 3))))
			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](2, 4, 6))
		})
	})

	Context("Multi-Channels Calculation", func() {
		var (
			inCh1 = channel.Channel{
				Leaseholder: 1,
				LocalKey:    1,
				Name:        "in_ch_1",
				DataType:    telem.Float32T,
			}
			inCh2 = channel.Channel{
				Leaseholder: 1,
				LocalKey:    2,
				Name:        "in_ch_2",
				DataType:    telem.Float32T,
			}
			out = channel.Channel{
				Leaseholder: 1,
				LocalKey:    1,
				Name:        "out",
				DataType:    telem.Float32T,
				Expression:  "return in_ch_1 * in_ch_2",
			}
			calc *calculation.Calculator
		)
		BeforeEach(func() {
			calc = MustSucceed(calculation.OpenCalculator(
				out,
				[]channel.Channel{inCh1, inCh2},
			))
		})
		AfterEach(func() {
			calc.Close()
		})

		Context("Aligned", func() {
			It("Should correctly calculate the output value", func() {
				inSeries1 := telem.NewSeriesV[float32](1, 2, 3)
				inSeries1.TimeRange = telem.NewRangeSeconds(5, 10)
				inSeries2 := telem.NewSeriesV[float32](1, 2, 3)
				inSeries2.TimeRange = telem.NewRangeSeconds(5, 10)
				outSeries := MustSucceed(calc.Next(core.MultiFrame(
					[]channel.Key{inCh1.Key(), inCh2.Key()},
					[]telem.Series{inSeries1, inSeries2},
				)))
				Expect(outSeries.Len()).To(Equal(int64(3)))
				Expect(outSeries.Alignment).To(Equal(telem.Alignment(0)))
				Expect(outSeries.TimeRange).To(Equal(telem.NewRangeSeconds(5, 10)))
				Expect(outSeries.AlignmentBounds().Upper).To(Equal(telem.Alignment(3)))
				Expect(outSeries).To(telem.MatchSeriesDataV[float32](1, 4, 9))
			})
		})

		Context("Misaligned", func() {
			It("Should correctly align the series and calculate the output value", func() {
				inCh1Series := telem.NewSeriesV[float32](1, 2, 3)
				inCh1Series.Alignment = 3
				inCh1Series.TimeRange = telem.NewRangeSeconds(5, 10)

				inCh2Series := telem.NewSeriesV[float32](1, 2, 3)
				inCh2Series.Alignment = 3
				inCh2Series.TimeRange = telem.NewRangeSeconds(5, 10)
				outSeries := MustSucceed(calc.Next(core.UnaryFrame(
					inCh1.Key(),
					inCh1Series,
				)))
				Expect(outSeries.Len()).To(Equal(int64(0)))
				Expect(outSeries.Alignment).To(Equal(telem.Alignment(0)))
				Expect(outSeries.TimeRange).To(Equal(telem.TimeRangeZero))

				outSeries = MustSucceed(calc.Next(core.UnaryFrame(
					inCh2.Key(),
					inCh2Series,
				)))
				Expect(outSeries.Len()).To(Equal(int64(3)))
				Expect(outSeries.Alignment).To(Equal(telem.Alignment(3)))
				Expect(outSeries.TimeRange).To(Equal(telem.NewRangeSeconds(5, 10)))
				Expect(outSeries.AlignmentBounds().Upper).To(Equal(telem.Alignment(6)))

				Expect(outSeries).To(telem.MatchSeriesDataV[float32](1, 4, 9))
			})
		})
	})

	Describe("Channels", func() {
		It("Should return information about the channel being calculated", func() {
			in := channel.Channel{
				Name:     "Cat",
				LocalKey: 12,
				DataType: telem.Float32T,
			}
			c := MustSucceed(calculation.OpenCalculator(in, []channel.Channel{}))
			Expect(c.Channel().Name).To(Equal("Cat"))
			Expect(c.Channel().LocalKey).To(BeEquivalentTo(12))
		})
	})

	Describe("Error Handling", func() {
		It("Should return an error if the calculation fails to compile", func() {
			out := channel.Channel{
				Name:       "Cat",
				LocalKey:   12,
				DataType:   telem.Float32T,
				Expression: "/////",
			}
			c, err := calculation.OpenCalculator(out, []channel.Channel{})
			Expect(err).To(MatchError(ContainSubstring("syntax error")))
			Expect(c).To(BeNil())
		})

		It("Should return an error if the calculation has an undefined variable", func() {
			out := channel.Channel{
				Name:       "Cat",
				LocalKey:   12,
				DataType:   telem.Float32T,
				Expression: "return dog * cat",
			}
			in := channel.Channel{
				Name:     "cat",
				LocalKey: 12,
				DataType: telem.Float32T,
			}
			c := MustSucceed(calculation.OpenCalculator(out, []channel.Channel{in}))
			o, err := c.Next(core.UnaryFrame(in.Key(), telem.NewSeriesV[float32](1, 2, 3)))
			Expect(o.Len()).To(Equal(int64(0)))
			Expect(err).To(MatchError(ContainSubstring("nil")))
		})
	})
})

func BenchmarkCalculator(b *testing.B) {
	inCh1 := channel.Channel{
		Leaseholder: 1,
		LocalKey:    1,
		Name:        "in_ch_1",
		DataType:    telem.Float32T,
	}
	inCh2 := channel.Channel{
		Leaseholder: 1,
		LocalKey:    2,
		Name:        "in_ch_2",
		DataType:    telem.Float32T,
	}
	out := channel.Channel{
		Leaseholder: 1,
		LocalKey:    3,
		Name:        "out",
		DataType:    telem.Float32T,
		Expression:  "return in_ch_1 * in_ch_2",
	}
	calc := MustSucceed(calculation.OpenCalculator(
		out,
		[]channel.Channel{inCh1, inCh2},
	))
	data1 := telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	data2 := telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	for b.Loop() {
		_, _ = calc.Next(core.MultiFrame(
			[]channel.Key{inCh1.Key(), inCh2.Key()},
			[]telem.Series{data1, data2},
		))
		data1.Alignment = data1.AlignmentBounds().Upper
		data2.Alignment = data2.AlignmentBounds().Upper
	}
}
