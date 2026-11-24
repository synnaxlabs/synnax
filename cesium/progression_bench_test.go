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
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// encodedBytes is the file size of a version 2 clock of one block: the block and its
// trailer.
const encodedBytes = 65_540

// clockBytes returns the bytes of the largest segment file of key in dir.
func clockBytes(b *testing.B, dir string, key cesium.ChannelKey) int64 {
	b.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, strconv.Itoa(int(key))))
	if err != nil {
		b.Fatal(err)
	}
	var most int64
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".seg") {
			continue
		}
		info, err := e.Info()
		if errors.Is(err, os.ErrNotExist) {
			// A rewrite unlinked the file after the listing.
			continue
		}
		if err != nil {
			b.Fatal(err)
		}
		most = max(most, info.Size())
	}
	return most
}

// awaitEncoded returns once the largest segment file of each of keys in dir holds at
// most limit bytes, which only an encoding gives the clocks of these benchmarks.
func awaitEncoded(b *testing.B, dir string, limit int64, keys ...cesium.ChannelKey) {
	b.Helper()
	for _, key := range keys {
		for clockBytes(b, dir, key) > limit {
			time.Sleep(time.Millisecond)
		}
	}
}

// writeFrames writes frames of per stamps step apart, with a gap before each, to the
// index keys[0], and zeros to each data key of keys. Then it writes one stamp an hour
// later, whose writer seals the segments of the others. It returns the stamps of the
// frames.
func writeFrames(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	frames, per int,
	step telem.TimeStamp,
) []telem.TimeStamp {
	b.Helper()
	all := make([]telem.TimeStamp, 0, frames*per)
	x := telem.SecondTS
	for f := range frames {
		x += step + telem.TimeStamp(1+f%7)
		for range per {
			all = append(all, x)
			x += step
		}
	}
	write := func(stamps []telem.TimeStamp) {
		w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
			Channels: keys,
			Clock:    keys[0],
			Start:    stamps[0],
		})
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < len(stamps); i += per {
			s := stamps[i:min(i+per, len(stamps))]
			series := make([]telem.Series, 0, len(keys))
			series = append(series, telem.NewSeries(s))
			for range keys[1:] {
				series = append(series, telem.NewSeries(make([]float32, len(s))))
			}
			if _, _, err = w.Write(
				b.Context(),
				telem.MultiFrame(keys, series),
				nil,
			); err != nil {
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
	write(all)
	write([]telem.TimeStamp{x + telem.HourTS})
	return all
}

// BenchmarkProgressionSize writes the clock of a Driver card, 400 writes of 125 stamps
// 40 µs apart, and reports the bytes of its segment file before and after the compactor
// stores it as progressions. Each iteration reads the clock once.
func BenchmarkProgressionSize(b *testing.B) {
	dir := b.TempDir()
	db := runCompactDB(b, dir)
	keys := createIndexedChannels(b, db, 1, 0)
	stamps := writeFrames(b, db, keys, 400, 125, 40_000)
	before := int64(len(stamps))*8 + int64(len(stamps))*8/65_536*4
	awaitEncoded(b, dir, encodedBytes, keys[0])
	after := clockBytes(b, dir, keys[0])
	for b.Loop() {
		if read := readAll(b, db, keys); read != len(stamps)+1 {
			b.Fatalf("read %d stamps, want %d", read, len(stamps)+1)
		}
	}
	b.ReportMetric(float64(before), "v1-bytes")
	b.ReportMetric(float64(after), "v2-bytes")
}

// BenchmarkReadProgressions reads an index of 1M stamps 1 ms apart and a data channel,
// with the index at version 1 or as progressions: a scan of both, and a read of one
// sample of the data at a random stamp, which looks the stamp up in its clock.
func BenchmarkReadProgressions(b *testing.B) {
	for _, version := range []int{1, 2} {
		dir := b.TempDir()
		db := runCompactDB(b, dir)
		keys := createIndexedChannels(b, db, 1, 1)
		var stamps []telem.TimeStamp
		if version == 1 {
			stamps = writeBenchData(b, db, keys, 1_000_000, 0, true)
		} else {
			stamps = writeFrames(b, db, keys, 1, 1_000_000, telem.MillisecondTS)
			awaitEncoded(b, dir, encodedBytes, keys[0])
		}
		b.Run(fmt.Sprintf("version=%d/read=scan", version), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if read := readAll(b, db, keys); read < 2*len(stamps) {
					b.Fatalf("read %d samples, want %d", read, 2*len(stamps))
				}
			}
		})
		b.Run(fmt.Sprintf("version=%d/read=point", version), func(b *testing.B) {
			r := rand.New(rand.NewPCG(1, 2))
			b.ReportAllocs()
			for b.Loop() {
				s := stamps[r.IntN(len(stamps))]
				fr, err := db.Read(b.Context(), s.Range(s+1), keys[1])
				if err != nil {
					b.Fatal(err)
				}
				if read := fr.Get(keys[1]).Len(); read != 1 {
					b.Fatalf("read %d samples at %d, want 1", read, s)
				}
			}
		})
	}
}

