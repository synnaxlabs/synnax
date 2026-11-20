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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Index load", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			// The index write extends the file before it writes the new pointer. When
			// the write fails, as on a full disk, the file keeps a zero-filled tail.
			It("Should not load a pointer that a failed index write left", func(
				ctx SpecContext,
			) {
				inner := openFS()
				faulty := WrapFaultyFS(inner)
				cfg := domain.Config{
					FS:              faulty,
					FileSize:        telem.Megabyte,
					Instrumentation: PanicLogger(),
				}
				db := MustSucceed(domain.Open(cfg))
				first := (1 * telem.SecondTS).Range(1*telem.SecondTS + 1)
				second := (2 * telem.SecondTS).Range(2*telem.SecondTS + 1)
				Expect(testutil.WriteDomain(ctx, db, first, []byte{1})).To(Succeed())
				faulty.SetOptions(WithFailWriteAt("index.domain"))
				Expect(testutil.WriteDomain(ctx, db, second, []byte{2})).
					To(MatchError(ErrFault))
				faulty.SetOptions()
				Expect(db.Close()).To(Succeed())

				stored := MustSucceed(xfs.ReadFile(inner, "index.domain"))
				GinkgoWriter.Printf("index.domain after the failed write: %x\n", stored)

				db = MustSucceed(domain.Open(cfg))
				defer func() { Expect(db.Close()).To(Succeed()) }()
				early := (500 * telem.MillisecondTS).Range(500*telem.MillisecondTS + 1)
				Expect(testutil.WriteDomain(ctx, db, early, []byte{5})).To(Succeed())
				i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
				var ranges []telem.TimeRange
				for i.SeekFirst(ctx); i.Valid(); i.Next() {
					ranges = append(ranges, i.TimeRange())
				}
				Expect(i.Close()).To(Succeed())
				Expect(ranges).To(Equal([]telem.TimeRange{early, first}))
				Expect(testutil.ReadDomain(ctx, db, telem.TimeRangeMax)).
					To(Equal([]byte{5, 1}))
			})

			// A delete shrinks the file before it writes the pointers after the
			// deleted ones. When the write fails, the file keeps the old pointers up to
			// the new length.
			It("Should keep the domains after a delete whose index write failed", func(
				ctx SpecContext,
			) {
				inner := openFS()
				faulty := WrapFaultyFS(inner)
				cfg := domain.Config{
					FS:              faulty,
					FileSize:        telem.Megabyte,
					Instrumentation: PanicLogger(),
				}
				db := MustSucceed(domain.Open(cfg))
				for _, s := range []telem.TimeStamp{10, 20, 30, 40} {
					Expect(testutil.WriteDomain(
						ctx,
						db,
						(s * telem.SecondTS).Range((s+2)*telem.SecondTS),
						[]byte{byte(s), byte(s + 1)},
					)).To(Succeed())
				}
				faulty.SetOptions(WithFailWriteAt("index.domain"))
				Expect(db.Delete(
					ctx,
					(20 * telem.SecondTS).Range(32*telem.SecondTS),
					fixedOffset(0),
					fixedOffset(2),
				)).To(MatchError(ErrFault))
				faulty.SetOptions()
				Expect(db.Close()).To(Succeed())

				db = MustSucceed(domain.Open(cfg))
				defer func() { Expect(db.Close()).To(Succeed()) }()
				data := MustSucceed(testutil.ReadDomain(ctx, db, telem.TimeRangeMax))
				Expect(data).To(Or(
					Equal([]byte{10, 11, 20, 21, 30, 31, 40, 41}),
					Equal([]byte{10, 11, 40, 41}),
				))
			})
		})
	}
})
