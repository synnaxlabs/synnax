// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete", Ordered, func() {
	var (
		db        *unary.DB
		indexDB   *unary.DB
		rateDB    *unary.DB
		index     cesium.ChannelKey = 11
		data      cesium.ChannelKey = 12
		rate      cesium.ChannelKey = 13
		defaultFS                   = fs.Default
		pth1                        = "./tests/synnaxUnaryTestsIndex"
		pth2                        = "./tests/synnaxUnaryTests"
		pth3                        = "./tests/synnaxUnaryTestsRate"
	)
	BeforeEach(func() {
		By("Creating channels")
		indexDB = MustSucceed(unary.Open(unary.Config{
			FS: MustSucceed(fs.Default.Sub(pth1)),
			Channel: core.Channel{
				Key:      index,
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Index:    index,
			},
		}))
		db = MustSucceed(unary.Open(unary.Config{
			FS: MustSucceed(fs.Default.Sub(pth2)),
			Channel: core.Channel{
				Key:      data,
				DataType: telem.Int64T,
				Index:    index,
			},
		}))
		rateDB = MustSucceed(unary.Open(unary.Config{
			FS: MustSucceed(fs.Default.Sub(pth3)),
			Channel: core.Channel{
				Key:      rate,
				DataType: telem.Int64T,
				Rate:     1 * telem.Hz,
			},
		}))
		db.SetIndex(indexDB.Index())
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
		Expect(indexDB.Close()).To(Succeed())
		Expect(rateDB.Close()).To(Succeed())
		Expect(defaultFS.Remove("tests")).To(Succeed())
	})
	Describe("Simple Rate-based channel", func() {
		It("Should delete chunks of a channel", func() {
			By("Writing data to the channel")
			Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18))).To(Succeed())

			// Data before deletion: 10, 11, 12, 13, 14, 15, 16, 17, 18

			By("Deleting channel data")
			Expect(rateDB.Delete(ctx, telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   15 * telem.SecondTS,
			})).To(Succeed())

			// Data after deletion: 10, 11, __, __, __, 15, 16, 17, 18
			iter := rateDB.OpenIterator(unary.IterRange((10 * telem.SecondTS).SpanRange(10 * telem.Second)))
			Expect(iter.SeekFirst(ctx)).To(BeTrue())

			frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(2))

			Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
			series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
			Expect(series0Data).To(ContainElement(10))
			Expect(series0Data).To(ContainElement(11))
			Expect(series0Data).ToNot(ContainElement(12))

			Expect(frame.Series[1].TimeRange.Start).To(Equal(15 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
			Expect(series1Data).ToNot(ContainElement(14))
			Expect(series1Data).To(ContainElement(15))
			Expect(series1Data).To(ContainElement(16))
			Expect(series1Data).To(ContainElement(17))
			Expect(series1Data).To(ContainElement(18))
		})
	})

	Describe("Simple Index-based channel", func() {
		It("Should delete chunks of a channel", func() {
			By("Writing data to the channel")
			Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())
			Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9))).To(Succeed())

			// Before deletion:
			// 10 11 12 13 14 15 16 17 18 19
			//  0  1  2  3  4  5  6  7  8  9

			By("Deleting channel data")
			Expect(db.Delete(ctx, telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   17 * telem.SecondTS,
			})).To(Succeed())

			// After deletion:
			// 10 11 12 13 14 15 16 17 18 19
			//  0  1                 7  8  9

			frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(2))
			Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))

			series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
			Expect(series0Data).To(ContainElement(0))
			Expect(series0Data).To(ContainElement(1))
			Expect(series0Data).ToNot(ContainElement(2))

			Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)

			Expect(series1Data).ToNot(ContainElement(6))
			Expect(series1Data).To(ContainElement(7))
			Expect(series1Data).To(ContainElement(8))
			Expect(series1Data).To(ContainElement(9))
		})
	})

	Describe("Deleting simple index channel", func() {
		It("Should Delete chunks off the index channel", func() {
			By("Writing data to the channel")
			// Before deletion:
			// 10 11 12 13 14 15 16 17 18 19
			//  0  1  2  3  4  5  6  7  8  9
			Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())

			By("Deleting channel data")
			Expect(indexDB.Delete(ctx, telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   17 * telem.SecondTS,
			})).To(Succeed())

			// After deletion:
			// 10 11                17 18 19

			frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(2))

			series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
			Expect(series0Data).To(ContainElement(10 * telem.SecondTS))
			Expect(series0Data).To(ContainElement(11 * telem.SecondTS))
			Expect(series0Data).ToNot(ContainElement(12 * telem.SecondTS))

			Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)

			Expect(series1Data).ToNot(ContainElement(16 * telem.SecondTS))
			Expect(series1Data).To(ContainElement(17 * telem.SecondTS))
			Expect(series1Data).To(ContainElement(18 * telem.SecondTS))
			Expect(series1Data).To(ContainElement(19 * telem.SecondTS))
		})
	})

	Describe("Deleting Time-based channel across multiple pointers", func() {
		It("Should complete such deletions with the appropriate pointers and tombstones", func() {
			By("Writing data to the channel")
			for i := 1; i <= 9; i++ {
				var data []int64
				for j := 0; j <= 9; j++ {
					data = append(data, int64(i*10+j))
				}
				Expect(unary.Write(ctx, rateDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
			}

			// should have been written to 10 - 99
			By("Deleting channel data")
			Expect(rateDB.Delete(ctx, telem.TimeRange{
				Start: 33 * telem.SecondTS,
				End:   75 * telem.SecondTS,
			})).To(Succeed())

			frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(6))

			Expect(frame.Series[2].TimeRange.End).To(Equal(33 * telem.SecondTS))
			series0Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
			Expect(series0Data).To(ContainElement(31))
			Expect(series0Data).To(ContainElement(32))
			Expect(series0Data).ToNot(ContainElement(33))

			Expect(frame.Series[3].TimeRange.Start).To(Equal(75 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[int](frame.Series[3].Data, telem.Int64T)
			Expect(series1Data).ToNot(ContainElement(74))
			Expect(series1Data).To(ContainElement(75))

			Expect(frame.Series[5].TimeRange.End).To(BeNumerically("<", 100*telem.SecondTS))
		})

		It("Should work for deleting whole pointers", func() {
			By("Writing data to the channel")
			for i := 1; i <= 9; i++ {
				var data []int64
				for j := 0; j <= 9; j++ {
					data = append(data, int64(i*10+j))
				}
				Expect(unary.Write(ctx, rateDB, telem.TimeStamp(10*i)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
			}

			// should have been written to 10 - 99
			By("Deleting channel data")
			Expect(rateDB.Delete(ctx, telem.TimeRange{
				Start: 20 * telem.SecondTS,
				End:   50 * telem.SecondTS,
			})).To(Succeed())

			frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(6))

			series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
			Expect(series0Data).ToNot(ContainElement(20))

			Expect(frame.Series[1].TimeRange.Start).To(Equal(50 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
			Expect(series1Data).ToNot(ContainElement(49))
			Expect(series1Data).To(ContainElement(50))

			Expect(frame.Series[5].TimeRange.End).To(BeNumerically("<", 100*telem.SecondTS))
		})

		It("Should work for deleting whole pointers in an indexed channel", func() {
			By("Writing data to the channel")
			for i := 1; i <= 9; i++ {
				var index []telem.TimeStamp
				var content []int64
				for j := 0; j <= 9; j++ {
					index = append(index, telem.TimeStamp(i*10+j))
					content = append(content, int64(i*100+j*10))
				}
				Expect(unary.Write(ctx, indexDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSecondsTSV(index...))).To(Succeed())
				Expect(unary.Write(ctx, db, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSeriesV[int64](content...))).To(Succeed())
			}

			// should have been written to 10 - 99
			By("Deleting channel data")
			Expect(db.Delete(ctx, telem.TimeRange{
				Start: 20 * telem.SecondTS,
				End:   50 * telem.SecondTS,
			})).To(Succeed())

			frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(6))

			series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
			Expect(series0Data).ToNot(ContainElement(200))

			Expect(frame.Series[1].TimeRange.Start).To(Equal(50 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
			Expect(series1Data).ToNot(ContainElement(490))
			Expect(series1Data).To(ContainElement(500))

			Expect(frame.Series[5].TimeRange.End).To(BeNumerically("<", 100*telem.SecondTS))
		})
	})
})
