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
	"math/rand/v2"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// hold holds its first caller while armed, until free.
type hold struct {
	armed atomic.Bool
	held  chan struct{}
	freed chan struct{}
	free  func()
}

func newHold() *hold {
	h := &hold{held: make(chan struct{}), freed: make(chan struct{})}
	h.free = sync.OnceFunc(func() { close(h.freed) })
	return h
}

func (h *hold) wait() {
	if h.armed.CompareAndSwap(true, false) {
		close(h.held)
		<-h.freed
	}
}

// inLogCheck reports whether the check pass logs a check in the caller.
func inLogCheck() bool {
	return strings.Contains(string(debug.Stack()), "cesium.(*checker).logCheck")
}

// errOpen is the error of the removal of an open file on a disk that refuses it.
var errOpen = errors.New("the file is open")

// passGate gates the check pass in dir. While refusing, it lets the pass create one
// check file and refuses the rest. While windows, it refuses to remove an open check
// file. The holds hold a check log at its first open or sub of dir, at its first sync
// of dir, and at its first close of a check file. The rmdir hold holds a removal of
// dir after its files go, and dir then goes only while empty, as with os.RemoveAll.
type passGate struct {
	dir                        string
	refusing, windows          atomic.Bool
	created                    atomic.Int32
	openLog, syncLog, closeLog *hold
	rmdir                      *hold
	mu                         sync.Mutex
	open                       map[string]int
}

func newPassGate() *passGate {
	return &passGate{
		dir:      "2",
		openLog:  newHold(),
		syncLog:  newHold(),
		closeLog: newHold(),
		rmdir:    newHold(),
		open:     map[string]int{},
	}
}

func (g *passGate) inDir(name string) bool {
	return name == g.dir || path.Dir(name) == g.dir
}

func (g *passGate) opened(name string, n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open[name] += n
}

func (g *passGate) refuseOpen(name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.windows.Load() && g.open[name] > 0 {
		return &os.PathError{Op: "remove", Path: name, Err: errOpen}
	}
	return nil
}

type passGateFS struct {
	xfs.FS
	g *passGate
}

func (f passGateFS) Sub(name string) (xfs.FS, error) {
	if f.g.openLog.armed.Load() && f.g.inDir(name) && inLogCheck() {
		f.g.openLog.wait()
	}
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return passGateFS{FS: sub, g: f.g}, nil
}

func (f passGateFS) Open(name string, flag int) (xfs.File, error) {
	g := f.g
	if g.openLog.armed.Load() && g.inDir(name) && inLogCheck() {
		g.openLog.wait()
	}
	check := flag&os.O_CREATE != 0 && path.Dir(name) == g.dir &&
		strings.HasSuffix(name, ".chk")
	if check && g.refusing.Load() && g.created.Add(1) > 1 {
		return nil, os.ErrPermission
	}
	file, err := f.FS.Open(name, flag)
	if err != nil || !check {
		return file, err
	}
	g.opened(name, 1)
	return checkFile{File: file, name: name, g: g}, nil
}

func (f passGateFS) SyncDir(name string) error {
	if f.g.syncLog.armed.Load() && name == f.g.dir && inLogCheck() {
		f.g.syncLog.wait()
	}
	return f.FS.SyncDir(name)
}

func (f passGateFS) Remove(name string) error {
	g := f.g
	if err := g.refuseOpen(name); err != nil {
		return err
	}
	if name != g.dir || !g.rmdir.armed.Load() {
		return f.FS.Remove(name)
	}
	infos, err := f.List(name)
	if err != nil {
		return err
	}
	for _, info := range infos {
		child := path.Join(name, info.Name())
		if err = g.refuseOpen(child); err != nil {
			return err
		}
		if err = f.FS.Remove(child); err != nil {
			return err
		}
	}
	g.rmdir.wait()
	if infos, err = f.List(name); err != nil {
		return err
	}
	if len(infos) > 0 {
		return &os.PathError{Op: "unlinkat", Path: name, Err: syscall.ENOTEMPTY}
	}
	return f.FS.Remove(name)
}

