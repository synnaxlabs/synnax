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
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	cfg    = segment.Config{DB: 7, ID: 3, DataType: 4, Clock: 0x00200001}
	varCfg = segment.Config{
		DB:       7,
		ID:       3,
		DataType: 9,
		Clock:    0x00200001,
		Variable: true,
	}
)

// pattern returns n bytes that differ from block to block.
func pattern(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*7 + i/segment.BlockSize)
	}
	return b
}

// placeAt returns the place of bytes at file offset off of the segment of c.
func placeAt(c segment.Config, off int64) place.Place {
	return place.Segment(c.DB, c.ID, uint64(off), segment.Version, c.DataType, c.Clock)
}

// samples returns the stream of variable-density samples with the body sizes, and the
// stream offset where each starts.
func samples(sizes ...int) (stream []byte, starts []int64) {
	for i, size := range sizes {
		starts = append(starts, int64(len(stream)))
		stream = binary.LittleEndian.AppendUint32(stream, uint32(size))
		for j := range size {
			stream = append(stream, byte(i+j*3))
		}
	}
	return stream, starts
}

// expectVariableTrailers expects the trailer of each full block in file bytes b of a
// stream whose samples start at starts.
func expectVariableTrailers(b, stream []byte, starts []int64) {
	GinkgoHelper()
	full := len(stream) / segment.BlockSize
	Expect(b).To(HaveLen(len(stream) + full*segment.VariableTrailer))
	for blk := range full {
		lo, hi := int64(blk*segment.BlockSize), int64((blk+1)*segment.BlockSize)
		var pos uint32
		off := segment.NoStart
		for _, s := range starts {
			if s < lo {
				pos++
			} else if s < hi && off == segment.NoStart {
				off = uint32(s - lo)
			}
		}
		fileOff := varCfg.FileOffset(lo)
		block := b[fileOff : fileOff+segment.BlockSize]
		Expect(block).To(Equal(stream[lo:hi]))
		t := b[fileOff+segment.BlockSize : fileOff+segment.BlockSize+16]
		Expect(binary.LittleEndian.Uint32(t)).To(Equal(pos), "first_pos of %d", blk)
		Expect(binary.LittleEndian.Uint32(t[4:])).To(Equal(off), "first_off of %d", blk)
		Expect(binary.LittleEndian.Uint32(t[8:])).
			To(Equal(placeAt(varCfg, fileOff).Sum(block)))
		Expect(binary.LittleEndian.Uint32(t[12:])).
			To(Equal(placeAt(varCfg, fileOff+segment.BlockSize).Sum(t[:12])))
	}
}

