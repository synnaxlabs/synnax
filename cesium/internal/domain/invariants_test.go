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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// pointerRecord returns the index file record of a domain over tr.
func pointerRecord(tr telem.TimeRange) []byte {
	b := make([]byte, 26)
	binary.LittleEndian.PutUint64(b[0:], uint64(tr.Start))
	binary.LittleEndian.PutUint64(b[8:], uint64(tr.End))
	binary.LittleEndian.PutUint16(b[16:], 1)
	return b
}

var _ = Describe("Invariants", func() {
	It(
		"Should panic when a commit leaves two domains that overlap",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			Expect(xfs.WriteFile(fs, "index.domain", append(
				pointerRecord((10*telem.SecondTS).Range(20*telem.SecondTS)),
				pointerRecord((15*telem.SecondTS).Range(25*telem.SecondTS))...,
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
})
