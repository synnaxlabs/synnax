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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	testutil2 "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS:"+fsName, func() {
			var (
				db      *unary.DB
				indexDB *unary.DB
				rateDB  *unary.DB
				index   uint32 = 1
				data    uint32 = 2
				rate    uint32 = 3
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeEach(func() {
				By("Creating channels")
				fs, cleanUp = makeFS()
				indexDB = MustSucceed(unary.Open(unary.Config{
					FS:        MustSucceed(fs.Sub("index")),
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      index,
						DataType: telem.TimeStampT,
						IsIndex:  true,
						Index:    index,
					},
					Instrumentation: PanicLogger(),
				}))
				db = MustSucceed(unary.Open(unary.Config{
					FS:        MustSucceed(fs.Sub("data")),
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      data,
						DataType: telem.Int64T,
						Index:    index,
					},
					Instrumentation: PanicLogger(),
				}))
				rateDB = MustSucceed(unary.Open(unary.Config{
					FS:        MustSucceed(fs.Sub("rate")),
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      rate,
						DataType: telem.Int64T,
						Rate:     1 * telem.Hz,
					},
					Instrumentation: PanicLogger(),
				}))
				db.SetIndex(indexDB.Index())
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(indexDB.Close()).To(Succeed())
				Expect(rateDB.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Single-domain deletion", func() {
				Context("Rate-based channels", func() {
					BeforeEach(func() {
						Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18))).To(Succeed())
					})
					It("Should delete chunks of a channel with both exact timestamps", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   15 * telem.SecondTS,
						})).To(Succeed())

						// Data after deletion: 10, 11, __, __, __, 15, 16, 17, 18
						frame := MustSucceed(rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}))
						Expect(frame.Series).To(HaveLen(2))

						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(10, 11))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(15 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).To(ConsistOf(15, 16, 17, 18))
					})
					It("Should delete a chunk even when the time range is not exact", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 13*telem.SecondTS + 500*telem.MillisecondTS,
							End:   16*telem.SecondTS + 500*telem.MillisecondTS,
						})).To(Succeed())

						frame := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
						Expect(frame.Series).To(HaveLen(2))

						Expect(frame.Series[0].TimeRange.End).To(Equal(13*telem.SecondTS + 1))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(10, 11, 12, 13))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).To(ConsistOf(17, 18))
					})
					It("Should delete a chunk when the end of the delete is the end of the pointer", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 13*telem.SecondTS + 500*telem.MillisecondTS,
							End:   18*telem.SecondTS + 1,
						})).To(Succeed())

						frame := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.End).To(Equal(13*telem.SecondTS + 1))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(10, 11, 12, 13))
					})
					It("Should delete a chunk when the start of the delete is the start of the pointer", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 10 * telem.SecondTS,
							End:   16*telem.SecondTS + 500*telem.MillisecondTS,
						})).To(Succeed())

						frame := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(17, 18))
					})
					It("Should delete a chunk when the start is out of the pointer", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 8 * telem.SecondTS,
							End:   17*telem.SecondTS + 500*telem.MillisecondTS,
						})).To(Succeed())

						frame := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.Start).To(Equal(18 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(18))
					})
					It("Should delete a chunk when the end is out of the pointer", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 13*telem.SecondTS + 500*telem.MillisecondTS,
							End:   20 * telem.SecondTS,
						})).To(Succeed())

						frame := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.End).To(Equal(13*telem.SecondTS + 1))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(10, 11, 12, 13))
					})
					It("Should delete a whole domain when both timestamps are out of the pointer", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 8 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						})).To(Succeed())

						iter := rateDB.OpenIterator(unary.IterRange((10 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.SeekFirst(ctx)).To(BeFalse())
						Expect(iter.Close()).To(Succeed())
					})
					It("Should not delete the first element of the domain when the start index is start + 1", func() {
						By("Deleting channel data")
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 10*telem.SecondTS + 1,
							End:   14*telem.SecondTS + 1,
						})).To(Succeed())

						frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())

						Expect(frame.Series).To(HaveLen(2))
						Expect(frame.Series[0].TimeRange.End).To(Equal(10*telem.SecondTS + 1))
						series0Data := telem.UnmarshalSlice[int64](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(int64(10)))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(15 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).To(ConsistOf(15, 16, 17, 18))
					})
				})

				Context("Index channels", func() {
					It("Should delete chunks of a channel with both exact timestamps", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9))).To(Succeed())

						By("Deleting channel data")
						Expect(db.Delete(ctx, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})).To(Succeed())

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())

						Expect(frame.Series).To(HaveLen(2))
						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(0, 1))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).To(ConsistOf(7, 8, 9))
					})
					It("Should delete chunks of a channel without exact timestamps", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 12, 15, 17, 19, 20, 23))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 12, 15, 17, 19, 20, 23))).To(Succeed())

						By("Deleting channel data")
						// 10 12 / 17 19 20 23
						Expect(db.Delete(ctx, (13 * telem.SecondTS).Range(16*telem.SecondTS))).To(Succeed())
						Expect(db.Delete(ctx, (19 * telem.SecondTS).Range(21*telem.SecondTS))).To(Succeed())
						Expect(db.Delete(ctx, (11 * telem.SecondTS).Range(14*telem.SecondTS))).To(Succeed())

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 25 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())
						Expect(frame.Series).To(HaveLen(3))

						Expect(frame.Series[0].TimeRange.End).To(Equal(10*telem.SecondTS + 1))
						Expect(frame.Series[0].Data).To(Equal(telem.NewSeriesV[int64](10).Data))
						Expect(frame.Series[1].TimeRange).To(Equal((17 * telem.SecondTS).Range(19 * telem.SecondTS)))
						Expect(frame.Series[1].Data).To(Equal(telem.NewSeriesV[int64](17).Data))
						Expect(frame.Series[2].TimeRange).To(Equal((23 * telem.SecondTS).Range(23*telem.SecondTS + 1)))
						Expect(frame.Series[2].Data).To(Equal(telem.NewSeriesV[int64](23).Data))
					})
					It("Should delete chunks of a channel if the start is out of the pointer", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9))).To(Succeed())

						By("Deleting channel data")
						Expect(db.Delete(ctx, telem.TimeRange{
							Start: 9 * telem.SecondTS,
							End:   17*telem.SecondTS + 1,
						})).To(Succeed())

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.Start).To(Equal(18 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series1Data).To(ConsistOf(8, 9))
					})
					It("Should delete chunks of a channel if the end is out of the pointer", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9))).To(Succeed())

						By("Deleting channel data")
						Expect(db.Delete(ctx, telem.TimeRange{
							Start: 13 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						})).To(Succeed())

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.End).To(Equal(13 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(0, 1, 2))
					})
					It("Should delete the whole channel", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9))).To(Succeed())

						By("Deleting channel data")
						Expect(db.Delete(ctx, telem.TimeRange{
							Start: 10 * telem.SecondTS,
							End:   19*telem.SecondTS + 1,
						})).To(Succeed())

						i := db.OpenIterator(unary.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})

				Context("Indexed channels", func() {
					It("Should delete chunks of a channel with both exact timestamps", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())

						By("Deleting channel data")
						Expect(indexDB.Delete(ctx, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})).To(Succeed())

						frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())

						Expect(frame.Series).To(HaveLen(2))
						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
						Expect(series0Data).To(ConsistOf(10*telem.SecondTS, 11*telem.SecondTS))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)
						Expect(series1Data).To(ConsistOf(17*telem.SecondTS, 18*telem.SecondTS, 19*telem.SecondTS))
					})
					It("Should delete chunks of a channel without exact timestamps", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())

						By("Deleting channel data")
						Expect(indexDB.Delete(ctx, telem.TimeRange{
							Start: 12*telem.SecondTS + 300*telem.MillisecondTS,
							End:   17*telem.SecondTS + 1,
						})).To(Succeed())

						frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())

						Expect(frame.Series).To(HaveLen(2))
						Expect(frame.Series[0].TimeRange.End).To(Equal(12*telem.SecondTS + 1))
						series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
						Expect(series0Data).To(ConsistOf(10*telem.SecondTS, 11*telem.SecondTS, 12*telem.SecondTS))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(18 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)
						Expect(series1Data).To(ConsistOf(18*telem.SecondTS, 19*telem.SecondTS))
					})
					It("Should delete chunks of a channel if the start is out of the pointer", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())

						By("Deleting channel data")
						Expect(indexDB.Delete(ctx, telem.TimeRange{
							Start: 9 * telem.SecondTS,
							End:   17*telem.SecondTS + 1,
						})).To(Succeed())

						frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())

						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.Start).To(Equal(18 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
						Expect(series1Data).To(ConsistOf(18*telem.SecondTS, 19*telem.SecondTS))
					})
					It("Should delete chunks of a channel if the end is out of the pointer", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())

						By("Deleting channel data")
						Expect(indexDB.Delete(ctx, telem.TimeRange{
							Start: 12*telem.SecondTS + 300*telem.MillisecondTS,
							End:   33 * telem.SecondTS,
						})).To(Succeed())

						frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())

						Expect(frame.Series).To(HaveLen(1))
						Expect(frame.Series[0].TimeRange.End).To(Equal(12*telem.SecondTS + 1))
						series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
						Expect(series0Data).To(ConsistOf(10*telem.SecondTS, 11*telem.SecondTS, 12*telem.SecondTS))
					})
					It("Should delete the whole channel", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())

						By("Deleting channel data")
						Expect(indexDB.Delete(ctx, telem.TimeRange{
							Start: 10 * telem.SecondTS,
							End:   19*telem.SecondTS + 1,
						})).To(Succeed())

						i := indexDB.OpenIterator(unary.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeFalse())
						Expect(i.Close()).To(Succeed())
					})
				})
			})

			Describe("Cross-domain deletion", func() {
				Context("Rate-based channels", func() {
					BeforeEach(func() {
						By("Writing data to the channel")
						Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13)))
						Expect(unary.Write(ctx, rateDB, 20*telem.SecondTS, telem.NewSeriesV[int64](20, 21, 22, 23, 24)))
						Expect(unary.Write(ctx, rateDB, 30*telem.SecondTS, telem.NewSeriesV[int64](30, 31, 32, 33, 34, 35, 36, 37)))
					})
					It("Should delete across two such domains", func() {
						By("Deleting channel data")
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   32 * telem.SecondTS,
						})).To(Succeed())

						frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())
						Expect(frame.Series).To(HaveLen(2))

						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ContainElement(10))
						Expect(series0Data).To(ContainElement(11))
						Expect(series0Data).ToNot(ContainElement(12))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(32 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).ToNot(ContainElement(31))
						Expect(series1Data).To(ContainElement(32))
					})
					It("Should delete full domains", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 10 * telem.SecondTS,
							End:   30 * telem.SecondTS,
						})).To(Succeed())

						frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
						Expect(err).ToNot(HaveOccurred())
						Expect(frame.Series).To(HaveLen(1))

						Expect(frame.Series[0].TimeRange.Start).To(Equal(30 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ContainElement(30))
						Expect(series0Data).To(ContainElement(31))
						Expect(series0Data).ToNot(ContainElement(29))
					})
					It("Should delete entire db", func() {
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 10 * telem.SecondTS,
							End:   38 * telem.SecondTS,
						})).To(Succeed())

						frame, err := rateDB.Read(ctx, telem.TimeRangeMax)
						Expect(err).ToNot(HaveOccurred())
						Expect(frame.Series).To(HaveLen(0))
					})
				})
				Context("Indexed channels", func() {
					Context("Two pointers", func() {
						BeforeEach(func() {
							By("Writing data to the channel")
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesV[telem.TimeStamp](10*telem.SecondTS, 13*telem.SecondTS, 13*telem.SecondTS+500*telem.MillisecondTS, 18*telem.SecondTS, 19*telem.SecondTS))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 13, 131, 18, 19))).To(Succeed())

							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesV[telem.TimeStamp](20*telem.SecondTS, 23500*telem.MillisecondTS, 23600*telem.MillisecondTS, 23800*telem.MillisecondTS, 25100*telem.MillisecondTS, 27800*telem.MillisecondTS))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](200, 235, 236, 238, 251, 278))).To(Succeed())
						})
						It("Should delete across two such domains", func() {
							By("Deleting channel data")
							Expect(db.Delete(ctx, telem.TimeRange{
								Start: 13*telem.SecondTS + 400*telem.MillisecondTS,
								End:   24 * telem.SecondTS,
							})).To(Succeed())

							frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
							Expect(err).ToNot(HaveOccurred())
							Expect(frame.Series).To(HaveLen(2))

							Expect(frame.Series[0].TimeRange.End).To(Equal(13*telem.SecondTS + 1))
							series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
							Expect(series0Data).To(ConsistOf(10, 13))

							Expect(frame.Series[1].TimeRange.Start).To(Equal(25100 * telem.MillisecondTS))
							series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
							Expect(series1Data).To(ConsistOf(251, 278))
						})
						It("Should delete full domains", func() {
							Expect(db.Delete(ctx, telem.TimeRange{
								Start: 10 * telem.SecondTS,
								End:   20 * telem.SecondTS,
							})).To(Succeed())

							frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
							Expect(err).ToNot(HaveOccurred())
							Expect(frame.Series).To(HaveLen(1))

							Expect(frame.Series[0].TimeRange.Start).To(Equal(20 * telem.SecondTS))
							series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
							Expect(series0Data).To(ConsistOf(200, 235, 236, 238, 251, 278))
						})
						It("Should delete entire db", func() {
							Expect(db.Delete(ctx, telem.TimeRangeMax)).To(Succeed())

							frame, err := rateDB.Read(ctx, telem.TimeRangeMax)
							Expect(err).ToNot(HaveOccurred())
							Expect(frame.Series).To(HaveLen(0))
						})
					})
					Context("Multiple pointers", func() {
						It("Should complete such deletions with the appropriate pointers and tombstones", func() {
							By("Writing data to the channel")
							for i := 1; i <= 9; i++ {
								var data []int64
								for j := 0; j <= 9; j++ {
									data = append(data, int64(i*10+j))
								}
								Expect(unary.Write(ctx, rateDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
							}

							By("Deleting channel data")
							Expect(rateDB.Delete(ctx, telem.TimeRange{
								Start: 33 * telem.SecondTS,
								End:   75 * telem.SecondTS,
							})).To(Succeed())

							frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
							Expect(err).ToNot(HaveOccurred())
							Expect(frame.Series).To(HaveLen(6))

							Expect(frame.Series[2].TimeRange.End).To(Equal(33 * telem.SecondTS))
							series2Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
							Expect(series2Data).To(ConsistOf(30, 31, 32))

							Expect(frame.Series[3].TimeRange.Start).To(Equal(75 * telem.SecondTS))
							series3Data := telem.UnmarshalSlice[int](frame.Series[3].Data, telem.Int64T)
							Expect(series3Data).To(ConsistOf(75, 76, 77, 78, 79))

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

							By("Deleting channel data")
							Expect(rateDB.Delete(ctx, telem.TimeRange{
								Start: 20 * telem.SecondTS,
								End:   50 * telem.SecondTS,
							})).To(Succeed())

							frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
							Expect(err).ToNot(HaveOccurred())
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

							By("Deleting channel data")
							Expect(db.Delete(ctx, telem.TimeRange{
								Start: 20 * telem.SecondTS,
								End:   50 * telem.SecondTS,
							})).To(Succeed())

							frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
							Expect(err).ToNot(HaveOccurred())
							Expect(frame.Series).To(HaveLen(6))

							series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
							Expect(series0Data).ToNot(ContainElement(200))

							Expect(frame.Series[1].TimeRange.Start).To(Equal(50 * telem.SecondTS))
							series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
							Expect(series1Data).ToNot(ContainElement(490))
							Expect(series1Data).To(ContainElement(500))

							Expect(frame.Series[5].TimeRange.End).To(BeNumerically("<", 100*telem.SecondTS))
						})

						It("Should add up to delete the whole channel", func() {
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

							timeRanges := []telem.TimeRange{
								{Start: 20 * telem.SecondTS, End: 50 * telem.SecondTS},
								{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS},
								{Start: 50 * telem.SecondTS, End: 57 * telem.SecondTS},
								{Start: 65 * telem.SecondTS, End: 79 * telem.SecondTS},
								{Start: 79 * telem.SecondTS, End: 86 * telem.SecondTS},
								{Start: 87 * telem.SecondTS, End: 120 * telem.SecondTS},
								{Start: 20 * telem.SecondTS, End: 50 * telem.SecondTS},
								{Start: 12 * telem.SecondTS, End: 32 * telem.SecondTS},
								{Start: 86 * telem.SecondTS, End: 87 * telem.SecondTS},
								{Start: 55 * telem.SecondTS, End: 66 * telem.SecondTS},
							}

							By("Deleting channel data")
							for _, tr := range timeRanges {
								Expect(db.Delete(ctx, tr)).To(Succeed())
							}

							f := MustSucceed(db.Read(ctx, telem.TimeRangeMax))
							Expect(f.Series).To(HaveLen(0))
						})
					})
				})
			})
			Context("Index channels", func() {
				BeforeEach(func() {
					By("Writing data to the channel")
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13)))
					Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 24)))
					Expect(unary.Write(ctx, indexDB, 30*telem.SecondTS, telem.NewSecondsTSV(30, 31, 33, 34, 35, 36, 37)))
				})

				It("Should delete between two domains", func() {
					Expect(indexDB.Delete(ctx, telem.TimeRange{Start: 12*telem.SecondTS + 500*telem.MillisecondTS, End: 24 * telem.SecondTS})).To(Succeed())
					frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(3))

					Expect(frame.Series[0].TimeRange.End).To(Equal(12*telem.SecondTS + 1))
					series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
					Expect(series0Data).To(ConsistOf(10*telem.SecondTS, 11*telem.SecondTS, 12*telem.SecondTS))

					Expect(frame.Series[1].TimeRange.Start).To(Equal(24 * telem.SecondTS))
					Expect(frame.Series[1].TimeRange.End).To(Equal(24*telem.SecondTS + 1))
					series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)
					Expect(series1Data).To(ConsistOf(24 * telem.SecondTS))
				})

				It("Should delete between multiple domains", func() {
					Expect(indexDB.Delete(ctx, telem.TimeRange{Start: 12*telem.SecondTS + 500*telem.MillisecondTS, End: 32 * telem.SecondTS})).To(Succeed())
					frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(2))

					Expect(frame.Series[0].TimeRange.End).To(Equal(12*telem.SecondTS + 1))
					series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
					Expect(series0Data).To(ConsistOf(10*telem.SecondTS, 11*telem.SecondTS, 12*telem.SecondTS))

					Expect(frame.Series[1].TimeRange.Start).To(Equal(33 * telem.SecondTS))
					Expect(frame.Series[1].TimeRange.End).To(Equal(37*telem.SecondTS + 1))
					series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)
					Expect(series1Data).To(ConsistOf(33*telem.SecondTS, 34*telem.SecondTS, 35*telem.SecondTS, 36*telem.SecondTS, 37*telem.SecondTS))
				})

				It("Should delete a domain entirely", func() {
					Expect(indexDB.Delete(ctx, telem.TimeRange{Start: 12*telem.SecondTS + 500*telem.MillisecondTS, End: 25 * telem.SecondTS})).To(Succeed())
					frame, err := indexDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(2))

					Expect(frame.Series[0].TimeRange.End).To(Equal(12*telem.SecondTS + 1))
					series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
					Expect(series0Data).To(ConsistOf(10*telem.SecondTS, 11*telem.SecondTS, 12*telem.SecondTS))

					Expect(frame.Series[1].TimeRange.Start).To(Equal(30 * telem.SecondTS))
					series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)
					Expect(series1Data).To(ConsistOf(30*telem.SecondTS, 31*telem.SecondTS, 33*telem.SecondTS, 34*telem.SecondTS, 35*telem.SecondTS, 36*telem.SecondTS, 37*telem.SecondTS))
				})
			})
			Context("Overshooting time range", func() {
				BeforeEach(func() {
					By("Writing data to the channel")
					Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13)))
					Expect(unary.Write(ctx, rateDB, 20*telem.SecondTS, telem.NewSeriesV[int64](20, 21, 22, 23, 24)))
					Expect(unary.Write(ctx, rateDB, 30*telem.SecondTS, telem.NewSeriesV[int64](30, 31, 32, 33, 34, 35, 36, 37)))
				})
				It("Should delete even when the start timestamp is not in bounds of a pointer", func() {
					By("Deleting channel data")
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 18 * telem.SecondTS,
						End:   32 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(2))

					Expect(frame.Series[0].TimeRange.End).To(BeNumerically(">", 13*telem.SecondTS))
					Expect(frame.Series[0].TimeRange.End).To(BeNumerically("<", 14*telem.SecondTS))
					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(series0Data).To(ConsistOf(10, 11, 12, 13))

					Expect(frame.Series[1].TimeRange.Start).To(Equal(32 * telem.SecondTS))
					series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
					Expect(series1Data).To(ConsistOf(32, 33, 34, 35, 36, 37))
				})
				It("Should delete even when the start timestamp is not in bounds of the db", func() {
					By("Deleting channel data")
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 8 * telem.SecondTS,
						End:   32 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 3 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(1))

					Expect(frame.Series[0].TimeRange.Start).To(Equal(32 * telem.SecondTS))
					series1Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(series1Data).ToNot(ContainElement(31))
					Expect(series1Data).To(ContainElement(32))
				})
				It("Should delete even when the end timestamp is not in bounds of a pointer", func() {
					By("Deleting channel data")
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 23 * telem.SecondTS,
						End:   26 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 20 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(2))

					Expect(frame.Series[0].TimeRange.End).To(Equal(23 * telem.SecondTS))
					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(series0Data).To(ConsistOf(20, 21, 22))

					Expect(frame.Series[1].TimeRange.Start).To(Equal(30 * telem.SecondTS))
					series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
					Expect(series1Data).To(ContainElement(30))
					Expect(series1Data).To(ContainElement(31))
				})
				It("Should delete even when the end timestamp is not in bounds of a pointer", func() {
					By("Deleting channel data")
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 12 * telem.SecondTS,
						End:   10123 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 3 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(1))

					Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
					series1Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(series1Data).ToNot(ContainElement(12))
					Expect(series1Data).To(ContainElement(11))
				})
				It("Should delete even when both timestamps are not in bounds of a pointer", func() {
					By("Deleting channel data")
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 16 * telem.SecondTS,
						End:   29 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 2 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())
					Expect(frame.Series).To(HaveLen(2))

					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(series0Data).To(ConsistOf(10, 11, 12, 13))

					Expect(frame.Series[1].TimeRange.Start).To(Equal(30 * telem.SecondTS))
					series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
					Expect(series1Data).To(ConsistOf(30, 31, 32, 33, 34, 35, 36, 37))
				})
			})

			Context("Delete Nothing", func() {
				BeforeEach(func() {
					By("Writing data to the channel")
					Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13)))
					Expect(unary.Write(ctx, rateDB, 20*telem.SecondTS, telem.NewSeriesV[int64](20, 21, 22, 23, 24)))
					Expect(unary.Write(ctx, rateDB, 30*telem.SecondTS, telem.NewSeriesV[int64](30, 31, 32, 33, 34, 35, 36, 37)))
				})

				It("Should only delete one sample when Start = End - 1", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 20 * telem.SecondTS,
						End:   21 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 20 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(frame.Series[0].TimeRange.Start).To(Equal(21 * telem.SecondTS))
					Expect(series0Data).ToNot(ContainElement(20))
					Expect(series0Data).To(ContainElement(21))
				})

				It("Should delete nothing when there is no data before end", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 0,
						End:   5 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRangeMax)
					Expect(err).ToNot(HaveOccurred())

					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(frame.Series[0].TimeRange.Start).To(Equal(10 * telem.SecondTS))
					Expect(series0Data).To(ContainElement(10))
				})

				It("Should delete nothing when there is no data after start", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 41 * telem.SecondTS,
						End:   45 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRangeMax)
					Expect(err).ToNot(HaveOccurred())

					series2Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
					Expect(frame.Series[2].TimeRange.End).To(Equal(37*telem.SecondTS + 1))
					Expect(series2Data).To(ContainElement(37))
				})

				It("Should delete no element when Start = End", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 20 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 20 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(frame.Series[0].TimeRange.Start).To(Equal(20 * telem.SecondTS))
					Expect(series0Data).To(ContainElement(20))
				})

				It("Should delete no element when given a range where both start and end are out of any pointer", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 14 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 20 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(frame.Series[0].TimeRange.Start).To(Equal(20 * telem.SecondTS))
					Expect(series0Data).To(ContainElement(20))

					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 14 * telem.SecondTS,
						End:   19 * telem.SecondTS,
					})).To(Succeed())

					frame, err = rateDB.Read(ctx, telem.TimeRange{Start: 20 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					series0Data = telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(frame.Series[0].TimeRange.Start).To(Equal(20 * telem.SecondTS))
					Expect(series0Data).To(ContainElement(20))
				})

				It("Should delete no element when given a range that contains no samples", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 10 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					})).To(Succeed())

					w, _ := MustSucceed2(rateDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 18 * telem.SecondTS, Subject: control.Subject{Key: "test_writer"}}))
					_, err := w.Write(telem.NewSeriesV[int64](10, 11, 12))
					Expect(err).ToNot(HaveOccurred())
					_, err = w.Commit(ctx)
					Expect(err).ToNot(HaveOccurred())
					_, err = w.Close()
					Expect(err).ToNot(HaveOccurred())

					Expect(rateDB.Delete(ctx, telem.TimeRange{
						Start: 14 * telem.SecondTS,
						End:   20 * telem.SecondTS,
					})).To(Succeed())

					frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 20 * telem.SecondTS, End: 100 * telem.SecondTS})
					Expect(err).ToNot(HaveOccurred())

					series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
					Expect(frame.Series[0].TimeRange.Start).To(Equal(20 * telem.SecondTS))
					Expect(series0Data).To(ContainElement(20))
				})
			})

			Context("Error paths", func() {
				It("Should error when the end timestamp is earlier than start timestamp", func() {
					Expect(rateDB.Delete(ctx, telem.TimeRange{Start: 30 * telem.SecondTS, End: 20 * telem.SecondTS})).To(MatchError(ContainSubstring("after delete end")))
				})
			})

			Describe("Regression", func() {
				// This test addresses a bug where if an index is split into two domains
				// to describe a data channel, and a call to delete that crosses the
				// two domains would result in a discontinuous error.
				//
				// This was critical since while for smaller sample sizes, a data domain
				// would not cross two indices, file cutoff makes this a very common
				// case: writing float32 data, which has a higher density than TimeStamps,
				// always gets cut off after Timestamp data. This makes it so that
				// after enough samples, the index will almost always be split into two
				// for one data domain.
				It("Should work when the index is split into two domains", func() {
					var (
						iKey     = testutil.GenerateChannelKey()
						dbKey    = testutil.GenerateChannelKey()
						indexDB2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("index")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      iKey,
								DataType: telem.TimeStampT,
								IsIndex:  true,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        40 * telem.ByteSize,
						}))
						db2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("data")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      dbKey,
								DataType: telem.Int64T,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        40 * telem.ByteSize,
						}))
					)
					db2.SetIndex(indexDB2.Index())
					w, _ := MustSucceed2(indexDB2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test"}}))
					MustSucceed(w.Write(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSecondsTSV(16, 17, 18, 19, 20)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					Expect(unary.Write(ctx, db2, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))).To(Succeed())
					Expect(db2.Delete(ctx, (13 * telem.SecondTS).Range(18*telem.SecondTS))).To(Succeed())

					Expect(indexDB2.Close()).To(Succeed())
					Expect(db2.Close()).To(Succeed())
				})

				// This test addresses a bug of the same origin as the previous one:
				// it is important to realize that the start of the domain used to find
				// the offset in delete is NOT necessarily the start of the domain in
				// the index channel as well. Previously in the codebase we checked that
				// the ensuing Stamp operation must be exact since we made that assumption,
				// as a result, the codebase DPanic'ed when it did not need to.
				It("Should work when the index approximation is not exact", func() {
					var (
						iKey     = testutil.GenerateChannelKey()
						dbKey    = testutil.GenerateChannelKey()
						indexDB2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("index")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      iKey,
								DataType: telem.TimeStampT,
								IsIndex:  true,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        40 * telem.ByteSize,
						}))
						db2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("data")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      dbKey,
								DataType: telem.Int64T,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        17 * telem.ByteSize,
						}))
					)
					db2.SetIndex(indexDB2.Index())
					w, _ := MustSucceed2(indexDB2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test"}}))
					MustSucceed(w.Write(telem.NewSecondsTSV(10, 12, 14, 16, 18, 20)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSecondsTSV(22, 24, 26, 28, 30)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					w, _ = MustSucceed2(db2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test2"}}))
					MustSucceed(w.Write(telem.NewSeriesV[int64](10, 12, 14, 16, 18)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSeriesV[int64](20, 22, 24, 26, 28, 30)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					Expect(db2.Delete(ctx, (20 * telem.SecondTS).Range(27*telem.SecondTS))).To(Succeed())

					f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax))
					Expect(f.Series).To(HaveLen(2))
					Expect(f.Series[0].Data).To(testutil2.EqualUnmarshal([]int64{10, 12, 14, 16, 18}))
					Expect(f.Series[1].Data).To(testutil2.EqualUnmarshal([]int64{28, 30}))

					Expect(indexDB2.Close()).To(Succeed())
					Expect(db2.Close()).To(Succeed())
				})

				// This test addresses an edge case in the previous test, where we attempt
				// to delete before the first element in a cut-off domain.
				It("Should work when we delete before the first element in a cut-off domain", func() {
					var (
						iKey     = testutil.GenerateChannelKey()
						dbKey    = testutil.GenerateChannelKey()
						indexDB2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("index")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      iKey,
								DataType: telem.TimeStampT,
								IsIndex:  true,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        40 * telem.ByteSize,
						}))
						db2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("data")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      dbKey,
								DataType: telem.Int64T,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        17 * telem.ByteSize,
						}))
					)
					db2.SetIndex(indexDB2.Index())
					w, _ := MustSucceed2(indexDB2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test"}}))
					MustSucceed(w.Write(telem.NewSecondsTSV(10, 12, 14, 16, 18, 20)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSecondsTSV(22, 24, 26, 28, 30)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					w, _ = MustSucceed2(db2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test2"}}))
					MustSucceed(w.Write(telem.NewSeriesV[int64](10, 12, 14, 16, 18, 20)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSeriesV[int64](22, 24, 26, 28, 30)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					Expect(db2.Delete(ctx, (21 * telem.SecondTS).Range(27*telem.SecondTS))).To(Succeed())

					f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax))
					Expect(f.Series).To(HaveLen(2))
					Expect(f.Series[0].Data).To(testutil2.EqualUnmarshal([]int64{10, 12, 14, 16, 18, 20}))
					Expect(f.Series[1].Data).To(testutil2.EqualUnmarshal([]int64{28, 30}))

					Expect(indexDB2.Close()).To(Succeed())
					Expect(db2.Close()).To(Succeed())
				})
				It("Should work when we delete at the first element in a cut-off domain", func() {
					var (
						iKey     = testutil.GenerateChannelKey()
						dbKey    = testutil.GenerateChannelKey()
						indexDB2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("index")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      iKey,
								DataType: telem.TimeStampT,
								IsIndex:  true,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        40 * telem.ByteSize,
						}))
						db2 = MustSucceed(unary.Open(unary.Config{
							FS:        MustSucceed(fs.Sub("data")),
							MetaCodec: codec,
							Channel: core.Channel{
								Key:      dbKey,
								DataType: telem.Int64T,
								Index:    iKey,
							},
							Instrumentation: PanicLogger(),
							FileSize:        17 * telem.ByteSize,
						}))
					)
					db2.SetIndex(indexDB2.Index())
					w, _ := MustSucceed2(indexDB2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test"}}))
					MustSucceed(w.Write(telem.NewSecondsTSV(10, 12, 14, 16, 18, 20)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSecondsTSV(22, 24, 26, 28, 30)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					w, _ = MustSucceed2(db2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test2"}}))
					MustSucceed(w.Write(telem.NewSeriesV[int64](10, 12, 14, 16, 18, 20)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSeriesV[int64](22, 24, 26, 28, 30)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
					Expect(db2.Delete(ctx, (23 * telem.SecondTS).Range(27*telem.SecondTS))).To(Succeed())

					f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax))
					Expect(f.Series).To(HaveLen(3))
					Expect(f.Series[0].Data).To(testutil2.EqualUnmarshal([]int64{10, 12, 14, 16, 18, 20}))
					Expect(f.Series[1].Data).To(testutil2.EqualUnmarshal([]int64{22}))
					Expect(f.Series[2].Data).To(testutil2.EqualUnmarshal([]int64{28, 30}))

					Expect(indexDB2.Close()).To(Succeed())
					Expect(db2.Close()).To(Succeed())
				})
			})

			Describe("HasDataFor", func() {
				It("Should return whether there is data for the given range", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
					Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6))).To(Succeed())
					hasData, err := db.HasDataFor(ctx, (12 * telem.SecondTS).Range(23*telem.SecondTS))
					Expect(hasData).To(BeTrue())
					Expect(err).ToNot(HaveOccurred())

					hasData, err = db.HasDataFor(ctx, (16*telem.SecondTS + 1).Range(25*telem.SecondTS))
					Expect(hasData).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())

					hasData, err = db.HasDataFor(ctx, (5 * telem.SecondTS).Range(10*telem.SecondTS))
					Expect(hasData).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())
				})
				It("Should return true when there is a writer starting before the given time range", func() {
					w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   5 * telem.SecondTS,
						Subject: control.Subject{Key: "foo_writer"},
					}))

					hasData, err := db.HasDataFor(ctx, (12 * telem.SecondTS).Range(23*telem.SecondTS))
					Expect(hasData).To(BeTrue())
					Expect(err).ToNot(HaveOccurred())

					MustSucceed(w.Close())

					hasData, err = db.HasDataFor(ctx, (12 * telem.SecondTS).Range(23*telem.SecondTS))
					Expect(hasData).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())
				})
				It("Should return true when there is a writer starting in the middle of the given time range", func() {
					w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   15 * telem.SecondTS,
						Subject: control.Subject{Key: "foo_writer"},
					}))

					hasData, err := db.HasDataFor(ctx, (12 * telem.SecondTS).Range(23*telem.SecondTS))
					Expect(hasData).To(BeTrue())
					Expect(err).ToNot(HaveOccurred())

					MustSucceed(w.Close())

					hasData, err = db.HasDataFor(ctx, (12 * telem.SecondTS).Range(23*telem.SecondTS))
					Expect(hasData).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())
				})
				It("Should return false when there is a writer starting after the given time range", func() {
					w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   25 * telem.SecondTS,
						Subject: control.Subject{Key: "foo_writer"},
					}))

					hasData, err := db.HasDataFor(ctx, (12 * telem.SecondTS).Range(23*telem.SecondTS))
					Expect(hasData).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())

					MustSucceed(w.Close())
				})
			})
		})
	}
})
