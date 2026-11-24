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
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// passGate gates the check files that the check pass creates in dir. While refusing,
// it lets the pass create one and refuses the rest. While holding, it holds the pass
// at the sync of dir after its next check file, until release closes.
type passGate struct {
	dir      string
	refusing atomic.Bool
	created  atomic.Int32
	holding  atomic.Bool
	wrote    atomic.Bool
	held     chan struct{}
	release  chan struct{}
}

type passGateFS struct {
	xfs.FS
	g *passGate
}

func (f passGateFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return passGateFS{FS: sub, g: f.g}, nil
}

func (f passGateFS) Open(name string, flag int) (xfs.File, error) {
	if flag&os.O_CREATE != 0 && path.Dir(name) == f.g.dir &&
		strings.HasSuffix(name, ".chk") {
		if f.g.refusing.Load() && f.g.created.Add(1) > 1 {
			return nil, os.ErrPermission
		}
		if f.g.holding.Load() {
			f.g.wrote.Store(true)
		}
	}
	return f.FS.Open(name, flag)
}

func (f passGateFS) SyncDir(name string) error {
	if name == f.g.dir && f.g.wrote.Load() && f.g.holding.CompareAndSwap(true, false) {
		close(f.g.held)
		<-f.g.release
	}
	return f.FS.SyncDir(name)
}

var _ = Describe("Drop during the check pass", func() {
	const (
		idx cesium.ChannelKey = 1
		i32 cesium.ChannelKey = 2
		str cesium.ChannelKey = 3
	)
	keys := []cesium.ChannelKey{idx, i32, str}
	// writeOld writes the old layout of the conversion specs to disk.
	writeOld := func(ctx context.Context, disk *fault.FS) {
		GinkgoHelper()
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
	}

	It("Should unlink the old files of a dropped key at the Close of its last "+
		"iterator while the check pass checks a segment of it", func(ctx SpecContext) {
		disk := MustSucceed(fault.New(fault.Config{}))
		writeOld(ctx, disk)
		g := &passGate{
			dir:     "2",
			held:    make(chan struct{}),
			release: make(chan struct{}),
		}
		parted := func() (*cesium.DB, func() error) {
			GinkgoHelper()
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(passGateFS{FS: disk, g: g}),
				cesium.WithPointerLog(),
				cesium.WithPartSize(1),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
			))
			stop := GoRun(ctx, db.Run)
			return db, func() error { return errors.Join(stop(), db.Close()) }
		}
		// The first open checks one segment of i32, as when its close cuts the pass
		// short.
		g.refusing.Store(true)
		_, closeDB := parted()
		Expect(closeDB()).To(Succeed())
		g.refusing.Store(false)
		g.holding.Store(true)
		db, closeDB := parted()
		DeferCleanup(func() {
			close(g.release)
			Expect(closeDB()).To(Succeed())
		})
		Eventually(g.held).WithTimeout(10*time.Second).Should(
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
})
