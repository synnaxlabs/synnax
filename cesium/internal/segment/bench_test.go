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
	"fmt"
	"testing"

	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// BenchmarkWrite measures a write of one series to an open fixed-density segment in
// memory, which a commit then puts back so the file stays one block long.
func BenchmarkWrite(b *testing.B) {
	for _, size := range []int{512, 16 << 10, 1 << 20} {
		b.Run(fmt.Sprintf("bytes=%d", size), func(b *testing.B) {
			budget, err := segment.NewBudget(1, 4, segment.GoExecutor)
			if err != nil {
				b.Fatal(err)
			}
			h, err := budget.Create(xfs.NewMem(), "1.seg")
			if err != nil {
				b.Fatal(err)
			}
			w := segment.NewWriter(h, cfg)
			data := pattern(size)
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for b.Loop() {
				if err = w.Write(data); err != nil {
					b.Fatal(err)
				}
				w.PutBack()
			}
		})
	}
}

// readSegment returns the committed part of a fixed-density segment of size bytes in
// memory, and its handle.
func readSegment(b *testing.B, size int) (*segment.Handle, segment.Committed) {
	b.Helper()
	budget, err := segment.NewBudget(1, 1, segment.GoExecutor)
	if err != nil {
		b.Fatal(err)
	}
	h, err := budget.Create(xfs.NewMem(), "1.seg")
	if err != nil {
		b.Fatal(err)
	}
	w := segment.NewWriter(h, cfg)
	if err = w.Write(pattern(size)); err != nil {
		b.Fatal(err)
	}
	length, tail := w.Commit()
	return h, segment.Committed{Config: cfg, Length: int64(length), TailCRC: tail}
}

// BenchmarkReadScan measures a checked read of a whole segment of 16 MiB in memory,
// which costs one copy and one checksum pass per byte.
func BenchmarkReadScan(b *testing.B) {
	h, com := readSegment(b, 16<<20)
	rd := segment.NewReader()
	p := make([]byte, com.Length)
	b.SetBytes(com.Length)
	b.ReportAllocs()
	for b.Loop() {
		if err := rd.Read(h, com, p, 0); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadStride measures a wide-stride read: one 8-byte sample from each of 64
// blocks, which checks one block per kept sample.
func BenchmarkReadStride(b *testing.B) {
	h, com := readSegment(b, 64*segment.BlockSize)
	rd := segment.NewReader()
	p := make([]byte, 8)
	b.ReportAllocs()
	for b.Loop() {
		for blk := range int64(64) {
			if err := rd.Read(h, com, p, blk*segment.BlockSize+100); err != nil {
				b.Fatal(err)
			}
		}
	}
}
