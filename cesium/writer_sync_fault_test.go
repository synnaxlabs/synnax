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
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	alamostestutil "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xerrors "github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var errSyncFault = xerrors.New("sync fault")

var _ = Describe("Writer storage faults on the pointer log", func() {
	var (
		faults    syncFaults
		mem       xfs.FS
		db        *cesium.DB
		ran       chan error
		cancel    context.CancelFunc
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
		logs      *observer.ObservedLogs
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		ins, observed := alamostestutil.ObservedInstrumentation(zapcore.ErrorLevel)
		logs = observed
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(syncFaultFS{FS: mem, faults: &faults}),
			cesium.WithPointerLog(),
			cesium.WithLogFileSize(telem.Size(log.MinFileSize)),
			cesium.WithInstrumentation(ins),
		))
		var runCtx context.Context
		runCtx, cancel = context.WithCancel(context.Background())
		ran = make(chan error, 1)
		go func() { ran <- db.Run(runCtx) }()
	}
	// shut stops Run and closes the DB, and returns the errors of both.
	shut := func() error {
		GinkgoHelper()
		cancel()
		var runErr error
		Eventually(ran).Should(Receive(&runErr))
		return xerrors.Join(runErr, db.Close())
	}
	BeforeEach(func(ctx SpecContext) {
		for _, b := range []*atomic.Bool{
			&faults.dirs,
			&faults.logSyncs,
			&faults.lose,
			&faults.segmentsFull,
			&faults.logFull,
		} {
			b.Store(false)
		}
		mem = xfs.NewMem()
		open(ctx)
		DeferCleanup(func() { Expect(shut()).To(Succeed()) })
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
			},
		)).To(Succeed())
	})
	// write opens a writer at the first of seconds, writes a sample of each key at each
	// of them, commits, and returns the first error of its write, commit, and Sync. It
	// closes the writer.
	write := func(ctx context.Context, seconds ...telem.TimeStamp) (err error) {
		GinkgoHelper()
		stamps := make([]telem.TimeStamp, len(seconds))
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			stamps[i], values[i] = s*telem.SecondTS, int64(s)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    stamps[0],
		}))
		defer func() { err = xerrors.Join(err, w.Close()) }()
		if _, _, err := w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
		}), nil); err != nil {
			return err
		}
		if _, _, err := w.Commit(ctx, nil); err != nil {
			return err
		}
		return w.Sync()
	}
	stored := func(ctx context.Context) []int64 {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
		values := make([]int64, 0, fr.Get(data).Len())
		for _, s := range fr.Get(data).Series {
			values = append(values, s.Unmarshal[int64]()...)
		}
		return values
	}
	running := func() {
		GinkgoHelper()
		select {
		case err := <-ran:
			Fail("Run ended: " + fmt.Sprint(err))
		case <-time.After(200 * time.Millisecond):
		}
	}

	It("Should keep running after a failed directory sync of a new segment, and sync "+
		"the directory at the next segment", func(ctx SpecContext) {
		faults.dirs.Store(true)
		Expect(write(ctx, 10, 11, 12)).To(MatchError(errSyncFault))
		running()
		faults.dirs.Store(false)
		before := faults.synced.Load()
		Expect(write(ctx, 20, 21)).To(Succeed())
		Expect(faults.synced.Load()).To(BeNumerically(">", before))
		running()
		Expect(stored(ctx)).To(Equal([]int64{20, 21}))
	})

	DescribeTable("Should keep running after a failed log sync, fail every later Sync "+
		"until a restart, and replay or drop the failed frame at the restart", func(
		ctx SpecContext,
		lost bool,
		want []int64,
	) {
		Expect(write(ctx, 10, 11, 12)).To(Succeed())
		faults.logSyncs.Store(true)
		faults.lose.Store(lost)
		Expect(write(ctx, 20, 21)).To(MatchError(errSyncFault))
		running()
		faults.logSyncs.Store(false)
		Expect(write(ctx, 30, 31)).To(MatchError(errSyncFault))
		running()
		Expect(shut()).To(MatchError(errSyncFault))
		open(ctx)
		Expect(stored(ctx)).To(Equal(want))
		segments := func() int {
			infos := MustSucceed(mem.List(strconv.Itoa(int(data))))
			return len(lo.Filter(infos, func(info xfs.FileInfo, _ int) bool {
				return strings.HasSuffix(info.Name(), ".seg")
			}))
		}
		Eventually(segments).Should(Equal(len(want) / 2))
		Expect(write(ctx, 40, 41)).To(Succeed())
		Expect(stored(ctx)).To(Equal(append(want, 40, 41)))
	},
		Entry("that the disk holds", false, []int64{10, 11, 12, 20, 21}),
		Entry("that the disk lost", true, []int64{10, 11, 12}),
	)

	It("Should log a failed log sync once, and a full log disk not at all", func(
		ctx SpecContext,
	) {
		const failed = "the log failed a sync"
		Expect(write(ctx, 10, 11, 12)).To(Succeed())
		faults.logFull.Store(true)
		Expect(write(ctx, 20, 21)).To(MatchError(syscall.ENOSPC))
		faults.logFull.Store(false)
		Expect(logs.FilterMessageSnippet(failed).Len()).To(BeZero())
		faults.logSyncs.Store(true)
		Expect(write(ctx, 30, 31)).To(MatchError(errSyncFault))
		Expect(write(ctx, 40, 41)).To(MatchError(errSyncFault))
		Expect(logs.FilterMessageSnippet(failed).Len()).To(Equal(1))
		faults.logSyncs.Store(false)
		Expect(shut()).To(MatchError(errSyncFault))
		open(ctx)
	})

	DescribeTable("Should fail only the write that a full disk refuses, and write "+
		"again with no restart once space frees", func(
		ctx SpecContext,
		full *atomic.Bool,
	) {
		Expect(write(ctx, 10, 11, 12)).To(Succeed())
		full.Store(true)
		Expect(write(ctx, 20, 21)).To(MatchError(syscall.ENOSPC))
		running()
		full.Store(false)
		Expect(write(ctx, 30, 31)).To(Succeed())
		running()
		Expect(stored(ctx)).To(Equal([]int64{10, 11, 12, 30, 31}))
	},
		Entry("on a segment", &faults.segmentsFull),
		Entry("on the log", &faults.logFull),
	)
})

