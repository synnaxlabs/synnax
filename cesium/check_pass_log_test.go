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
	"maps"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// flipFS changes the bytes that a read of an old file returns while it is armed: the
// first read at each offset, or, when never is true, every read, each one in another
// way, so that no two reads agree. With only set, it changes only the reads of that
// file. The bytes on the disk do not change.
type flipFS struct {
	xfs.FS
	mu    sync.Mutex
	armed bool
	never bool
	only  string
	reads map[string]int
}

func (f *flipFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *flipFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".domain") {
		return file, err
	}
	return flipFile{File: file, fs: f, name: name}, nil
}

func (f *flipFS) arm(never bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed, f.never, f.reads = true, never, make(map[string]int)
}

func (f *flipFS) disarm() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed = false
}

// flip returns the mask of the first byte of a read at off of name.
func (f *flipFS) flip(name string, off int64) byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.armed || (f.only != "" && name != f.only) {
		return 0
	}
	at := name + "@" + strconv.FormatInt(off, 10)
	f.reads[at]++
	switch n := f.reads[at]; {
	case f.never:
		return byte(n)
	case n == 1:
		return 0xFF
	}
	return 0
}

type flipFile struct {
	xfs.File
	fs   *flipFS
	name string
}

func (f flipFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	if n > 0 {
		p[0] ^= f.fs.flip(f.name, off)
	}
	return n, err
}

// readsFS counts the reads of the old files on it.
type readsFS struct {
	xfs.FS
	n atomic.Int64
}

func (f *readsFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *readsFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".domain") {
		return file, err
	}
	return readsFile{File: file, fs: f}, nil
}

type readsFile struct {
	xfs.File
	fs *readsFS
}

func (f readsFile) ReadAt(p []byte, off int64) (int, error) {
	f.fs.n.Add(1)
	return f.File.ReadAt(p, off)
}

// passFS holds the first call of op on a name with suffix, once armed, until release
// closes. op is "open", "close" for the Close of an old file or a check file, or
// "syncdir". It counts the opens of check files.
type passFS struct {
	xfs.FS
	mu               sync.Mutex
	op, suffix       string
	waiting, release chan struct{}
	checks           atomic.Int64
}

func (f *passFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *passFS) arm(op, suffix string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.op, f.suffix = op, suffix
	f.waiting, f.release = make(chan struct{}, 1), make(chan struct{})
}

func (f *passFS) hold(op, name string) {
	f.mu.Lock()
	held := f.op == op && strings.HasSuffix(name, f.suffix)
	if held {
		f.op = ""
	}
	waiting, release := f.waiting, f.release
	f.mu.Unlock()
	if held {
		waiting <- struct{}{}
		<-release
	}
}

func (f *passFS) Open(name string, flag int) (xfs.File, error) {
	checks := strings.HasSuffix(name, ".chk")
	if checks {
		f.checks.Add(1)
	}
	f.hold("open", name)
	file, err := f.FS.Open(name, flag)
	if err != nil || !checks && !strings.HasSuffix(name, ".domain") {
		return file, err
	}
	return passFile{File: file, fs: f, name: name}, nil
}

func (f *passFS) SyncDir(name string) error {
	f.hold("syncdir", name)
	return f.FS.SyncDir(name)
}

type passFile struct {
	xfs.File
	fs   *passFS
	name string
}

func (f passFile) Close() error {
	f.fs.hold("close", f.name)
	return f.File.Close()
}

// writeStamps writes n rising stamps of an index key on the old layout to a new fault
// FS, and returns it.
func writeStamps(ctx context.Context, key cesium.ChannelKey, n int) *fault.FS {
	GinkgoHelper()
	disk := MustSucceed(fault.New(fault.Config{}))
	db := openDBOnFS(ctx, disk)
	Expect(db.CreateChannel(ctx, cesium.Channel{
		Key:      key,
		Name:     "time",
		IsIndex:  true,
		DataType: telem.TimestampT,
	})).To(Succeed())
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
	}
	Expect(write(ctx, db, stamps[0], telem.MultiFrame(
		[]cesium.ChannelKey{key},
		[]telem.Series{telem.NewSeries(stamps)},
	))).To(Succeed())
	Expect(db.Close()).To(Succeed())
	disk.Writeback()
	return disk
}