// BenchmarkProgressionBacklog writes 1,000 clocks of 16,400 stamps 1 ms apart, a sealed
// segment of 2 blocks each, while a rate of 1 byte a second holds off their encoding.
// Then it opens the database again, as a restart does. It reports the time of that open
// and the time until the compactor stores every clock as progressions at the default
// rate, the time per clock of the same backlog at no rate limit (the sync of the new
// file and the durable entry of its Replace), the time of an open of the cleared root,
// and the bytes of the newest head.
func BenchmarkProgressionBacklog(b *testing.B) {
	const clocks = 1_000
	// backlog writes the clocks to a new root, opens it again at rate, and waits until
	// every clock is encoded. It returns the root, the time of the open, and the time
	// until the encodings end, with the database closed.
	backlog := func(rate telem.Size) (string, time.Duration, time.Duration) {
		dir := b.TempDir()
		db, stop := openRun(b, dir, 1)
		keys := createIndexedChannels(b, db, clocks, 0)
		for _, key := range keys {
			writeFrames(b, db, []cesium.ChannelKey{key}, 1, 16_400, telem.MillisecondTS)
		}
		closeRun(b, db, stop)
		began := time.Now()
		db, stop = openRun(b, dir, rate)
		opened := time.Since(began)
		awaitEncoded(b, dir, encodedBytes, keys...)
		cleared := time.Since(began)
		closeRun(b, db, stop)
		return dir, opened, cleared
	}
	dir, opened, cleared := backlog(0)
	_, _, unlimited := backlog(1 << 50)
	began := time.Now()
	db, stop := openRun(b, dir, 0)
	openedClear := time.Since(began)
	closeRun(b, db, stop)
	for b.Loop() {
	}
	b.ReportMetric(float64(opened.Nanoseconds()), "open-ns")
	b.ReportMetric(float64(cleared.Nanoseconds()), "backlog-ns")
	b.ReportMetric(float64(unlimited.Nanoseconds())/clocks, "replace-wait-ns")
	b.ReportMetric(float64(openedClear.Nanoseconds()), "open-clear-ns")
	b.ReportMetric(float64(newestHead(b, dir)), "head-bytes")
}

// openRun opens the pointer log database in dir with the background read rate rate, or
// the default rate when rate is zero, and runs it. It returns the stop of the run.
func openRun(b *testing.B, dir string, rate telem.Size) (*cesium.DB, func() error) {
	b.Helper()
	db, err := cesium.Open(b.Context(), dir, cesium.WithPointerLog(),
		cesium.WithGCConfig(cesium.GCConfig{Rate: rate}))
	if err != nil {
		b.Fatal(err)
	}
	return db, testutil.GoRun(b.Context(), db.Run)
}

// closeRun stops the run of db and closes it.
func closeRun(b *testing.B, db *cesium.DB, stop func() error) {
	b.Helper()
	if err := errors.Join(stop(), db.Close()); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkEncodeCommit is BenchmarkCompactCommit with the compactor encoding six clock
// segments of 256 MiB at the default rate, 48 s of work, or with none.
func BenchmarkEncodeCommit(b *testing.B) {
	encode := func(b *testing.B, db *cesium.DB, dir string) func() bool {
		b.Helper()
		return startEncode(b, db, dir, 6)
	}
	for _, on := range []bool{false, true} {
		b.Run(fmt.Sprintf("encode=%t", on), func(b *testing.B) {
			start := encode
			if !on {
				start = nil
			}
			benchmarkCompactCommit(b, start)
		})
	}
}

// newestHead returns the bytes of the head file of the highest generation in dir.
func newestHead(b *testing.B, dir string) int64 {
	b.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatal(err)
	}
	var newest os.DirEntry
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".head") &&
			(newest == nil || e.Name() > newest.Name()) {
			newest = e
		}
	}
	if newest == nil {
		b.Fatal("no head")
	}
	info, err := newest.Info()
	if err != nil {
		b.Fatal(err)
	}
	return info.Size()
}

// startEncode writes segments sealed clock segments of 32M stamps 1 ms apart, 256 MiB
// each, and returns a function that reports whether one of them is still at version 1.
// At the default rate of 64 MiB a second, the two passes of each take 8 s.
func startEncode(b *testing.B, db *cesium.DB, dir string, segments int) func() bool {
	b.Helper()
	keys := make([]cesium.ChannelKey, segments)
	for i := range keys {
		keys[i] = cesium.ChannelKey(3_000 + i)
	}
	for _, key := range keys {
		if err := db.CreateChannel(b.Context(), cesium.Channel{
			Key: key, Name: fmt.Sprintf("encode_%d", key), IsIndex: true,
			DataType: telem.TimestampT,
		}); err != nil {
			b.Fatal(err)
		}
		writeFrames(b, db, []cesium.ChannelKey{key}, 64, 1<<19, telem.MillisecondTS)
	}
	return func() bool {
		for _, key := range keys {
			if clockBytes(b, dir, key) > encodedBytes {
				return true
			}
		}
		return false
	}
}
