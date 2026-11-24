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
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/testutil/oldlayout"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkConvert times the first open of a root of the old layout on the pointer log,
// one conversion per iteration. The old path of this binary writes the root once, in
// CESIUM_BENCH_DIR when it is set, so that a run after a page cache drop reads it cold.
// Each iteration first removes, untimed, what the conversion before it wrote. It
// reports the terms of the conversion target: the bytes of the old metadata files, the
// old data files, the stamp reads, the bytes of head 1, and the bytes of both log
// files.
func BenchmarkConvert(b *testing.B) {
	for _, size := range []struct{ keys, files int }{
		{100, 1}, {100, 16}, {1_000, 1}, {1_000, 16}, {10_000, 1},
	} {
		keys, files := size.keys, size.files
		b.Run(fmt.Sprintf("keys=%d/files=%d", keys, files), func(b *testing.B) {
			root := convertFixture(b, keys, files)
			meta, old := oldTerms(b, root)
			fs := &stampCounter{FS: root, reads: set.New[stampRead]()}
			var head, logs int64
			for b.Loop() {
				b.StopTimer()
				undoConversion(b, root)
				fs.reset()
				b.StartTimer()
				db, err := cesium.Open(
					b.Context(),
					"",
					cesium.WithFS(fs),
					cesium.WithPointerLog(),
				)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				if err = db.Close(); err != nil {
					b.Fatal(err)
				}
				head = fileSize(b, root, checkpoint.Name(1))
				logs = fileSize(b, root, log.Name(1)) + fileSize(b, root, log.Name(2))
				b.StartTimer()
			}
			b.ReportMetric(float64(meta), "meta-B")
			b.ReportMetric(float64(old), "old-files")
			b.ReportMetric(float64(fs.count()), "stamp-reads")
			b.ReportMetric(float64(head), "head-B")
			b.ReportMetric(float64(logs), "logs-B")
		})
	}
}

// convertFixture returns the root of an old layout of keys keys, in groups of an index
// and 9 float64 keys that it times, where each key holds about files old data files. It
// writes the root once, syncs each of its files, and keeps it in benchDir.
func convertFixture(b *testing.B, keys, files int) xfs.FS {
	b.Helper()
	const (
		group   = 10
		samples = 128
	)
	dir := filepath.Join(
		benchDir(b),
		fmt.Sprintf("convert-keys=%d-files=%d", keys, files),
	)
	done := dir + ".done"
	if _, err := os.Stat(done); err == nil {
		return mustSub(b, dir)
	}
	if err := os.RemoveAll(dir); err != nil {
		b.Fatal(err)
	}
	root := mustSub(b, dir)
	for g := range keys / group {
		group := fixtureGroup(g*group+1, group, files, samples)
		if err := oldlayout.Write(root, group...); err != nil {
			b.Fatal(err)
		}
	}
	syncTree(b, root)
	if err := os.WriteFile(done, nil, 0o644); err != nil {
		b.Fatal(err)
	}
	return root
}

// fixtureGroup returns the old layout of an index at key first and the group-1 float64
// keys after it that it times: files old files of each key, each with one pointer of 8
// commits of samples samples. Sample n has the stamp 1 s + n ms and the value n.
func fixtureGroup(first, group, files, samples int) []oldlayout.Key {
	perFile := 8 * samples
	out := make([]oldlayout.Key, group)
	for i := range out {
		key := cesium.ChannelKey(first + i)
		ch := cesium.Channel{
			Key:      key,
			Name:     strconv.FormatUint(uint64(key), 10),
			DataType: telem.Float64T,
			Index:    cesium.ChannelKey(first),
		}
		if i == 0 {
			ch.DataType, ch.Index, ch.IsIndex = telem.TimestampT, 0, true
		}
		ch.Version = channel.VersionCurrent
		k := oldlayout.Key{
			Channel: ch,
			Counter: uint32(files),
			Files:   make(map[uint16][]byte, files),
		}
		for f := range files {
			stamps := make([]telem.TimeStamp, perFile)
			values := make([]float64, perFile)
			for j := range perFile {
				n := f*perFile + j
				stamps[j] = telem.SecondTS + telem.TimeStamp(n)*telem.MillisecondTS
				values[j] = float64(n)
			}
			data := telem.NewSeries(values).Data
			if i == 0 {
				data = telem.NewSeries(stamps).Data
			}
			k.Files[uint16(f+1)] = data
			k.Pointers = append(k.Pointers, oldlayout.Pointer{
				TimeRange: telem.TimeRange{
					Start: stamps[0],
					End:   stamps[perFile-1] + 1,
				},
				File: uint16(f + 1),
				Size: uint32(len(data)),
			})
		}
		out[i] = k
	}
	return out
}