// checkFiles returns the number of check files in the directory of each of keys on
// fs. A missing directory holds none.
func checkFiles(
	fs xfs.FS,
	keys ...cesium.ChannelKey,
) (map[cesium.ChannelKey]int, error) {
	out := make(map[cesium.ChannelKey]int, len(keys))
	for _, key := range keys {
		out[key] = 0
		dir := strconv.FormatUint(uint64(key), 10)
		ok, err := fs.Exists(dir)
		if err != nil || !ok {
			if err != nil {
				return nil, err
			}
			continue
		}
		infos, err := fs.List(dir)
		if err != nil {
			return nil, err
		}
		for _, info := range infos {
			if strings.HasSuffix(info.Name(), ".chk") {
				out[key]++
			}
		}
	}
	return out, nil
}

// oldSegments returns the version 0 segments of each key in the head of generation on
// fs.
func oldSegments(fs xfs.FS, generation uint64) map[cesium.ChannelKey][]state.Segment {
	GinkgoHelper()
	_, s := MustSucceed2(checkpoint.Read(fs, generation))
	out := make(map[cesium.ChannelKey][]state.Segment)
	for _, ch := range s.Channels {
		for _, seg := range ch.Segments {
			if seg.Version == 0 {
				out[ch.Key] = append(out[ch.Key], seg)
			}
		}
	}
	return out
}

// startConverted opens fs on the pointer log as openConverted does, with extra, calls
// opened once the open, and with it the conversion, is done, and then runs the DB until
// the returned close.
func startConverted(
	ctx context.Context,
	fs xfs.FS,
	ins alamos.Instrumentation,
	opened func(),
	extra ...cesium.Option,
) (*cesium.DB, func() error) {
	GinkgoHelper()
	db := MustSucceed(cesium.Open(ctx, "", append([]cesium.Option{
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithRandom(rand.NewPCG(1, 2)),
		cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
		cesium.WithInstrumentation(ins),
	}, extra...)...))
	opened()
	stop := GoRun(ctx, db.Run)
	return db, func() error { return errors.Join(stop(), db.Close()) }
}

// without returns the bytes of each key of o without its samples from from to below to.
func (o oldSamples) without(from, to int) map[cesium.ChannelKey][]byte {
	return map[cesium.ChannelKey][]byte{
		oldIdx: telem.NewSeries(slices.Concat(o.stamps[:from], o.stamps[to:])).Data,
		oldI32: telem.NewSeries(slices.Concat(o.ints[:from], o.ints[to:])).Data,
		oldStr: telem.NewSeries(slices.Concat(o.strs[:from], o.strs[to:])).Data,
	}
}

