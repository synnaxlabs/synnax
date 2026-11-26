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
	"testing"

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

// readGateFS counts the opens of each file of its tree, and holds the first ReadAt of a
// file under prefix, once armed, until release closes.
type readGateFS struct {
	xfs.FS
	dir string
	g   *readGate
}

type readGate struct {
	mu               sync.Mutex
	opens            map[string]int
	prefix           string
	armed            bool
	entered, release chan struct{}
}

func newReadGateFS(fs xfs.FS) *readGateFS {
	return &readGateFS{FS: fs, g: &readGate{
		opens:   map[string]int{},
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}}
}

func (f *readGateFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	full := path.Join(f.dir, name)
	f.g.mu.Lock()
	f.g.opens[full]++
	f.g.mu.Unlock()
	return &readGateFile{File: file, name: full, g: f.g}, nil
}

func (f *readGateFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &readGateFS{FS: sub, dir: path.Join(f.dir, name), g: f.g}, nil
}

// arm makes the next ReadAt of a file under prefix wait for release, and counts opens
// from zero.
func (f *readGateFS) arm(prefix string) {
	f.g.mu.Lock()
	defer f.g.mu.Unlock()
	f.g.prefix, f.g.armed = prefix, true
	clear(f.g.opens)
}

// segmentOpensUnder returns the opens of the segment files under prefix since arm.
func (f *readGateFS) segmentOpensUnder(prefix string) int {
	f.g.mu.Lock()
	defer f.g.mu.Unlock()
	n := 0
	for name, opens := range f.g.opens {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".seg") {
			n += opens
		}
	}
	return n
}

type readGateFile struct {
	xfs.File
	name string
	g    *readGate
}

func (f *readGateFile) ReadAt(p []byte, off int64) (int, error) {
	g := f.g
	g.mu.Lock()
	hold := g.armed && strings.HasPrefix(f.name, g.prefix)
	if hold {
		g.armed = false
	}
	g.mu.Unlock()
	if hold {
		close(g.entered)
		<-g.release
	}
	return f.File.ReadAt(p, off)
}

var _ = Describe("Reads on the pointer log", func() {
	It("Should give a read of a segment its own open of the file while another read "+
		"of it waits inside a read", func(ctx SpecContext) {
		fs := newReadGateFS(xfs.NewMem())
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
			Start:    10 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11, 12),
			telem.NewSeries([]int64{10, 11, 12}),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		dir := strconv.FormatUint(uint64(data), 10) + "/"
		fs.arm(dir)
		first := make(chan error, 1)
		go func() {
			_, err := db.Read(ctx, telem.TimeRangeMax, data)
			first <- err
		}()
		Eventually(fs.g.entered).Should(BeClosed())
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
		Expect(fr.Get(data).Series).To(HaveLen(1))
		close(fs.g.release)
		Eventually(first).Should(Receive(BeNil()))
		Expect(fs.segmentOpensUnder(dir)).To(Equal(2))
	})

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
		channels := make([]cesium.Channel, 1, 1+limit)
		channels[0] = cesium.Channel{
			Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		}
		keys := make([]cesium.ChannelKey, 1, 1+limit)
		keys[0] = idx
		series := make([]telem.Series, 1, 1+limit)
		series[0] = telem.NewSeriesSecondsTSV(1)
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

	It("Should read nothing at an automatic Next after a seek that finds no range, "+
		"once a Drop took the data of the key", func(ctx SpecContext) {
		db := mustOpenDBOnFS(ctx, xfs.NewMem(), cesium.WithPointerLog())
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
			Start:    10 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11),
			telem.NewSeries([]int64{10, 11}),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{data},
		}))
		DeferCleanup(func() { Expect(iter.Close()).To(Succeed()) })
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(db.Drop(ctx, []cesium.ChannelKey{data})).To(Succeed())
		Expect(iter.SeekGE(10 * telem.SecondTS)).To(BeFalse())
		iter.Next(cesium.AutoSpan)
		Expect(iter.Value().Get(data).Series).To(BeEmpty())
		Expect(iter.Error()).ToNot(HaveOccurred())
	})
	It("Should allocate no range reader at a seek and a step of an open read", func(
		ctx SpecContext,
	) {
		db := mustOpenDBOnFS(ctx, xfs.NewMem(), cesium.WithPointerLog())
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		for _, start := range []telem.TimeStamp{10 * telem.SecondTS, 20 * telem.SecondTS} {
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Start:    start,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesV(start, start+telem.SecondTS, start+2*telem.SecondTS),
				telem.NewSeriesV[int64](1, 2, 3),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
		}
		i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Channels: []cesium.ChannelKey{data},
			Bounds:   telem.TimeRangeMax,
		}))
		Expect(i.SeekFirst()).To(BeTrue())
		// A step allocates its frame, its series data, and its index lookups.
		allocs := testing.AllocsPerRun(100, func() {
			i.SeekGE(20 * telem.SecondTS)
			i.Next(3 * telem.Second)
		})
		series := i.Value().Get(data).Series
		Expect(series).To(HaveLen(1))
		Expect(series[0].Len()).To(Equal(int64(3)))
		Expect(allocs).To(BeNumerically("<=", 9))
		Expect(i.Close()).To(Succeed())
	})
})
