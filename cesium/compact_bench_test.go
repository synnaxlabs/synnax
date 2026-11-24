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
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// unlinkFS signals unlinked each time it removes a segment file.
type unlinkFS struct {
	xfs.FS
	unlinked chan struct{}
}

func (f *unlinkFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *unlinkFS) Remove(name string) error {
	err := f.FS.Remove(name)
	if strings.HasSuffix(name, ".seg") {
		select {
		case f.unlinked <- struct{}{}:
		default:
		}
	}
	return err
}

// BenchmarkCompact rewrites one sealed segment of int64 samples whose middle half a
// delete took out, per iteration, on the OS file system with no rate limit. The time
// of an iteration runs from the return of the delete to the unlink of the old file, and
// its bytes are the live bytes of the copy.
func BenchmarkCompact(b *testing.B) {
	for _, size := range []int{1 << 20, 64 << 20} {
		b.Run(fmt.Sprintf("size=%dMiB", size>>20), func(b *testing.B) {
			benchmarkCompact(b, size)
		})
	}
}

func benchmarkCompact(b *testing.B, size int) {
	b.Helper()
	root, err := xfs.Default.Sub(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	fs := &unlinkFS{FS: root, unlinked: make(chan struct{}, 1)}
	db := runCompactDB(b, "", cesium.WithFS(fs), cesium.WithGCConfig(cesium.GCConfig{
		Rate: 1 << 50,
	}))
	keys := []cesium.ChannelKey{1, 2}
	if err := db.CreateChannel(
		b.Context(),
		cesium.Channel{Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT},
		cesium.Channel{Key: 2, Name: "data", Index: 1, DataType: telem.Int64T},
	); err != nil {
		b.Fatal(err)
	}
	n := size / 8
	data := telem.NewSeries(make([]int64, n))
	stamps := make([]telem.TimeStamp, n)
	b.SetBytes(int64(size / 2))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		b.StopTimer()
		first := telem.TimeStamp(i*(n+1)+1) * telem.SecondTS
		for j := range stamps {
			stamps[j] = first + telem.TimeStamp(j)*telem.SecondTS
		}
		writeCompactFrame(b, db, keys, first, telem.NewSeries(stamps), data)
		last := first + telem.TimeStamp(n)*telem.SecondTS
		writeCompactFrame(
			b,
			db,
			keys,
			last,
			telem.NewSeries([]telem.TimeStamp{last}),
			telem.NewSeries([]int64{0}),
		)
		quarter := telem.TimeStamp(n/4) * telem.SecondTS
		middle := (first + quarter).Range(first + 3*quarter)
		if err := db.DeleteTimeRange(b.Context(), keys[1:], middle); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		<-fs.unlinked
		b.StopTimer()
		// The delete of the whole stretch unlinks both segments, which keeps the memory
		// file system small.
		whole := first.Range(last)
		if err := db.DeleteTimeRange(b.Context(), keys, whole); err != nil {
			b.Fatal(err)
		}
		select {
		case <-fs.unlinked:
		default:
		}
		b.StartTimer()
	}
}

// BenchmarkCompactCommit writes one frame of float32 samples to 32 channels, then
// commits and syncs it, 20,000 times per iteration, on the OS file system. With rewrite
// set, the compactor rewrites seven segments of another key at the default rate, and
// the run fails when the rewrites end before the commits. It reports the 99th
// percentile of the time of a commit.
func BenchmarkCompactCommit(b *testing.B) {
	for _, rewrite := range []bool{false, true} {
		b.Run(fmt.Sprintf("rewrite=%t", rewrite), func(b *testing.B) {
			benchmarkCompactCommit(b, rewrite)
		})
	}
}

func benchmarkCompactCommit(b *testing.B, rewrite bool) {
	b.Helper()
	dir := b.TempDir()
	db := runCompactDB(b, dir)
	running := func() bool { return true }
	if rewrite {
		running = startRewrite(b, db, dir, 7)
	}
	benchmarkCommits(b, db, running, "the rewrites ended before the commits")
}

