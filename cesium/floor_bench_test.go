// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"fmt"
	"hash/crc32"
	"os"
	"sync"
	"testing"
)

// The floor benchmarks measure what the host can do, with the standard library only.
// They sit in this package so that a run measures them on the same disk and in the same
// process as the storage benchmarks they bound.

const (
	kib = 1 << 10
	mib = 1 << 20
	// floorFileSize is the size of the file that BenchmarkFloorWrite writes.
	floorFileSize = 256 * mib
)

func sizeName(size int) string {
	switch {
	case size >= mib:
		return fmt.Sprintf("size=%dMiB", size/mib)
	case size >= kib:
		return fmt.Sprintf("size=%dKiB", size/kib)
	}
	return fmt.Sprintf("size=%dB", size)
}

func createFloorFile(b *testing.B, dir string) *os.File {
	b.Helper()
	f, err := os.CreateTemp(dir, "floor")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := f.Close(); err != nil {
			b.Error(err)
		}
	})
	return f
}

// BenchmarkFloorSync appends size bytes to a file and syncs it, per iteration.
func BenchmarkFloorSync(b *testing.B) {
	for _, size := range []int{4 * kib, 16 * kib, 64 * kib, mib} {
		b.Run(sizeName(size), func(b *testing.B) {
			f := createFloorFile(b, b.TempDir())
			block := make([]byte, size)
			for b.Loop() {
				if _, err := f.Write(block); err != nil {
					b.Fatal(err)
				}
				if err := f.Sync(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFloorSyncFiles syncs 1,000 files that each have 4 KiB of new data, per
// iteration, with the given number of syncs in flight. The appends are not timed.
func BenchmarkFloorSyncFiles(b *testing.B) {
	const count = 1_000
	for _, concurrent := range []int{1, 8, 32, 64} {
		name := fmt.Sprintf("files=%d/concurrent=%d", count, concurrent)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			files := make([]*os.File, count)
			for i := range files {
				files[i] = createFloorFile(b, dir)
			}
			block := make([]byte, 4*kib)
			for b.Loop() {
				b.StopTimer()
				for _, f := range files {
					if _, err := f.Write(block); err != nil {
						b.Fatal(err)
					}
				}
				b.StartTimer()
				syncFiles(b, files, concurrent)
			}
		})
	}
}

// syncFiles syncs every file, with the given number of syncs in flight.
func syncFiles(b *testing.B, files []*os.File, concurrent int) {
	b.Helper()
	var wg sync.WaitGroup
	for first := range concurrent {
		wg.Go(func() {
			for i := first; i < len(files); i += concurrent {
				if err := files[i].Sync(); err != nil {
					b.Error(err)
				}
			}
		})
	}
	wg.Wait()
}

// BenchmarkFloorWrite writes a file of floorFileSize in 1 MiB blocks and syncs it, per
// iteration.
func BenchmarkFloorWrite(b *testing.B) {
	f := createFloorFile(b, b.TempDir())
	block := make([]byte, mib)
	b.SetBytes(floorFileSize)
	for b.Loop() {
		if _, err := f.Seek(0, 0); err != nil {
			b.Fatal(err)
		}
		if err := f.Truncate(0); err != nil {
			b.Fatal(err)
		}
		for range floorFileSize / mib {
			if _, err := f.Write(block); err != nil {
				b.Fatal(err)
			}
		}
		if err := f.Sync(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFloorCopy copies size bytes in memory, per iteration.
func BenchmarkFloorCopy(b *testing.B) {
	for _, size := range []int{64, 4 * kib, 64 * kib, mib} {
		b.Run(sizeName(size), func(b *testing.B) {
			src, dst := make([]byte, size), make([]byte, size)
			b.SetBytes(int64(size))
			for b.Loop() {
				copy(dst, src)
			}
		})
	}
}

// BenchmarkFloorCRC32C checksums size bytes with CRC-32C, per iteration.
func BenchmarkFloorCRC32C(b *testing.B) {
	table := crc32.MakeTable(crc32.Castagnoli)
	for _, size := range []int{64, 4 * kib, 64 * kib, mib} {
		b.Run(sizeName(size), func(b *testing.B) {
			data := make([]byte, size)
			b.SetBytes(int64(size))
			for b.Loop() {
				crc32.Checksum(data, table)
			}
		})
	}
}