var _ = Describe("Segment", func() {
	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var fs xfs.FS
			BeforeEach(func() { fs = makeFS() })

			fileBytes := func() []byte {
				GinkgoHelper()
				f := MustSucceed(fs.Open("3.seg", os.O_RDONLY))
				defer func() { Expect(f.Close()).To(Succeed()) }()
				return MustSucceed(io.ReadAll(f))
			}

			Describe("Fixed density", func() {
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
							To(Equal(placeAt(cfg, 0).Sum(first)))
						second := b[65_540 : 65_540+segment.BlockSize]
						Expect(
							binary.LittleEndian.Uint32(b[65_540+segment.BlockSize:]),
						).
							To(Equal(placeAt(cfg, 65_540).Sum(second)))
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
						Expect(tail).
							To(Equal(placeAt(cfg, 65_540).Sum(data[segment.BlockSize:])))
						Expect(w.Close()).To(Succeed())
					},
				)

				It(
					"Should give an empty block the checksum of its place alone",
					func() {
						w := MustSucceed(segment.Create(fs, "3.seg", cfg))
						Expect(w.Write(pattern(segment.BlockSize))).To(Succeed())
						_, tail := w.Commit()
						Expect(tail).To(Equal(placeAt(cfg, 65_540).Sum(nil)))
						Expect(w.Close()).To(Succeed())
					},
				)

				It(
					"Should put the length and checksum back to the last commit",
					func() {
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
							To(Equal(placeAt(cfg, 0).Sum(data[:segment.BlockSize])))
					},
				)

				It(
					"Should read a stream range across blocks with no trailer bytes",
					func() {
						w := MustSucceed(segment.Create(fs, "3.seg", cfg))
						data := pattern(3*segment.BlockSize + 5)
						Expect(w.Write(data)).To(Succeed())
						Expect(w.Sync()).To(Succeed())
						Expect(w.Close()).To(Succeed())
						f := MustSucceed(fs.Open("3.seg", os.O_RDONLY))
						defer func() { Expect(f.Close()).To(Succeed()) }()
						got := make([]byte, 2*segment.BlockSize)
						Expect(cfg.ReadAt(f, got, 100)).To(Succeed())
						Expect(got).To(Equal(data[100 : 100+2*segment.BlockSize]))
					},
				)
			})

			Describe("Variable density", func() {
				DescribeTable(
					"Should give each trailer the position and offset of the first sample "+
						"that starts in its block",
					func(sizes []int, split bool) {
						stream, starts := samples(sizes...)
						w := MustSucceed(segment.Create(fs, "3.seg", varCfg))
						if split {
							for i, s := range starts {
								end := int64(len(stream))
								if i+1 < len(starts) {
									end = starts[i+1]
								}
								Expect(w.Write(stream[s:end])).To(Succeed())
							}
						} else {
							Expect(w.Write(stream)).To(Succeed())
						}
						length, tail := w.Commit()
						Expect(w.Close()).To(Succeed())
						Expect(length).To(Equal(uint32(len(stream))))
						full := int64(
							len(stream) / segment.BlockSize * segment.BlockSize,
						)
						Expect(tail).To(Equal(
							placeAt(varCfg, varCfg.FileOffset(full)).Sum(stream[full:]),
						))
						expectVariableTrailers(fileBytes(), stream, starts)
					},
					Entry(
						"a sample over a whole block",
						[]int{70_000, 10, 200_000, 9},
						false,
					),
					Entry(
						"the same samples in one write each",
						[]int{70_000, 10, 200_000, 9},
						true,
					),
					Entry(
						"a length that crosses a block end",
						[]int{segment.BlockSize - 6, 100, 66_000},
						true,
					),
					Entry(
						"a sample that starts at a block start",
						[]int{segment.BlockSize - 4, 5, segment.BlockSize - 9, 1},
						false,
					),
					Entry("many small samples", repeat(3, 50_000), false),
					Entry("empty samples", repeat(0, 40_000), true),
				)

				It(
					"Should refuse a write that ends inside a sample and change nothing",
					func() {
						stream, starts := samples(10, 70_000, 20)
						w := MustSucceed(segment.Create(fs, "3.seg", varCfg))
						Expect(w.Write(stream[:starts[1]])).To(Succeed())
						Expect(w.Write(stream[starts[1] : starts[1]+3])).
							To(MatchError(ContainSubstring("ends inside the length at 0")))
						Expect(w.Write(stream[starts[1] : starts[2]-1])).
							To(MatchError(ContainSubstring("ends inside a sample")))
						Expect(w.Len()).To(Equal(starts[1]))
						Expect(w.Write(stream[starts[1]:])).To(Succeed())
						Expect(w.Close()).To(Succeed())
						expectVariableTrailers(fileBytes(), stream, starts)
					},
				)

				It(
					"Should put the sample count and block state back to the last commit",
					func() {
						first, _ := samples(5, 70_000)
						w := MustSucceed(segment.Create(fs, "3.seg", varCfg))
						Expect(w.Write(first[:9])).To(Succeed())
						w.Commit()
						Expect(w.Write(first[9:])).To(Succeed())
						w.PutBack()
						stream, starts := samples(5, 100, 140_000, 7)
						Expect(w.Write(stream[9:])).To(Succeed())
						Expect(w.Close()).To(Succeed())
						expectVariableTrailers(fileBytes(), stream, starts)
					},
				)

				It(
					"Should read a stream range across blocks with no trailer bytes",
					func() {
						stream, _ := samples(200_000)
						w := MustSucceed(segment.Create(fs, "3.seg", varCfg))
						Expect(w.Write(stream)).To(Succeed())
						Expect(w.Close()).To(Succeed())
						f := MustSucceed(fs.Open("3.seg", os.O_RDONLY))
						defer func() { Expect(f.Close()).To(Succeed()) }()
						got := make([]byte, 2*segment.BlockSize)
						Expect(varCfg.ReadAt(f, got, 1000)).To(Succeed())
						Expect(got).To(Equal(stream[1000 : 1000+2*segment.BlockSize]))
					},
				)
			})

			It("Should refuse to create a segment file that exists", func() {
				w := MustSucceed(segment.Create(fs, "3.seg", cfg))
				Expect(w.Close()).To(Succeed())
				Expect(segment.Create(fs, "3.seg", cfg)).Error().
					To(MatchError(ContainSubstring("exists")))
			})
		})
	}

	DescribeTable(
		"Should place stream offsets past the trailers of the blocks before them",
		func(c segment.Config, x, want int64) { Expect(c.FileOffset(x)).To(Equal(want)) },
		Entry("start", cfg, int64(0), int64(0)),
		Entry("inside the first block", cfg, int64(65_535), int64(65_535)),
		Entry("start of the second block", cfg, int64(65_536), int64(65_540)),
		Entry("third block", cfg, int64(131_073), int64(131_081)),
		Entry("variable, inside the first block", varCfg, int64(65_535), int64(65_535)),
		Entry("variable, second block", varCfg, int64(65_536), int64(65_552)),
		Entry("variable, third block", varCfg, int64(131_073), int64(131_105)),
	)
})

func repeat(size, n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = size
	}
	return s
}
