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

var _ = Describe("DB", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			Describe("HasDataFor", func() {
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

				It("Should return true if the domain DB has data for particular time range", func() {
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(domain.Write(ctx, db, tr, []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(db.HasDataFor(ctx, tr)).To(BeTrue())
				})

				It("Should return false if the domain DB does not have data for particular time range", func() {
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(domain.Write(ctx, db, tr, []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(db.HasDataFor(ctx, (20 * telem.SecondTS).SpanRange(10*telem.Second))).To(BeFalse())
				})

				It("Should return false if the DB is empty", func() {
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(db.HasDataFor(ctx, tr)).To(BeFalse())
				})
			})

			Describe("Close", func() {
				It("Should return an error if there are open writers on the DB", func() {
					fs, cleanUp := makeFS()
					db := MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						Instrumentation: PanicLogger(),
					}))
					w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{}))
					Expect(db.Close()).To(MatchError(core.ErrOpenResource))
					Expect(w.Close()).To(Succeed())
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())

				})
			})

			Describe("Size", func() {
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

				It("Should return zero for an empty database", func() {
					Expect(db.Size()).To(Equal(telem.Size(0)))
				})

				It("Should return the correct size after writing data", func() {
					data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(domain.Write(ctx, db, tr, data)).To(Succeed())
					Expect(db.Size()).To(Equal(telem.Size(len(data))))
				})

				It("Should accumulate size across multiple writes", func() {
					data1 := []byte{1, 2, 3, 4, 5}
					tr1 := (10 * telem.SecondTS).SpanRange(5 * telem.Second)
					Expect(domain.Write(ctx, db, tr1, data1)).To(Succeed())

					data2 := []byte{6, 7, 8, 9, 10, 11, 12}
					tr2 := (20 * telem.SecondTS).SpanRange(7 * telem.Second)
					Expect(domain.Write(ctx, db, tr2, data2)).To(Succeed())

					expectedSize := telem.Size(len(data1) + len(data2))
					Expect(db.Size()).To(Equal(expectedSize))
				})
			})
		})
	}

	Describe("Attempting Operations on a Closed DB", func() {
		Describe("HasDataFor", func() {
			It("Should return ErrDBClosed", func() {
				db := MustSucceed(domain.Open(domain.Config{
					FS:              xfs.NewMem(),
					Instrumentation: PanicLogger(),
				}))
				Expect(db.Close()).To(Succeed())
				_, err := db.HasDataFor(ctx, telem.TimeRange{})
				Expect(err).To(HaveOccurredAs(domain.ErrDBClosed))
			})
		})
	})
})
