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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// BenchmarkRepair times a confirmed repair of a root of 1 GiB of segments whose log
// holds one bad frame between valid ones, once per iteration. Each iteration first
// restores the heads and logs of the damaged root, untimed.
func BenchmarkRepair(b *testing.B) {
	dir, saved := b.TempDir(), b.TempDir()
	id := writeRepairRoot(b, dir)
	spoilFrame(b, dir)
	copyRootFiles(b, dir, saved)
	for b.Loop() {
		b.StopTimer()
		removeRootFiles(b, dir)
		copyRootFiles(b, saved, dir)
		b.StartTimer()
		rep, err := cesium.Repair(
			b.Context(),
			dir,
			func(cesium.Report) bool { return true },
			cesium.WithInstrumentation(alamos.New("bench")),
			cesium.WithIdentity(id),
		)
		if err != nil {
			b.Fatal(err)
		}
		if !rep.Repaired || len(rep.Damage) == 0 || len(rep.Unrecovered) > 0 {
			b.Fatalf("repaired %t, with %d damaged frames and %d unrecovered files",
				rep.Repaired, len(rep.Damage), len(rep.Unrecovered))
		}
	}
}

// writeRepairRoot writes two sealed segments of 512 MiB on the pointer log layout in
// dir, and returns the identity of the database.
func writeRepairRoot(b *testing.B, dir string) uint64 {
	b.Helper()
	db, err := cesium.Open(
		b.Context(),
		dir,
		cesium.WithInstrumentation(alamos.New("bench")),
		cesium.WithPointerLog(),
	)
	if err != nil {
		b.Fatal(err)
	}
	stop := testutil.GoRun(b.Context(), db.Run)
	keys := []cesium.ChannelKey{1000, 1001}
	if err = db.CreateChannel(
		b.Context(),
		cesium.Channel{
			Key:      keys[0],
			Name:     "repair_time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
		cesium.Channel{
			Key:      keys[1],
			Name:     "repair_text",
			Index:    keys[0],
			DataType: telem.StringT,
		},
	); err != nil {
		b.Fatal(err)
	}
	writeRewriteSegments(b, db, keys, 2)
	id := db.Identity()
	if err = errors.Join(stop(), db.Close()); err != nil {
		b.Fatal(err)
	}
	return id
}

// spoilFrame changes a byte of the middle frame of the newest log in dir that holds at
// least three frames. A frame starts with the magic CPLF at a block of 4096 bytes from
// offset 8192 on.
func spoilFrame(b *testing.B, dir string) {
	b.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatal(err)
	}
	var logs []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			logs = append(logs, e.Name())
		}
	}
	slices.Sort(logs)
	for _, name := range slices.Backward(logs) {
		path := filepath.Clean(filepath.Join(dir, name))
		data, err := os.ReadFile(path)
		if err != nil {
			b.Fatal(err)
		}
		var frames []int
		for off := 8192; off+4 <= len(data); off += 4096 {
			if string(data[off:off+4]) == "CPLF" {
				frames = append(frames, off)
			}
		}
		if len(frames) < 3 {
			continue
		}
		at := frames[len(frames)/2] + 4
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err == nil {
			_, err = f.WriteAt([]byte{data[at] ^ 0xFF}, int64(at))
			err = errors.Join(err, f.Close())
		}
		if err != nil {
			b.Fatal(err)
		}
		return
	}
	b.Fatal("no log holds three frames")
}

// removeRootFiles removes each file in the root dir, and keeps its directories.
func removeRootFiles(b *testing.B, dir string) {
	b.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatal(err)
	}
	for _, e := range entries {
		if e.Type().IsRegular() {
			if err = os.Remove(filepath.Join(dir, e.Name())); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// copyRootFiles copies each file in the root from to the root to.
func copyRootFiles(b *testing.B, from, to string) {
	b.Helper()
	entries, err := os.ReadDir(from)
	if err != nil {
		b.Fatal(err)
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		src, err := os.Open(filepath.Clean(filepath.Join(from, e.Name())))
		if err != nil {
			b.Fatal(err)
		}
		dst, err := os.Create(filepath.Clean(filepath.Join(to, e.Name())))
		if err == nil {
			_, err = io.Copy(dst, src)
			err = errors.Join(err, dst.Close())
		}
		if err = errors.Join(err, src.Close()); err != nil {
			b.Fatal(err)
		}
	}
}