// benchmarkCommits writes one frame of float32 samples to 32 new channels of db, then
// commits and syncs it, 20,000 times per iteration, and reports the 99th percentile of
// the time of a commit. It fails with ended when running reports false at the end.
func benchmarkCommits(b *testing.B, db *cesium.DB, running func() bool, ended string) {
	b.Helper()
	keys := createIndexedChannels(b, db, 1, 32)
	w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
		Channels: keys,
		Clock:    keys[0],
		Start:    telem.SecondTS,
	})
	if err != nil {
		b.Fatal(err)
	}
	const samples, commits = 125, 20_000
	data := telem.NewSeries(make([]float32, samples))
	stamps := make([]telem.TimeStamp, samples)
	start := telem.SecondTS
	took := make([]time.Duration, 0, commits)
	b.ReportAllocs()
	for b.Loop() {
		for range commits {
			began := time.Now()
			for i := range stamps {
				stamps[i] = start + telem.TimeStamp(i)
			}
			series := make([]telem.Series, len(keys))
			series[0] = telem.NewSeries(stamps)
			for i := 1; i < len(series); i++ {
				series[i] = data
			}
			if _, _, err := w.Write(
				b.Context(),
				telem.MultiFrame(keys, series),
				nil,
			); err != nil {
				b.Fatal(err)
			}
			if _, _, err := w.Commit(b.Context(), nil); err != nil {
				b.Fatal(err)
			}
			if err := w.Sync(); err != nil {
				b.Fatal(err)
			}
			start += telem.TimeStamp(samples)
			took = append(took, time.Since(began))
		}
	}
	b.StopTimer()
	if !running() {
		b.Fatal(ended)
	}
	reportP99(b, took)
	if err := w.Close(); err != nil {
		b.Error(err)
	}
}

// BenchmarkCompactRead reads an index and a data channel of 100 stretches of 100
// samples, per iteration, on the OS file system. With rewrite set, the compactor
// rewrites a segment of another key at the default rate, and the run fails when the
// rewrite ends before the reads. It reports the 99th percentile of the time of an
// iteration.
func BenchmarkCompactRead(b *testing.B) {
	for _, rewrite := range []bool{false, true} {
		b.Run(fmt.Sprintf("rewrite=%t", rewrite), func(b *testing.B) {
			benchmarkCompactRead(b, rewrite)
		})
	}
}

func benchmarkCompactRead(b *testing.B, rewrite bool) {
	b.Helper()
	dir := b.TempDir()
	db := runCompactDB(b, dir)
	keys := writeReadStretches(b, db)
	running := func() bool { return true }
	if rewrite {
		running = startRewrite(b, db, dir, 1)
	}
	benchmarkReads(b, db, keys, running, "the rewrite ended before the reads")
}

// readStretches and readSamples are the stretches that writeReadStretches writes, and
// the samples of each.
const readStretches, readSamples = 100, 100

// writeReadStretches writes readStretches stretches of readSamples samples to a new
// index and data channel of db, keys 1 and 2, and returns their keys.
func writeReadStretches(b *testing.B, db *cesium.DB) []cesium.ChannelKey {
	b.Helper()
	keys := []cesium.ChannelKey{1, 2}
	if err := db.CreateChannel(
		b.Context(),
		cesium.Channel{Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT},
		cesium.Channel{Key: 2, Name: "data", Index: 1, DataType: telem.Int64T},
	); err != nil {
		b.Fatal(err)
	}
	for s := range readStretches {
		writePartsStretch(b, b.Context(), db, keys, s*readSamples, readSamples)
	}
	return keys
}

// benchmarkReads reads keys of db, from writeReadStretches, per iteration, and reports
// the 99th percentile of the time of an iteration. It fails with ended when running
// reports false at the end.
func benchmarkReads(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	running func() bool,
	ended string,
) {
	b.Helper()
	const stretches, samples = readStretches, readSamples
	took := make([]time.Duration, 0, 1<<20)
	b.ReportAllocs()
	for b.Loop() {
		began := time.Now()
		if read := readAll(b, db, keys); read != 2*stretches*samples {
			b.Fatalf("read %d samples, want %d", read, 2*stretches*samples)
		}
		if len(took) < cap(took) {
			took = append(took, time.Since(began))
		}
	}
	b.StopTimer()
	if !running() {
		b.Fatal(ended)
	}
	reportP99(b, took)
}

// runCompactDB opens a database of the pointer log layout in dir with opts, and runs it
// until the benchmark ends.
func runCompactDB(b *testing.B, dir string, opts ...cesium.Option) *cesium.DB {
	b.Helper()
	db, err := cesium.Open(b.Context(), dir, append([]cesium.Option{
		cesium.WithInstrumentation(alamos.New("bench")),
		cesium.WithPointerLog(),
	}, opts...)...)
	if err != nil {
		b.Fatal(err)
	}
	stop := testutil.GoRun(b.Context(), db.Run)
	b.Cleanup(func() {
		if err := errors.Join(stop(), db.Close()); err != nil {
			b.Error(err)
		}
	})
	return db
}

// writeCompactFrame writes stamps to keys[0] and data to keys[1] with one writer from
// start.
func writeCompactFrame(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	start telem.TimeStamp,
	stamps, data telem.Series,
) {
	b.Helper()
	w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
		Channels: keys,
		Clock:    keys[0],
		Start:    start,
	})
	if err != nil {
		b.Fatal(err)
	}
	if _, _, err = w.Write(
		b.Context(),
		telem.MultiFrame(keys, []telem.Series{stamps, data}),
		nil,
	); err != nil {
		b.Fatal(err)
	}
	if _, _, err = w.Commit(b.Context(), nil); err != nil {
		b.Fatal(err)
	}
	if err = w.Close(); err != nil {
		b.Fatal(err)
	}
}

