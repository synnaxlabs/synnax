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
	"io"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
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

// flipReader flips a bit at file offset at in the next left reads that cover it, or in
// every read when left is negative, and counts the reads that cover it.
type flipReader struct {
	io.ReaderAt
	mu    sync.Mutex
	at    int64
	left  int
	reads int
}

func (r *flipReader) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.ReaderAt.ReadAt(p, off)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.at >= off && r.at < off+int64(n) {
		r.reads++
		if r.left != 0 {
			p[r.at-off] ^= 1
			if r.left > 0 {
				r.left--
			}
		}
	}
	return n, err
}

// destReader records the address of the first byte of each read's destination.
type destReader struct {
	io.ReaderAt
	dests []*byte
}

func (r *destReader) ReadAt(p []byte, off int64) (int, error) {
	r.dests = append(r.dests, &p[0])
	return r.ReaderAt.ReadAt(p, off)
}

// failReader fails every read with err.
type failReader struct{ err error }

func (r failReader) ReadAt([]byte, int64) (int, error) { return 0, r.err }

var _ = Describe("Reader", func() {
	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var (
				budget *segment.Budget
				h      *segment.Handle
				rd     *segment.Reader
			)
			BeforeEach(func() {
				budget = MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor))
				rd = segment.NewReader()
			})

			// write writes stream to a new segment of c and returns its committed part.
			write := func(c segment.Config, stream []byte) segment.Committed {
				GinkgoHelper()
				h = MustSucceed(budget.Create(makeFS(), "3.seg"))
				DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
				w := segment.NewWriter(h, c)
				Expect(w.Write(stream)).To(Succeed())
				length, tail := w.Commit()
				return segment.Committed{
					Config:  c,
					Length:  int64(length),
					TailCRC: tail,
				}
			}

			DescribeTable(
				"Should read every range of a segment and check each block",
				func(c segment.Config, stream []byte, ranges [][2]int) {
					com := write(c, stream)
					for _, r := range ranges {
						p := make([]byte, r[1]-r[0])
						Expect(rd.Read(h, com, p, int64(r[0]))).To(Succeed())
						Expect(p).To(Equal(stream[r[0]:r[1]]), "range %v", r)
					}
				},
				Entry("fixed, inside the tail", cfg, pattern(1000),
					[][2]int{{0, 1000}, {10, 20}}),
				Entry(
					"fixed, over blocks",
					cfg,
					pattern(3*segment.BlockSize+77),
					[][2]int{
						{0, 3*segment.BlockSize + 77},
						{segment.BlockSize, 2 * segment.BlockSize},
						{100, 2*segment.BlockSize + 50},
						{3 * segment.BlockSize, 3*segment.BlockSize + 77},
						{segment.BlockSize - 1, segment.BlockSize + 1},
					},
				),
				Entry("fixed, ending at a block end", cfg, pattern(2*segment.BlockSize),
					[][2]int{
						{0, 2 * segment.BlockSize},
						{segment.BlockSize + 5, 2 * segment.BlockSize},
					}),
				Entry("variable, over blocks", varCfg, first(samples(70_000, 61_000)),
					[][2]int{
						{0, 131_008},
						{segment.BlockSize - 3, 131_008},
					}),
			)

			It("Should read each block once, into place when it fits", func() {
				com := write(cfg, pattern(4*segment.BlockSize+10))
				r := &countingReader{ReaderAt: h}
				p := make([]byte, 4*segment.BlockSize+10)
				Expect(rd.Read(r, com, p, 0)).To(Succeed())
				Expect(r.reads.Load()).To(Equal(int64(5)))
			})

			It("Should read a whole block into the caller's bytes, and a part through "+
				"scratch space", func() {
				com := write(cfg, pattern(3*segment.BlockSize+10))
				r := &destReader{ReaderAt: h}
				p := make([]byte, 2*segment.BlockSize+10)
				Expect(rd.Read(r, com, p, segment.BlockSize)).To(Succeed())
				Expect(r.dests).To(HaveLen(3))
				Expect(r.dests[0]).To(BeIdenticalTo(&p[0]))
				Expect(r.dests[1]).To(BeIdenticalTo(&p[segment.BlockSize]))
				Expect(r.dests[2]).ToNot(BeIdenticalTo(&p[2*segment.BlockSize]))
			})

			It(
				"Should pass a block that fails its check on the first read only",
				func() {
					stream := pattern(2*segment.BlockSize + 10)
					com := write(cfg, stream)
					r := &flipReader{
						ReaderAt: h,
						at:       segment.BlockSize + 4 + 100,
						left:     1,
					}
					p := make([]byte, len(stream))
					Expect(rd.Read(r, com, p, 0)).To(Succeed())
					Expect(p).To(Equal(stream))
					Expect(r.reads).To(Equal(2))
				},
			)

			DescribeTable(
				"Should refuse a block that fails its check on two reads",
				func(at func(com segment.Committed) int64, msg string) {
					com := write(cfg, pattern(2*segment.BlockSize+10))
					r := &flipReader{ReaderAt: h, at: at(com), left: -1}
					p := make([]byte, com.Length)
					Expect(rd.Read(r, com, p, 0)).To(SatisfyAll(
						MatchError(place.ErrCorrupt),
						MatchError(ContainSubstring(msg)),
					))
					Expect(r.reads).To(Equal(2))
				},
				Entry("a full block", func(segment.Committed) int64 { return 5 },
					"block at stream offset 0 of segment 3 of key 6 fails two reads"),
				Entry("its trailer", func(segment.Committed) int64 {
					return segment.BlockSize + 1
				}, "block at stream offset 0 of segment 3"),
				Entry("the unfinished last block", func(segment.Committed) int64 {
					return 2*segment.BlockSize + 8 + 3
				}, "block at stream offset 131072 of segment 3"),
			)

			It("Should refuse a tail whose checksum is not the committed one", func() {
				com := write(cfg, pattern(100))
				com.TailCRC++
				Expect(rd.Read(h, com, make([]byte, 100), 0)).
					To(MatchError(place.ErrCorrupt))
			})

			It("Should refuse a file shorter than the committed length", func() {
				com := write(cfg, pattern(100))
				com.Length = 200
				Expect(rd.Read(h, com, make([]byte, 150), 0)).To(SatisfyAll(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("fails two reads")),
				))
			})

			It("Should refuse a missing file as corrupt", func() {
				missing := budget.Open(makeFS(), "9.seg")
				DeferCleanup(func() { Expect(missing.Close()).To(Succeed()) })
				com := segment.Committed{Config: cfg, Length: 10}
				Expect(rd.Read(missing, com, make([]byte, 10), 0)).
					To(MatchError(place.ErrCorrupt))
			})

			It("Should return a read error that is not corruption as it is", func() {
				fault := errors.New("device fault")
				com := segment.Committed{Config: cfg, Length: 10}
				Expect(rd.Read(failReader{err: fault}, com, make([]byte, 10), 0)).
					To(SatisfyAll(MatchError(fault), Not(MatchError(place.ErrCorrupt))))
			})

			It("Should refuse a read past the committed length", func() {
				com := write(cfg, pattern(100))
				Expect(rd.Read(h, com, make([]byte, 10), 95)).
					To(MatchError(ContainSubstring("past the committed length 100")))
			})

			It("Should read with no allocation", func() {
				com := write(cfg, pattern(3*segment.BlockSize+5))
				p := make([]byte, 2*segment.BlockSize)
				var err error
				Expect(testing.AllocsPerRun(10, func() {
					err = errors.Join(err, rd.Read(h, com, p[:10], 100))
					err = errors.Join(err, rd.Read(h, com, p, segment.BlockSize))
				})).To(BeZero())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	}
})
