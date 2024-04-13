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
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", func() {
	Describe("Channel Indexed", func() {
		var (
			db      *unary.DB
			indexDB *unary.DB
			index   uint32 = 1
			data    uint32 = 2
		)
		BeforeEach(func() {
			indexDB = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
				Channel: core.Channel{
					Key:      index,
					DataType: telem.TimeStampT,
					IsIndex:  true,
					Index:    index,
				},
			}))
			db = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
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
				Specify("Single TimeRange", func() {
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
				Specify("Partial TimeRange", func() {
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
				//FSpecify("Partial Time Range 2 - Reg", func() {
				//	Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
				//	Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
				//	iter := db.OpenIterator(unary.IteratorConfig{
				//		Bounds:        (12 * telem.SecondTS).SpanRange(3 * telem.Second),
				//		AutoChunkSize: 2,
				//	})
				//	Expect(iter.SeekFirst(ctx)).To(BeTrue())
				//	//Expect(iter.View()).To(Equal((12 * telem.SecondTS).SpanRange(0)))
				//	Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
				//	Expect(iter.Value().Series[0]).To(Equal(telem.NewSeriesV[int64](3, 4)))
				//	Expect(iter.Len()).To(Equal(int64(2)))
				//	Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
				//	Expect(iter.Len()).To(Equal(int64(1)))
				//	Expect(iter.Next(ctx, unary.AutoSpan)).To(BeFalse())
				//	Expect(iter.Close()).To(Succeed())
				//})
				Specify("Multi TimeRange - Uneven Crossing", func() {
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
			FS: fs.NewMem(),
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
	})
})