type checkFile struct {
	xfs.File
	name string
	g    *passGate
}

func (f checkFile) Close() error {
	if f.g.closeLog.armed.Load() && inLogCheck() {
		f.g.closeLog.wait()
	}
	err := f.File.Close()
	f.g.opened(f.name, -1)
	return err
}

var _ = Describe("Drop during the check pass", func() {
	const (
		idx cesium.ChannelKey = 1
		i32 cesium.ChannelKey = 2
		str cesium.ChannelKey = 3
	)
	keys := []cesium.ChannelKey{idx, i32, str}
	var (
		disk *fault.FS
		g    *passGate
		logs *observer.ObservedLogs
		ins  alamos.Instrumentation
	)
	// open opens a database on disk through g, and returns it with its close, which
	// closes it once.
	open := func(ctx context.Context) (*cesium.DB, func() error) {
		GinkgoHelper()
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(passGateFS{FS: disk, g: g}),
			cesium.WithPointerLog(),
			cesium.WithPartSize(1),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
			cesium.WithInstrumentation(ins),
		))
		stop := GoRun(ctx, db.Run)
		return db, sync.OnceValue(
			func() error { return errors.Join(stop(), db.Close()) },
		)
	}
	// dropAround drops i32 while h holds the check pass. It frees h once the Drop
	// returns, or after a second, since the Drop can wait for the check log that h
	// holds. It returns the error of the Drop.
	dropAround := func(ctx context.Context, db *cesium.DB, h *hold) error {
		GinkgoHelper()
		dropped := make(chan error, 1)
		go func() { dropped <- db.Drop(ctx, []cesium.ChannelKey{i32}) }()
		select {
		case err := <-dropped:
			h.free()
			return err
		case <-time.After(time.Second):
		}
		h.free()
		var err error
		Eventually(dropped).WithTimeout(10*time.Second).Should(
			Receive(&err),
			"the Drop returns once the check pass goes on",
		)
		return err
	}
	// warnings returns the logs at warn level or above.
	warnings := func() []observer.LoggedEntry {
		return logs.Filter(func(e observer.LoggedEntry) bool {
			return e.Level >= zapcore.WarnLevel
		}).All()
	}
	// The first open writes the old layout of the conversion specs, and checks one
	// segment of i32, as when its close cuts the pass short. The open after it checks
	// the next one.
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		g = newPassGate()
		core, observed := observer.New(zapcore.DebugLevel)
		logs = observed
		ins = alamos.New("cesium", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
		db := openDBOnFS(ctx, disk, cesium.WithFileSizeCap(48*telem.Byte))
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      idx,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: i32, Name: "i32", Index: idx, DataType: telem.Int32T},
			cesium.Channel{Key: str, Name: "str", Index: idx, DataType: telem.StringT},
		)).To(Succeed())
		for _, start := range []telem.TimeStamp{1, 1000} {
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Clock:    idx,
				Start:    start * telem.SecondTS,
			}))
			for c := range telem.TimeStamp(4) {
				var (
					stamps = make([]telem.TimeStamp, 5)
					ints   = make([]int32, 5)
					strs   = make([]string, 5)
				)
				for j := range telem.TimeStamp(5) {
					n := c*5 + j
					stamps[j] = (start + n) * telem.SecondTS
					ints[j] = int32(start + n)
					strs[j] = strings.Repeat("x", int(n%7))
				}
				MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeries(stamps),
					telem.NewSeries(ints),
					telem.NewSeries(strs),
				}), nil))
				MustSucceed2(w.Commit(ctx, nil))
			}
			Expect(w.Close()).To(Succeed())
		}
		Expect(db.Close()).To(Succeed())
		disk.Writeback()
		g.refusing.Store(true)
		_, closeDB := open(ctx)
		Expect(closeDB()).To(Succeed())
		g.refusing.Store(false)
		logs.TakeAll()
	})

	It("Should unlink the old files of a dropped key at the Close of its last "+
		"iterator while the check pass checks a segment of it", func(ctx SpecContext) {
		g.syncLog.armed.Store(true)
		db, closeDB := open(ctx)
		DeferCleanup(func() {
			g.syncLog.free()
			Expect(closeDB()).To(Succeed())
		})
		Eventually(g.syncLog.held).WithTimeout(10*time.Second).Should(
			BeClosed(),
			"the check pass writes the check file of a segment of i32",
		)
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{i32},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(db.Drop(ctx, []cesium.ChannelKey{i32})).To(Succeed())
		Expect(iter.Close()).To(Succeed())
		Expect(disk.List(g.dir)).To(BeEmpty())
	})

	It("Should not make the directory of a dropped key again when the check pass "+
		"syncs it after the Drop", func(ctx SpecContext) {
		g.syncLog.armed.Store(true)
		db, closeDB := open(ctx)
		DeferCleanup(func() {
			g.syncLog.free()
			Expect(closeDB()).To(Succeed())
		})
		Eventually(g.syncLog.held).WithTimeout(10*time.Second).Should(
			BeClosed(),
			"the check pass writes the check file of a segment of i32",
		)
		dropped := make(chan error, 1)
		go func() { dropped <- db.Drop(ctx, []cesium.ChannelKey{i32}) }()
		var err error
		Eventually(dropped).WithTimeout(10*time.Second).Should(
			Receive(&err),
			"the Drop does not wait for the sync of a check file",
		)
		Expect(err).To(Succeed())
		g.syncLog.free()
		Expect(closeDB()).To(Succeed())
		Expect(disk.Exists(g.dir)).To(BeFalse())
		Expect(warnings()).To(BeEmpty())
	})

	It("Should remove the directory of a dropped key when the check pass makes a "+
		"check file in it while the Drop removes it", func(ctx SpecContext) {
		g.openLog.armed.Store(true)
		g.syncLog.armed.Store(true)
		g.rmdir.armed.Store(true)
		db, closeDB := open(ctx)
		DeferCleanup(func() {
			g.openLog.free()
			g.syncLog.free()
			g.rmdir.free()
			Expect(closeDB()).To(Succeed())
		})
		Eventually(g.openLog.held).WithTimeout(10*time.Second).Should(
			BeClosed(),
			"the check pass begins the check log of a segment of i32",
		)
		dropped := make(chan error, 1)
		go func() { dropped <- db.Drop(ctx, []cesium.ChannelKey{i32}) }()
		// The Drop removes the files of i32 before the pass makes its check file,
		// unless the Drop waits for the check log.
		select {
		case <-g.rmdir.held:
		case <-time.After(time.Second):
		}
		g.openLog.free()
		Eventually(g.syncLog.held).WithTimeout(10*time.Second).Should(
			BeClosed(),
			"the check pass writes the check file of a segment of i32",
		)
		Eventually(g.rmdir.held).WithTimeout(10*time.Second).Should(
			BeClosed(),
			"the Drop removes the files of i32",
		)
		g.rmdir.free()
		var err error
		Eventually(dropped).WithTimeout(10 * time.Second).Should(Receive(&err))
		Expect(err).To(Succeed())
		g.syncLog.free()
		Expect(closeDB()).To(Succeed())
		Expect(disk.Exists(g.dir)).To(BeFalse())
	})

	It("Should drop a key while the check pass writes a check file of it, on a disk "+
		"that refuses to remove an open file", func(ctx SpecContext) {
		g.windows.Store(true)
		g.closeLog.armed.Store(true)
		db, closeDB := open(ctx)
		DeferCleanup(func() {
			g.closeLog.free()
			Expect(closeDB()).To(Succeed())
		})
		Eventually(g.closeLog.held).WithTimeout(10*time.Second).Should(
			BeClosed(),
			"the check pass writes the check file of a segment of i32",
		)
		Expect(dropAround(ctx, db, g.closeLog)).To(Succeed())
		Expect(closeDB()).To(Succeed())
		Expect(disk.Exists(g.dir)).To(BeFalse())
	})
})
