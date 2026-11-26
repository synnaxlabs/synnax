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
	"path"
	"strconv"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// openFS counts the open files of its tree, and refuses to remove a file or directory
// that holds an open file, as Windows does.
type openFS struct {
	xfs.FS
	dir   string
	count *openCount
}

type openCount struct {
	mu    sync.Mutex
	files map[string]int
	// segments counts the open segment files, and peak is the most at once.
	segments, peak int
}

func newOpenFS(fs xfs.FS) *openFS {
	return &openFS{FS: fs, count: &openCount{files: map[string]int{}}}
}

func (f *openFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	full, c := path.Join(f.dir, name), f.count
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[full]++
	if strings.HasSuffix(full, ".seg") {
		c.segments++
		c.peak = max(c.peak, c.segments)
	}
	return &openFile{File: file, name: full, count: c}, nil
}

func (f *openFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &openFS{FS: sub, dir: path.Join(f.dir, name), count: f.count}, nil
}

func (f *openFS) Remove(name string) error {
	if n := f.openUnder(path.Join(f.dir, name)); n != "" {
		return errors.Newf("remove of %s, which holds the open file %s", name, n)
	}
	return f.FS.Remove(name)
}

// openUnder returns an open file at or under name, or "" when none is open.
func (f *openFS) openUnder(name string) string {
	f.count.mu.Lock()
	defer f.count.mu.Unlock()
	for n, open := range f.count.files {
		if open > 0 && (n == name || strings.HasPrefix(n, name+"/")) {
			return n
		}
	}
	return ""
}

// resetPeak makes the open segment files now the peak.
func (f *openFS) resetPeak() {
	f.count.mu.Lock()
	defer f.count.mu.Unlock()
	f.count.peak = f.count.segments
}

func (f *openFS) peak() int {
	f.count.mu.Lock()
	defer f.count.mu.Unlock()
	return f.count.peak
}

type openFile struct {
	xfs.File
	name  string
	count *openCount
}

func (f *openFile) Close() error {
	c := f.count
	c.mu.Lock()
	c.files[f.name]--
	if strings.HasSuffix(f.name, ".seg") {
		c.segments--
	}
	c.mu.Unlock()
	return f.File.Close()
}

var _ = Describe("Reads on the pointer log", func() {
	It("Should open each segment file once across reads of its ranges", func(
		ctx SpecContext,
	) {
		rec := NewRecorder(xfs.NewMem())
		db := mustOpenDBOnFS(ctx, rec, cesium.WithPointerLog())
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    telem.SecondTS,
		}))
		const commits = 10
		for i := range commits {
			s := telem.TimeStamp(i + 1)
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(s),
				telem.NewSeries([]int64{int64(s)}),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
		rec.Reset()
		for i := range commits {
			s := telem.TimeStamp(i+1) * telem.SecondTS
			fr := MustSucceed(db.Read(ctx, s.Range(s+1), data))
			Expect(fr.Get(data).Series).To(HaveLen(1))
			Expect(fr.Get(data).Series[0].Len()).To(BeEquivalentTo(1))
		}
		opens := map[string]int{}
		for _, e := range rec.Events() {
			if e.Op == OpOpen && strings.HasSuffix(e.Name, ".seg") {
				opens[e.Name]++
			}
		}
		Expect(opens).To(HaveLen(len(keys)))
		for name, n := range opens {
			Expect(n).To(Equal(1), name)
		}
	})

	It("Should hold no more segment files open than its limit across reads of more "+
		"segments", func(ctx SpecContext) {
		fs := newOpenFS(xfs.NewMem())
		db := mustOpenDBOnFS(ctx, fs, cesium.WithPointerLog())
		// A database holds at most 1024 segment files open. The index makes one more.
		const limit = 1024
		idx := GenerateChannelKey()
		channels := []cesium.Channel{
			{Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT},
		}
		keys := []cesium.ChannelKey{idx}
		series := []telem.Series{telem.NewSeriesSecondsTSV(1)}
		for range limit {
			k := GenerateChannelKey()
			channels = append(channels, cesium.Channel{
				Key: k, Name: "data", Index: idx, DataType: telem.Int64T,
			})
			keys = append(keys, k)
			series = append(series, telem.NewSeries([]int64{1}))
		}
		Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, series), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		fs.resetPeak()
		for _, k := range keys[1:] {
			fr := MustSucceed(db.Read(ctx, telem.SecondTS.Range(telem.SecondTS+1), k))
			Expect(fr.Get(k).Series).To(HaveLen(1))
		}
		Expect(fs.peak()).To(Equal(limit))
	})

	It("Should close the segment files that reads of a key opened before a Drop "+
		"removes them", func(ctx SpecContext) {
		fs := newOpenFS(xfs.NewMem())
		db := mustOpenDBOnFS(ctx, fs, cesium.WithPointerLog())
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(1),
			telem.NewSeries([]int64{1}),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		fr := MustSucceed(db.Read(ctx, telem.SecondTS.Range(telem.SecondTS+1), data))
		Expect(fr.Get(data).Series).To(HaveLen(1))
		dir := strconv.FormatUint(uint64(data), 10)
		Expect(fs.openUnder(dir)).ToNot(BeEmpty())
		Expect(db.Drop(ctx, []cesium.ChannelKey{data})).To(Succeed())
		Expect(fs.openUnder(dir)).To(BeEmpty())
		Expect(fs.Exists(dir)).To(BeFalse())
	})
})