var _ = Describe("Check pass on the pointer log", func() {
	const (
		idx = oldIdx
		i32 = oldI32
		str = oldStr
	)
	var (
		keys = []cesium.ChannelKey{idx, i32, str}
		disk *fault.FS
		old  oldSamples
		logs *observer.ObservedLogs
		ins  alamos.Instrumentation
	)
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		old = writeOld(ctx, disk)
		core, observed := observer.New(zapcore.DebugLevel)
		logs = observed
		ins = alamos.New("cesium", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
	})
	// segmentsOf returns the number of version 0 segments of each key: one for each old
	// data file that writeOld wrote with data in it, since the conversion makes one
	// segment of each. Heads cannot count them once parts hold their records.
	segmentsOf := func() map[cesium.ChannelKey]int {
		GinkgoHelper()
		out := make(map[cesium.ChannelKey]int, len(keys))
		for _, key := range keys {
			dir := strconv.FormatUint(uint64(key), 10)
			for _, info := range MustSucceed(disk.List(dir)) {
				name := info.Name()
				if strings.HasSuffix(name, ".domain") && name != "index.domain" &&
					name != "counter.domain" && info.Size() > 0 {
					out[key]++
				}
			}
		}
		return out
	}
	// checked waits until the pass gave each version 0 segment of head 1 a check file,
	// then switches the log of db to 2, and returns the version 0 segments of head 2.
	checked := func(db *cesium.DB) map[cesium.ChannelKey][]state.Segment {
		GinkgoHelper()
		want := segmentsOf()
		Eventually(func() (map[cesium.ChannelKey]int, error) {
			return checkFiles(disk, keys...)
		}).Should(Equal(want))
		Expect(switchLog(db, disk, 2)).To(Succeed())
		waitHead(disk, 2)
		return oldSegments(disk, 2)
	}
	none := map[cesium.ChannelKey]int{idx: 0, i32: 0, str: 0}
	// messages returns the number of logs of message.
	messages := func(message string) func() int {
		return func() int { return logs.FilterMessage(message).Len() }
	}
	allChecked := func(segs map[cesium.ChannelKey][]state.Segment) {
		GinkgoHelper()
		for key, ss := range segs {
			for _, s := range ss {
				Expect(s.Checked).To(BeTrue(), "segment %d of key %d", s.ID, key)
			}
		}
	}

	It("Should give each converted segment its check file and its checks, and read "+
		"each sample as before", func(ctx SpecContext) {
		db, closeDB := openConverted(ctx, disk, ins)
		allChecked(checked(db))
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(closeDB()).To(Succeed())
		// The reopen reads checked, and the pass finds no target.
		db, closeDB = openConverted(ctx, disk, ins)
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(closeDB()).To(Succeed())
		Expect(checkFiles(disk, keys...)).To(Equal(segmentsOf()))
	})

	It("Should check each block whose second read is right", func(ctx SpecContext) {
		flip := &flipFS{FS: disk}
		db, closeDB := startConverted(ctx, flip, ins, func() { flip.arm(false) })
		allChecked(checked(db))
		flip.disarm()
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(closeDB()).To(Succeed())
	})

	It("Should leave a segment unchecked when no two reads of a block agree, and log "+
		"it once", func(ctx SpecContext) {
		flip := &flipFS{FS: disk}
		db, closeDB := startConverted(ctx, flip, ins, func() { flip.arm(true) })
		total := 0
		for _, n := range segmentsOf() {
			total += n
		}
		failed := messages("segment failed its block reads and stays unchecked")
		Eventually(failed).Should(Equal(total))
		Consistently(failed).Should(Equal(total))
		Expect(checkFiles(disk, keys...)).To(Equal(none))
		flip.disarm()
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(closeDB()).To(Succeed())
	})

	It("Should check a segment at the next open when a stray check file shares its "+
		"directory", func(ctx SpecContext) {
		flip := &flipFS{FS: disk, only: oldName(i32, oldPointers(disk, i32)[0])}
		_, closeDB := startConverted(ctx, flip, ins, func() { flip.arm(true) })
		failed := messages("segment failed its block reads and stays unchecked")
		Eventually(failed).Should(Equal(1))
		want := segmentsOf()
		left := maps.Clone(want)
		left[i32]--
		Eventually(func() (map[cesium.ChannelKey]int, error) {
			return checkFiles(disk, keys...)
		}).Should(Equal(left))
		Expect(closeDB()).To(Succeed())
		// The stray file makes as many check files as old files in the directory.
		stray := strconv.FormatUint(uint64(i32), 10) + "/00000000000000ff.chk"
		Expect(xfs.WriteFile(disk, stray, make([]byte, 8))).To(Succeed())
		db, closeDB := openConverted(ctx, disk, ins)
		Eventually(func() (map[cesium.ChannelKey]int, error) {
			return checkFiles(disk, keys...)
		}).Should(Equal(want))
		Expect(MustSucceed(disk.Exists(stray))).To(BeFalse())
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(closeDB()).To(Succeed())
	})

	const riseCut = "cut the samples of a converted segment whose stamps do not rise"
	// write puts stamp at sample i of the old index pointer p.
	write := func(p oldPointer, i int, stamp telem.TimeStamp) {
		GinkgoHelper()
		f := MustSucceed(disk.Open(oldName(idx, p), os.O_RDWR))
		b := binary.LittleEndian.AppendUint64(nil, uint64(stamp))
		MustSucceed(f.WriteAt(b, p.offset+8*int64(i)))
		Expect(f.Close()).To(Succeed())
	}
	DescribeTable(
		"Should cut the samples of an older index pointer whose stamps do not "+
			"rise, alone and with the samples that they time, and warn of the cut",
		func(ctx SpecContext, run func(n int) (from, to int), repeat bool) {
			p := oldPointers(disk, idx)[0]
			n := int(p.size / 8)
			from, to := run(n)
			Expect(from).To(BeNumerically(">", 0))
			for i := from; i < to; i++ {
				var stamp telem.TimeStamp
				if repeat {
					stamp = old.stamps[from-1]
				}
				write(p, i, stamp)
			}
			db, closeDB := openConverted(ctx, disk, ins)
			head := checked(db)
			Expect(dataOf(ctx, db, keys)).To(Equal(old.without(from, to)))
			for _, key := range keys {
				var kept uint32
				for _, s := range head[key] {
					for _, ptr := range s.Pointers {
						if ptr.First >= p.start && ptr.Last <= old.stamps[n-1] {
							kept += ptr.Samples
						}
					}
				}
				Expect(kept).To(Equal(uint32(n-to+from)), "key %d", key)
			}
			cuts := logs.FilterMessage(riseCut).All()
			Expect(cuts).To(HaveLen(1))
			fields := cuts[0].ContextMap()
			Expect(fields).To(SatisfyAll(
				HaveKeyWithValue("key", uint32(idx)),
				HaveKeyWithValue("keys", ConsistOf(uint32(i32), uint32(str))),
				HaveKeyWithValue("after", old.stamps[from-1].String()),
				HaveKeyWithValue("samples", uint32(to-from)),
			))
			if to < n {
				Expect(fields).To(HaveKeyWithValue("before", old.stamps[to].String()))
			} else {
				Expect(fields).ToNot(HaveKey("before"))
			}
			Expect(closeDB()).To(Succeed())
		},
		Entry("a zero", func(int) (int, int) { return 2, 3 }, false),
		Entry("two zeros", func(int) (int, int) { return 2, 4 }, false),
		Entry(
			"a repeat of the stamp before it",
			func(int) (int, int) { return 3, 4 },
			true,
		),
		Entry(
			"a repeat at the end of the pointer",
			func(n int) (int, int) { return n - 1, n },
			true,
		),
	)

	It("Should read rising stamps, none of them zero, and each sample at its own "+
		"stamp, from each crash image of a write of the old layout", func(
		ctx SpecContext,
	) {
		rounds := messages("finished a round of the check pass")
		for run := range uint64(32) {
			written := MustSucceed(fault.New(fault.Config{}))
			writeOldLost(ctx, written)
			image := MustSucceed(written.Crash(fault.CrashConfig{
				RunKey:            run,
				EntrySurvivalRate: 0.5,
				DataSurvivalRate:  0.5,
			}))
			db, closeDB := openConverted(ctx, image, ins)
			Eventually(rounds).Should(BeNumerically(">", int(run)), "image %d", run)
			// An image can lose a channel with all of its samples.
			var present []cesium.ChannelKey
			for _, key := range keys {
				if _, err := db.RetrieveChannel(ctx, key); err == nil {
					present = append(present, key)
				}
			}
			if !slices.Contains(present, idx) {
				Expect(closeDB()).To(Succeed())
				continue
			}
			// Each key reads alone, since a read of several keys refuses a data key
			// with no samples.
			read := func(key cesium.ChannelKey) []telem.Series {
				GinkgoHelper()
				if !slices.Contains(present, key) {
					return nil
				}
				return MustSucceed(
					db.Read(ctx, telem.TimeRangeMax, key),
				).Get(key).
					Series
			}
			stampAt := make(map[telem.Alignment]telem.TimeStamp)
			var prev telem.TimeStamp
			for _, series := range read(idx) {
				for j := range int(series.Len()) {
					stamp := series.ValueAt[telem.TimeStamp](j)
					Expect(stamp).To(BeNumerically(">", prev), "image %d", run)
					prev = stamp
					stampAt[series.Alignment.AddSamples(uint32(j))] = stamp
				}
			}
			// writeOld gives sample i the stamp i + 1 seconds, the int32 i + 1, and a
			// string of i % 7 + 1 bytes.
			for _, series := range read(i32) {
				for j := range int(series.Len()) {
					v := telem.TimeStamp(series.ValueAt[int32](j)) * telem.SecondTS
					Expect(stampAt).To(HaveKeyWithValue(
						series.Alignment.AddSamples(uint32(j)),
						v,
					), "image %d", run)
				}
			}
			for _, series := range read(str) {
				for j := range int(series.Len()) {
					stamp, ok := stampAt[series.Alignment.AddSamples(uint32(j))]
					Expect(ok).To(BeTrue(), "image %d", run)
					i := int(stamp/telem.SecondTS) - 1
					Expect(series.At(j)).To(HaveLen(i%7+1), "image %d", run)
				}
			}
			Expect(closeDB()).To(Succeed())
		}
	})

	It("Should finish the pass over parted segments after a crash at each of its "+
		"durable events, and keep the cuts of the rise check", func(ctx SpecContext) {
		// A zero inside the oldest index pointer gives the pass two entries: the
		// Tombstones of the rise check, then the Check.
		p := oldPointers(disk, idx)[0]
		zero(disk, oldName(idx, p), p.offset+16, 8)
		parted := cesium.WithPartSize(1)
		// The first open converts and moves every record to the parts of head 1, so the
		// reopen below holds the version 0 segments only in parts. It never runs, so
		// the
		// pass starts at the reopen.
		Expect(MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(disk),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
			cesium.WithInstrumentation(ins),
			parted,
		)).Close()).To(Succeed())
		want := old.without(2, int(p.size/8))
		segs := segmentsOf()
		// finished waits up to 30 seconds for the pass on fs to give each version 0
		// segment its check file. It runs inside an event of the DB, so it returns an
		// error instead of failing the spec.
		finished := func(fs xfs.FS) error {
			for deadline := time.Now().Add(30 * time.Second); ; {
				got, err := checkFiles(fs, keys...)
				if err != nil || maps.Equal(got, segs) {
					return err
				}
				if time.Now().After(deadline) {
					return errors.Newf("the pass did not finish: %v", got)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		// Each image replays the entries of the pass that it kept over a head whose
		// parts hold their segments.
		c := &crashes{disk: disk, check: func(image *fault.FS) error {
			db, closeDB, err := openSeeded(ctx, image)
			if err != nil {
				return err
			}
			err = finished(image)
			got, readErr := dataOf(ctx, db, keys)
			if err = errors.Join(err, readErr, closeDB()); err != nil {
				return err
			}
			if !maps.EqualFunc(got, want, bytes.Equal) {
				return errors.New("the read differs from the cut samples")
			}
			return nil
		}}
		db, closeDB := startConverted(
			ctx,
			durableFS{FS: disk, onEvent: c.take},
			ins,
			func() {
				c.mu.Lock()
				c.taking = true
				c.mu.Unlock()
			},
			parted,
		)
		// Parts hold the records, so the check files show the end of the pass. Each
		// event
		// waits for the check of its image.
		Eventually(func() (map[cesium.ChannelKey]int, error) {
			return checkFiles(disk, keys...)
		}).WithTimeout(5 * time.Minute).Should(Equal(segs))
		c.mu.Lock()
		c.taking = false
		c.mu.Unlock()
		allChecked(checked(db))
		Expect(dataOf(ctx, db, keys)).To(Equal(want))
		Expect(closeDB()).To(Succeed())
		Expect(c.images).To(BeNumerically(">", 3))
		Expect(c.failures).To(BeEmpty())
	})

	DescribeTable("Should rewrite a checked old file that a delete leaves mostly dead "+
		"only when version 0 is rewritable",
		func(ctx SpecContext, extra []cesium.Option, rewritten bool) {
			db, closeDB := startConverted(ctx, disk, ins, func() {}, extra...)
			allChecked(checked(db))
			// The first old file of i32 holds samples 1 to 15.
			oldFile := oldName(i32, oldPointers(disk, i32)[0])
			Expect(oldFile).To(Equal("2/1.domain"))
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{i32}, telem.TimeRange{
				Start: old.stamps[0],
				End:   old.stamps[12],
			})).To(Succeed())
			exists := func() bool { return MustSucceed(disk.Exists(oldFile)) }
			if rewritten {
				Eventually(exists).Should(BeFalse())
				Eventually(func() (map[cesium.ChannelKey]int, error) {
					return checkFiles(disk, i32)
				}).Should(Equal(map[cesium.ChannelKey]int{i32: segmentsOf()[i32]}))
			} else {
				Consistently(exists).Should(BeTrue())
			}
			want := old.bytes(nil)
			want[i32] = telem.NewSeries(old.ints[12:]).Data
			Expect(dataOf(ctx, db, keys)).To(Equal(want))
			Expect(closeDB()).To(Succeed())
		},
		Entry("rewritable", []cesium.Option{cesium.WithVersion0Rewritable()}, true),
		Entry("not rewritable", nil, false),
	)

	It("Should keep the check file of a segment that a reader pins across a Drop of "+
		"its key", func(ctx SpecContext) {
		db, closeDB := openConverted(ctx, disk, ins)
		allChecked(checked(db))
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{i32},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(db.Drop(ctx, []cesium.ChannelKey{i32})).To(Succeed())
		var got []byte
		for iter.Next(cesium.AutoSpan) {
			for _, s := range iter.Value().Get(i32).Series {
				got = append(got, s.Data...)
			}
		}
		Expect(iter.Error()).To(Succeed())
		Expect(got).To(Equal(old.bytes(nil)[i32]))
		Expect(iter.Close()).To(Succeed())
		Expect(closeDB()).To(Succeed())
	})

	It("Should drop the check of a segment that a delete buries during its pass",
		func(ctx SpecContext) {
			gate := &gateFS{FS: disk}
			db, closeDB := startConverted(
				ctx,
				gate,
				ins,
				func() { gate.arm(".domain") },
			)
			Eventually(gate.waiting).Should(Receive())
			Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRangeMax)).To(Succeed())
			close(gate.release)
			Eventually(
				messages("dropped the check of a segment that changed during its pass"),
			).
				Should(Equal(1))
			Expect(checkFiles(disk, keys...)).To(Equal(none))
			Expect(logs.FilterLevelExact(zapcore.ErrorLevel).Len()).To(BeZero())
			Expect(dataOf(ctx, db, keys)).To(BeEmpty())
			Expect(closeDB()).To(Succeed())
		})

	It("Should drop the check of a segment whose key a Drop takes during its pass",
		func(ctx SpecContext) {
			gate := &gateFS{FS: disk}
			db, closeDB := startConverted(
				ctx,
				gate,
				ins,
				func() { gate.arm(".domain") },
			)
			Eventually(gate.waiting).Should(Receive())
			Expect(db.Drop(ctx, keys)).To(Succeed())
			close(gate.release)
			Eventually(
				messages("dropped the check of a segment that changed during its pass"),
			).
				Should(Equal(1))
			// The directory of each key goes at a later Sweep, since the pass pinned
			// its files during the Drop.
			Expect(checkFiles(disk, keys...)).To(Equal(none))
			for _, key := range keys {
				Expect(db.RetrieveChannel(ctx, key)).Error().
					To(MatchError(query.ErrNotFound))
			}
			Expect(logs.FilterLevelExact(zapcore.ErrorLevel).Len()).To(BeZero())
			Expect(closeDB()).To(Succeed())
		})

	It("Should free the files of a key that a Drop takes during the read of its "+
		"segment within one block", func(ctx SpecContext) {
		const key cesium.ChannelKey = 21
		big := writeStamps(ctx, key, 4*segment.BlockSize/8)
		reads := &readsFS{FS: big}
		gate := &gateFS{FS: reads}
		db, closeDB := startConverted(ctx, gate, ins, func() { gate.arm(".domain") })
		Eventually(gate.waiting).Should(Receive())
		reads.n.Store(0)
		Expect(db.Drop(ctx, []cesium.ChannelKey{key})).To(Succeed())
		close(gate.release)
		dir := strconv.FormatUint(uint64(key), 10)
		Eventually(func() ([]string, error) {
			infos, err := big.List(dir)
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			var names []string
			for _, info := range infos {
				if strings.HasSuffix(info.Name(), ".domain") {
					names = append(names, info.Name())
				}
			}
			return names, err
		}).Should(BeEmpty())
		Expect(reads.n.Load()).To(BeNumerically("<=", 3))
		Expect(logs.FilterLevelExact(zapcore.ErrorLevel).Len()).To(BeZero())
		Expect(closeDB()).To(Succeed())
	})

	Describe("Drop at the write of a check file", func() {
		const key cesium.ChannelKey = 21
		var (
			small *fault.FS
			hold  *passFS
			dir   = strconv.FormatUint(uint64(key), 10)
		)
		BeforeEach(func(ctx SpecContext) {
			small = writeStamps(ctx, key, 64)
			hold = &passFS{FS: small}
		})
		// files returns the names in the directory of key, or nil when it is gone.
		files := func() ([]string, error) {
			infos, err := small.List(dir)
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			var names []string
			for _, info := range infos {
				names = append(names, info.Name())
			}
			return names, err
		}
		// drop drops key in a goroutine, and returns the channel of its error.
		drop := func(ctx context.Context, db *cesium.DB) <-chan error {
			done := make(chan error, 1)
			go func() { done <- db.Drop(ctx, []cesium.ChannelKey{key}) }()
			return done
		}
		quiet := func() {
			GinkgoHelper()
			Eventually(
				messages("dropped the check of a segment that changed during its pass"),
			).Should(Equal(1))
			Expect(logs.FilterLevelExact(zapcore.WarnLevel).Len()).To(BeZero())
			Expect(logs.FilterLevelExact(zapcore.ErrorLevel).Len()).To(BeZero())
		}

		It("Should write no check file of a key that a Drop takes before the write",
			func(ctx SpecContext) {
				db, closeDB := startConverted(
					ctx,
					hold,
					ins,
					func() { hold.arm("close", ".domain") },
				)
				Eventually(hold.waiting).Should(Receive())
				Expect(db.Drop(ctx, []cesium.ChannelKey{key})).To(Succeed())
				hold.checks.Store(0)
				close(hold.release)
				quiet()
				Expect(hold.checks.Load()).To(BeZero())
				Eventually(files).Should(BeEmpty())
				Expect(closeDB()).To(Succeed())
			})

		It("Should hold a Drop that meets the write of a check file until the file is "+
			"closed, then remove the directory", func(ctx SpecContext) {
			db, closeDB := startConverted(
				ctx,
				hold,
				ins,
				func() { hold.arm("open", ".chk") },
			)
			Eventually(hold.waiting).Should(Receive())
			done := drop(ctx, db)
			Consistently(done, 200*time.Millisecond).ShouldNot(Receive())
			close(hold.release)
			Eventually(done).Should(Receive(BeNil()))
			Expect(files()).To(BeEmpty())
			quiet()
			Expect(closeDB()).To(Succeed())
		})

		It("Should remove the directory of a key that a Drop takes during the sync of "+
			"its check file", func(ctx SpecContext) {
			db, closeDB := startConverted(
				ctx,
				hold,
				ins,
				func() { hold.arm("syncdir", dir) },
			)
			Eventually(hold.waiting).Should(Receive())
			Eventually(drop(ctx, db)).Should(Receive(BeNil()))
			Expect(files()).To(BeEmpty())
			close(hold.release)
			quiet()
			Expect(closeDB()).To(Succeed())
		})
	})
})