// syncFaults are the faults that a syncFaultFS injects while each is set: a failed
// SyncDir, a failed Sync of a log file, which also zeros the writes since the last sync
// when lose is set, and a full disk for writes of segment files or log files. synced
// counts each SyncDir that passes.
type syncFaults struct {
	dirs, logSyncs, lose  atomic.Bool
	segmentsFull, logFull atomic.Bool
	synced                atomic.Int64
}

type syncFaultFS struct {
	xfs.FS
	faults *syncFaults
}

func (f syncFaultFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return syncFaultFS{FS: sub, faults: f.faults}, nil
}

func (f syncFaultFS) SyncDir(name string) error {
	if f.faults.dirs.Load() {
		return errSyncFault
	}
	if err := f.FS.SyncDir(name); err != nil {
		return err
	}
	f.faults.synced.Add(1)
	return nil
}

func (f syncFaultFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return file, err
	}
	switch {
	case strings.HasSuffix(name, ".log"):
		return &syncFaultFile{
			File: file,
			sync: &f.faults.logSyncs,
			lose: &f.faults.lose,
			full: &f.faults.logFull,
		}, nil
	case strings.HasSuffix(name, ".seg"):
		return &syncFaultFile{File: file, full: &f.faults.segmentsFull}, nil
	}
	return file, nil
}

// syncFaultFile fails each Sync while sync is set, and zeros the writes since its last
// Sync then when lose is set. It fails each WriteAt with ENOSPC while full is set.
type syncFaultFile struct {
	xfs.File
	sync, lose, full *atomic.Bool
	// unsynced holds the offsets and lengths of the writes since the last Sync.
	unsynced [][2]int64
}

func (f *syncFaultFile) Sync() error {
	if f.sync == nil || !f.sync.Load() {
		f.unsynced = f.unsynced[:0]
		return f.File.Sync()
	}
	if f.lose.Load() {
		for _, w := range f.unsynced {
			if _, err := f.File.WriteAt(make([]byte, w[1]), w[0]); err != nil {
				return err
			}
		}
	}
	f.unsynced = f.unsynced[:0]
	return errSyncFault
}

func (f *syncFaultFile) WriteAt(p []byte, off int64) (int, error) {
	if f.full.Load() {
		return 0, syscall.ENOSPC
	}
	f.unsynced = append(f.unsynced, [2]int64{off, int64(len(p))})
	return f.File.WriteAt(p, off)
}
