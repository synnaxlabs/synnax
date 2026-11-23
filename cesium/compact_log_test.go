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
	"context"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// failFS fails each write to a segment file that it creates while armed.
type failFS struct {
	xfs.FS
	armed atomic.Bool
}

func (f *failFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *failFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || flag&os.O_CREATE == 0 || !strings.HasSuffix(name, ".seg") ||
		!f.armed.Load() {
		return file, err
	}
	return failFile{File: file}, nil
}

type failFile struct{ xfs.File }

func (failFile) WriteAt([]byte, int64) (int, error) {
	return 0, errors.New("disk failed")
}

var _ = Describe("Compaction on the pointer log", func() {
	var (
		fs        *failFS
		clock     xtime.Clock
		gc        cesium.GCConfig
		db        *cesium.DB
		stop      func()
		logs      *observer.ObservedLogs
		idx, data cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		core, observed := observer.New(zapcore.DebugLevel)
		logs = observed
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithClock(clock),
			cesium.WithGCConfig(gc),
			cesium.WithInstrumentation(alamos.New("cesium", alamos.WithLogger(logger))),
		))
		stop = StartRun(db.Run)
	}
	reopen := func(ctx context.Context) {
		GinkgoHelper()
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
	}
	BeforeEach(func() {
		fs, clock, gc = &failFS{FS: xfs.NewMem()}, xtime.Real, cesium.GCConfig{}
	})
	JustBeforeEach(func(ctx SpecContext) {
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	// stamp returns the stamp of sample i of a key.
	stamp := func(i int) telem.TimeStamp {
		return telem.TimeStamp(i+1) * telem.SecondTS
	}
	// write writes s to key, and the stamps of samples from first on to the index.
	write := func(
		ctx context.Context,
		key cesium.ChannelKey,
		s telem.Series,
		first int,
	) {
		GinkgoHelper()
		stamps := make([]telem.TimeStamp, s.Len())
		for i := range stamps {
			stamps[i] = stamp(first + i)
		}
		keys := []cesium.ChannelKey{idx, key}
		w := MustSucceed(db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Clock: keys[0], Start: stamp(first)},
		))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeries(stamps), s},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	// segments returns the names and sizes of the segment files of key.
	segments := func(key cesium.ChannelKey) map[string]int64 {
		GinkgoHelper()
		out := make(map[string]int64)
		dir := strconv.FormatUint(uint64(key), 10)
		for _, e := range MustSucceed(fs.List(dir)) {
			if strings.HasSuffix(e.Name(), ".seg") {
				out[e.Name()] = e.Size()
			}
		}
		return out
	}
	files := func(key cesium.ChannelKey) func() map[string]int64 {
		return func() map[string]int64 { return segments(key) }
	}
	// written writes s to key in a segment that a later write of end seals, and returns
	// the name of its file.
	written := func(
		ctx context.Context,
		key cesium.ChannelKey,
		s, end telem.Series,
	) string {
		GinkgoHelper()
		write(ctx, key, s, 0)
		write(ctx, key, end, 1<<22)
		names := slices.Sorted(maps.Keys(segments(key)))
		Expect(names).To(HaveLen(2))
		return names[0]
	}
	// rewritten returns a function that reports whether the segment file old of key is
	// gone, and another took its place.
	rewritten := func(key cesium.ChannelKey, old string) func() bool {
		n := len(segments(key))
		return func() bool {
			files := segments(key)
			_, kept := files[old]
			return len(files) == n && !kept
		}
	}
	// count returns the integers from 0 to n.
	count := func(n int) []int64 {
		out := make([]int64, n)
		for i := range out {
			out[i] = int64(i)
		}
		return out
	}
	end := telem.NewSeries([]int64{-1})
	// samples returns the samples of key.
	samples := func(ctx context.Context, key cesium.ChannelKey) [][]byte {
		GinkgoHelper()
		var out [][]byte
		for _, s := range MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).
			Get(key).Series {
			for b := range s.Samples() {
				out = append(out, slices.Clone(b))
			}
		}
		return out
	}
	values := func(ctx context.Context) []int64 {
		GinkgoHelper()
		var out []int64
		for _, s := range MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).
			Get(data).Series {
			for i := range int(s.Len()) {
				out = append(out, s.ValueAt[int64](i))
			}
		}
		return out
	}
	// deleteRange deletes the samples from start to end of key.
	deleteRange := func(ctx context.Context, key cesium.ChannelKey, start, end int) {
		GinkgoHelper()
		tr := stamp(start).Range(stamp(end))
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{key}, tr)).To(Succeed())
	}
	messages := func(message string) func() int {
		return func() int { return logs.FilterMessage(message).Len() }
	}

	It("Should rewrite a segment that a delete leaves above the dead share with its "+
		"live samples", func(ctx SpecContext) {
		old := written(ctx, data, telem.NewSeries(count(10)), end)
		done := rewritten(data, old)
		deleteRange(ctx, data, 2, 6)
		Eventually(done).Should(BeTrue())
		Expect(segments(data)).To(ConsistOf(int64(6*8), int64(8)))
		want := []int64{0, 1, 6, 7, 8, 9, -1}
		Expect(values(ctx)).To(Equal(want))
		reopen(ctx)
		Expect(values(ctx)).To(Equal(want))
	})

	It("Should rewrite a segment that a delete cut while open once a write seals it",
		func(ctx SpecContext) {
			write(ctx, data, telem.NewSeries(count(10)), 0)
			deleteRange(ctx, data, 2, 6)
			names := slices.Sorted(maps.Keys(segments(data)))
			Expect(names).To(HaveLen(1))
			Consistently(files(data)).Should(HaveKey(names[0]))
			write(ctx, data, end, 1<<22)
			Eventually(files(data)).ShouldNot(HaveKey(names[0]))
			Expect(values(ctx)).To(Equal([]int64{0, 1, 6, 7, 8, 9, -1}))
		})

	It("Should copy a segment over many jobs, with a whole sample past a job",
		func(ctx SpecContext) {
			text := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: text, Name: "text", Index: idx, DataType: telem.StringT,
			})).To(Succeed())
			strs := make([]string, 400)
			for i := range strs {
				strs[i] = strings.Repeat(string(rune('a'+i%26)), 5000+i*7)
			}
			strs[350] = strings.Repeat("z", 3<<19)
			old := written(
				ctx,
				text,
				telem.NewSeriesV(strs...),
				telem.NewSeriesV("end"),
			)
			done := rewritten(text, old)
			deleteRange(ctx, text, 100, 300)
			Eventually(done).Should(BeTrue())
			kept := append(strs[:100:100], strs[300:]...)
			want := make([][]byte, 0, len(kept)+1)
			for _, s := range kept {
				want = append(want, []byte(s))
			}
			want = append(want, []byte("end"))
			Expect(samples(ctx, text)).To(Equal(want))
			reopen(ctx)
			Expect(samples(ctx, text)).To(Equal(want))
		})

	It("Should copy a fixed-density segment over many jobs", func(ctx SpecContext) {
		old := written(ctx, data, telem.NewSeries(count(300_000)), end)
		done := rewritten(data, old)
		deleteRange(ctx, data, 100_000, 200_000)
		Eventually(done).Should(BeTrue())
		want := append(count(100_000), count(300_000)[200_000:]...)
		Expect(values(ctx)).To(Equal(append(want, -1)))
	})

	It("Should keep a segment that fails a block check out of the rewrites until the "+
		"next open", func(ctx SpecContext) {
		name := written(ctx, data, telem.NewSeries(count(10)), end)
		f := MustSucceed(fs.Open(
			strconv.FormatUint(uint64(data), 10)+"/"+name,
			os.O_RDWR,
		))
		MustSucceed(f.WriteAt([]byte{0xFF}, 0))
		Expect(f.Close()).To(Succeed())
		failed := messages("segment failed a block check and is not rewritten")
		deleteRange(ctx, data, 2, 6)
		Eventually(failed).Should(Equal(1))
		Expect(segments(data)).To(HaveKey(name))
		deleteRange(ctx, data, 7, 8)
		Consistently(failed).Should(Equal(1))
		Expect(segments(data)).To(HaveKey(name))
		reopen(ctx)
		Eventually(failed).Should(Equal(1))
		Expect(segments(data)).To(SatisfyAll(HaveLen(2), HaveKey(name)))
	})

	It("Should remove the file of a rewrite that fails to write, and rewrite the "+
		"segment at a later signal", func(ctx SpecContext) {
		old := written(ctx, data, telem.NewSeries(count(10)), end)
		done := rewritten(data, old)
		fs.armed.Store(true)
		deleteRange(ctx, data, 2, 6)
		Eventually(messages("rewrite of a segment failed")).Should(Equal(1))
		Expect(segments(data)).To(SatisfyAll(HaveLen(2), HaveKey(old)))
		fs.armed.Store(false)
		deleteRange(ctx, data, 7, 8)
		Eventually(done).Should(BeTrue())
		Expect(values(ctx)).To(Equal([]int64{0, 1, 6, 8, 9, -1}))
	})

	It("Should keep the old file for an iterator that reads it, until the iterator "+
		"closes", func(ctx SpecContext) {
		old := written(ctx, data, telem.NewSeries(count(10)), end)
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{data},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		deleteRange(ctx, data, 2, 6)
		Eventually(files(data)).Should(HaveLen(3))
		Consistently(files(data)).Should(HaveKey(old))
		Expect(iter.Close()).To(Succeed())
		Expect(segments(data)).To(SatisfyAll(HaveLen(2), Not(HaveKey(old))))
	})

	Describe("Rate", func() {
		var fake *timetestutil.Clock
		BeforeEach(func() {
			fake = timetestutil.NewClock(time.Unix(0, 0))
			clock, gc.Rate = fake, 1<<20
		})
		// started writes 4 MiB of samples to data, then deletes the first quarter of
		// them, which starts a rewrite of three jobs of 1 MiB. It returns the name of
		// the old segment file once the first job waits for the clock.
		started := func(ctx context.Context) string {
			GinkgoHelper()
			old := written(ctx, data, telem.NewSeries(count(1<<19)), end)
			deleteRange(ctx, data, 0, 1<<17)
			Eventually(fake.Pending).Should(Equal(1))
			return old
		}
		// replaced reports whether data holds the new segment of a rewrite of old, and
		// the segment that seals old.
		replaced := func(old string) bool {
			files := segments(data)
			_, kept := files[old]
			return len(files) == 2 && !kept
		}

		It("Should read at most Rate bytes a second of the clock", func(
			ctx SpecContext,
		) {
			old := started(ctx)
			for range 3 {
				Eventually(fake.Pending).Should(Equal(1))
				Consistently(files(data)).Should(HaveLen(3))
				fake.Advance(time.Second)
			}
			Eventually(func() bool { return replaced(old) }).Should(BeTrue())
			Expect(values(ctx)).To(Equal(append(count(1 << 19)[1<<17:], -1)))
		})

		It("Should drop a copy that a delete makes stale, and rewrite the segment "+
			"again", func(ctx SpecContext) {
			old := started(ctx)
			deleteRange(ctx, data, 400_000, 400_010)
			Eventually(func() bool {
				if fake.Pending() > 0 {
					fake.Advance(time.Second)
				}
				return replaced(old)
			}).Should(BeTrue())
			Expect(messages("segment changed during its rewrite")()).To(Equal(1))
			want := count(1 << 19)[1<<17:]
			want = slices.Delete(want, 400_000-1<<17, 400_010-1<<17)
			Expect(values(ctx)).To(Equal(append(want, -1)))
		})

		It("Should drop a rewrite in progress at close, and keep the old segment",
			func(ctx SpecContext) {
				old := started(ctx)
				stop()
				Expect(db.Close()).To(Succeed())
				Expect(segments(data)).To(SatisfyAll(HaveLen(2), HaveKey(old)))
			})
	})
})
