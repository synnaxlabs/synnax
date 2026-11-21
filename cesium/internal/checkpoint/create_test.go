// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Create", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })
	// partFiles returns the names of the part files in fs.
	partFiles := func() []string {
		GinkgoHelper()
		var names []string
		for _, e := range MustSucceed(fs.List("")) {
			if strings.HasSuffix(e.Name(), ".part") {
				names = append(names, e.Name())
			}
		}
		return names
	}
	// reopen reads head 1 and merges the parts it names into its state.
	reopen := func() (checkpoint.Header, *state.Snapshot) {
		GinkgoHelper()
		h, s := MustSucceed2(checkpoint.Read(fs, 1))
		MustSucceed(checkpoint.LoadParts(fs, db, h)).Merge(s)
		return h, s
	}
	// segOf is the size in a head of a segment record with one pointer.
	const segOf = 32 + 36

	It("Should write every segment to head 1 when the sealed records do not pass the "+
		"part size", func() {
		s := withSealed(sealed(1, 10), sealed(2, 20))
		MustSucceed(checkpoint.Create(fs, db, s, 1<<20))
		h, got := reopen()
		Expect(h.Parts).To(BeEmpty())
		Expect(h.Limits.Parts).To(Equal(uint64(1)))
		Expect(got.Channels).To(Equal(s.Channels))
		Expect(partFiles()).To(BeEmpty())
	})

	It("Should write the sealed records to parts of about the part size, and the "+
		"rest to head 1", func() {
		s := withSealed(
			sealed(1, 10), sealed(2, 20), sealed(3, 30),
			sealed(4, 40), sealed(5, 50), sealed(6, 60),
		)
		MustSucceed(checkpoint.Create(fs, db, s, 2*segOf))
		h, head := MustSucceed2(checkpoint.Read(fs, 1))
		Expect(h.Parts).To(HaveLen(3))
		for i, p := range h.Parts {
			Expect(p.ID).To(Equal(uint64(i + 1)))
			Expect(p.Generation).To(Equal(uint64(1)))
		}
		Expect(h.Parts[0].First).To(Equal(telem.TimeStamp(10)))
		Expect(h.Parts[2].Last).To(Equal(telem.TimeStamp(60)))
		Expect(h.Limits.Parts).To(Equal(uint64(4)))
		Expect(head.Channels).To(HaveLen(1))
		Expect(head.Channels[0].Segments).To(HaveLen(1), "only the open segment")
		Expect(head.Channels[0].Segments[0].ID).To(Equal(uint64(100)))
		_, got := reopen()
		Expect(got.Channels).To(Equal(s.Channels))
	})

	It("Should keep the old file number of a version 0 segment in its part", func() {
		old := sealed(1, 10)
		old.Version, old.File = 0, 7
		s := withSealed(old, sealed(2, 20))
		MustSucceed(checkpoint.Create(fs, db, s, 1))
		_, got := reopen()
		Expect(got.Channels[0].Segments[0].File).To(Equal(uint16(7)))
		Expect(got.Channels).To(Equal(s.Channels))
	})

	It("Should remove the part files that a cut-short create left", func() {
		f := MustSucceed(fs.Open("0000000000000009.part", os.O_CREATE|os.O_RDWR))
		Expect(f.Close()).To(Succeed())
		MustSucceed(checkpoint.Create(fs, db, withSealed(sealed(1, 10)), 1<<20))
		Expect(partFiles()).To(BeEmpty())
	})
})
