// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *domain.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeEach(func() {
				fs, cleanUp = makeFS()
				db = MustSucceed(domain.Open(domain.Config{
					FS:              fs,
					Instrumentation: PanicLogger(),
				}))
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Valid", func() {
				It("Should return false on an iterator with zero span bounds", func() {
					i := db.OpenIterator(domain.IteratorConfig{
						Bounds: (10 * telem.SecondTS).SpanRange(0),
					})
					Expect(i.Valid()).To(BeFalse())
					Expect(i.Close()).To(Succeed())
				})
			})
			Describe("SeekFirst + SeekLast", Ordered, func() {
				BeforeEach(func() {
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).SpanRange(10*telem.Second), []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
				})
				DescribeTable("SeekFirst",
					func(
						ts telem.TimeStamp,
						expectedResult bool,
						expectedFirst telem.TimeRange,
					) {
						i := db.OpenIterator(domain.IterRange(ts.SpanRange(telem.TimeSpanMax)))
						Expect(i.SeekFirst(ctx)).To(Equal(expectedResult))
						if expectedResult {
							Expect(i.TimeRange()).To(Equal(expectedFirst))
						}
						Expect(i.Close()).To(Succeed())
					},
					Entry("Bound start equal to domain start",
						10*telem.SecondTS,
						true,
						(10*telem.SecondTS).SpanRange(10*telem.Second),
					),
					Entry("Bound end equal to domain start",
						20*telem.SecondTS,
						true,
						(30*telem.SecondTS).SpanRange(10*telem.Second),
					),
					Entry(`
				Bound start strictly greater than domain start and strictly less than
				domain end
			`,
						15*telem.SecondTS,
						true,
						(10*telem.SecondTS).SpanRange(10*telem.Second),
					),
					Entry("Bound start strictly less than start of first defined domain",
						5*telem.SecondTS,
						true,
						(10*telem.SecondTS).SpanRange(10*telem.Second),
					),
					Entry("Bound start strictly greater than end of last defined domain",
						40*telem.SecondTS,
						false,
						telem.TimeRangeZero,
					),
				)
				DescribeTable("SeekLast",
					func(
						ts telem.TimeStamp,
						expectedResult bool,
						expectedLast telem.TimeRange,
					) {
						tr := telem.TimeRange{Start: 0, End: ts}
						i := db.OpenIterator(domain.IterRange(tr))
						Expect(i.SeekLast(ctx)).To(Equal(expectedResult))
						Expect(i.TimeRange()).To(Equal(expectedLast))
						Expect(i.Close()).To(Succeed())
					},
					Entry("Bound end equal to domain end",
						40*telem.SecondTS,
						true,
						(30*telem.SecondTS).SpanRange(10*telem.Second),
					),
					Entry("Bound end equal to domain start",
						30*telem.SecondTS,
						true,
						(10*telem.SecondTS).SpanRange(10*telem.Second),
					),
					Entry(`
					Bound end strictly greater than domain start and strictly less than
					domain end
			`,
						35*telem.SecondTS,
						true,
						(30*telem.SecondTS).SpanRange(10*telem.Second),
					),
				)
			})

			Describe("Exhaustion", func() {
				BeforeEach(func() {
					Expect(domain.Write(ctx, db, (50 * telem.SecondTS).SpanRange(10*telem.Second), []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(domain.Write(ctx, db, (60 * telem.SecondTS).SpanRange(10*telem.Second), []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(domain.Write(ctx, db, (30 * telem.SecondTS).SpanRange(10*telem.Second), []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
				})
				Context("Requests", func() {
					It("Should return false when the iterator is exhausted", func() {
						iter := db.OpenIterator(domain.IteratorConfig{
							Bounds: (15 * telem.SecondTS).SpanRange(45 * telem.Second),
						})
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.Position()).To(Equal(uint32(0)))
						Expect(iter.Next()).To(BeTrue())
						Expect(iter.TimeRange()).To(Equal((30 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Next()).To(BeTrue())
						Expect(iter.Position()).To(Equal(uint32(2)))
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.TimeRange()).To(Equal((50 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Next()).To(BeFalse())
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.Position()).To(Equal(uint32(2)))
						Expect(iter.Close()).To(Succeed())
					})
				})
				Context("Responses", func() {
					It("Should return false when the iterator is exhausted", func() {
						iter := db.OpenIterator(domain.IteratorConfig{
							Bounds: (15 * telem.SecondTS).SpanRange(45 * telem.Second),
						})
						Expect(iter.SeekLast(ctx)).To(BeTrue())
						Expect(iter.TimeRange()).To(Equal((50 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Prev()).To(BeTrue())
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.TimeRange()).To(Equal((30 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Prev()).To(BeTrue())
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).SpanRange(10 * telem.Second)))
						Expect(iter.Prev()).To(BeFalse())
						Expect(iter.Len()).To(Equal(int64(6)))
						Expect(iter.Position()).To(Equal(uint32(0)))
						Expect(iter.Close()).To(Succeed())
					})
				})
			})

			Describe("Close", func() {
				It("Should not allow operations on a closed iterator", func() {
					var (
						i = db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
						e = core.EntityClosed("domain.iterator")
					)
					Expect(i.Close()).To(Succeed())
					Expect(i.SeekFirst(ctx)).To(BeFalse())
					Expect(i.Valid()).To(BeFalse())
					_, err := i.OpenReader(ctx)
					Expect(err).To(HaveOccurredAs(e))
					Expect(i.Close()).To(Succeed())
				})

				It("Should give an iterator that cannot be used when the db is closed", func() {
					Expect(domain.Write(ctx, db, (0 * telem.SecondTS).Range(10*telem.SecondTS), []byte{1, 2, 3, 4})).To(Succeed())
					Expect(db.Close()).To(Succeed())
					r := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
					Expect(r.SeekFirst(ctx)).To(BeFalse())
					Expect(r.Close()).To(Succeed())
				})
			})
		})
	}
})