// startRewrite writes segments sealed segments of 512 samples of 1 MiB to a text key,
// deletes the last quarter of each, and returns once the compactor starts to write the
// first new segment. At the default rate of 64 MiB a second, the copy of each segment
// takes 6 s. The function it returns reports whether a rewrite still runs.
func startRewrite(b *testing.B, db *cesium.DB, dir string, segments int) func() bool {
	b.Helper()
	keys := []cesium.ChannelKey{1000, 1001}
	if err := db.CreateChannel(
		b.Context(),
		cesium.Channel{
			Key: 1000, Name: "rewrite_time", IsIndex: true, DataType: telem.TimestampT,
		},
		cesium.Channel{
			Key: 1001, Name: "rewrite_text", Index: 1000, DataType: telem.StringT,
		},
	); err != nil {
		b.Fatal(err)
	}
	writeRewriteSegments(b, db, keys, segments)
	for seg := range segments {
		from, to := seg*rewritePer+rewritePer*3/4, (seg+1)*rewritePer
		tail := rewriteStamp(from).Range(rewriteStamp(to))
		if err := db.DeleteTimeRange(b.Context(), keys[1:], tail); err != nil {
			b.Fatal(err)
		}
	}
	// The old segments, the segment of the end frame, and the first new segment.
	names := awaitSegments(b, dir, keys[1], segments+2)
	slices.Sort(names)
	old := names[:segments]
	return func() bool {
		return slices.ContainsFunc(old, func(name string) bool {
			_, err := os.Stat(filepath.Join(dir, strconv.Itoa(int(keys[1])), name))
			return err == nil
		})
	}
}

// rewritePer is the count of 1 MiB samples in each segment that startRewrite writes.
const rewritePer = 512

// rewriteStamp returns the stamp of sample i of the segments that startRewrite writes.
func rewriteStamp(i int) telem.TimeStamp {
	return telem.TimeStamp(i+1) * telem.SecondTS
}

// writeRewriteSegments writes segments sealed segments of rewritePer samples of 1 MiB
// to the text key of keys, timed by its index, from second 1 on. The open of each
// writer seals the segment of the one before it, and an end frame seals the last.
func writeRewriteSegments(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	segments int,
) {
	b.Helper()
	const chunk = 64
	sample := strings.Repeat("x", 1<<20)
	strs := make([]string, chunk)
	for i := range strs {
		strs[i] = sample
	}
	text := telem.NewSeriesV(strs...)
	stamps := make([]telem.TimeStamp, chunk)
	for seg := range segments {
		w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
			Channels: keys,
			Clock:    keys[0],
			Start:    rewriteStamp(seg * rewritePer),
		})
		if err != nil {
			b.Fatal(err)
		}
		for c := range rewritePer / chunk {
			for i := range stamps {
				stamps[i] = rewriteStamp(seg*rewritePer + c*chunk + i)
			}
			if _, _, err = w.Write(b.Context(), telem.MultiFrame(
				keys,
				[]telem.Series{telem.NewSeries(stamps), text},
			), nil); err != nil {
				b.Fatal(err)
			}
			if _, _, err = w.Commit(b.Context(), nil); err != nil {
				b.Fatal(err)
			}
		}
		if err = w.Close(); err != nil {
			b.Fatal(err)
		}
	}
	end := rewriteStamp(segments * rewritePer)
	writeCompactFrame(
		b,
		db,
		keys,
		end,
		telem.NewSeries([]telem.TimeStamp{end}),
		telem.NewSeriesV("end"),
	)
}

// awaitSegments returns the names of the segment files of key in dir once there are n.
func awaitSegments(b *testing.B, dir string, key cesium.ChannelKey, n int) []string {
	b.Helper()
	segments := filepath.Join(dir, strconv.Itoa(int(key)))
	for {
		entries, err := os.ReadDir(segments)
		if err != nil {
			b.Fatal(err)
		}
		var names []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".seg") {
				names = append(names, e.Name())
			}
		}
		if len(names) == n {
			return names
		}
		time.Sleep(time.Millisecond)
	}
}

// reportP99 reports the 99th percentile of took as p99-ns/op.
func reportP99(b *testing.B, took []time.Duration) {
	b.Helper()
	if len(took) == 0 {
		return
	}
	slices.Sort(took)
	b.ReportMetric(float64(took[len(took)*99/100].Nanoseconds()), "p99-ns/op")
}
