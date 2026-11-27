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
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xerrors "github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var errSyncFault = xerrors.New("sync fault")

var _ = Describe("Writer sync faults on the pointer log", func() {
	var (
		dirs, logs atomic.Bool
		synced     atomic.Int64
		db         *cesium.DB
		ran        chan error
		idx, data  cesium.ChannelKey
		keys       []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		dirs.Store(false)
		logs.Store(false)
		fs := syncFaultFS{FS: xfs.NewMem(), dirs: &dirs, logs: &logs, synced: &synced}
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
		))
		runCtx, cancel := context.WithCancel(context.Background())
		ran = make(chan error, 1)
		go func() { ran <- db.Run(runCtx) }()
		DeferCleanup(func() {
			cancel()
			Eventually(ran).Should(Receive(BeNil()))
			Expect(db.Close()).To(Succeed())
		})
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
	// write opens a writer at the second start, writes a sample of each key at each of
	// seconds, commits, and returns the error of its Sync after it closes.
	write := func(ctx context.Context, seconds ...telem.TimeStamp) error {
		GinkgoHelper()
		stamps := make([]telem.TimeStamp, len(seconds))
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			stamps[i], values[i] = s*telem.SecondTS, int64(s)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    stamps[0],
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		err := w.Sync()
		_ = w.Close()
		return err
	}
	stored := func(ctx context.Context) []int64 {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
		var values []int64
		for _, s := range fr.Get(data).Series {
			values = append(values, s.Unmarshal[int64]()...)
		}
		return values
	}
	running := func() {
		GinkgoHelper()
		Consistently(ran, 200*time.Millisecond).ShouldNot(Receive())
	}

	It("Should keep the engine running after a failed directory sync of a new "+
		"segment, and sync once the fault clears", func(ctx SpecContext) {
		dirs.Store(true)
		Expect(write(ctx, 10, 11, 12)).To(MatchError(errSyncFault))
		running()
		dirs.Store(false)
		before := synced.Load()
		Expect(write(ctx, 20, 21)).To(Succeed())
		Expect(synced.Load()).To(BeNumerically(">", before))
		running()
		Expect(stored(ctx)).To(ContainElements(int64(20), int64(21)))
	})

	It("Should keep the engine running after a failed log sync, and sync once the "+
		"fault clears", func(ctx SpecContext) {
		Expect(write(ctx, 10, 11, 12)).To(Succeed())
		logs.Store(true)
		Expect(write(ctx, 20, 21)).To(MatchError(errSyncFault))
		running()
		logs.Store(false)
		Expect(write(ctx, 30, 31)).To(Succeed())
		running()
		Expect(stored(ctx)).To(ContainElements(int64(10), int64(30), int64(31)))
	})
})

// syncFaultFS fails each SyncDir while dirs is set, and each Sync of a log file while
// logs is set. It counts each SyncDir that passes in synced.
type syncFaultFS struct {
	xfs.FS
	dirs, logs *atomic.Bool
	synced     *atomic.Int64
}

func (f syncFaultFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return syncFaultFS{FS: sub, dirs: f.dirs, logs: f.logs, synced: f.synced}, nil
}

func (f syncFaultFS) SyncDir(name string) error {
	if f.dirs.Load() {
		return errSyncFault
	}
	if err := f.FS.SyncDir(name); err != nil {
		return err
	}
	f.synced.Add(1)
	return nil
}

func (f syncFaultFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".log") {
		return file, err
	}
	return syncFaultFile{File: file, fail: f.logs}, nil
}

type syncFaultFile struct {
	xfs.File
	fail *atomic.Bool
}

func (f syncFaultFile) Sync() error {
	if f.fail.Load() {
		return errSyncFault
	}
	return f.File.Sync()
}
