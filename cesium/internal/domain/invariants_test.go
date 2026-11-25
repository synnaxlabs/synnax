// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build invariants || race

package domain_test

import (
	"encoding/binary"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// pointerRecord returns the index file record of a domain over tr.
// pointerRecord encodes the index record of a domain in file 1 that holds size bytes
// from offset.
func pointerRecord(tr telem.TimeRange, offset, size uint32) []byte {
	b := make([]byte, 26)
	binary.LittleEndian.PutUint64(b[0:], uint64(tr.Start))
	binary.LittleEndian.PutUint64(b[8:], uint64(tr.End))
	binary.LittleEndian.PutUint16(b[16:], 1)
	binary.LittleEndian.PutUint32(b[18:], offset)
	binary.LittleEndian.PutUint32(b[22:], size)
	return b
}

var _ = Describe("Invariants", func() {
	It(
		"Should panic when a commit leaves two domains that overlap",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			Expect(xfs.WriteFile(fs, "index.domain", append(
				pointerRecord((10*telem.SecondTS).Range(20*telem.SecondTS), 0, 0),
				pointerRecord((15*telem.SecondTS).Range(25*telem.SecondTS), 0, 0)...,
			))).To(Succeed())
			// The panic leaves the index locked, so the spec does not close the DB.
			db := MustSucceed(domain.Open(domain.Config{
				FS:              fs,
				Instrumentation: PanicLogger(),
			}))
			// The writer stays open: its close would wait on the index that the panic
			// leaves locked.
			w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
				End: 31 * telem.SecondTS,
			}))
			Expect(w.SetStart(30 * telem.SecondTS)).To(Succeed())
			MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
			Expect(func() {
				Expect(w.Commit(ctx, 31*telem.SecondTS, nil)).To(Succeed())
			}).To(PanicWith(MatchError(ContainSubstring("are not sorted and apart"))))
		},
	)

	It("Should panic when a commit sits next to a domain that ends before it starts",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			Expect(xfs.WriteFile(
				fs,
				"index.domain",
				pointerRecord(telem.TimeRange{
					Start: 20 * telem.SecondTS,
					End:   10 * telem.SecondTS,
				}, 0, 0),
			)).To(Succeed())
			// The panic leaves the index locked, so the spec closes neither the DB nor
			// the writer.
			db := MustSucceed(domain.Open(domain.Config{
				FS:              fs,
				Instrumentation: PanicLogger(),
			}))
			w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
				End: 31 * telem.SecondTS,
			}))
			Expect(w.SetStart(30 * telem.SecondTS)).To(Succeed())
			MustSucceed(w.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
			Expect(func() {
				Expect(w.Commit(ctx, 31*telem.SecondTS, nil)).To(Succeed())
			}).To(PanicWith(MatchError(ContainSubstring("ends before it starts"))))
		},
	)

	// open opens a DB over four domains of 8 bytes each in file 1, the last two of
	// which
	// overlap.
	open := func() *domain.DB {
		GinkgoHelper()
		fs := xfs.NewMem()
		var index []byte
		for i, tr := range []telem.TimeRange{
			(10 * telem.SecondTS).Range(20 * telem.SecondTS),
			(30 * telem.SecondTS).Range(40 * telem.SecondTS),
			(50 * telem.SecondTS).Range(60 * telem.SecondTS),
			(55 * telem.SecondTS).Range(65 * telem.SecondTS),
		} {
			index = append(index, pointerRecord(tr, uint32(i*8), 8)...)
		}
		Expect(xfs.WriteFile(fs, "index.domain", index)).To(Succeed())
		Expect(xfs.WriteFile(fs, "1.domain", make([]byte, 32))).To(Succeed())
		// The counter names file 1, so a collection visits it.
		Expect(xfs.WriteFile(fs, "counter.domain", []byte{1, 0, 0, 0})).To(Succeed())
		// The panic leaves the index locked, so the spec does not close the DB.
		return MustSucceed(domain.Open(domain.Config{
			FS:              fs,
			FileSize:        9 * telem.Byte,
			GCThreshold:     math.SmallestNonzeroFloat32,
			Instrumentation: PanicLogger(),
		}))
	}

	It("Should panic when a delete splices a domain next to two that overlap",
		func(ctx SpecContext) {
			db := open()
			Expect(func() {
				Expect(db.Delete(
					ctx,
					(55 * telem.SecondTS).Range(57*telem.SecondTS),
					fixedOffset(0),
					fixedOffset(2),
				)).To(Succeed())
			}).To(PanicWith(MatchError(ContainSubstring("are not sorted and apart"))))
		},
	)

	It("Should panic when a collection keeps two domains that overlap", func(
		ctx SpecContext,
	) {
		db := open()
		// The delete splices the first domain, whose neighbors do not overlap.
		Expect(db.Delete(
			ctx,
			(10 * telem.SecondTS).Range(12*telem.SecondTS),
			fixedOffset(0),
			fixedOffset(2),
		)).To(Succeed())
		Expect(func() {
			Expect(db.GarbageCollect(ctx)).To(Succeed())
		}).To(PanicWith(MatchError(ContainSubstring("are not sorted and apart"))))
	})
})
