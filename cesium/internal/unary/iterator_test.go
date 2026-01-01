// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			Describe("Channel Indexed", func() {
				var (
					db      *unary.DB
					indexDB *unary.DB
					index   uint32 = 1
					data    uint32 = 2
					fs      fs.FS
					cleanUp func() error
				)
				BeforeEach(func() {
					fs, cleanUp = makeFS()
					indexDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(fs.Sub("index")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Name:     "Alex",
							Key:      index,
							DataType: telem.TimeStampT,
							IsIndex:  true,
							Index:    index,
						},
						Instrumentation: PanicLogger(),
					}))
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(fs.Sub("data")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Name:     "Megos",
							Key:      data,
							DataType: telem.Int64T,
							Index:    index,
						},
						Instrumentation: PanicLogger(),
					}))
					db.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(indexDB.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})

				Describe("Happy Path", func() {
					Specify("Next", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())

						Expect(unary.Write(ctx, indexDB, 16*telem.SecondTS, telem.NewSeriesSecondsTSV(16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 16*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10))).To(Succeed())

						iter := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
						Expect(iter.Next(ctx, 5*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(5 * telem.Second)))
						Expect(iter.Next(ctx, 3*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((15 * telem.SecondTS).SpanRange(3 * telem.Second)))
						Expect(iter.Next(ctx, 5*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((18 * telem.SecondTS).SpanRange(5 * telem.Second)))
						Expect(iter.Value().Len()).To(Equal(int64(2)))
						Expect(iter.Close()).To(Succeed())
					})
					Specify("Prev", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())

						Expect(unary.Write(ctx, indexDB, 16*telem.SecondTS, telem.NewSeriesSecondsTSV(16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 16*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10))).To(Succeed())

						iter := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))
						Expect(iter.SeekLast(ctx)).To(BeTrue())
						Expect(iter.View()).To(Equal((19*telem.SecondTS + 1).SpanRange(0)))
						Expect(iter.Prev(ctx, 5*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((14*telem.SecondTS + 1).SpanRange(5 * telem.Second)))
						Expect(iter.Prev(ctx, 5*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((9*telem.SecondTS + 1).SpanRange(5 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(5)))
						Expect(iter.Prev(ctx, 1*telem.Second)).To(BeFalse())
						Expect(iter.Close()).To(Succeed())
					})
					Specify("Next and Prev", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15))).To(Succeed())

						Expect(unary.Write(ctx, indexDB, 16*telem.SecondTS, telem.NewSeriesSecondsTSV(16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 16*telem.SecondTS, telem.NewSeriesV[int64](16, 17, 18, 19))).To(Succeed())

						Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 23))).To(Succeed())
						Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](20, 23))).To(Succeed())

						iter := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))
						// Iter window: [15*telem.SecondTS, 18*telem.SecondTS)
						Expect(iter.SeekGE(ctx, 15*telem.SecondTS)).To(BeTrue())
						Expect(iter.Next(ctx, 3*telem.Second)).To(BeTrue())
						Expect(iter.Value().Count()).To(Equal(2))
						Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](15))
						Expect(iter.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](16, 17))

						// Iter window: [12*telem.SecondTS, 15*telem.SecondTS)
						Expect(iter.Prev(ctx, 3*telem.Second)).To(BeTrue())
						Expect(iter.Value().Count()).To(Equal(1))
						Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](12, 13, 14))

						// Iter window: [15*telem.SecondTS, 22*telem.SecondTS)
						Expect(iter.Next(ctx, 7*telem.Second)).To(BeTrue())
						Expect(iter.Value().Count()).To(Equal(3))
						Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](15))
						Expect(iter.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](16, 17, 18, 19))
						Expect(iter.Value().SeriesAt(2)).To(telem.MatchSeriesDataV[int64](20))

						Expect(iter.Close()).To(Succeed())
					})
					Specify("Sample", func() {
						// Test case added to fix the bug where immediately contiguous
						// domains get flipped in order by read.
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18))).To(Succeed())
						w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 17 * telem.SecondTS, Subject: control.Subject{Key: "test_writer"}}))
						Expect(w.Write(telem.NewSeriesV[telem.TimeStamp](10, 11, 12, 13, 14, 15, 16)))
						_, err := w.Commit(ctx)
						Expect(err).ToNot(HaveOccurred())
						_, err = w.Close()
						Expect(err).ToNot(HaveOccurred())

						w, _ = MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{Start: 17 * telem.SecondTS, Subject: control.Subject{Key: "test_writer"}}))
						Expect(w.Write(telem.NewSeriesV[int64](17, 18)))
						_, err = w.Commit(ctx)
						Expect(err).ToNot(HaveOccurred())
						_, err = w.Close()
						Expect(err).ToNot(HaveOccurred())

						i := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())

						f := i.Value()
						Expect(f.Count()).To(Equal(2))
						Expect(f.SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(17 * telem.SecondTS)))
						Expect(f.SeriesAt(1).TimeRange).To(Equal((17 * telem.SecondTS).Range(18*telem.SecondTS + 1)))
						Expect(i.Close()).To(Succeed())
					})
				})

				Describe("Exhaustion", func() {
					Specify("Single Time Range", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
						iter := MustSucceed(db.OpenIterator(unary.IterRange((5 * telem.SecondTS).SpanRange(10 * telem.Second))))
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
						Expect(iter.Next(ctx, 3*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(3)))
						Expect(iter.Next(ctx, 4*telem.Second)).To(BeTrue())
						Expect(iter.Len()).To(Equal(int64(2)))
						Expect(iter.Next(ctx, 1*telem.Second)).To(BeFalse())
						Expect(iter.Len()).To(Equal(int64(0)))
						Expect(iter.Close()).To(Succeed())
					})
					Specify("Multi Time Range", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
						Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24, 25))).To(Succeed())
						Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10, 11, 12))).To(Succeed())
						iter := MustSucceed(db.OpenIterator(unary.IterRange((5 * telem.SecondTS).SpanRange(30 * telem.Second))))
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
						Expect(iter.Next(ctx, 3*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(3)))
						Expect(iter.Next(ctx, 10*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((13 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.Next(ctx, 10*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((23 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(3)))
						Expect(iter.Close()).To(Succeed())

					})
					Describe("Auto Span", func() {
						Specify("Single Domain - Leftover chunk", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 2,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})
						Specify("Single Domain - Full number chunks in domain", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 2,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})
						Specify("Partial Domain", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (10 * telem.SecondTS).SpanRange(3 * telem.Second),
								AutoChunkSize: 2,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})
						Specify("Partial Domain 2 - Regression", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (12 * telem.SecondTS).SpanRange(3 * telem.Second),
								AutoChunkSize: 2,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.View()).To(Equal((12 * telem.SecondTS).SpanRange(0)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((12 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((14 * telem.SecondTS).SpanRange(1 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.View()).To(Equal((15 * telem.SecondTS).SpanRange(0)))

							Expect(iter.SeekLast(ctx)).To(BeTrue())
							Expect(iter.View()).To(Equal((15 * telem.SecondTS).SpanRange(0)))
							Expect(iter.Prev(ctx, 2*telem.Second)).To(BeTrue())
							Expect(iter.View()).To(Equal((13 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Prev(ctx, 3*telem.Second)).To(BeTrue())
							Expect(iter.View()).To(Equal((12 * telem.SecondTS).Range(13 * telem.SecondTS)))
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Value().SeriesAt(0).Data).To(Equal([]byte{3, 0, 0, 0, 0, 0, 0, 0}))
							Expect(iter.Prev(ctx, 5*telem.Second)).To(BeFalse())

							Expect(iter.Close()).To(Succeed())
						})
						// This spec was added due to a bug in the SeekFirst and SeekLast methods
						// that would cause the iterator view to immediately go out of bounds,
						// and then cause iter.Sample() to return duplicate data even after
						// calling iter.Next(ctx, unary.AutoSpan)
						//
						// In this case (before the fix), calling iter.SeekFirst(ctx) would
						// return an invalid view of (6 * telem.SecondTS).SpanRange(0), and then
						// advancing the iterator the first time would cause it to go to
						// (10 * telem.SecondTS).SpanRange(0), and then calling iter.Sample()
						// would still return 2 values, and then calling Next(ctx, unary.AutoSpan)
						// would advance the iterator to (10 * telem.SecondTS).SpanRange(2 * telem.Second),
						// returning the same 2 values again.
						//
						// This spec asserts that this behavior is fixed.
						Specify("Partial Domain 3 - Regression", func() {
							Expect(unary.Write(ctx, indexDB, 6*telem.SecondTS, telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 6*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9, 10))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (10 * telem.SecondTS).SpanRange(4 * telem.Second),
								AutoChunkSize: 2,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Valid()).To(BeFalse())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](5, 6))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((12 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](7, 8))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())

							Expect(iter.SeekLast(ctx)).To(BeTrue())
							Expect(iter.Prev(ctx, 3*telem.Second)).To(BeTrue())
							Expect(iter.View()).To(Equal((11 * telem.SecondTS).SpanRange(3 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Prev(ctx, 10*telem.Second)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(1 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(1)))

							Expect(iter.Close()).To(Succeed())
						})
						// The problem mentioned in the above spec also arises in the SeekGE and
						// SeekLE methods, for example, iter.SeekGE(ctx, 5*telem.SecondTS) would
						// return true, but result in an invalid view of  (5 * telem.SecondTS).SpanRange(0)
						Specify("Partial Domain 4 - Regression", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24, 25, 26))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](8, 9, 10, 11, 12, 13, 14))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (10 * telem.SecondTS).SpanRange(5 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekGE(ctx, 5*telem.SecondTS)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
							Expect(iter.Valid()).To(BeFalse())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((13 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Value().SeriesAt(0).Data).To(Equal([]byte{4, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0}))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())

							Expect(iter.SeekLE(ctx, 0*telem.SecondTS)).To(BeFalse())
							Expect(iter.SeekLE(ctx, 40*telem.SecondTS)).To(BeFalse())
							Expect(iter.View()).To(Equal((15 * telem.SecondTS).SpanRange(0)))
							Expect(iter.Prev(ctx, 4*telem.Second)).To(BeTrue())
							Expect(iter.View()).To(Equal((11 * telem.SecondTS).SpanRange(4 * telem.Second)))
							Expect(iter.Prev(ctx, 5*telem.Second)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(1 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Value().SeriesAt(0).Data).To(Equal([]byte{1, 0, 0, 0, 0, 0, 0, 0}))
							Expect(iter.Prev(ctx, 10*telem.Second)).To(BeFalse())

							Expect(iter.Close()).To(Succeed())
						})
						Specify("Multi domain - Uneven Crossing", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24, 25, 26))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](8, 9, 10, 11, 12, 13, 14))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})
						Specify("Multi TimeRange - Even Crossing", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10, 11))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})

						Specify("Prev Auto Span - Basic", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekLast(ctx)).To(BeTrue())
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](6, 7, 8))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](3, 4, 5))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](1, 2))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})

						Specify("Prev Auto Span - Domain Crossing", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10, 11))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekLast(ctx)).To(BeTrue())
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](9, 10, 11))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](6))
							Expect(iter.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](7, 8))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](3, 4, 5))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](1, 2))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})

						Specify("Prev Auto Span - Cut-off Domain", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10, 11))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (12 * telem.SecondTS).SpanRange(15 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekLast(ctx)).To(BeTrue())
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](9, 10, 11))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](6))
							Expect(iter.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](7, 8))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](3, 4, 5))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})

						Specify("Prev Auto Span - Partial Domain", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        (12 * telem.SecondTS).SpanRange(4 * telem.Second),
								AutoChunkSize: 3,
							}))
							Expect(iter.SeekLast(ctx)).To(BeTrue())
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(3)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](4, 5, 6))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](3))
							Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})

						Specify("Multi domain - Regression 1", func() {
							var i telem.TimeStamp
							for i = 1; i < 6; i++ {
								Expect(unary.Write(ctx, indexDB, telem.SecondTS*i, telem.NewSeriesSecondsTSV(i))).To(Succeed())
								Expect(unary.Write(ctx, db, telem.SecondTS*i, telem.NewSeriesV(int64(i)))).To(Succeed())
							}
							iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
								Bounds:        telem.TimeRangeMax,
								AutoChunkSize: 5,
							}))
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(5)))
							Expect(iter.Close()).To(Succeed())
						})
						Describe("Regression tests: discontinuity", func() {
							BeforeEach(func() {
								// 0  1  4  6 / 10  11  12 / 13  15  17
								// 0  1  4  6 / 10  11  12 / 13  15  17
								Expect(unary.Write(ctx, indexDB, 0, telem.NewSeriesSecondsTSV(0, 1, 4, 6))).To(Succeed())
								Expect(unary.Write(ctx, db, 0, telem.NewSeriesV[int64](0, 1, 4, 6))).To(Succeed())
								Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12))).To(Succeed())
								Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12))).To(Succeed())
								Expect(unary.Write(ctx, indexDB, 13*telem.SecondTS, telem.NewSeriesSecondsTSV(13, 15, 17))).To(Succeed())
								Expect(unary.Write(ctx, db, 13*telem.SecondTS, telem.NewSeriesV[int64](13, 15, 17))).To(Succeed())
							})
							Specify("Multiple Domain - Forward - View in discontinuity", func() {
								// This test is to address an error where if an iterator
								// first moves to a discontinuity in the index (no data),
								// then moves to a view that overlaps more than one domain,
								// it is unable to parse the first domain in the second view.
								// 0  1  4  6 / 10  11  12 / 13  15  17
								// 0  1  4  6 / 10  11  12 / 13  15  17
								By("Opening an iterator")
								iter := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))
								Expect(iter.SeekFirst(ctx)).To(BeTrue())
								Expect(iter.Next(ctx, 7*telem.Second)).To(BeTrue())
								f := iter.Value()
								Expect(f.Count()).To(Equal(1))
								Expect(f.SeriesAt(0)).To(telem.MatchSeriesDataV[int64](0, 1, 4, 6))
								Expect(f.SeriesAt(0).TimeRange).To(Equal((0 * telem.SecondTS).Range(6*telem.SecondTS + 1)))

								By("Placing the iterator in the discontinuity")
								// Iterator now has view [7*telem.SecondTS, 9*telem.SecondTS)
								Expect(iter.Next(ctx, 2*telem.Second)).To(BeFalse())

								By("Moving it out")
								// Iterator now has view [9*telem.SecondTS, 15*telem.SecondTS)
								Expect(iter.Next(ctx, 6*telem.Second)).To(BeTrue())
								f = iter.Value()
								Expect(f.Count()).To(Equal(2))
								Expect(f.SeriesAt(0)).To(telem.MatchSeriesDataV[int64](10, 11, 12))
								Expect(f.SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
								Expect(f.SeriesAt(1)).To(telem.MatchSeriesDataV[int64](13))
								Expect(f.SeriesAt(1).TimeRange).To(Equal((13 * telem.SecondTS).Range(15 * telem.SecondTS)))

								// Iterator now has view [15*telem.SecondTS, 20*telem.SecondTS)
								Expect(iter.Next(ctx, 5*telem.Second)).To(BeTrue())
								f = iter.Value()
								Expect(f.Count()).To(Equal(1))
								Expect(f.SeriesAt(0)).To(telem.MatchSeriesDataV[int64](15, 17))
								Expect(f.SeriesAt(0).TimeRange).To(Equal((15 * telem.SecondTS).Range(17*telem.SecondTS + 1)))

								Expect(iter.Next(ctx, 1*telem.Second)).To(BeFalse())
								Expect(iter.Close()).To(Succeed())
							})
							Specify("Multiple Domain - Forward - uneven crossing", func() {
								// This test addresses the bug where if an iterator reads
								// a domain but does not read all of it, the internal
								// iterator still moves on to the next domain.
								By("Opening an iterator")
								i := MustSucceed(db.OpenIterator(unary.IterRange((2 * telem.SecondTS).Range(15 * telem.SecondTS))))

								// 0  1  || 4  6 / 10  11  12 / 13  || 15  17
								// 0  1  || 4  6 / 10  11  12 / 13  || 15  17

								Expect(i.SeekFirst(ctx)).To(BeTrue())
								Expect(i.Valid()).To(BeFalse())

								// Current iterator view: [2*telem.SecondTS, 11*telem.SecondTS)
								Expect(i.Next(ctx, 9*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(2))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](4, 6))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((2 * telem.SecondTS).Range(6*telem.SecondTS + 1)))
								Expect(i.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](10))
								Expect(i.Value().SeriesAt(1).TimeRange).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))

								// Current iterator view: [11*telem.SecondTS, 14*telem.SecondTS)
								Expect(i.Next(ctx, 3*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(2))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](11, 12))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((11 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
								Expect(i.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](13))
								Expect(i.Value().SeriesAt(1).TimeRange).To(Equal((13 * telem.SecondTS).Range(14 * telem.SecondTS)))

								Expect(i.Next(ctx, 5*telem.Second)).To(BeFalse())
								Expect(i.Valid()).To(BeFalse())

								Expect(i.Close()).To(Succeed())
							})
							Specify("View is full domain", func() {
								// This test tests that if a view is an entire domain, the
								// iterator will not move on to the next domain unnecessarily.
								By("Opening an iterator")
								i := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))

								// 0  1  4  6 / 10  11  12 / 13  15  17
								// 0  1  4  6 / 10  11  12 / 13  15  17

								Expect(i.SeekFirst(ctx)).To(BeTrue())
								Expect(i.Valid()).To(BeFalse())

								// Current iterator view: [0*telem.SecondTS, 13*telem.SecondTS)
								Expect(i.Next(ctx, 13*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(2))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](0, 1, 4, 6))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((0 * telem.SecondTS).Range(6*telem.SecondTS + 1)))
								Expect(i.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](10, 11, 12))
								Expect(i.Value().SeriesAt(1).TimeRange).To(Equal((10 * telem.SecondTS).Range(12*telem.SecondTS + 1)))

								// Current iterator view: [13*telem.SecondTS, 14*telem.SecondTS+1)
								Expect(i.Next(ctx, 1*telem.Second+1))
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](13))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((13 * telem.SecondTS).Range(14*telem.SecondTS + 1)))

								Expect(i.Next(ctx, 4*telem.Second))
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](15, 17))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((14*telem.SecondTS + 1).Range(17*telem.SecondTS + 1)))

								Expect(i.Next(ctx, 1*telem.Second)).To(BeFalse())
								Expect(i.Close()).To(Succeed())
							})
							Specify("Multiple Domain - Backward - view in discontinuity", func() {
								// This test is to address an error where if an iterator
								// first moves to a discontinuity in the index (no data),
								// then moves to a view that overlaps more than one domain,
								// it is unable to parse the first domain in the second view.
								By("Opening an iterator")
								i := MustSucceed(db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax}))
								// 0  1  4  6 / 10  11  12 / 13  15  17
								// 0  1  4  6 / 10  11  12 / 13  15  17

								Expect(i.SeekLast(ctx)).To(BeTrue())
								// Open iterator view: [10*telem.SecondTS+1, 17*telem.SecondTS+1)
								Expect(i.Prev(ctx, 7*telem.Second)).To(BeTrue())
								Expect(i.Valid()).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(2))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](11, 12))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((10*telem.SecondTS + 1).Range(12*telem.SecondTS + 1)))
								Expect(i.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](13, 15, 17))
								Expect(i.Value().SeriesAt(1).TimeRange).To(Equal((13 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
								// Open iterator view: [9*telem.SecondTS+1, 10*telem.SecondTS+1)
								Expect(i.Prev(ctx, 1*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](10))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(10*telem.SecondTS + 1)))
								// Open iterator view: [7*telem.SecondTS, 9*telem.SecondTS + 1)
								Expect(i.Prev(ctx, 2*telem.Second+1)).To(BeFalse())
								// Open iterator view: [-1*telem.SecondTS, 7*telem.SecondTS)
								Expect(i.Prev(ctx, 8*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](0, 1, 4, 6))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((0 * telem.SecondTS).Range(6*telem.SecondTS + 1)))
								Expect(i.Prev(ctx, 1*telem.Nanosecond)).To(BeFalse())
								Expect(i.Close()).To(Succeed())
							})
							Specify("Multiple Domain - Backward - uneven crossing", func() {
								// This test addresses the bug where if an iterator reads
								// a domain but does not read all of it, the internal
								// iterator still moves on to the previous domain.
								By("Opening an iterator")
								i := MustSucceed(db.OpenIterator(unary.IteratorConfig{
									Bounds: (2 * telem.SecondTS).Range(15 * telem.SecondTS),
								}))

								// 0  1  || 4  6 / 10  11  12 / 13  || 15  17
								// 0  1  || 4  6 / 10  11  12 / 13  || 15  17

								Expect(i.SeekLast(ctx)).To(BeTrue())
								Expect(i.Valid()).To(BeFalse())

								// Current iterator view: [11*telem.SecondTS, 15*telem.SecondTS)
								Expect(i.Prev(ctx, 4*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(2))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](11, 12))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((11 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
								Expect(i.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](13))
								Expect(i.Value().SeriesAt(1).TimeRange).To(Equal((13 * telem.SecondTS).Range(15 * telem.SecondTS)))

								// Current iterator view: [4*telem.SecondTS, 11*telem.SecondTS)
								Expect(i.Prev(ctx, 7*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(2))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](4, 6))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((4 * telem.SecondTS).Range(6*telem.SecondTS + 1)))
								Expect(i.Value().SeriesAt(1)).To(telem.MatchSeriesDataV[int64](10))
								Expect(i.Value().SeriesAt(1).TimeRange).To(Equal((10 * telem.SecondTS).Range(11 * telem.SecondTS)))

								Expect(i.Prev(ctx, 1*telem.Second)).To(BeFalse())
								Expect(i.Valid()).To(BeFalse())

								Expect(i.Close()).To(Succeed())
							})
							Specify("View is full domain", func() {
								// This test tests that if a view is an entire domain, the
								// iterator will not move on to the next domain unnecessarily.
								By("Opening an iterator")
								i := MustSucceed(db.OpenIterator(unary.IterRange(telem.TimeRangeMax)))

								// 0  1  4  6 / 10  11  12 / 13  15  17
								// 0  1  4  6 / 10  11  12 / 13  15  17

								Expect(i.SeekLast(ctx)).To(BeTrue())
								Expect(i.Valid()).To(BeFalse())

								// Current iterator view: [12*telem.SecondTS + 1, 17*telem.SecondTS + 1)
								Expect(i.Prev(ctx, 5*telem.Second)).To(BeTrue())
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](13, 15, 17))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((13 * telem.SecondTS).Range(17*telem.SecondTS + 1)))

								// Current iterator view: [10*telem.SecondTS, 12*telem.SecondTS+1)
								Expect(i.Prev(ctx, 2*telem.Second+1))
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](10, 11, 12))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(12*telem.SecondTS + 1)))

								Expect(i.Prev(ctx, 10*telem.Second))
								Expect(i.Value().Count()).To(Equal(1))
								Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](0, 1, 4, 6))
								Expect(i.Value().SeriesAt(0).TimeRange).To(Equal((0 * telem.SecondTS).Range(6*telem.SecondTS + 1)))

								Expect(i.Prev(ctx, 1*telem.Second)).To(BeFalse())
								Expect(i.Close()).To(Succeed())
							})
						})
					})
				})

				Describe("Regressions", func() {
					// This spec was added due to a bug in the line plotting mechanics
					// due to misalignment between actually-related domains. Let's
					// say you create an index and write a domain to it
					//
					// idx [1, 2, 3, 4]
					//
					// this domain will have alignment (0d0p - 0d3p). Now, if you write
					// another domain to the index
					//
					// idx [1, 2, 3, 4] [5, 6, 7, 8]
					//
					// the new domain will have alignment (1d0p - 1d3p). Then, we
					// write to a data channel aligned at the second domain
					//
					// idx [1, 2, 3, 4] [5, 6, 7, 8]
					// data 			[8, 9, 10, 11]
					//
					// the data domain will have alignment (0d0p - 0d3p), but it actually
					// aligns with the index domain at (1d0p - 1d3p). This spec tests
					// a fix made to ensure that the data domain has the alignment (1d0p - 1d3p)
					It("Should correctly define the alignment of a series when a domain has already been written to the index channel", func() {
						Expect(unary.Write(ctx, indexDB, 6*telem.SecondTS, telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24, 25, 26))).To(Succeed())
						Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](8, 9, 10, 11, 12, 13, 14))).To(Succeed())
						iter := MustSucceed(db.OpenIterator(unary.IteratorConfig{
							Bounds:        (20 * telem.SecondTS).SpanRange(15 * telem.Second),
							AutoChunkSize: 3,
						}))
						defer func() {
							Expect(iter.Close()).To(Succeed())
						}()
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
						fr := iter.Value()
						Expect(fr.Len()).To(Equal(int64(3)))
						s := fr.SeriesAt(0)
						Expect(s.Alignment).To(Equal(telem.NewAlignment(1, 0)))
					})

					// This test case is added due to a behavior change in the iterator.
					// Originally, when SeekGE finds a domain that is greater than the
					// provided timestamp but does not contain it, it leaves the iterator's
					// view at the provided timestamp.
					// For example, if the domains are [0-5] [10-15] [25-30], and we
					// seekGE(20), the iterator's view is set at 20, meaning when we
					// call next(3), we would read from [20, 23), resulting in a false
					// next call. This is confusing for the caller since a next call
					// following a seek call should read data, in this case, from
					// [25, 28).
					//
					// A similar behavior adjustment is put in place for SeekLE as well.
					It("Should bound an iterator's view to a sought domain", func() {
						Expect(unary.Write(ctx, indexDB, 0*telem.SecondTS, telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4, 5))).To(Succeed())
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13))).To(Succeed())
						Expect(unary.Write(ctx, indexDB, 15*telem.SecondTS, telem.NewSeriesSecondsTSV(15, 16, 17))).To(Succeed())
						Expect(unary.Write(ctx, db, 0*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13))).To(Succeed())
						Expect(unary.Write(ctx, db, 15*telem.SecondTS, telem.NewSeriesV[int64](15, 16, 17))).To(Succeed())

						i := MustSucceed(db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax}))
						Expect(i.SeekLE(ctx, 9*telem.SecondTS)).To(BeTrue())
						Expect(i.Prev(ctx, 3*telem.Second)).To(BeTrue())
						Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](3, 4, 5))

						Expect(i.SeekGE(ctx, 7*telem.SecondTS)).To(BeTrue())
						Expect(i.Next(ctx, 2*telem.Second)).To(BeTrue())
						Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](10, 11))

						Expect(i.Close()).To(Succeed())
					})
					It("Should auto-span through a domain split between two indices", func() {
						var (
							iKey     = GenerateChannelKey()
							dbKey    = GenerateChannelKey()
							indexDB2 = MustSucceed(unary.Open(ctx, unary.Config{
								FS:        MustSucceed(fs.Sub("index")),
								MetaCodec: codec,
								Channel: channel.Channel{
									Key:      iKey,
									DataType: telem.TimeStampT,
									IsIndex:  true,
									Index:    iKey,
								},
								Instrumentation: PanicLogger(),
								FileSize:        40 * telem.Byte,
							}))
							dataDB2 = MustSucceed(unary.Open(ctx, unary.Config{
								FS:        MustSucceed(fs.Sub("data")),
								MetaCodec: codec,
								Channel: channel.Channel{
									Key:      dbKey,
									DataType: telem.Int64T,
									Index:    iKey,
								},
								Instrumentation: PanicLogger(),
								FileSize:        40 * telem.Byte,
							}))
						)
						dataDB2.SetIndex(indexDB2.Index())
						w, _ := MustSucceed2(indexDB2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test"}}))
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15)))
						MustSucceed(w.Commit(ctx))
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(16, 17, 18, 19, 20)))
						MustSucceed(w.Commit(ctx))
						MustSucceed(w.Close())
						Expect(unary.Write(ctx, dataDB2, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))).To(Succeed())
						i := MustSucceed(dataDB2.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax, AutoChunkSize: 8}))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
						Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](10, 11, 12, 13, 14, 15, 16, 17))
						Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
						Expect(i.Value().SeriesAt(0)).To(telem.MatchSeriesDataV[int64](18, 19, 20))
						Expect(i.Close()).To(Succeed())

						Expect(dataDB2.Close()).To(Succeed())
						Expect(indexDB2.Close()).To(Succeed())
					})

					Context("Cut-off domain on index channel", func() {
						var (
							iKey     cesium.ChannelKey
							indexDB2 *unary.DB
						)
						BeforeEach(func() {
							iKey = GenerateChannelKey()
							indexDB2 = MustSucceed(unary.Open(ctx, unary.Config{
								FS:        MustSucceed(fs.Sub("index3")),
								MetaCodec: codec,
								Channel: channel.Channel{
									Name:     "Ozturk",
									Key:      iKey,
									DataType: telem.TimeStampT,
									IsIndex:  true,
									Index:    iKey,
								},
								Instrumentation: PanicLogger(),
								FileSize:        40 * telem.Byte,
							}))
							w, _ := MustSucceed2(indexDB2.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "test"}}))
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15)))
							MustSucceed(w.Commit(ctx))
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(16, 17, 18, 19, 20, 21, 22, 23)))
							MustSucceed(w.Commit(ctx))
							MustSucceed(w.Close())
						})
						AfterEach(func() { Expect(indexDB2.Close()).To(Succeed()) })
						// The following regression test is used to assert that upon reading
						// the second auto-span in a domain that begins with an inexact
						// start (cut off), the iterator behaves properly. The broken
						// behavior was that it was unable to find the correct start/end
						// approximations due to the inexact start.
						It("Should auto-span with a cut-off domain", func() {

							i := MustSucceed(indexDB2.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax, AutoChunkSize: 7}))
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15).Data))
							Expect(i.Value().SeriesAt(1).Data).To(Equal(telem.NewSeriesSecondsTSV(16).Data))
							Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(17, 18, 19, 20, 21, 22, 23).Data))
							Expect(i.Close()).To(Succeed())
						})

						// The following regression test is used to assert that upon reading
						// the second auto-span in a domain that begins with an inexact
						// start (cut off), the iterator behaves properly. The broken
						// behavior was that it was unable to find the correct start/end
						// approximations due to the inexact start.
						It("Should call next properly with a cut-off domain", func() {
							i := MustSucceed(indexDB2.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax, AutoChunkSize: 7}))
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, 7*telem.Second)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15).Data))
							Expect(i.Value().SeriesAt(1).Data).To(Equal(telem.NewSeriesSecondsTSV(16).Data))
							Expect(i.Next(ctx, 3*telem.Second)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(17, 18, 19).Data))
							Expect(i.Next(ctx, 4*telem.Second)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(20, 21, 22, 23).Data))
							Expect(i.Close()).To(Succeed())

							Expect(indexDB2.Close()).To(Succeed())
						})

						It("Should call prev properly with a cut-off domain", func() {
							i := MustSucceed(indexDB2.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax, AutoChunkSize: 7}))
							Expect(i.SeekLast(ctx)).To(BeTrue())
							Expect(i.Prev(ctx, 9*telem.Second)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(15).Data))
							Expect(i.Value().SeriesAt(1).Data).To(Equal(telem.NewSeriesSecondsTSV(16, 17, 18, 19, 20, 21, 22, 23).Data))
							Expect(i.Prev(ctx, 2*telem.Second)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(13, 14).Data))
							Expect(i.Prev(ctx, 3*telem.Second)).To(BeTrue())
							Expect(i.Value().SeriesAt(0).Data).To(Equal(telem.NewSeriesSecondsTSV(10, 11, 12).Data))
							Expect(i.Close()).To(Succeed())

							Expect(indexDB2.Close()).To(Succeed())
						})
					})
				})
			})
			Describe("Close", func() {
				var (
					fs      fs.FS
					cleanUp func() error
					db      *unary.DB
					key     cesium.ChannelKey
				)
				BeforeEach(func() {
					fs, cleanUp = makeFS()
					key = GenerateChannelKey()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      key,
							Name:     "ludwig",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				It("Should not allow operations on a closed iterator", func() {
					var (
						i = MustSucceed(db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax}))
						e = channel.NewErrResourceClosed("unary.iterator")
					)
					Expect(i.Close()).To(Succeed())
					Expect(i.SeekFirst(ctx)).To(BeFalse())
					Expect(i.Error()).To(HaveOccurredAs(e))
					Expect(i.Error()).To(MatchError(ContainSubstring("[ludwig]<%d>", key)))
					Expect(i.Valid()).To(BeFalse())
					Expect(i.Close()).To(Succeed())

					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})

				It("Should not allow an iterator to operate on a closed db", func() {
					Expect(unary.Write(ctx, db, 0, telem.NewSeriesV[int64](3, 4, 5, 6))).To(Succeed())
					Expect(db.Close()).To(Succeed())
					i, err := db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
					Expect(i).To(BeNil())
					Expect(err).To(HaveOccurredAs(unary.ErrDBClosed))
				})
			})
		})
	}
	Describe("Regressions", func() {
		// Tests a scenario where enough data is written to an 8-byte
		// density index channel that it causes a file-rollover, splitting the domain
		// into two effectively contiguous domains. The 4-byte density data channel
		// does _not_ reach the rollover point.
		//
		// At this point, the index channel has two effectively contiguous domains while
		// the data channel has a single domain. When querying data within the second
		// effectively contiguous domain, the iterator would return the correct data,
		// but the alignments of the domains would not match.
		//
		// This test ensures that changes to correct the alignment calculations behave
		// properly.
		Describe(`Correctly aligning effectively contiguous domains across index
						and data channels with different densities`, func() {

			It("Should correctly align effectively contiguous domains with different densities", func() {
				var (
					indexKey cesium.ChannelKey = 1
					dataKey  cesium.ChannelKey = 2
				)
				fileSizeLimit := 8 * 4 * telem.Byte
				fs := fs.NewMem()
				indexFS := MustSucceed(fs.Sub("index"))
				unaryFS := MustSucceed(fs.Sub("data"))
				indexDB := MustSucceed(unary.Open(ctx, unary.Config{
					FS:        indexFS,
					MetaCodec: codec,
					Channel: channel.Channel{
						Name:     "Fred",
						Key:      indexKey,
						DataType: telem.TimeStampT,
						IsIndex:  true,
						Index:    indexKey,
					},
					Instrumentation: PanicLogger(),
					FileSize:        fileSizeLimit,
				}))
				db := MustSucceed(unary.Open(ctx, unary.Config{
					FS:        unaryFS,
					MetaCodec: codec,
					Channel: channel.Channel{
						Name:     "Armisen",
						Key:      dataKey,
						DataType: telem.Float32T,
						Index:    indexKey,
					},
					Instrumentation: PanicLogger(),
					FileSize:        fileSizeLimit,
				}))
				db.SetIndex(indexDB.Index())

				indexW, _ := MustSucceed2(indexDB.OpenWriter(
					ctx,
					unary.WriterConfig{
						Start:   1 * telem.SecondTS,
						Subject: control.Subject{Key: "test"},
					},
				))
				MustSucceed(indexW.Write(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
				MustSucceed(indexW.Commit(ctx))
				MustSucceed(indexW.Write(telem.NewSeriesSecondsTSV(6, 7, 8)))
				MustSucceed(indexW.Commit(ctx))
				MustSucceed(indexW.Close())
				Expect(indexFS.List(".")).To(HaveLen(5 /* meta, index, counter, and 2 data files*/))

				unaryW, _ := MustSucceed2(db.OpenWriter(
					ctx,
					unary.WriterConfig{
						Start:   1 * telem.SecondTS,
						Subject: control.Subject{Key: "test"},
					},
				))
				MustSucceed(unaryW.Write(telem.NewSeriesV[float32](1, 2, 3, 4, 5)))
				// Rollover 1
				MustSucceed(unaryW.Commit(ctx))
				MustSucceed(unaryW.Write(telem.NewSeriesV[float32](6, 7, 8)))
				MustSucceed(unaryW.Commit(ctx))
				MustSucceed(unaryW.Close())
				// Assert that we've rolled over the correct number of files
				Expect(unaryFS.List(".")).To(HaveLen(4 /* meta, index, counter, and 1 data file*/))

				tr := telem.TimeRange{Start: 7 * telem.SecondTS, End: 8 * telem.SecondTS}
				iterCfg := unary.IteratorConfig{Bounds: tr}
				i := MustSucceed(indexDB.OpenIterator(iterCfg))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
				firstSeries := i.Value().SeriesAt(0)
				Expect(firstSeries.Alignment.DomainIndex()).To(Equal(uint32(1)))
				Expect(firstSeries.Alignment.SampleIndex()).To(Equal(uint32(1)))
				Expect(i.Close()).To(Succeed())

				i = MustSucceed(db.OpenIterator(iterCfg))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
				firstSeries = i.Value().SeriesAt(0)
				Expect(firstSeries.Alignment.DomainIndex()).To(Equal(uint32(1)))
				Expect(firstSeries.Alignment.SampleIndex()).To(Equal(uint32(1)))
				Expect(i.Close()).To(Succeed())

			})

			It("Should correctly align across three different densities", func() {
				var (
					indexKey cesium.ChannelKey = 1
					data1Key cesium.ChannelKey = 2
					data2Key cesium.ChannelKey = 3
				)
				// 4 timestamps sample file size
				fileSizeLimit := 8 * 4 * telem.Byte
				fs := fs.NewMem()
				indexFS := MustSucceed(fs.Sub("index"))
				uFS1 := MustSucceed(fs.Sub("data1"))
				uFS2 := MustSucceed(fs.Sub("data2"))
				indexDB := MustSucceed(unary.Open(ctx, unary.Config{
					FS:        indexFS,
					MetaCodec: codec,
					Channel: channel.Channel{
						Name:     "GI",
						Key:      indexKey,
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
					FileSize: fileSizeLimit,
				}))
				db1 := MustSucceed(unary.Open(ctx, unary.Config{
					FS:        uFS1,
					MetaCodec: codec,
					Channel: channel.Channel{
						Name:     "Joe",
						Key:      data1Key,
						DataType: telem.Float32T,
						Index:    indexKey,
					},
					FileSize: fileSizeLimit,
				}))
				db1.SetIndex(indexDB.Index())
				db2 := MustSucceed(unary.Open(ctx, unary.Config{
					FS:        uFS2,
					MetaCodec: codec,
					Channel: channel.Channel{
						Name:     "Jon",
						Key:      data2Key,
						DataType: telem.Uint8T,
						Index:    indexKey,
					},
					FileSize: fileSizeLimit,
				}))
				db2.SetIndex(indexDB.Index())

				indexW, _ := MustSucceed2(indexDB.OpenWriter(
					ctx,
					unary.WriterConfig{
						Start:   1 * telem.SecondTS,
						Subject: control.Subject{Key: "test"},
					}),
				)
				MustSucceed(indexW.Write(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
				// Rollover 1
				MustSucceed(indexW.Commit(ctx))
				MustSucceed(indexW.Write(telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10)))
				MustSucceed(indexW.Commit(ctx))
				// Rollover 2
				MustSucceed(indexW.Write(telem.NewSeriesSecondsTSV(11)))
				MustSucceed(indexW.Commit(ctx))

				MustSucceed(indexW.Close())
				// Assert that we've rolled over the correct number of files
				Expect(indexFS.List(".")).To(HaveLen(6 /* meta, index, counter, and 3 data files*/))

				// Write to the first data channel
				unaryW, _ := MustSucceed2(db1.OpenWriter(
					ctx,
					unary.WriterConfig{
						Start:   1 * telem.SecondTS,
						Subject: control.Subject{Key: "test"},
					}),
				)

				MustSucceed(unaryW.Write(telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8, 9)))
				// Rollover 1
				MustSucceed(unaryW.Commit(ctx))
				// Rollover 2
				MustSucceed(unaryW.Write(telem.NewSeriesV[float32](10)))
				MustSucceed(unaryW.Commit(ctx))

				MustSucceed(unaryW.Write(telem.NewSeriesV[float32](11)))
				MustSucceed(unaryW.Commit(ctx))

				MustSucceed(unaryW.Close())
				// Assert that we've rolled over the correct number of files
				Expect(uFS1.List(".")).To(HaveLen(5 /* meta, index, counter, and 2 data files*/))

				// Write to the second data channel
				unaryW, _ = MustSucceed2(db2.OpenWriter(
					ctx,
					unary.WriterConfig{
						Start:   1 * telem.SecondTS,
						Subject: control.Subject{Key: "test"},
					}),
				)

				MustSucceed(unaryW.Write(telem.NewSeriesV[uint8](1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)))
				MustSucceed(unaryW.Commit(ctx))
				MustSucceed(unaryW.Close())
				// Assert that we've rolled over the correct number of files
				Expect(uFS2.List(".")).To(HaveLen(4 /* meta, index, counter, and 1 data file*/))

				tr := telem.TimeRange{Start: 11 * telem.SecondTS, End: 12 * telem.SecondTS}
				iterCfg := unary.IteratorConfig{Bounds: tr}
				i := MustSucceed(indexDB.OpenIterator(iterCfg))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
				firstSeries := i.Value().SeriesAt(0)
				Expect(firstSeries.Alignment.DomainIndex()).To(Equal(uint32(2)))
				Expect(firstSeries.Alignment.SampleIndex()).To(Equal(uint32(0)))
				Expect(firstSeries.Data).To(Equal(telem.NewSeriesSecondsTSV(11).Data))

				i = MustSucceed(db1.OpenIterator(iterCfg))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
				firstSeries = i.Value().SeriesAt(0)
				Expect(firstSeries.Alignment.DomainIndex()).To(Equal(uint32(2)))
				Expect(firstSeries.Alignment.SampleIndex()).To(Equal(uint32(0)))
				Expect(firstSeries.Data).To(Equal(telem.NewSeriesV[float32](11).Data))

				i = MustSucceed(db2.OpenIterator(iterCfg))
				Expect(i.SeekFirst(ctx)).To(BeTrue())
				Expect(i.Next(ctx, cesium.AutoSpan)).To(BeTrue())
				firstSeries = i.Value().SeriesAt(0)
				Expect(firstSeries.Alignment.DomainIndex()).To(Equal(uint32(2)))
				Expect(firstSeries.Alignment.SampleIndex()).To(Equal(uint32(0)))
				Expect(firstSeries.Data).To(Equal(telem.NewSeriesV[uint8](11).Data))
			})

		})
	})
})
