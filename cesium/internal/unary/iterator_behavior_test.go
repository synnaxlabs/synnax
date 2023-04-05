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
	"github.com/apache/arrow/go/v10/arrow/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var alloc = memory.NewGoAllocator()

var _ = Describe("Iterator Behavior", func() {
	Describe("Channel Indexed", func() {
		var (
			db      *unary.DB
			indexDB *unary.DB
		)
		BeforeEach(func() {
			indexDB = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
				Channel: core.Channel{
					Key:      "index",
					DataType: telem.TimeStampT,
					IsIndex:  true,
					Index:    "index",
				},
			}))
			db = MustSucceed(unary.Open(unary.Config{
				FS: fs.NewMem(),
				Channel: core.Channel{
					Key:      "data",
					DataType: telem.Int64T,
					Index:    "index",
				},
			}))
			db.SetIndex(indexDB.Index())
		})
		AfterEach(func() {
			Expect(db.Close()).To(Succeed())
			Expect(indexDB.Close()).To(Succeed())
		})
		Describe("Forward Exhaustion", func() {
			Specify("Single Range", func() {
				Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
				Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewArrayV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
				iter := db.NewIterator(ctx, unary.IterRange((5 * telem.SecondTS).SpanRange(10*telem.Second)))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
				Expect(iter.Next(3 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
				Expect(iter.Len()).To(Equal(int64(3)))
				Expect(iter.Next(4 * telem.Second)).To(BeTrue())
				Expect(iter.Len()).To(Equal(int64(2)))
				Expect(iter.Next(1 * telem.Second)).To(BeFalse())
			})
			Specify("Multi Range", func() {
				Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
				Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewArrayV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
				Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24, 25))).To(Succeed())
				Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewArrayV[int64](7, 8, 9, 10, 11, 12))).To(Succeed())
				iter := db.NewIterator(ctx, unary.IterRange((5 * telem.SecondTS).SpanRange(30*telem.Second)))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(0)))
				Expect(iter.Next(3 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal((10 * telem.SecondTS).SpanRange(3 * telem.Second)))
				Expect(iter.Len()).To(Equal(int64(3)))
				Expect(iter.Next(10 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal((13 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Len()).To(Equal(int64(6)))
				Expect(iter.Next(10 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal((23 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Len()).To(Equal(int64(3)))
			})
			Describe("Auto Exhaustion", func() {
				Specify("Single Range", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
					Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewArrayV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
					iter := db.NewIterator(ctx, unary.IteratorConfig{
						Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
						AutoChunkSize: 2,
					})
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(2)))
					Expect(iter.Close()).To(Succeed())
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(2)))
					Expect(iter.Close()).To(Succeed())
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(2)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(1)))
					Expect(iter.Next(unary.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
				Specify("Partial Range", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
					Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewArrayV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
					iter := db.NewIterator(ctx, unary.IteratorConfig{
						Bounds:        (10 * telem.SecondTS).SpanRange(3 * telem.Second),
						AutoChunkSize: 2,
					})
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(2)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(1)))
					Expect(iter.Next(unary.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
				Specify("Multi Range - Uneven Crossing", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16))).To(Succeed())
					Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewArrayV[int64](1, 2, 3, 4, 5, 6, 7))).To(Succeed())
					Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26))).To(Succeed())
					Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewArrayV[int64](8, 9, 10, 11, 12, 13, 14))).To(Succeed())
					iter := db.NewIterator(ctx, unary.IteratorConfig{
						Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
						AutoChunkSize: 3,
					})
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(2)))
					Expect(iter.Next(unary.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
				Specify("Multi Range - Even Crossing", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15))).To(Succeed())
					Expect(unary.Write(ctx, db, 10*telem.SecondTS, telem.NewArrayV[int64](1, 2, 3, 4, 5, 6))).To(Succeed())
					Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24))).To(Succeed())
					Expect(unary.Write(ctx, db, 20*telem.SecondTS, telem.NewArrayV[int64](7, 8, 9, 10, 11))).To(Succeed())
					iter := db.NewIterator(ctx, unary.IteratorConfig{
						Bounds:        (5 * telem.SecondTS).SpanRange(30 * telem.Second),
						AutoChunkSize: 3,
					})
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(3)))
					Expect(iter.Next(unary.AutoSpan)).To(BeTrue())
					Expect(iter.Len()).To(Equal(int64(2)))
					Expect(iter.Next(unary.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})

			})
		})
	})
})
