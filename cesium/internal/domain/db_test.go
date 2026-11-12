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
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			Describe("HasDataFor", func() {
				var (
					db *domain.DB
					fs fs.FS
				)
				BeforeEach(func() {
					fs = openFS()
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						Instrumentation: PanicLogger(),
					}))
				})

				It(
					"Should return true if the domain DB has data for particular time range",
					func(ctx SpecContext) {
						tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
						Expect(
							testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3, 4, 5, 6}),
						).To(Succeed())
						Expect(db.HasDataFor(ctx, tr)).To(BeTrue())
					},
				)

				It(
					"Should return false if the domain DB does not have data for particular time range",
					func(ctx SpecContext) {
						tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
						Expect(
							testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3, 4, 5, 6}),
						).To(Succeed())
						Expect(
							db.HasDataFor(
								ctx,
								(20 * telem.SecondTS).SpanRange(10*telem.Second),
							),
						).To(BeFalse())
					},
				)

				It("Should return false if the DB is empty", func(ctx SpecContext) {
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(db.HasDataFor(ctx, tr)).To(BeFalse())
				})
			})

			Describe("Close", func() {
				It(
					"Should return an error if there are open writers on the DB",
					func(ctx SpecContext) {
						fs := openFS()
						db := MustSucceed(domain.Open(domain.Config{
							FS:              fs,
							Instrumentation: PanicLogger(),
						}))
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{}))
						Expect(db.Close()).To(MatchError(resource.ErrOpen))
						Expect(w.Close()).To(Succeed())
						Expect(db.Close()).To(Succeed())
					},
				)
			})

			Describe("MarkClosed", func() {
				var db *domain.DB
				BeforeEach(func() {
					db = MustOpen(domain.Open(domain.Config{
						FS:              openFS(),
						Instrumentation: PanicLogger(),
					}))
				})

				It(
					"Should refuse new writers until UnmarkClosed",
					func(ctx SpecContext) {
						Expect(db.MarkClosed()).To(Succeed())
						Expect(
							db.OpenWriter(ctx, domain.WriterConfig{}),
						).Error().To(MatchError(domain.ErrDBClosed))
						db.UnmarkClosed()
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{}))
						Expect(w.Close()).To(Succeed())
					},
				)

				It(
					"Should leave the DB open if a writer is open",
					func(ctx SpecContext) {
						w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{}))
						Expect(db.MarkClosed()).To(MatchError(resource.ErrOpen))
						Expect(w.Close()).To(Succeed())
						w = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{}))
						Expect(w.Close()).To(Succeed())
					},
				)

				It("Should return ErrDBClosed if the DB is closed", func() {
					Expect(db.MarkClosed()).To(Succeed())
					Expect(db.MarkClosed()).To(MatchError(domain.ErrDBClosed))
					db.UnmarkClosed()
				})
			})

			Describe("Release", func() {
				It(
					"Should persist the data of a marked DB",
					func(ctx SpecContext) {
						fs := openFS()
						cfg := domain.Config{FS: fs, Instrumentation: PanicLogger()}
						db := MustSucceed(domain.Open(cfg))
						tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
						Expect(
							testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3}),
						).To(Succeed())
						Expect(db.MarkClosed()).To(Succeed())
						Expect(db.Release()).To(Succeed())
						db = MustOpen(domain.Open(cfg))
						Expect(db.HasDataFor(ctx, tr)).To(BeTrue())
					},
				)
			})

			Describe("Size", func() {
				var (
					db *domain.DB
					fs fs.FS
				)
				BeforeEach(func() {
					fs = openFS()
					db = MustOpen(domain.Open(domain.Config{
						FS:              fs,
						Instrumentation: PanicLogger(),
					}))
				})

				It("Should return zero for an empty database", func() {
					Expect(db.Size()).To(Equal(telem.Size(0)))
				})

				It(
					"Should return the correct size after writing data",
					func(ctx SpecContext) {
						data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
						tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
						Expect(testutil.WriteDomain(ctx, db, tr, data)).To(Succeed())
						Expect(db.Size()).To(Equal(telem.Size(len(data))))
					},
				)

				It(
					"Should accumulate size across multiple writes",
					func(ctx SpecContext) {
						data1 := []byte{1, 2, 3, 4, 5}
						tr1 := (10 * telem.SecondTS).SpanRange(5 * telem.Second)
						Expect(testutil.WriteDomain(ctx, db, tr1, data1)).To(Succeed())

						data2 := []byte{6, 7, 8, 9, 10, 11, 12}
						tr2 := (20 * telem.SecondTS).SpanRange(7 * telem.Second)
						Expect(testutil.WriteDomain(ctx, db, tr2, data2)).To(Succeed())

						expectedSize := telem.Size(len(data1) + len(data2))
						Expect(db.Size()).To(Equal(expectedSize))
					},
				)
			})
		})
	}

	Describe("Open", func() {
		It("Should reject a file size that does not fit in 32 bits", func() {
			Expect(domain.Open(domain.Config{
				FS:              fs.NewMem(),
				Instrumentation: PanicLogger(),
				FileSize:        5 * telem.Gigabyte,
			})).Error().To(MatchError(ContainSubstring("file_size")))
		})
	})

	Describe("Attempting Operations on a Closed DB", func() {
		Describe("HasDataFor", func() {
			It("Should return ErrDBClosed", func(ctx SpecContext) {
				db := MustSucceed(domain.Open(domain.Config{
					FS:              fs.NewMem(),
					Instrumentation: PanicLogger(),
				}))
				Expect(db.Close()).To(Succeed())
				Expect(
					db.HasDataFor(ctx, telem.TimeRange{}),
				).Error().
					To(MatchError(domain.ErrDBClosed))
			})
		})
	})
})
