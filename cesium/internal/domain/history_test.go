// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OnHistoryChange", func() {
	var (
		db      *domain.DB
		changes []telem.TimeRange
	)
	BeforeEach(func() {
		changes = nil
		db = MustOpen(domain.Open(domain.Config{
			FS: fs.NewMem(),
			OnHistoryChange: func(_ context.Context, tr telem.TimeRange) {
				changes = append(changes, tr)
			},
		}))
	})
	write := func(ctx context.Context, start, end telem.TimeStamp) {
		GinkgoHelper()
		tr := (start * telem.SecondTS).Range(end * telem.SecondTS)
		Expect(testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3})).To(Succeed())
	}

	It("Should not report domains written at the end", func(ctx SpecContext) {
		write(ctx, 10, 20)
		write(ctx, 30, 40)
		Expect(changes).To(BeEmpty())
	})

	It("Should report a domain written before the last", func(ctx SpecContext) {
		write(ctx, 30, 40)
		write(ctx, 10, 20)
		Expect(changes).To(Equal([]telem.TimeRange{
			(10 * telem.SecondTS).Range(20 * telem.SecondTS),
		}))
	})

	It("Should report a commit to a domain before the last", func(ctx SpecContext) {
		cfg := domain.WriterConfig{Start: 10 * telem.SecondTS}
		w := MustOpen(db.OpenWriter(ctx, cfg))
		MustSucceed(w.Write([]byte{1, 2}))
		Expect(w.Commit(ctx, 15*telem.SecondTS, nil)).To(Succeed())
		write(ctx, 30, 40)
		MustSucceed(w.Write([]byte{3, 4}))
		Expect(w.Commit(ctx, 20*telem.SecondTS, nil)).To(Succeed())
		Expect(changes).To(Equal([]telem.TimeRange{
			(15 * telem.SecondTS).Range(20 * telem.SecondTS),
		}))
	})

	It("Should report a delete", func(ctx SpecContext) {
		write(ctx, 10, 20)
		tr := (12 * telem.SecondTS).Range(15 * telem.SecondTS)
		Expect(db.Delete(ctx, tr, fixedOffset(1), fixedOffset(2))).To(Succeed())
		Expect(changes).To(Equal([]telem.TimeRange{tr}))
	})

	It("Should not report a delete between two domains", func(ctx SpecContext) {
		write(ctx, 10, 20)
		write(ctx, 30, 40)
		tr := (22 * telem.SecondTS).Range(25 * telem.SecondTS)
		Expect(db.Delete(ctx, tr, fixedOffset(0), fixedOffset(0))).To(Succeed())
		Expect(changes).To(BeEmpty())
	})

	It(
		"Should not report a delete between domains of different sizes",
		func(ctx SpecContext) {
			write(ctx, 10, 20)
			tr := (30 * telem.SecondTS).Range(40 * telem.SecondTS)
			Expect(
				testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3, 4, 5}),
			).To(Succeed())
			del := (15 * telem.SecondTS).Range(30 * telem.SecondTS)
			Expect(db.Delete(ctx, del, fixedOffset(3), fixedOffset(0))).To(Succeed())
			Expect(changes).To(BeEmpty())
		},
	)

	It(
		"Should report a delete ending at a smaller domain",
		func(ctx SpecContext) {
			first := (10 * telem.SecondTS).Range(20 * telem.SecondTS)
			Expect(testutil.WriteDomain(
				ctx,
				db,
				first,
				[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			)).To(Succeed())
			second := (30 * telem.SecondTS).Range(40 * telem.SecondTS)
			Expect(
				testutil.WriteDomain(ctx, db, second, []byte{1, 2, 3, 4, 5}),
			).To(Succeed())
			del := (15 * telem.SecondTS).Range(30 * telem.SecondTS)
			Expect(db.Delete(ctx, del, fixedOffset(5), fixedOffset(0))).To(Succeed())
			Expect(changes).To(Equal([]telem.TimeRange{del}))
			i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
			Expect(i.SeekFirst(ctx)).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(first.Start.Range(del.Start)))
			r := MustSucceed(i.OpenReader(ctx))
			buf := make([]byte, 5)
			MustSucceed(r.ReadAt(buf, 0))
			Expect(buf).To(Equal([]byte{0, 1, 2, 3, 4}))
			Expect(r.Close()).To(Succeed())
			Expect(i.Next()).To(BeTrue())
			Expect(i.TimeRange()).To(Equal(second))
			Expect(i.Size()).To(Equal(telem.Size(5)))
			Expect(i.Close()).To(Succeed())
		},
	)
})
