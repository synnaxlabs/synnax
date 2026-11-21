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
			var h *segment.Handle
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
				w.Commit()
				return stream, starts
			}

			DescribeTable(
				"Should find where each sample starts, and the end after the last",
				func(sizes []int, bufSize int) {
					stream, starts := write(sizes...)
					buf := make([]byte, bufSize)
					n := uint32(len(starts))
					for pos := range n + 1 {
						want := int64(len(stream))
						if pos < n {
							want = starts[pos]
						}
						Expect(varCfg.Seek(h, buf, pos, int64(len(stream)), n)).
							To(Equal(want), "position %d", pos)
					}
				},
				Entry("a sample over a whole block", []int{70_000, 10, 200_000, 9}, 16),
				Entry(
					"a run of blocks with no start",
					[]int{3, 4, 400_000, 5, 6},
					segment.BlockSize,
				),
				Entry("a length that crosses a block end",
					[]int{segment.BlockSize - 6, 100, 66_000, 2}, 16),
				Entry("a sample that starts at a block start",
					[]int{segment.BlockSize - 4, 5, segment.BlockSize - 9, 1}, 64),
				Entry("many small samples", repeat(13, 20_000), segment.BlockSize),
				Entry("samples in the tail block only", []int{1, 2, 3}, 16),
				Entry("one block per sample", repeat(segment.BlockSize-4, 6), 16),
			)

			It("Should read the trailers of a few blocks and one window", func() {
				stream, starts := write(repeat(segment.BlockSize/2-4, 200)...)
				r := &countingReader{ReaderAt: h}
				buf := make([]byte, segment.BlockSize)
				Expect(varCfg.Seek(r, buf, 141, int64(len(stream)), 200)).
					To(Equal(starts[141]))
				Expect(r.reads.Load()).To(BeNumerically("<=", 10))
			})

			It("Should refuse a position past the samples", func() {
				stream, _ := write(1, 2)
				Expect(varCfg.Seek(h, make([]byte, 16), 3, int64(len(stream)), 2)).
					Error().To(MatchError(ContainSubstring("position 3 is past the 2")))
			})

			It(
				"Should refuse a trailer that counts more samples than the segment",
				func() {
					stream, _ := write(repeat(1000, 200)...)
					at := varCfg.FileOffset(2 * segment.BlockSize)
					Expect(
						h.WriteAt(binary.LittleEndian.AppendUint32(nil, 9999), at-16),
					).
						To(Equal(4))
					Expect(
						varCfg.Seek(h, make([]byte, 16), 150, int64(len(stream)), 200),
					).
						Error().
						To(And(
							MatchError(place.ErrCorrupt),
							MatchError(ContainSubstring("trailer of segment block 1")),
						))
				},
			)

			// corrupt writes v at field off (0 first_pos, 4 first_off) of the trailer
			// of
			// full block b.
			corrupt := func(b, off int64, v uint32) {
				GinkgoHelper()
				at := varCfg.FileOffset((b+1)*segment.BlockSize) - 16 + off
				Expect(h.WriteAt(binary.LittleEndian.AppendUint32(nil, v), at)).
					To(Equal(4))
			}

			DescribeTable(
				"Should refuse trailers that disagree with one another",
				func(sizes []int, block, field int64, v, pos uint32, msg string) {
					stream, starts := write(sizes...)
					corrupt(block, field, v)
					Expect(varCfg.Seek(h, make([]byte, 16), pos, int64(len(stream)),
						uint32(len(starts)))).Error().To(SatisfyAll(
						MatchError(place.ErrCorrupt),
						MatchError(ContainSubstring(msg)),
					))
				},
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

			It("Should refuse a sample length that runs past the segment", func() {
				stream, starts := write(5, 5, 5)
				Expect(
					h.WriteAt(binary.LittleEndian.AppendUint32(nil, 1<<20), starts[1]),
				).
					To(Equal(4))
				Expect(varCfg.Seek(h, make([]byte, 16), 2, int64(len(stream)), 3)).
					Error().To(And(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("sample 2 at stream offset")),
				))
			})
		})
	}
})
