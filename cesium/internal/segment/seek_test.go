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
				"Should find where each sample starts from the start of the stream",
				func(sizes []int, bufSize int) {
					stream, starts := write(sizes...)
					buf := make([]byte, bufSize)
					for pos := range uint32(len(starts)) {
						Expect(varCfg.SeekFrom(h, buf, 0, pos, int64(len(stream)))).
							To(Equal(starts[pos]), "position %d", pos)
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

			DescribeTable(
				"Should find the start of the sample k positions after each sample",
				func(sizes []int, bufSize int) {
					stream, starts := write(sizes...)
					buf := make([]byte, bufSize)
					n := len(starts)
					step := max(1, n/50)
					for i := 0; i < n; i += step {
						for _, k := range []int{0, 1, n/3 - i, n - 1 - i} {
							if k < 0 || i+k >= n {
								continue
							}
							Expect(varCfg.SeekFrom(
								h,
								buf,
								starts[i],
								uint32(k),
								int64(len(stream)),
							)).To(Equal(starts[i+k]), "sample %d, k %d", i, k)
						}
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

			It(
				"Should find a sample far after another by the trailers of a few blocks "+
					"and two windows",
				func() {
					stream, starts := write(repeat(segment.BlockSize/2-4, 200)...)
					r := &countingReader{ReaderAt: h}
					buf := make([]byte, segment.BlockSize)
					Expect(
						varCfg.SeekFrom(r, buf, starts[11], 130, int64(len(stream))),
					).
						To(Equal(starts[141]))
					Expect(r.reads.Load()).To(BeNumerically("<=", 12))
				},
			)

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
						varCfg.SeekFrom(
							h,
							make([]byte, 16),
							0,
							150,
							int64(len(stream)),
						),
					).
						Error().
						To(And(
							MatchError(place.ErrCorrupt),
							MatchError(ContainSubstring("starts past the length")),
						))
				},
			)

			// trailer is a change to field off (0 first_pos, 4 first_off) of the
			// trailer
			// of full block b.
			type trailer struct {
				b, off int64
				v      uint32
			}
			// allBut sets first_pos to v in each of n full blocks but block b.
			allBut := func(n, b int64, v uint32) []trailer {
				var t []trailer
				for i := range n {
					if i != b {
						t = append(t, trailer{i, 0, v})
					}
				}
				return t
			}

			// Each entry seeks 10 samples from sample 2, at the start of block 1. The
			// walk ends at sample 4, the start of block 2, whose trailer stays good.
			// The search
			// for sample 12 then reads only the changed trailers that lead it astray.
			DescribeTable(
				"Should refuse trailers that disagree with one another past the first "+
					"block that the walk reaches",
				func(changes []trailer, msg string) {
					stream, starts := write(repeat(segment.BlockSize/2-4, 200)...)
					for _, c := range changes {
						at := varCfg.FileOffset((c.b+1)*segment.BlockSize) - 16 + c.off
						Expect(
							h.WriteAt(binary.LittleEndian.AppendUint32(nil, c.v), at),
						).To(Equal(4))
					}
					Expect(
						varCfg.SeekFrom(
							h,
							make([]byte, 16),
							starts[2],
							10,
							int64(len(stream)),
						),
					).Error().To(SatisfyAll(
						MatchError(place.ErrCorrupt),
						MatchError(ContainSubstring(msg)),
					))
				},
				Entry("every block but block 2 counts more samples before it",
					allBut(100, 2, 1<<31), "segment block 0 counts samples before it"),
				Entry(
					"block 6 starts no sample, and blocks 0, 1, and 3 count too many",
					[]trailer{
						{0, 0, 12},
						{1, 0, 12},
						{3, 0, 12},
						{6, 4, segment.NoStart},
					},
					"segment block 0 starts no sample",
				),
				Entry(
					"blocks 5 and 6 start no sample",
					[]trailer{{5, 4, segment.NoStart}, {6, 4, segment.NoStart}},
					"segment block 5 starts no sample, but the block after it counts more",
				),
			)

			It("Should refuse a sample length that runs past the segment", func() {
				stream, starts := write(5, 5, 5)
				Expect(
					h.WriteAt(binary.LittleEndian.AppendUint32(nil, 1<<20), starts[1]),
				).
					To(Equal(4))
				Expect(varCfg.SeekFrom(h, make([]byte, 16), 0, 2, int64(len(stream)))).
					Error().To(And(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("sample 2 at stream offset")),
				))
			})
		})
	}
})
