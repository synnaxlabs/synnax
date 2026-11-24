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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/segment"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xerrors "github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// holdFS holds the write of a log frame that holds marker, until unhold.
type holdFS struct {
	xfs.FS
	mu      sync.Mutex
	marker  []byte
	waiting chan struct{}
	release chan struct{}
	// removeDir and removeErr fail each removal of a segment file under removeDir.
	removeDir string
	removeErr error
}

// failRemoves makes each later removal of a segment file under dir fail with err, or
// lets removals through when err is nil.
func (f *holdFS) failRemoves(dir string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeDir, f.removeErr = dir, err
}

func (f *holdFS) Remove(name string) error {
	f.mu.Lock()
	dir, err := f.removeDir, f.removeErr
	f.mu.Unlock()
	if err != nil && strings.HasPrefix(name, dir+"/") &&
		strings.HasSuffix(name, ".seg") {
		return err
	}
	return f.FS.Remove(name)
}

func (f *holdFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *holdFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".log") {
		return file, err
	}
	return &holdFile{File: file, fs: f}, nil
}

// hold makes the next log write that holds marker wait for unhold.
func (f *holdFS) hold(marker []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.marker = marker
	f.waiting, f.release = make(chan struct{}, 1), make(chan struct{})
}

// unhold lets a held write go on. It does nothing when no hold is set.
func (f *holdFS) unhold() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.marker != nil {
		f.marker = nil
		close(f.release)
	}
}

type holdFile struct {
	xfs.File
	fs *holdFS
}

func (f *holdFile) WriteAt(p []byte, off int64) (int, error) {
	f.fs.mu.Lock()
	marker, waiting, release := f.fs.marker, f.fs.waiting, f.fs.release
	f.fs.mu.Unlock()
	if marker != nil && bytes.Contains(p, marker) {
		waiting <- struct{}{}
		<-release
	}
	return f.File.WriteAt(p, off)
}

