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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// checkKeys are the index and the text key of the old root that writeOldRoot writes.
var checkKeys = []cesium.ChannelKey{1000, 1001}

// BenchmarkCheckCommit is the commit run of BenchmarkCompactCommit. With pass set, the
// check pass reads 1.5 GiB of old files at the default rate during the commits, and the
// run fails when the pass ends before them.
func BenchmarkCheckCommit(b *testing.B) {
	for _, pass := range []bool{false, true} {
		b.Run(fmt.Sprintf("pass=%t", pass), func(b *testing.B) {
			dir := b.TempDir()
			running := func() bool { return true }
			if pass {
				running = writeOldRoot(b, dir, 3)
			}
			benchmarkCommits(b, runCompactDB(b, dir), running,
				"the check pass ended before the commits")
		})
	}
}

// BenchmarkCheckRead is the read run of BenchmarkCompactRead. With pass set, the check
// pass reads 512 MiB of old files at the default rate during the reads, and the run
// fails when the pass ends before them.
func BenchmarkCheckRead(b *testing.B) {
	for _, pass := range []bool{false, true} {
		b.Run(fmt.Sprintf("pass=%t", pass), func(b *testing.B) {
			dir := b.TempDir()
			running := func() bool { return true }
			if pass {
				running = writeOldRoot(b, dir, 1)
			}
			db := runCompactDB(b, dir)
			benchmarkReads(b, db, writeReadStretches(b, db), running,
				"the check pass ended before the reads")
		})
	}
}

// BenchmarkCheckPass times the check pass of 512 MiB of synced old files at the default
// rate, from the end of the open that converts them to the last check file, once per
// iteration. It reports the bound of the pass, two reads of each old byte at the rate,
// as bound-s, and the time of the pass as pass-s.
func BenchmarkCheckPass(b *testing.B) {
	const rate = 64 << 20
	var took, bound time.Duration
	for b.Loop() {
		b.StopTimer()
		dir := b.TempDir()
		running := writeOldRoot(b, dir, 1)
		syncTree(b, mustSub(b, dir))
		bound = time.Duration(
			2 * float64(oldDataBytes(b, dir)) / rate * float64(time.Second),
		)
		db, err := cesium.Open(
			b.Context(),
			dir,
			cesium.WithInstrumentation(alamos.New("bench")),
			cesium.WithPointerLog(),
		)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		began := time.Now()
		stop := testutil.GoRun(b.Context(), db.Run)
		for running() {
			time.Sleep(10 * time.Millisecond)
		}
		took = time.Since(began)
		b.StopTimer()
		if err = errors.Join(stop(), db.Close()); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
	b.ReportMetric(took.Seconds(), "pass-s")
	b.ReportMetric(bound.Seconds(), "bound-s")
}

// writeOldRoot writes segments times 512 samples of 1 MiB to the text key of
// checkKeys, timed by its index, on the old layout in dir, in old files of 64 MiB. The
// function it returns reports whether an old file of the text key lacks its check file.
func writeOldRoot(b *testing.B, dir string, segments int) func() bool {
	b.Helper()
	old, err := cesium.Open(
		b.Context(),
		dir,
		cesium.WithInstrumentation(alamos.New("bench")),
		cesium.WithFileSizeCap(64<<20*telem.Byte),
	)
	if err != nil {
		b.Fatal(err)
	}
	if err = old.CreateChannel(
		b.Context(),
		cesium.Channel{
			Key:      checkKeys[0],
			Name:     "old_time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
		cesium.Channel{
			Key:      checkKeys[1],
			Name:     "old_text",
			Index:    checkKeys[0],
			DataType: telem.StringT,
		},
	); err != nil {
		b.Fatal(err)
	}
	writeRewriteSegments(b, old, checkKeys, segments)
	if err = old.Close(); err != nil {
		b.Fatal(err)
	}
	keyDir := filepath.Join(dir, strconv.Itoa(int(checkKeys[1])))
	return func() bool {
		entries, err := os.ReadDir(keyDir)
		if err != nil {
			b.Fatal(err)
		}
		olds, checks := 0, 0
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				b.Fatal(err)
			}
			switch name := e.Name(); {
			case strings.HasSuffix(name, ".chk"):
				checks++
			case isOldData(name) && info.Size() > 0:
				olds++
			}
		}
		return checks < olds
	}
}

// oldDataBytes returns the bytes of the old data files of the text key of checkKeys in
// dir.
func oldDataBytes(b *testing.B, dir string) int64 {
	b.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, strconv.Itoa(int(checkKeys[1]))))
	if err != nil {
		b.Fatal(err)
	}
	var n int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			b.Fatal(err)
		}
		if isOldData(e.Name()) {
			n += info.Size()
		}
	}
	return n
}

// isOldData reports whether name is an old data file N.domain.
func isOldData(name string) bool {
	n, ok := strings.CutSuffix(name, ".domain")
	_, err := strconv.ParseUint(n, 10, 16)
	return ok && err == nil
}
