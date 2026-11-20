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
	"math/rand/v2"
	"os"
	"slices"
	"sync"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// shortFS fails the next write after it writes a prefix of fail bytes, when fail is
// zero or more.
type shortFS struct {
	xfs.FS
	mu   sync.Mutex
	fail int
}

func (s *shortFS) Open(name string, flag int) (xfs.File, error) {
	f, err := s.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &shortFile{File: f, s: s}, nil
}

type shortFile struct {
	xfs.File
	s *shortFS
}

func (f *shortFile) WriteAt(p []byte, off int64) (int, error) {
	f.s.mu.Lock()
	n := f.s.fail
	f.s.fail = -1
	f.s.mu.Unlock()
	if n >= 0 {
		n = min(n, len(p))
		m, _ := f.File.WriteAt(p[:n], off)
		return m, syscall.ENOSPC
	}
	return f.File.WriteAt(p, off)
}

// reference returns the file bytes of a stream with samples at starts, built from the
// format table.
func reference(c segment.Config, stream []byte, starts []int64) []byte {
	var out []byte
	full := len(stream) / segment.BlockSize
	for blk := range full {
		lo, hi := blk*segment.BlockSize, (blk+1)*segment.BlockSize
		fileOff := int64(len(out))
		block := stream[lo:hi]
		out = append(out, block...)
		crc := placeAt(c, fileOff).Sum(block)
		if !c.Variable {
			out = binary.LittleEndian.AppendUint32(out, crc)
			continue
		}
		var pos uint32
		off := segment.NoStart
		for _, s := range starts {
			if s < int64(lo) {
				pos++
			} else if s < int64(hi) && off == segment.NoStart {
				off = uint32(s - int64(lo))
			}
		}
		t := binary.LittleEndian.AppendUint32(nil, pos)
		t = binary.LittleEndian.AppendUint32(t, off)
		t = binary.LittleEndian.AppendUint32(t, crc)
		t = binary.LittleEndian.AppendUint32(
			t,
			placeAt(c, fileOff+segment.BlockSize).Sum(t),
		)
		out = append(out, t...)
	}
	return append(out, stream[full*segment.BlockSize:]...)
}

// batchOf returns a write of random whole samples that starts at stream length at.
func batchOf(r *rand.Rand, c segment.Config, at int, n *int) ([]byte, []int64) {
	var (
		batch  []byte
		starts []int64
	)
	for range r.IntN(5) {
		var size int
		switch r.IntN(4) {
		case 0:
		case 1:
			size = r.IntN(40)
		case 2:
			// End the sample, or its length, at or near a block end.
			size = segment.BlockSize - 4 - (at+len(batch))%segment.BlockSize -
				r.IntN(6) + 2
			size = max(size, 0)
		default:
			size = r.IntN(3 * segment.BlockSize)
		}
		starts = append(starts, int64(at+len(batch)))
		if !c.Variable {
			batch = append(batch, pattern(size/8*8)...)
			continue
		}
		batch = binary.LittleEndian.AppendUint32(batch, uint32(size))
		for j := range size {
			batch = append(batch, byte(*n+j*5))
		}
		*n++
	}
	return batch, starts
}

var _ = Describe("Write model check", func() {
	DescribeTable(
		"Should write the reference bytes over random writes, commits, and short writes",
		func(c segment.Config) {
			for runKey := range uint64(48) {
				r := rand.New(rand.NewPCG(runKey, 1))
				mem := MustSucceed(xfs.NewMem().Sub("d"))
				sfs := &shortFS{FS: mem, fail: -1}
				b := MustSucceed(segment.NewBudget(1))
				h := MustSucceed(b.Create(sfs, "1.seg"))
				w := segment.NewWriter(h, c)
				var (
					stream, committed       []byte
					starts, committedStarts []int64
					count                   int
				)
				for range 60 {
					batch, bStarts := batchOf(r, c, len(stream), &count)
					if r.IntN(5) == 0 {
						sfs.mu.Lock()
						sfs.fail = r.IntN(len(batch) + segment.VariableTrailer + 1)
						sfs.mu.Unlock()
						Expect(w.Write(batch)).To(MatchError(syscall.ENOSPC))
						stream = slices.Clone(committed)
						starts = slices.Clone(committedStarts)
						Expect(
							w.Len(),
						).To(Equal(int64(len(stream))), "run key %d", runKey)
						continue
					}
					Expect(w.Write(batch)).To(Succeed())
					stream = append(stream, batch...)
					starts = append(starts, bStarts...)
					switch r.IntN(6) {
					case 0, 1:
						length, _ := w.Commit()
						Expect(length).To(Equal(uint32(len(stream))))
						committed = slices.Clone(stream)
						committedStarts = slices.Clone(starts)
					case 2:
						w.PutBack()
						stream = slices.Clone(committed)
						starts = slices.Clone(committedStarts)
					}
				}
				length, tail := w.Commit()
				Expect(length).To(Equal(uint32(len(stream))))
				full := len(stream) / segment.BlockSize * segment.BlockSize
				Expect(tail).To(Equal(
					placeAt(c, c.FileOffset(int64(full))).Sum(stream[full:]),
				), "run key %d", runKey)
				Expect(h.Close()).To(Succeed())
				f := MustSucceed(mem.Open("1.seg", os.O_RDONLY))
				got := MustSucceed(io.ReadAll(f))
				Expect(f.Close()).To(Succeed())
				want := reference(c, stream, starts)
				Expect(
					len(got),
				).To(BeNumerically(">=", len(want)), "run key %d", runKey)
				Expect(got[:len(want)]).To(Equal(want), "run key %d", runKey)
			}
		},
		Entry("fixed", cfg),
		Entry("variable", varCfg),
	)
})
