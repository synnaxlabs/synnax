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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// countingReader counts the reads of a segment file.
type countingReader struct {
	io.ReaderAt
	reads atomic.Int64
}

func (r *countingReader) ReadAt(p []byte, off int64) (int, error) {
	r.reads.Add(1)
	return r.ReaderAt.ReadAt(p, off)
}

var _ = Describe("Seek", func() {
	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var (
				h   *segment.Handle
				com segment.Committed
				rd  *segment.Reader
			)
			BeforeEach(func() { rd = segment.NewReader() })

			// write writes the samples of sizes to a new variable-density segment, and
			// returns the stream and the stream offset where each sample starts.
			write := func(sizes ...int) ([]byte, []int64) {
				GinkgoHelper()
				b := MustSucceed(segment.NewBudget(1, 4, segment.GoExecutor))
				h = MustSucceed(b.Create(makeFS(), "3.seg"))
				DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
				stream, starts := samples(sizes...)
				w := segment.NewWriter(h, varCfg)
				Expect(w.Write(stream)).To(Succeed())
				length, tail := w.Commit()
				com = segment.Committed{
					Config:  varCfg,
					Length:  int64(length),
					TailCRC: tail,
				}
				return stream, starts
			}

			seek := func(r io.ReaderAt, pos, samples uint32) (int64, error) {
				return rd.Seek(r, com, pos, samples)
			}

			DescribeTable(
				"Should find where each sample starts, and the end after the last",
				func(sizes []int) {
					stream, starts := write(sizes...)
					n := uint32(len(starts))
					for pos := range n + 1 {
						want := int64(len(stream))
						if pos < n {
							want = starts[pos]
						}
						Expect(seek(h, pos, n)).To(Equal(want), "position %d", pos)
					}
				},
				Entry("a sample over a whole block", []int{70_000, 10, 200_000, 9}),
				Entry("a run of blocks with no start", []int{3, 4, 400_000, 5, 6}),
				Entry("a length that crosses a block end",
					[]int{segment.BlockSize - 6, 100, 66_000, 2}),
				Entry("a sample that starts at a block start",
					[]int{segment.BlockSize - 4, 5, segment.BlockSize - 9, 1}),
				Entry("many small samples", repeat(13, 20_000)),
				Entry("samples in the tail block only", []int{1, 2, 3}),
				Entry("one block per sample", repeat(segment.BlockSize-4, 6)),
			)

			It("Should read the trailers of a few blocks and one window", func() {
				_, starts := write(repeat(segment.BlockSize/2-4, 200)...)
				r := &countingReader{ReaderAt: h}
				Expect(seek(r, 141, 200)).To(Equal(starts[141]))
				Expect(r.reads.Load()).To(BeNumerically("<=", 12))
			})

			It("Should refuse a position past the samples", func() {
				write(1, 2)
				Expect(seek(h, 3, 2)).Error().
					To(MatchError(ContainSubstring("position 3 is past the 2")))
			})

			// corrupt writes v at field off (0 first_pos, 4 first_off) of the trailer
			// of full block b, and sets its tcrc again when reseal is true.
			corrupt := func(b, off int64, v uint32, reseal bool) {
				GinkgoHelper()
				at := varCfg.FileOffset((b+1)*segment.BlockSize) - 16
				t := make([]byte, 16)
				Expect(h.ReadAt(t, at)).To(Equal(16))
				binary.LittleEndian.PutUint32(t[off:], v)
				if reseal {
					binary.LittleEndian.PutUint32(
						t[12:],
						placeAt(varCfg, at).Sum(t[:12]),
					)
				}
				Expect(h.WriteAt(t, at)).To(Equal(16))
			}

			It("Should refuse a trailer that fails its check on two reads", func() {
				write(repeat(1000, 200)...)
				corrupt(1, 0, 9999, false)
				Expect(seek(h, 150, 200)).Error().To(SatisfyAll(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring(
						"trailer of block 1 of segment 3 of key 6 fails two reads",
					)),
				))
			})

			It(
				"Should pass a trailer that fails its check on the first read only",
				func() {
					_, starts := write(repeat(1000, 200)...)
					at := varCfg.FileOffset(2*segment.BlockSize) - 16
					r := &flipReader{ReaderAt: h, at: at + 2, left: 1}
					Expect(rd.Seek(r, com, 150, 200)).To(Equal(starts[150]))
				},
			)

			DescribeTable(
				"Should refuse trailers that pass their checks but do not fit",
				func(sizes []int, block, field int64, v, pos uint32, msg string) {
					_, starts := write(sizes...)
					corrupt(block, field, v, true)
					Expect(seek(h, pos, uint32(len(starts)))).Error().To(SatisfyAll(
						MatchError(place.ErrCorrupt),
						MatchError(ContainSubstring(msg)),
					))
				},
				Entry("more samples than the segment", repeat(1000, 200), int64(1),
					int64(0), uint32(9999), uint32(150), "trailer of segment block 1"),
				Entry("block 0 counts samples before it",
					[]int{70_000, 70_000}, int64(0), int64(0), uint32(1), uint32(0),
					"segment block 0 counts samples before it"),
				Entry("block 0 starts no sample",
					[]int{140_000, 3}, int64(0), int64(0), uint32(1), uint32(1),
					"segment block 0 starts no sample"),
				Entry(
					"the block before a run of no starts starts none",
					[]int{200_000, 3},
					int64(0),
					int64(4),
					segment.NoStart,
					uint32(1),
					"segment block 0 starts no sample, but the block after it "+
						"counts more",
				),
			)

			It("Should refuse a walk through a block that fails its check", func() {
				_, starts := write(5, 5, 5)
				Expect(h.WriteAt([]byte{0xFF}, starts[1]+6)).To(Equal(1))
				Expect(seek(h, 2, 3)).Error().To(SatisfyAll(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("fails two reads")),
				))
			})

			It(
				"Should refuse a checked sample length that runs past the segment",
				func() {
					stream, starts := write(5, 5, 5)
					binary.LittleEndian.PutUint32(stream[starts[1]:], 1<<20)
					Expect(h.WriteAt(stream, 0)).To(Equal(len(stream)))
					com.TailCRC = placeAt(varCfg, 0).Sum(stream)
					Expect(seek(h, 2, 3)).Error().To(SatisfyAll(
						MatchError(place.ErrCorrupt),
						MatchError(ContainSubstring("sample 2 at stream offset")),
					))
				},
			)
		})
	}
})
