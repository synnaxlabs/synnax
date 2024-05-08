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
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", func() {
	for fsName, fsMaker := range fileSystems {
		fs := fsMaker()
		Context("FS: "+fsName, func() {
			Describe("Channel Indexed", func() {
				var (
					db        *unary.DB
					indexDB   *unary.DB
					index     uint32 = 1
					data      uint32 = 2
					indexPath        = rootPath + "/iterator_test/index"
					dataPath         = rootPath + "/iterator_test/data"
				)
				BeforeEach(func() {
					indexDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(indexPath)),
						Channel: core.Channel{
							Key:      index,
							DataType: telem.TimeStampT,
							IsIndex:  true,
							Index:    index,
						},
					}))
					db = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(dataPath)),
						Channel: core.Channel{
							Key:      data,
							DataType: telem.Int64T,
							Index:    index,
						},
					}))
					db.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(indexDB.Close()).To(Succeed())
					Expect(fs.Remove(rootPath)).To(Succeed())
				})

				Describe("Happy Path", func() {
					Specify("Next", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())

						Expect(unary.Write(ctx, indexDB, 16*telem.SecondTS, telem.NewSecondsTSV(16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 16*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10))).To(Succeed())

						iter := db.OpenIterator(unary.IterRange(telem.TimeRangeMax))
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
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())

						Expect(unary.Write(ctx, indexDB, 16*telem.SecondTS, telem.NewSecondsTSV(16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, db, 16*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10))).To(Succeed())

						iter := db.OpenIterator(unary.IterRange(telem.TimeRangeMax))
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
				})

				Describe("Requests Exhaustion", func() {
					Specify("Single TimeRange", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
						iter := db.OpenIterator(unary.IterRange((5 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
						Expect(iter.Next(ctx, 3*telem.Second)).To(BeTrue())
						Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(3)))
						Expect(iter.Next(ctx, 4*telem.Second)).To(BeTrue())
						Expect(iter.Len()).To(Equal(int64(2)))
						Expect(iter.Next(ctx, 1*telem.Second)).To(BeFalse())
					})
					Specify("Multi TimeRange", func() {
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
						Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
						Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24, 25))).To(Succeed())
						Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10, 11, 12))).To(Succeed())
						iter := db.OpenIterator(unary.IterRange((5 * telem.SecondTS).SpanRange(30 * telem.Second)))
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

					})
					Describe("Auto Exhaustion", func() {
						Specify("Single Domain", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 2,
							})
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
						Specify("Partial Domain", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (10 * telem.SecondTS).SpanRange(3 * telem.Second),
								AutoChunkSize: 2,
							})
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.Len()).To(Equal(int64(1)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
							Expect(iter.Close()).To(Succeed())
						})
						Specify("Partial Domain 2 - Regression", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (12 * telem.SecondTS).SpanRange(3 * telem.Second),
								AutoChunkSize: 2,
							})
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
							Expect(iter.Value().Series[0].Data).To(Equal([]byte{3, 0, 0, 0, 0, 0, 0, 0}))
							Expect(iter.Prev(ctx, 5*telem.Second)).To(BeFalse())

							Expect(iter.Close()).To(Succeed())
						})
						// This spec was added due to a bug in the SeekFirst and SeekLast methods
						// that would cause the iterator view to immediately go out of bounds,
						// and then cause iter.Value() to return duplicate data even after
						// calling iter.Next(ctx, unary.AutoSpan)
						//
						// In this case (before the fix), calling iter.SeekFirst(ctx) would
						// return an invalid view of (6 * telem.SecondTS).SpanRange(0), and then
						// advancing the iterator the first time would cause it to go to
						// (10 * telem.SecondTS).SpanRange(0), and then calling iter.Value()
						// would still return 2 values, and then calling Next(ctx, unary.AutoSpan)
						// would advance the iterator to (10 * telem.SecondTS).SpanRange(2 * telem.Second),
						// returning the same 2 values again.
						//
						// This spec asserts that this behavior is fixed.
						Specify("Partial Domain 3 - Regression", func() {
							Expect(unary.Write(ctx, indexDB, 6*telem.SecondTS, telem.NewSecondsTSV(6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 6*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9, 10))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (10 * telem.SecondTS).SpanRange(4 * telem.Second),
								AutoChunkSize: 2,
							})
							Expect(iter.SeekFirst(ctx)).To(BeTrue())
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Value().Series[0].Data).To(Equal(telem.NewSeriesV[int64](5, 6).Data))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((12 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Len()).To(Equal(int64(2)))
							Expect(iter.Value().Series[0].Data).To(Equal(telem.NewSeriesV[int64](7, 8).Data))
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
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](8, 9, 10, 11, 12, 13, 14))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (10 * telem.SecondTS).SpanRange(5 * telem.Second),
								AutoChunkSize: 3,
							})
							Expect(iter.SeekGE(ctx, 5*telem.SecondTS)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
							Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
							Expect(iter.View()).To(Equal((13 * telem.SecondTS).SpanRange(2 * telem.Second)))
							Expect(iter.Value().Series[0].Data).To(Equal([]byte{4, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0}))
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
							Expect(iter.Value().Series[0].Data).To(Equal([]byte{1, 0, 0, 0, 0, 0, 0, 0}))
							Expect(iter.Prev(ctx, 10*telem.Second)).To(BeFalse())

							Expect(iter.Close()).To(Succeed())
						})
						Specify("Multi Domain - Uneven Crossing", func() {
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](8, 9, 10, 11, 12, 13, 14))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 3,
							})
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
							Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
							Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
							Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24))).To(Succeed())
							Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewSeriesV[int64](7, 8, 9, 10, 11))).To(Succeed())
							iter := db.OpenIterator(unary.IteratorConfig{
								Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
								AutoChunkSize: 3,
							})
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

					})
				})
			})
			Describe("Close", func() {
				var db = MustSucceed(unary.Open(unary.Config{
					FS: MustSucceed(fs.Sub(rootPath)),
					Channel: core.Channel{
						Key:      2,
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
				It("Should not allow operations on a closed iterator", func() {
					var (
						i = db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
						e = core.EntityClosed("unary.iterator")
					)
					Expect(i.Close()).To(Succeed())
					Expect(i.SeekFirst(ctx)).To(BeFalse())
					Expect(i.Error()).To(MatchError(e))
					Expect(i.Valid()).To(BeFalse())
					Expect(i.Close()).To(Succeed())
				})
				Expect(db.Close()).To(Succeed())
				Expect(fs.Remove(rootPath)).To(Succeed())
			})
		})
	}
})
