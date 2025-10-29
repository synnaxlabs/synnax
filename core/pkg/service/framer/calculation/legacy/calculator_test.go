// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	calculation "github.com/synnaxlabs/synnax/pkg/service/framer/calculation/legacy"
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

	Context("Multi-Channel Calculation", func() {
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

	Describe("Channel", func() {
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

	Describe("High Water Mark Behavior", func() {
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
				LocalKey:    3,
				Name:        "out",
				DataType:    telem.Float32T,
				Expression:  "return in_ch_1 + in_ch_2",
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

		It("Should maintain high water mark across multiple frames", func() {
			// First frame: send data for both channels
			inSeries1 := telem.NewSeriesV[float32](1, 2, 3)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 3)

			inSeries2 := telem.NewSeriesV[float32](10, 20, 30)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries := MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			// Should calculate all 3 samples
			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(0)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](11, 22, 33))

			// Second frame: send more data with alignment starting after high water mark
			inSeries1 = telem.NewSeriesV[float32](4, 5, 6)
			inSeries1.Alignment = 3
			inSeries1.TimeRange = telem.NewRangeSeconds(3, 6)

			inSeries2 = telem.NewSeriesV[float32](40, 50, 60)
			inSeries2.Alignment = 3
			inSeries2.TimeRange = telem.NewRangeSeconds(3, 6)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			// Should calculate 3 more samples
			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](44, 55, 66))
		})

		It("Should not recalculate data before high water mark", func() {
			// First frame
			inSeries1 := telem.NewSeriesV[float32](1, 2, 3, 4, 5)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 5)

			inSeries2 := telem.NewSeriesV[float32](10, 20, 30, 40, 50)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 5)

			outSeries := MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(5)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](11, 22, 33, 44, 55))

			// Second frame: send overlapping data
			// Data with alignment 2-6 (overlapping with 2-4 from previous)
			inSeries1 = telem.NewSeriesV[float32](3, 4, 5, 6, 7)
			inSeries1.Alignment = 2
			inSeries1.TimeRange = telem.NewRangeSeconds(2, 7)

			inSeries2 = telem.NewSeriesV[float32](30, 40, 50, 60, 70)
			inSeries2.Alignment = 2
			inSeries2.TimeRange = telem.NewRangeSeconds(2, 7)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			// Should only calculate new data (alignment 5-6)
			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(5)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](66, 77))
		})

		It("Should handle gaps in data with high water mark", func() {
			// First frame: ch1 only
			inSeries1 := telem.NewSeriesV[float32](1, 2, 3)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries := MustSucceed(calc.Next(core.UnaryFrame(
				inCh1.Key(),
				inSeries1,
			)))

			// No output yet, waiting for ch2
			Expect(outSeries.Len()).To(Equal(int64(0)))

			// Second frame: ch2 with partial overlap
			inSeries2 := telem.NewSeriesV[float32](10, 20)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 2)

			outSeries = MustSucceed(calc.Next(core.UnaryFrame(
				inCh2.Key(),
				inSeries2,
			)))

			// Should calculate first 2 samples
			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(0)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](11, 22))

			// Third frame: more ch2 data
			inSeries2 = telem.NewSeriesV[float32](30, 40, 50)
			inSeries2.Alignment = 2
			inSeries2.TimeRange = telem.NewRangeSeconds(2, 5)

			outSeries = MustSucceed(calc.Next(core.UnaryFrame(
				inCh2.Key(),
				inSeries2,
			)))

			// Should calculate third sample (alignment 2)
			Expect(outSeries.Len()).To(Equal(int64(1)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(2)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](33))

			// Fourth frame: more ch1 data
			inSeries1 = telem.NewSeriesV[float32](4, 5, 6)
			inSeries1.Alignment = 3
			inSeries1.TimeRange = telem.NewRangeSeconds(3, 6)

			outSeries = MustSucceed(calc.Next(core.UnaryFrame(
				inCh1.Key(),
				inSeries1,
			)))

			// Should calculate alignment 3-4
			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](44, 55))
		})

		It("Should correctly update high water mark with different domain indices", func() {
			// First frame with domain index 0
			inSeries1 := telem.NewSeriesV[float32](1, 2, 3)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 3)

			inSeries2 := telem.NewSeriesV[float32](10, 20, 30)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries := MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(3)))

			// Second frame with new domain index (simulating time jump)
			inSeries1 = telem.NewSeriesV[float32](4, 5, 6)
			inSeries1.Alignment = telem.NewAlignment(1, 0)
			inSeries1.TimeRange = telem.NewRangeSeconds(100, 103)

			inSeries2 = telem.NewSeriesV[float32](40, 50, 60)
			inSeries2.Alignment = telem.NewAlignment(1, 0)
			inSeries2.TimeRange = telem.NewRangeSeconds(100, 103)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			// Should calculate all 3 samples with new domain
			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries.Alignment.DomainIndex()).To(Equal(uint32(1)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](44, 55, 66))
		})

		It("Should handle single channel with high water mark", func() {
			singleOut := channel.Channel{
				Leaseholder: 1,
				LocalKey:    4,
				Name:        "single_out",
				DataType:    telem.Float32T,
				Expression:  "return in_ch_1 * 3",
			}
			singleCalc := MustSucceed(calculation.OpenCalculator(
				singleOut,
				[]channel.Channel{inCh1},
			))
			defer singleCalc.Close()

			// First frame
			inSeries := telem.NewSeriesV[float32](1, 2, 3)
			inSeries.Alignment = 0
			inSeries.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries := MustSucceed(singleCalc.Next(core.UnaryFrame(
				inCh1.Key(),
				inSeries,
			)))

			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](3, 6, 9))

			// Second frame with overlap
			inSeries = telem.NewSeriesV[float32](2, 3, 4, 5)
			inSeries.Alignment = 1
			inSeries.TimeRange = telem.NewRangeSeconds(1, 5)

			outSeries = MustSucceed(singleCalc.Next(core.UnaryFrame(
				inCh1.Key(),
				inSeries,
			)))

			// Should only calculate new data (alignment 3-4)
			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](12, 15))
		})

		It("Should handle empty frames without affecting high water mark", func() {
			// First frame with data
			inSeries1 := telem.NewSeriesV[float32](1, 2, 3)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 3)

			inSeries2 := telem.NewSeriesV[float32](10, 20, 30)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries := MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](11, 22, 33))

			// Empty frame
			emptyFrame := core.Frame{}
			outSeries = MustSucceed(calc.Next(emptyFrame))
			Expect(outSeries.Len()).To(Equal(int64(0)))

			// Third frame after empty - should continue from high water mark
			inSeries1 = telem.NewSeriesV[float32](4, 5)
			inSeries1.Alignment = 3
			inSeries1.TimeRange = telem.NewRangeSeconds(3, 5)

			inSeries2 = telem.NewSeriesV[float32](40, 50)
			inSeries2.Alignment = 3
			inSeries2.TimeRange = telem.NewRangeSeconds(3, 5)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](44, 55))
		})

		It("Should handle zero-length series", func() {
			// First frame with zero-length series
			emptySeriesCh1 := telem.NewSeriesV[float32]()
			emptySeriesCh1.Alignment = 0
			emptySeriesCh1.TimeRange = telem.TimeRangeZero

			emptySeriesCh2 := telem.NewSeriesV[float32]()
			emptySeriesCh2.Alignment = 0
			emptySeriesCh2.TimeRange = telem.TimeRangeZero

			outSeries := MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{emptySeriesCh1, emptySeriesCh2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(0)))

			// Second frame with actual data
			inSeries1 := telem.NewSeriesV[float32](1, 2, 3)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 3)

			inSeries2 := telem.NewSeriesV[float32](10, 20, 30)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](11, 22, 33))
		})

		It("Should handle backwards alignment (data arriving out of order)", func() {
			// First frame with alignment 5-8
			inSeries1 := telem.NewSeriesV[float32](5, 6, 7, 8)
			inSeries1.Alignment = 5
			inSeries1.TimeRange = telem.NewRangeSeconds(5, 9)

			inSeries2 := telem.NewSeriesV[float32](50, 60, 70, 80)
			inSeries2.Alignment = 5
			inSeries2.TimeRange = telem.NewRangeSeconds(5, 9)

			outSeries := MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			// Should calculate all 4 samples
			Expect(outSeries.Len()).To(Equal(int64(4)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(5)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](55, 66, 77, 88))

			// Second frame with earlier alignment (2-4) - should be ignored
			inSeries1 = telem.NewSeriesV[float32](2, 3, 4)
			inSeries1.Alignment = 2
			inSeries1.TimeRange = telem.NewRangeSeconds(2, 5)

			inSeries2 = telem.NewSeriesV[float32](20, 30, 40)
			inSeries2.Alignment = 2
			inSeries2.TimeRange = telem.NewRangeSeconds(2, 5)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			// Should return empty series as data is before high water mark
			Expect(outSeries.Len()).To(Equal(int64(0)))

			// Third frame continuing from alignment 9
			inSeries1 = telem.NewSeriesV[float32](9, 10)
			inSeries1.Alignment = 9
			inSeries1.TimeRange = telem.NewRangeSeconds(9, 11)

			inSeries2 = telem.NewSeriesV[float32](90, 100)
			inSeries2.Alignment = 9
			inSeries2.TimeRange = telem.NewRangeSeconds(9, 11)

			outSeries = MustSucceed(calc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(9)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](99, 110))
		})

		It("Should handle very large alignment jumps", func() {
			// Create a fresh calculator for this test
			localCalc := MustSucceed(calculation.OpenCalculator(
				out,
				[]channel.Channel{inCh1, inCh2},
			))
			defer localCalc.Close()

			// First frame
			inSeries1 := telem.NewSeriesV[float32](1, 2)
			inSeries1.Alignment = 0
			inSeries1.TimeRange = telem.NewRangeSeconds(0, 2)

			inSeries2 := telem.NewSeriesV[float32](10, 20)
			inSeries2.Alignment = 0
			inSeries2.TimeRange = telem.NewRangeSeconds(0, 2)

			outSeries := MustSucceed(localCalc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](11, 22))

			// Second frame with huge alignment jump (different domain)
			largeAlignment := telem.NewAlignment(1, 0)
			inSeries1 = telem.NewSeriesV[float32](3, 4)
			inSeries1.Alignment = largeAlignment
			inSeries1.TimeRange = telem.NewRangeSeconds(1000000, 1000002)

			inSeries2 = telem.NewSeriesV[float32](30, 40)
			inSeries2.Alignment = largeAlignment
			inSeries2.TimeRange = telem.NewRangeSeconds(1000000, 1000002)

			outSeries = MustSucceed(localCalc.Next(core.MultiFrame(
				[]channel.Key{inCh1.Key(), inCh2.Key()},
				[]telem.Series{inSeries1, inSeries2},
			)))

			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(largeAlignment))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](33, 44))
		})

		It("Should handle channels with no data requirement (constants only)", func() {
			constOut := channel.Channel{
				Leaseholder: 1,
				LocalKey:    5,
				Name:        "const_out",
				DataType:    telem.Float32T,
				Expression:  "return 42.0",
			}
			constCalc := MustSucceed(calculation.OpenCalculator(
				constOut,
				[]channel.Channel{}, // No input channels
			))
			defer constCalc.Close()

			// Send empty frame - should still produce no output since no inputs
			emptyFrame := core.Frame{}
			outSeries := MustSucceed(constCalc.Next(emptyFrame))
			Expect(outSeries.Len()).To(Equal(int64(0)))
		})

		It("Should handle mixed data types in multi-channel calculation", func() {
			intCh := channel.Channel{
				Leaseholder: 1,
				LocalKey:    10,
				Name:        "int_ch",
				DataType:    telem.Int32T,
			}
			floatCh := channel.Channel{
				Leaseholder: 1,
				LocalKey:    11,
				Name:        "float_ch",
				DataType:    telem.Float32T,
			}
			mixedOut := channel.Channel{
				Leaseholder: 1,
				LocalKey:    12,
				Name:        "mixed_out",
				DataType:    telem.Float32T,
				Expression:  "return int_ch + float_ch",
			}
			mixedCalc := MustSucceed(calculation.OpenCalculator(
				mixedOut,
				[]channel.Channel{intCh, floatCh},
			))
			defer mixedCalc.Close()

			intSeries := telem.NewSeriesV[int32](1, 2, 3)
			intSeries.Alignment = 0
			intSeries.TimeRange = telem.NewRangeSeconds(0, 3)

			floatSeries := telem.NewSeriesV[float32](0.5, 1.5, 2.5)
			floatSeries.Alignment = 0
			floatSeries.TimeRange = telem.NewRangeSeconds(0, 3)

			outSeries := MustSucceed(mixedCalc.Next(core.MultiFrame(
				[]channel.Key{intCh.Key(), floatCh.Key()},
				[]telem.Series{intSeries, floatSeries},
			)))

			Expect(outSeries.Len()).To(Equal(int64(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](1.5, 3.5, 5.5))

			// Test high water mark with mixed types
			intSeries = telem.NewSeriesV[int32](4, 5)
			intSeries.Alignment = 3
			intSeries.TimeRange = telem.NewRangeSeconds(3, 5)

			floatSeries = telem.NewSeriesV[float32](3.5, 4.5)
			floatSeries.Alignment = 3
			floatSeries.TimeRange = telem.NewRangeSeconds(3, 5)

			outSeries = MustSucceed(mixedCalc.Next(core.MultiFrame(
				[]channel.Key{intCh.Key(), floatCh.Key()},
				[]telem.Series{intSeries, floatSeries},
			)))

			Expect(outSeries.Len()).To(Equal(int64(2)))
			Expect(outSeries.Alignment).To(Equal(telem.Alignment(3)))
			Expect(outSeries).To(telem.MatchSeriesDataV[float32](7.5, 9.5))
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
