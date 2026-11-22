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
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// entry is the size of a block in the cache: the stream bytes and room for a trailer.
const entry = segment.BlockSize + segment.VariableTrailer

// nestedReader runs during before its first read.
type nestedReader struct {
	io.ReaderAt
	during func()
}

func (r *nestedReader) ReadAt(p []byte, off int64) (int, error) {
	if f := r.during; f != nil {
		r.during = nil
		f()
	}
	return r.ReaderAt.ReadAt(p, off)
}

var _ = Describe("Cache", func() {
	var (
		budget *segment.Budget
		stream []byte
		h      *segment.Handle
		com    segment.Committed
		r      *countingReader
	)
	// write writes stream to a new segment of c, in a file of its own.
	write := func(c segment.Config, b []byte) (*segment.Handle, segment.Committed) {
		GinkgoHelper()
		h := MustSucceed(budget.Create(xfs.NewMem(), "3.seg"))
		DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
		w := segment.NewWriter(h, c)
		Expect(w.Write(b)).To(Succeed())
		length, tail := w.Commit()
		return h, segment.Committed{Config: c, Length: int64(length), TailCRC: tail}
	}
	BeforeEach(func() {
		budget = MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor))
		stream = pattern(5*segment.BlockSize + 10)
		h, com = write(cfg, stream)
		r = &countingReader{ReaderAt: h}
	})
	// probe reads the 8 bytes at off from f with a new reader of cache, and resets it.
	probe := func(cache *segment.Cache, f io.ReaderAt, off int64) error {
		GinkgoHelper()
		rd := segment.NewReader(cache)
		defer rd.Reset()
		p := make([]byte, 8)
		if err := rd.Read(f, com, p, off); err != nil {
			return err
		}
		Expect(p).To(Equal(stream[off : off+8]))
		return nil
	}
	// block returns a stream offset inside full block i.
	block := func(i int64) int64 { return i*segment.BlockSize + 100 }

	It("Should give a full block that one reader checked to another with no read",
		func() {
			cache := segment.NewCache(4 * entry)
			holder := segment.NewReader(cache)
			p := make([]byte, 8)
			Expect(holder.Read(r, com, p, 100)).To(Succeed())
			Expect(probe(cache, r, 5000)).To(Succeed())
			holder.Reset()
			Expect(probe(cache, r, 9000)).To(Succeed())
			Expect(r.reads.Load()).To(Equal(int64(1)))
		})

	It("Should not cache a block that fails its check", func() {
		cache := segment.NewCache(4 * entry)
		bad := &flipReader{ReaderAt: h, at: 50, left: -1}
		Expect(probe(cache, bad, 100)).To(MatchError(place.ErrCorrupt))
		Expect(probe(cache, r, 100)).To(Succeed())
		Expect(probe(cache, r, 100)).To(Succeed())
		Expect(r.reads.Load()).To(Equal(int64(1)))
	})

	It("Should read the unfinished last block from the file for each reader", func() {
		cache := segment.NewCache(4 * entry)
		Expect(probe(cache, r, 5*segment.BlockSize)).To(Succeed())
		Expect(probe(cache, r, 5*segment.BlockSize)).To(Succeed())
		Expect(r.reads.Load()).To(Equal(int64(2)))
	})

	It("Should leave whole-block reads to the file, and find a block that went bad "+
		"after it was cached at its next read from the file", func() {
		cache := segment.NewCache(entry)
		// The read of block 4 and the tail takes block 4 whole.
		scan := segment.NewReader(cache)
		whole := make([]byte, segment.BlockSize+10)
		Expect(scan.Read(r, com, whole, 4*segment.BlockSize)).To(Succeed())
		Expect(probe(cache, r, block(4))).To(Succeed())
		Expect(r.reads.Load()).To(Equal(int64(3)), "a whole-block read fills nothing")
		bad := &flipReader{ReaderAt: h, at: com.FileOffset(4*segment.BlockSize) + 50}
		bad.left = -1
		Expect(probe(cache, bad, block(4))).To(Succeed())
		Expect(bad.reads).To(BeZero())
		Expect(scan.Read(bad, com, whole, 4*segment.BlockSize)).
			To(MatchError(place.ErrCorrupt))
		Expect(probe(cache, r, block(1))).To(Succeed())
		Expect(probe(cache, bad, block(4))).To(MatchError(place.ErrCorrupt))
	})

	It("Should drop the least recently used block", func() {
		cache := segment.NewCache(2 * entry)
		for _, i := range []int64{0, 1, 0, 2} {
			Expect(probe(cache, h, block(i))).To(Succeed())
		}
		steps := []struct{ block, reads int64 }{{0, 0}, {1, 1}, {0, 1}, {2, 2}}
		for _, step := range steps {
			Expect(probe(cache, r, block(step.block))).To(Succeed())
			Expect(r.reads.Load()).To(Equal(step.reads), "block %d", step.block)
		}
	})

	It("Should keep a block that a reader holds past the budget, and drop it once "+
		"released", func() {
		cache := segment.NewCache(entry)
		holder := segment.NewReader(cache)
		Expect(holder.Read(h, com, make([]byte, 8), block(0))).To(Succeed())
		Expect(probe(cache, h, block(1))).To(Succeed())
		Expect(probe(cache, r, block(0))).To(Succeed())
		Expect(r.reads.Load()).To(BeZero())
		holder.Reset()
		Expect(probe(cache, h, block(2))).To(Succeed())
		Expect(probe(cache, r, block(0))).To(Succeed())
		Expect(probe(cache, r, block(1))).To(Succeed())
		Expect(r.reads.Load()).To(Equal(int64(2)))
	})

	It("Should release the cached block of a reader that moves to the last block",
		func() {
			cache := segment.NewCache(entry)
			rd := segment.NewReader(cache)
			p := make([]byte, 8)
			Expect(rd.Read(h, com, p, block(0))).To(Succeed())
			Expect(rd.Read(h, com, p, 5*segment.BlockSize)).To(Succeed())
			Expect(probe(cache, h, block(1))).To(Succeed())
			Expect(probe(cache, r, block(0))).To(Succeed())
			Expect(r.reads.Load()).To(Equal(int64(1)))
		})

	It("Should count the holds of readers right after a read that fails its check",
		func() {
			cache := segment.NewCache(entry)
			rd := segment.NewReader(cache)
			p := make([]byte, 8)
			Expect(rd.Read(h, com, p, block(0))).To(Succeed())
			bad := &flipReader{ReaderAt: h, at: com.FileOffset(block(1)), left: -1}
			Expect(rd.Read(bad, com, p, block(1))).To(MatchError(place.ErrCorrupt))
			rd.Reset()
			holder := segment.NewReader(cache)
			Expect(holder.Read(h, com, p, block(0))).To(Succeed())
			Expect(probe(cache, h, block(2))).To(Succeed())
			Expect(probe(cache, r, block(0))).To(Succeed())
			Expect(r.reads.Load()).To(BeZero(), "the holder keeps block 0")
		})

	It("Should hold the block that another reader added while it read the same block",
		func() {
			cache := segment.NewCache(entry)
			rd := segment.NewReader(cache)
			nested := &nestedReader{ReaderAt: h, during: func() {
				Expect(probe(cache, h, block(0))).To(Succeed())
			}}
			p := make([]byte, 8)
			Expect(rd.Read(nested, com, p, block(0))).To(Succeed())
			Expect(probe(cache, h, block(1))).To(Succeed())
			Expect(probe(cache, r, block(0))).To(Succeed())
			Expect(r.reads.Load()).To(BeZero(), "the reader holds block 0")
		})

	It("Should read past the cache for a read that goes on from the last read",
		func() {
			cache := segment.NewCache(4 * entry)
			rd := segment.NewReader(cache)
			p := make([]byte, 8)
			Expect(rd.Read(h, com, p, segment.BlockSize-8)).To(Succeed())
			Expect(rd.Read(h, com, p, segment.BlockSize)).To(Succeed())
			Expect(p).To(Equal(stream[segment.BlockSize : segment.BlockSize+8]))
			Expect(probe(cache, r, block(1))).To(Succeed())
			Expect(probe(cache, r, block(0))).To(Succeed())
			Expect(r.reads.Load()).To(Equal(int64(1)), "only block 0 is cached")
		})

	It("Should read past the cache for a read over two blocks", func() {
		cache := segment.NewCache(4 * entry)
		p := make([]byte, 16)
		Expect(segment.NewReader(cache).Read(h, com, p, segment.BlockSize-8)).
			To(Succeed())
		Expect(p).To(Equal(stream[segment.BlockSize-8 : segment.BlockSize+8]))
		Expect(probe(cache, r, block(0))).To(Succeed())
		Expect(probe(cache, r, block(1))).To(Succeed())
		Expect(r.reads.Load()).To(Equal(int64(2)))
	})

	It("Should take the first read after a reset as a probe", func() {
		cache := segment.NewCache(4 * entry)
		rd := segment.NewReader(cache)
		p := make([]byte, 8)
		Expect(rd.Read(h, com, p, segment.BlockSize-8)).To(Succeed())
		rd.Reset()
		Expect(rd.Read(h, com, p, segment.BlockSize)).To(Succeed())
		Expect(probe(cache, r, block(1))).To(Succeed())
		Expect(r.reads.Load()).To(BeZero())
	})

	It("Should not give the block of one segment to another", func() {
		cache := segment.NewCache(4 * entry)
		Expect(probe(cache, h, 100)).To(Succeed())
		other := cfg
		other.ID = 4
		otherStream := pattern(segment.BlockSize + 200)[100:]
		oh, ocom := write(other, otherStream)
		or := &countingReader{ReaderAt: oh}
		p := make([]byte, 8)
		Expect(segment.NewReader(cache).Read(or, ocom, p, 100)).To(Succeed())
		Expect(p).To(Equal(otherStream[100:108]))
		Expect(or.reads.Load()).To(Equal(int64(1)))
	})

	It("Should read with no allocation once warm, when the reads pass the budget",
		func() {
			cache := segment.NewCache(2 * entry)
			rd := segment.NewReader(cache)
			p := make([]byte, 8)
			var err error
			cycle := func() {
				for i := range int64(4) {
					err = errors.Join(err, rd.Read(h, com, p, block(i)))
				}
			}
			cycle()
			Expect(testing.AllocsPerRun(10, cycle)).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
		})

	It("Should give parallel readers the right bytes while releases run against "+
		"evictions", func() {
		cache := segment.NewCache(2 * entry)
		var (
			wg     sync.WaitGroup
			failed atomic.Int64
		)
		for g := range int64(8) {
			wg.Go(func() {
				rd := segment.NewReader(cache)
				defer rd.Reset()
				p := make([]byte, 8)
				for i := range int64(200) {
					off := block((g+i)%5) + i%64
					if rd.Read(h, com, p, off) != nil ||
						string(p) != string(stream[off:off+8]) {
						failed.Add(1)
					}
					if i%3 == 0 {
						rd.Reset()
					}
				}
			})
		}
		wg.Wait()
		Expect(failed.Load()).To(BeZero())
	})
})
