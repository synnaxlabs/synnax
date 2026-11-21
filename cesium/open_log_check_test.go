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
	"math/rand/v2"
	"os"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// openLog opens and runs Cesium on the pointer log in fs, and returns its close.
func openLog(ctx context.Context, fs xfs.FS) (*cesium.DB, func() error, error) {
	db, err := cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithRandom(rand.NewPCG(1, 2)),
	)
	if err != nil {
		return nil, nil, err
	}
	stop := testutil.GoRun(ctx, db.Run)
	return db, func() error {
		if err := stop(); err != nil {
			return err
		}
		return db.Close()
	}, nil
}

// durableFS calls onEvent after each call that changes what a crash keeps: a create, a
// truncate, a sync, a remove, a rename, and a directory sync. It calls onWrite, when
// set, after each write.
type durableFS struct {
	xfs.FS
	onEvent func()
	onWrite func()
}

func (f durableFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		f.onEvent()
	}
	return durableFile{File: file, onEvent: f.onEvent, onWrite: f.onWrite}, nil
}

func (f durableFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	f.onEvent()
	return durableFS{FS: sub, onEvent: f.onEvent, onWrite: f.onWrite}, nil
}

func (f durableFS) Remove(name string) error {
	defer f.onEvent()
	return f.FS.Remove(name)
}

func (f durableFS) Rename(from, to string) error {
	defer f.onEvent()
	return f.FS.Rename(from, to)
}

func (f durableFS) SyncDir(name string) error {
	defer f.onEvent()
	return f.FS.SyncDir(name)
}

type durableFile struct {
	xfs.File
	onEvent func()
	onWrite func()
}

func (f durableFile) WriteAt(p []byte, off int64) (int, error) {
	if f.onWrite != nil {
		defer f.onWrite()
	}
	return f.File.WriteAt(p, off)
}

func (f durableFile) Truncate(size int64) error {
	defer f.onEvent()
	return f.File.Truncate(size)
}

func (f durableFile) Sync() error {
	defer f.onEvent()
	return f.File.Sync()
}

// images takes a crash image of disk at each call of take while taking holds, under one
// lock, since the owners of a DB take them from their own goroutines.
type images struct {
	mu     sync.Mutex
	disk   *fault.FS
	runKey uint64
	taking atomic.Bool
	list   []*fault.FS
}

func (im *images) take() {
	if !im.taking.Load() {
		return
	}
	im.mu.Lock()
	defer im.mu.Unlock()
	im.list = append(im.list, MustSucceed(im.disk.Crash(fault.CrashConfig{
		RunKey:            im.runKey*1_000 + uint64(len(im.list)),
		EntrySurvivalRate: 0.5,
		DataSurvivalRate:  0.5,
	})))
}

// failures opens each image and returns a line for each open that fails.
func (im *images) failures(ctx context.Context) []string {
	GinkgoHelper()
	im.mu.Lock()
	defer im.mu.Unlock()
	var out []string
	for i, image := range im.list {
		_, closeImage, err := openLog(ctx, image)
		if err != nil {
			out = append(out, fmt.Sprintf(
				"run key %d, image %d of %d: %v", im.runKey, i, len(im.list), err,
			))
			continue
		}
		Expect(closeImage()).To(Succeed())
	}
	return out
}

var _ = Describe("Pointer log open check", func() {
	It("Should open each crash image of the first open of a database", func(
		ctx SpecContext,
	) {
		var failures []string
		for runKey := range uint64(4) {
			im := &images{disk: MustSucceed(fault.New(fault.Config{})), runKey: runKey}
			im.taking.Store(true)
			_, closeDB := MustSucceed2(
				openLog(ctx, durableFS{FS: im.disk, onEvent: im.take}),
			)
			Expect(closeDB()).To(Succeed())
			im.taking.Store(false)
			Expect(im.list).ToNot(BeEmpty())
			failures = append(failures, im.failures(ctx)...)
		}
		Expect(failures).To(BeEmpty())
	})

	It("Should open each crash image of a channel create", func(ctx SpecContext) {
		var failures []string
		for runKey := range uint64(4) {
			im := &images{disk: MustSucceed(fault.New(fault.Config{})), runKey: runKey}
			db, closeDB := MustSucceed2(openLog(ctx, durableFS{
				FS:      im.disk,
				onEvent: im.take,
				onWrite: im.take,
			}))
			im.taking.Store(true)
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			im.taking.Store(false)
			Expect(closeDB()).To(Succeed())
			Expect(im.list).ToNot(BeEmpty())
			failures = append(failures, im.failures(ctx)...)
		}
		Expect(failures).To(BeEmpty())
	})

	It("Should open with the channel that it created after one bad read of each "+
		"block", func(ctx SpecContext) {
		ch := cesium.Channel{
			Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		}
		disk := MustSucceed(fault.New(fault.Config{}))
		db, closeDB := MustSucceed2(openLog(ctx, disk))
		Expect(db.CreateChannel(ctx, ch)).To(Succeed())
		ch = MustSucceed(db.RetrieveChannel(ctx, 1))
		Expect(closeDB()).To(Succeed())
		syncAll(disk, "")
		var failures []string
		for runKey := range uint64(8) {
			// The first read of each block of the image flips one bit. A second read
			// is good.
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				Image: fault.Config{RunKey: runKey, ReadCorruptionRate: 1},
			}))
			db, closeImage, err := openLog(ctx, image)
			if err != nil {
				failures = append(failures, fmt.Sprintf("run key %d: %v", runKey, err))
				continue
			}
			got, err := db.RetrieveChannel(ctx, 1)
			if err != nil || got != ch {
				failures = append(failures, fmt.Sprintf(
					"run key %d: key %d, name %q, data type %s, index %d: %v",
					runKey, got.Key, got.Name, got.DataType, got.Index, err,
				))
			}
			Expect(closeImage()).To(Succeed())
		}
		Expect(failures).To(BeEmpty())
	})

	It("Should put the Restart in log 1 when a crash past three quarters of it finds "+
		"no room for log 2", func(ctx SpecContext) {
		disks := fault.Config{Capacity: log.FileSize + 1<<20}
		disk := MustSucceed(fault.New(disks))
		db, closeDB := MustSucceed2(openLog(ctx, disk))
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		// Each Intend takes one block of the log: its frame, then a mark at the next
		// block boundary. 12,400 blocks pass three quarters of the log.
		for range 12_400 {
			Expect(db.Intend([]cesium.ChannelKey{1})).To(Succeed())
		}
		image := MustSucceed(disk.Crash(fault.CrashConfig{Image: disks}))
		Expect(closeDB()).To(Succeed())
		_, closeImage := MustSucceed2(openLog(ctx, image))
		Expect(closeImage()).To(Succeed())
	})
})
