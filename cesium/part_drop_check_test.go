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
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// headHoldFS holds the create of the head of generation 3 until release closes, and
// counts the lists of the root by the clean after a switch.
type headHoldFS struct {
	xfs.FS
	held, release chan struct{}
	cleans        *atomic.Int64
}

func (f headHoldFS) Open(name string, flag int) (xfs.File, error) {
	if name == checkpoint.Name(3) && flag&os.O_CREATE != 0 {
		f.held <- struct{}{}
		<-f.release
	}
	return f.FS.Open(name, flag)
}

func (f headHoldFS) List(name string) ([]xfs.FileInfo, error) {
	if byClean() {
		f.cleans.Add(1)
	}
	return f.FS.List(name)
}

// Sub keeps the hold on the root that Open gives the engine.
func (f headHoldFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

// byClean reports whether the clean of the checkpointer is a caller.
func byClean() bool {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, "internal/checkpoint.clean") {
			return true
		}
		if !more {
			return false
		}
	}
}

var _ = Describe("Check: a key dropped and created again during a switch", func() {
	It("Should not read the part records of the dropped key through the new key "+
		"once the head of the switch commits", func(ctx SpecContext) {
		fs := headHoldFS{
			FS:      xfs.NewMem(),
			held:    make(chan struct{}, 1),
			release: make(chan struct{}),
			cleans:  &atomic.Int64{},
		}
		db := partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		release := sync.OnceFunc(func() { close(fs.release) })
		DeferCleanup(release)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		dataCh := cesium.Channel{
			Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
		}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			dataCh,
		)).To(Succeed())
		writeParted(ctx, db, fs, idx, data, 3, 10)
		// Switch 3 takes its snapshot while data holds its parted records, then waits
		// to create its head.
		switched := make(chan error, 1)
		go func() { switched <- switchLog(db, fs, 3) }()
		Eventually(fs.held).WithTimeout(time.Minute).Should(Receive())
		Expect(db.Drop(ctx, []cesium.ChannelKey{data})).To(Succeed())
		Expect(db.CreateChannel(ctx, dataCh)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Start:    1_000 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{idx, data},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(1_000, 1_001),
				telem.NewSeriesV[int64](-1, -2),
			},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(int64s(ctx, db, data)).To(Equal([]int64{-1, -2}))
		cleans := fs.cleans.Load()
		release()
		Eventually(switched).WithTimeout(time.Minute).Should(Receive(Succeed()))
		// The checkpointer hands the head to the state before its clean, and a
		// writer's open waits for the state owner.
		Eventually(fs.cleans.Load).Should(BeNumerically(">", cleans))
		w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Start:    2_000 * telem.SecondTS,
		}))
		Expect(w.Close()).To(Succeed())
		Expect(int64s(ctx, db, data)).To(Equal([]int64{-1, -2}))
	})
})