var _ = Describe("Delete on the pointer log", func() {
	var (
		fs *holdFS
		db *cesium.DB
		// id is the identity of the database in fs, from the open that made it.
		id        uint64
		stop      func()
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
		changes   []cesium.HistoryChange
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(id),
		))
		id = db.Identity()
		stop = StartRun(db.Run)
		db.OnHistoryChange(func(_ context.Context, c cesium.HistoryChange) {
			changes = append(changes, c)
		})
	}
	BeforeEach(func(ctx SpecContext) {
		fs, changes, id = &holdFS{FS: xfs.NewMem()}, nil, 0
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	frame := func(seconds ...telem.TimeStamp) cesium.Frame {
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s)
		}
		return telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		})
	}
	write := func(ctx context.Context, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    seconds[0] * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, frame(seconds...), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	values := func(ctx context.Context, key cesium.ChannelKey) []int64 {
		GinkgoHelper()
		var out []int64
		for _, s := range MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).
			Get(key).Series {
			for i := range int(s.Len()) {
				if key == idx {
					out = append(
						out,
						int64(s.ValueAt[telem.TimeStamp](i)/telem.SecondTS),
					)
				} else {
					out = append(out, s.ValueAt[int64](i))
				}
			}
		}
		return out
	}
	span := func(start, end telem.TimeStamp) telem.TimeRange {
		return (start * telem.SecondTS).Range(end * telem.SecondTS)
	}

	Describe("Reads of the keys on one clock", func() {
		var (
			more cesium.ChannelKey
			all  []cesium.ChannelKey
			iter interface {
				SeekFirst() bool
				SeekGE(stamp telem.TimeStamp) bool
				Next(span telem.TimeSpan) bool
				Value() cesium.Frame
				Close() error
			}
		)
		// writeAll writes the stamps from second from, ten of them, to idx, and the
		// seconds as values to data and more.
		writeAll := func(ctx context.Context, from int) {
			GinkgoHelper()
			seconds := make([]telem.TimeStamp, 10)
			ints := make([]int64, 10)
			for i := range seconds {
				seconds[i], ints[i] = telem.TimeStamp(from+i), int64(from+i)
			}
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: all,
				Clock:    idx,
				Start:    telem.TimeStamp(from) * telem.SecondTS,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(all, []telem.Series{
				telem.NewSeriesSecondsTSV(seconds...),
				telem.NewSeries(ints),
				telem.NewSeries(ints),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
		}
		BeforeEach(func(ctx SpecContext) {
			more = GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: more, Name: "more", Index: idx, DataType: telem.Int64T,
			})).To(Succeed())
			all = []cesium.ChannelKey{idx, data, more}
			writeAll(ctx, 10)
			iter = MustSucceed(db.OpenIterator(cesium.IteratorConfig{
				Bounds:   telem.TimeRangeMax,
				Channels: []cesium.ChannelKey{data, more},
			}))
			DeferCleanup(func() { Expect(iter.Close()).To(Succeed()) })
		})
		// step returns the values of each key in the next span. Each value is the
		// second of its stamp, so a key whose samples and stamps came from two states
		// returns values outside the span.
		step := func(span telem.TimeSpan) map[cesium.ChannelKey][]int64 {
			GinkgoHelper()
			Expect(iter.Next(span)).To(BeTrue())
			out := make(map[cesium.ChannelKey][]int64)
			for _, key := range []cesium.ChannelKey{data, more} {
				for _, se := range iter.Value().Get(key).Series {
					for i := range int(se.Len()) {
						out[key] = append(out[key], se.ValueAt[int64](i))
					}
				}
			}
			return out
		}
		both := func(values ...int64) map[cesium.ChannelKey][]int64 {
			return map[cesium.ChannelKey][]int64{data: values, more: values}
		}

		It("Should read each key from the state of its last step when a delete takes "+
			"stamps of the clock between two steps", func() {
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(step(3 * telem.Second)).To(Equal(both(10, 11, 12)))
			Expect(db.DeleteTimeRange(context.Background(), all, span(13, 16))).
				To(Succeed())
			Expect(step(7 * telem.Second)).To(Equal(both(13, 14, 15, 16, 17, 18, 19)))
		})

		It("Should read the samples that a write adds between two steps at a seek of "+
			"the second", func(ctx SpecContext) {
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(step(3 * telem.Second)).To(Equal(both(10, 11, 12)))
			writeAll(ctx, 20)
			Expect(iter.SeekGE(20 * telem.SecondTS)).To(BeTrue())
			Expect(step(5 * telem.Second)).To(Equal(both(20, 21, 22, 23, 24)))
		})
	})

	It("Should delete a range from the middle of the data of a key", func(
		ctx SpecContext,
	) {
		write(ctx, 10, 11, 12, 13, 14, 15, 16)
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, span(12, 15))).
			To(Succeed())
		Expect(values(ctx, data)).To(Equal([]int64{10, 11, 15, 16}))
		Expect(values(ctx, idx)).To(Equal([]int64{10, 11, 12, 13, 14, 15, 16}))
		Expect(changes).To(Equal([]cesium.HistoryChange{
			{Key: data, TimeRange: span(12, 15)},
		}))
	})

	It("Should delete an index with its data, and keep the deletes across a reopen",
		func(ctx SpecContext) {
			write(ctx, 10, 11, 12, 13, 14)
			Expect(db.DeleteTimeRange(ctx, keys, span(11, 13))).To(Succeed())
			Expect(changes).To(ConsistOf(
				cesium.HistoryChange{Key: data, TimeRange: span(11, 13)},
				cesium.HistoryChange{Key: idx, TimeRange: span(11, 13)},
			))
			stop()
			Expect(db.Close()).To(Succeed())
			open(ctx)
			Expect(values(ctx, data)).To(Equal([]int64{10, 13, 14}))
			Expect(values(ctx, idx)).To(Equal([]int64{10, 13, 14}))
		})

	It("Should refuse a delete of an index range that a data key outside it uses",
		func(ctx SpecContext) {
			write(ctx, 10, 11, 12)
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{idx}, span(11, 12))).
				To(SatisfyAll(
					MatchError(ContainSubstring("refused by the state")),
					MatchError(
						ContainSubstring("key "+strconv.Itoa(int(data))+" uses"),
					),
				))
			Expect(values(ctx, idx)).To(Equal([]int64{10, 11, 12}))
			Expect(changes).To(BeEmpty())
		})

	It("Should change nothing for a range with no data", func(ctx SpecContext) {
		write(ctx, 10, 11)
		Expect(db.DeleteTimeRange(ctx, keys, span(20, 30))).To(Succeed())
		Expect(values(ctx, data)).To(Equal([]int64{10, 11}))
		Expect(changes).To(BeEmpty())
	})

	It("Should cut a key of variable density at its sample offsets", func(
		ctx SpecContext,
	) {
		text := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: text, Name: "text", Index: idx, DataType: telem.StringT,
		})).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, text},
			Clock:    idx,
			Start:    10 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{idx, text},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12, 13),
				telem.NewSeriesV("a", "bb", "ccc", "dddd"),
			},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{text}, span(11, 13))).
			To(Succeed())
		series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, text)).Get(text).Series
		Expect(series).To(HaveLen(2))
		Expect(series[0]).To(MatchSeriesDataV("a"))
		Expect(series[1]).To(MatchSeriesDataV("dddd"))
	})

	DescribeTable(
		"Should cut a pointer of several blocks of stamps on a block edge and "+
			"inside a block",
		func(ctx SpecContext, start, end int) {
			// perBlock is the number of stamps in a block of a clock segment.
			const perBlock = segment.BlockSize / 8
			seconds := make([]telem.TimeStamp, 3*perBlock+1000)
			for i := range seconds {
				seconds[i] = telem.TimeStamp(i + 1)
			}
			// The series of write scales its stamps in place.
			write(ctx, slices.Clone(seconds)...)
			Expect(db.DeleteTimeRange(ctx, keys, span(seconds[start], seconds[end]))).
				To(Succeed())
			want := make([]int64, 0, len(seconds))
			for i, s := range seconds {
				if i < start || i >= end {
					want = append(want, int64(s))
				}
			}
			Expect(values(ctx, data)).To(Equal(want))
			Expect(values(ctx, idx)).To(Equal(want))
		},
		Entry("left on an edge, right inside", 2*segment.BlockSize/8,
			2*segment.BlockSize/8+100),
		Entry("left inside, right on an edge", 8000, segment.BlockSize/8),
		Entry("in blocks far apart", 100, 3*segment.BlockSize/8+5),
	)

	DescribeTable("Should cut a key of variable density over several blocks on a "+
		"block edge and inside a block", func(ctx SpecContext, start, end int) {
		// Each sample is a 4 byte length and 60 bytes, so a block holds perBlock whole
		// samples, and sample perBlock * m starts block m.
		const perBlock = segment.BlockSize / 64
		text := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: text, Name: "text", Index: idx, DataType: telem.StringT,
		})).To(Succeed())
		n := 3*perBlock + 100
		stamps := make([]telem.TimeStamp, n)
		strs := make([]string, n)
		for i := range n {
			stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
			strs[i] = fmt.Sprintf("%060d", i)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, text},
			Clock:    idx,
			Start:    stamps[0],
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{idx, text},
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(strs)},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		// The first delete leaves a pointer that starts at sample 10, past offset 0.
		for _, tr := range []telem.TimeRange{
			stamps[0].Range(stamps[10]),
			stamps[start].Range(stamps[end]),
		} {
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{text}, tr)).
				To(Succeed())
		}
		var got []string
		for _, series := range MustSucceed(db.Read(ctx, telem.TimeRangeMax, text)).
			Get(text).Series {
			for i := range int(series.Len()) {
				got = append(got, string(series.At(i)))
			}
		}
		Expect(got).To(Equal(slices.Concat(strs[10:start], strs[end:])))
	},
		Entry("left on an edge, right inside", 2*segment.BlockSize/64,
			2*segment.BlockSize/64+10),
		Entry("left inside, right on an edge", 1000, segment.BlockSize/64),
		Entry("in blocks far apart", 17, 3*segment.BlockSize/64+50),
	)

	It("Should unlink a segment that a delete leaves empty once no reader pins it",
		func(ctx SpecContext) {
			write(ctx, 10, 11)
			write(ctx, 20, 21)
			dir := strconv.FormatUint(uint64(data), 10)
			segments := func() int {
				n := 0
				for _, e := range MustSucceed(fs.List(dir)) {
					if strings.HasSuffix(e.Name(), ".seg") {
						n++
					}
				}
				return n
			}
			Expect(segments()).To(Equal(2))
			iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
				Bounds:   telem.TimeRangeMax,
				Channels: []cesium.ChannelKey{data},
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(db.DeleteTimeRange(ctx, keys, span(0, 15))).To(Succeed())
			Expect(segments()).To(Equal(2), "the iterator pins the empty segment")
			Expect(iter.Close()).To(Succeed())
			Expect(segments()).To(Equal(1))
			Expect(values(ctx, data)).To(Equal([]int64{20, 21}))
		})

	It(
		"Should log a failed removal at the last unpin of a dropped key, keep its "+
			"error out of every read, and leave the file to the next open",
		func(ctx SpecContext) {
			write(ctx, 10, 11)
			stop()
			Expect(db.Close()).To(Succeed())
			core, logs := observer.New(zapcore.ErrorLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			db = MustOpen(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithInstrumentation(
					alamos.New("test", alamos.WithLogger(logger)),
				),
			))
			stop = StartRun(db.Run)
			iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
				Bounds:   telem.TimeRangeMax,
				Channels: []cesium.ChannelKey{data},
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			dir := strconv.FormatUint(uint64(data), 10)
			fs.failRemoves(dir, xerrors.New("remove failed"))
			Expect(db.Drop(ctx, []cesium.ChannelKey{data})).To(Succeed())
			// A seek loads the view again, which unpins the segments of the dropped
			// key.
			iter.SeekFirst()
			Expect(iter.Error()).ToNot(HaveOccurred())
			Expect(iter.Close()).To(Succeed())
			const message = "Failed to remove the file of a dead segment"
			entries := logs.FilterMessage(message).All()
			Expect(entries).ToNot(BeEmpty())
			fields := entries[0].ContextMap()
			Expect(fields).To(HaveKeyWithValue("key", data))
			Expect(fields["file"]).To(HavePrefix(dir + "/"))
			Expect(fields["error"]).To(ContainSubstring("remove failed"))
			Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx)).Get(idx).Len()).
				To(Equal(int64(2)))
			Expect(MustSucceed(fs.List(dir))).ToNot(BeEmpty())
			fs.failRemoves("", nil)
			stop()
			Expect(db.Close()).To(Succeed())
			open(ctx)
			Eventually(func() (int, error) {
				infos, err := fs.List(dir)
				return len(infos), err
			}).Should(BeZero())
		},
	)

	It("Should finish a read that pinned the segments of a key across its Drop",
		func(ctx SpecContext) {
			write(ctx, 10, 11)
			write(ctx, 20, 21)
			iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
				Bounds:   telem.TimeRangeMax,
				Channels: []cesium.ChannelKey{data},
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(db.Drop(ctx, []cesium.ChannelKey{data})).To(Succeed())
			var got []int64
			for iter.Next(cesium.AutoSpan) {
				for _, s := range iter.Value().Get(data).Series {
					for i := range int(s.Len()) {
						got = append(got, s.ValueAt[int64](i))
					}
				}
			}
			Expect(iter.Error()).ToNot(HaveOccurred())
			Expect(got).To(Equal([]int64{10, 11, 20, 21}))
			Expect(iter.Close()).To(Succeed())
			dir := strconv.FormatUint(uint64(data), 10)
			Expect(fs.List(dir)).To(BeEmpty(), "the last reader unlinks the segments")
		})

	It(
		"Should refuse a backfill that straddles an end of a delete in flight, and stay "+
			"healthy",
		func(ctx SpecContext) {
			write(ctx, 10, 11, 12, 13)
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Clock:    idx,
				Start:    30 * telem.SecondTS,
			}))
			tr := span(12, 15)
			fs.hold(binary.LittleEndian.AppendUint64(
				binary.LittleEndian.AppendUint64(nil, uint64(tr.Start)),
				uint64(tr.End),
			))
			DeferCleanup(fs.unhold)
			deleted := make(chan error, 1)
			go func() {
				deleted <- db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, tr)
			}()
			Eventually(fs.waiting).Should(Receive())
			MustSucceed2(w.Write(ctx, frame(14, 15, 16), nil))
			Expect(w.Commit(ctx, nil)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("reaches the claim of a delete")),
			))
			fs.unhold()
			Eventually(deleted).Should(Receive(BeNil()))
			Expect(w.Close()).To(MatchError(ContainSubstring("claim of a delete")))
			write(ctx, 31, 32)
			Expect(values(ctx, data)).To(Equal([]int64{10, 11, 31, 32}))
		},
	)

	It("Should refuse a delete of a missing channel", func(ctx SpecContext) {
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{9_999}, span(0, 1))).
			To(MatchError(cesium.ErrChannelNotFound))
	})
})