func fileSize(b *testing.B, fs xfs.FS, name string) int64 {
	b.Helper()
	info, err := fs.Stat(name)
	if err != nil {
		b.Fatal(err)
	}
	return info.Size()
}

func mustSub(b *testing.B, dir string) xfs.FS {
	b.Helper()
	fs, err := xfs.Default.Sub(dir)
	if err != nil {
		b.Fatal(err)
	}
	return fs
}

// syncTree syncs each file of each directory of root, then each directory, then root.
func syncTree(b *testing.B, root xfs.FS) {
	b.Helper()
	dirs, err := root.List("")
	if err != nil {
		b.Fatal(err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		infos, err := root.List(d.Name())
		if err != nil {
			b.Fatal(err)
		}
		for _, info := range infos {
			f, err := root.Open(path.Join(d.Name(), info.Name()), os.O_RDWR)
			if err != nil {
				b.Fatal(err)
			}
			if err = errors.Join(f.Sync(), f.Close()); err != nil {
				b.Fatal(err)
			}
		}
		if err = root.SyncDir(d.Name()); err != nil {
			b.Fatal(err)
		}
	}
	if err = root.SyncDir(""); err != nil {
		b.Fatal(err)
	}
}

// oldDataFile matches the name of an old data file.
var oldDataFile = regexp.MustCompile(`^[0-9]+\.domain$`)

// oldTerms returns the bytes of the meta.json and index.domain files of root, and its
// number of old data files.
func oldTerms(b *testing.B, root xfs.FS) (meta int64, files int) {
	b.Helper()
	dirs, err := root.List("")
	if err != nil {
		b.Fatal(err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		infos, err := root.List(d.Name())
		if err != nil {
			b.Fatal(err)
		}
		for _, info := range infos {
			switch name := info.Name(); {
			case name == "meta.json" || name == "index.domain":
				meta += info.Size()
			case oldDataFile.MatchString(name):
				files++
			}
		}
	}
	return meta, files
}

// undoConversion removes what a conversion of root wrote: its heads, its logs, and the
// channel file of each key.
func undoConversion(b *testing.B, root xfs.FS) {
	b.Helper()
	infos, err := root.List("")
	if err != nil {
		b.Fatal(err)
	}
	for _, info := range infos {
		name := info.Name()
		switch {
		case info.IsDir():
			err = root.Remove(path.Join(name, "channel"))
		case strings.HasSuffix(name, ".head") || strings.HasSuffix(name, ".log"):
			err = root.Remove(name)
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

// stampRead is a read of one stamp: an old data file and an offset.
type stampRead struct {
	name string
	off  int64
}

// stampCounter counts the distinct stamp reads of old data files: reads of 8 bytes.
type stampCounter struct {
	xfs.FS
	mu    sync.Mutex
	reads set.Set[stampRead]
}

func (s *stampCounter) Open(name string, flag int) (xfs.File, error) {
	f, err := s.FS.Open(name, flag)
	if err != nil || !oldDataFile.MatchString(path.Base(name)) {
		return f, err
	}
	return stampFile{File: f, name: name, s: s}, nil
}

// Sub returns s for the root, which Open takes, so that s sees every read.
func (s *stampCounter) Sub(name string) (xfs.FS, error) {
	if name == "" {
		return s, nil
	}
	return s.FS.Sub(name)
}

func (s *stampCounter) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads = set.New[stampRead]()
}

func (s *stampCounter) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reads)
}

type stampFile struct {
	xfs.File
	name string
	s    *stampCounter
}

func (f stampFile) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 8 {
		f.s.mu.Lock()
		f.s.reads.Add(stampRead{name: f.name, off: off})
		f.s.mu.Unlock()
	}
	return f.File.ReadAt(p, off)
}
