// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", func() {
	var db *domain.DB
	BeforeEach(func() {
		db = MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Valid", func() {
		It("Should return false on an iterator with zero span bounds", func() {
			r := db.NewIterator(domain.IteratorConfig{
				Bounds: (10 * telem.SecondTS).SpanRange(0),
			})
			Expect(r.Valid()).To(BeFalse())
		})
	})
	Describe("SeekFirst + SeekLast", func() {
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
				r := db.NewIterator(domain.IterRange(ts.SpanRange(telem.TimeSpanMax)))
				Expect(r.SeekFirst(ctx)).To(Equal(expectedResult))
				if expectedResult {
					Expect(r.TimeRange()).To(Equal(expectedFirst))
				}
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
				r := db.NewIterator(domain.IterRange(tr))
				Expect(r.SeekLast(ctx)).To(Equal(expectedResult))
				Expect(r.TimeRange()).To(Equal(expectedLast))
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
				iter := db.NewIterator(domain.IteratorConfig{
					Bounds: (15 * telem.SecondTS).SpanRange(45 * telem.Second),
				})
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Next()).To(BeTrue())
				Expect(iter.TimeRange()).To(Equal((30 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Next()).To(BeTrue())
				Expect(iter.TimeRange()).To(Equal((50 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Next()).To(BeFalse())
			})
		})
		Context("Responses", func() {
			It("Should return false when the iterator is exhausted", func() {
				iter := db.NewIterator(domain.IteratorConfig{
					Bounds: (15 * telem.SecondTS).SpanRange(45 * telem.Second),
				})
				Expect(iter.SeekLast(ctx)).To(BeTrue())
				Expect(iter.TimeRange()).To(Equal((50 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Prev()).To(BeTrue())
				Expect(iter.TimeRange()).To(Equal((30 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Prev()).To(BeTrue())
				Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).SpanRange(10 * telem.Second)))
				Expect(iter.Prev()).To(BeFalse())
			})
		})
	})

	Describe("Close", func() {
		It("Should not allow operations on a closed iterator", func() {
			var (
				i = db.NewIterator(domain.IterRange(telem.TimeRangeMax))
				e = core.EntityClosed("domain.iterator")
			)
			Expect(i.Close()).To(Succeed())
			Expect(i.SeekFirst(ctx)).To(BeFalse())
			Expect(i.Valid()).To(BeFalse())
			_, err := i.NewReader(ctx)
			Expect(err).To(MatchError(e))
			Expect(i.Close()).To(Succeed())
		})
	})
})
