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
	"io"
	"syscall"
	"testing"
)

// BenchmarkFloorRead reads a file of floorFileSize from the disk in 1 MiB blocks, per
// iteration. It drops the file from the page cache before each read, which only Linux
// lets an unprivileged process do.
func BenchmarkFloorRead(b *testing.B) {
	f := createFloorFile(b, b.TempDir())
	block := make([]byte, mib)
	for range floorFileSize / mib {
		if _, err := f.Write(block); err != nil {
			b.Fatal(err)
		}
	}
	if err := f.Sync(); err != nil {
		b.Fatal(err)
	}
	const fadvDontNeed = 4
	b.SetBytes(floorFileSize)
	for b.Loop() {
		b.StopTimer()
		if _, _, errno := syscall.Syscall6(
			syscall.SYS_FADVISE64, f.Fd(), 0, 0, fadvDontNeed, 0, 0,
		); errno != 0 {
			b.Fatal(errno)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		for range floorFileSize / mib {
			if _, err := io.ReadFull(f, block); err != nil {
				b.Fatal(err)
			}
		}
	}
}
