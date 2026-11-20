// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"encoding/binary"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var cfg = segment.Config{DB: 7, ID: 3, DataType: 4, Clock: 0x00200001}

// pattern returns n bytes that differ from block to block.
func pattern(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*7 + i/segment.BlockSize)
	}
	return b
}

// placeAt returns the place of the block at file offset off of the segment of cfg.
func placeAt(off int64) place.Place {
	return place.Segment(
		cfg.DB,
		cfg.ID,
		uint64(off),
		segment.Version,
		cfg.DataType,
		cfg.Clock,
	)
}

var _ = Describe("Segment", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })

	fileBytes := func() []byte {
		GinkgoHelper()
		f := MustSucceed(fs.Open("3.seg", os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		return MustSucceed(io.ReadAll(f))
	}

	DescribeTable(
		"Should place stream offsets past the trailers of the blocks before them",
		func(x, want int64) { Expect(segment.FileOffset(x)).To(Equal(want)) },
		Entry("start", int64(0), int64(0)),
		Entry("inside the first block", int64(65_535), int64(65_535)),
		Entry("start of the second block", int64(65_536), int64(65_540)),
		Entry("third block", int64(131_073), int64(131_081)),
	)

	It(
		"Should follow each full block with the checksum of the block at its place",
		func() {
			w := MustSucceed(segment.Create(fs, "3.seg", cfg))
			data := pattern(2*segment.BlockSize + 10)
			Expect(w.Write(data)).To(Succeed())
			Expect(w.Close()).To(Succeed())
			b := fileBytes()
			Expect(b).To(HaveLen(len(data) + 2*segment.FixedTrailer))
			first := b[:segment.BlockSize]
			Expect(binary.LittleEndian.Uint32(b[segment.BlockSize:])).
				To(Equal(placeAt(0).Sum(first)))
			second := b[65_540 : 65_540+segment.BlockSize]
			Expect(binary.LittleEndian.Uint32(b[65_540+segment.BlockSize:])).
				To(Equal(placeAt(65_540).Sum(second)))
		},
	)

	It(
		"Should return the running checksum of the unfinished block at a commit",
		func() {
			w := MustSucceed(segment.Create(fs, "3.seg", cfg))
			data := pattern(segment.BlockSize + 100)
			Expect(w.Write(data[:30])).To(Succeed())
			Expect(w.Write(data[30:])).To(Succeed())
			length, tail := w.Commit()
			Expect(length).To(Equal(uint32(len(data))))
			Expect(tail).To(Equal(placeAt(65_540).Sum(data[segment.BlockSize:])))
			Expect(w.Close()).To(Succeed())
		},
	)

	It("Should give an empty block the checksum of its place alone", func() {
		w := MustSucceed(segment.Create(fs, "3.seg", cfg))
		Expect(w.Write(pattern(segment.BlockSize))).To(Succeed())
		_, tail := w.Commit()
		Expect(tail).To(Equal(placeAt(65_540).Sum(nil)))
		Expect(w.Close()).To(Succeed())
	})

	It("Should put the length and checksum back to the last commit", func() {
		w := MustSucceed(segment.Create(fs, "3.seg", cfg))
		data := pattern(segment.BlockSize + 50)
		Expect(w.Write(data[:20])).To(Succeed())
		length, tail := w.Commit()
		Expect(w.Write(pattern(segment.BlockSize))).To(Succeed())
		w.PutBack()
		Expect(w.Len()).To(Equal(int64(length)))
		Expect(w.Write(data[20:])).To(Succeed())
		again, againTail := w.Commit()
		Expect(again).To(Equal(uint32(len(data))))
		Expect(againTail).ToNot(Equal(tail))
		Expect(w.Close()).To(Succeed())
		b := fileBytes()
		Expect(binary.LittleEndian.Uint32(b[segment.BlockSize:])).
			To(Equal(placeAt(0).Sum(data[:segment.BlockSize])))
	})

	It("Should read a stream range across blocks with no trailer bytes", func() {
		w := MustSucceed(segment.Create(fs, "3.seg", cfg))
		data := pattern(3*segment.BlockSize + 5)
		Expect(w.Write(data)).To(Succeed())
		Expect(w.Sync()).To(Succeed())
		Expect(w.Close()).To(Succeed())
		f := MustSucceed(fs.Open("3.seg", os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		got := make([]byte, 2*segment.BlockSize)
		Expect(segment.ReadAt(f, got, 100)).To(Succeed())
		Expect(got).To(Equal(data[100 : 100+2*segment.BlockSize]))
	})

	It("Should refuse to create a segment file that exists", func() {
		w := MustSucceed(segment.Create(fs, "3.seg", cfg))
		Expect(w.Close()).To(Succeed())
		Expect(segment.Create(fs, "3.seg", cfg)).Error().
			To(MatchError(ContainSubstring("exists")))
	})
})
