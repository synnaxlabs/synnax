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
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// frameHold lets the next write of a log through once armed, and then holds the write
// after it, which covers that frame, until let.
type frameHold struct {
	armed         atomic.Bool
	held, release chan struct{}
	once          sync.Once
	// framed, name, off, and size locate the frame that the hold keeps uncovered. Only
	// the goroutine of the log writes them, before held closes.
	framed    bool
	name      string
	off, size int64
}

func newFrameHold() *frameHold {
	return &frameHold{held: make(chan struct{}), release: make(chan struct{})}
}

// let ends the hold.
func (h *frameHold) let() { h.once.Do(func() { close(h.release) }) }

// cut flips the bytes of the held frame in its first block of image, as the lasting
// corruption of fault 5d does before a later frame or mark is durable.
func (h *frameHold) cut(image xfs.FS) {
	GinkgoHelper()
	f := MustSucceed(image.Open(h.name, os.O_RDWR))
	b := make([]byte, min(h.size, (h.off|(log.BlockSize-1))+1-h.off))
	MustSucceed(f.ReadAt(b, h.off))
	for i := range b {
		b[i] ^= 0xFF
	}
	MustSucceed(f.WriteAt(b, h.off))
	Expect(f.Close()).To(Succeed())
}

// frameHoldFS writes each log file through h.
type frameHoldFS struct {
	xfs.FS
	h *frameHold
}

// Sub keeps the hold on the root that Open gives the engine.
func (f frameHoldFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f frameHoldFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".log") {
		return file, err
	}
	return frameHoldFile{File: file, name: name, h: f.h}, nil
}

type frameHoldFile struct {
	xfs.File
	name string
	h    *frameHold
}

func (f frameHoldFile) WriteAt(p []byte, off int64) (int, error) {
	if f.h.armed.Load() {
		if !f.h.framed {
			f.h.framed, f.h.name, f.h.off, f.h.size = true, f.name, off, int64(len(p))
		} else {
			f.h.armed.Store(false)
			close(f.h.held)
			<-f.h.release
		}
	}
	return f.File.WriteAt(p, off)
}

const (
	coverSpare cesium.ChannelKey = 3
	coverIdle  cesium.ChannelKey = 4
)

var _ = Describe("Check: a record whose frame fault 5d cuts before the log covers it",
	func() {
		var (
			disk *fault.FS
			h    *frameHold
			db   *cesium.DB
			idle *cesium.Writer
		)
		BeforeEach(func(ctx SpecContext) {
			disk = MustSucceed(fault.New(fault.Config{}))
			h = newFrameHold()
			var closeDB func()
			db, closeDB = breakOpen(ctx, frameHoldFS{FS: disk, h: h})
			DeferCleanup(func() {
				h.let()
				closeDB()
			})
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      breakIdx,
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      breakData,
					Name:     "data",
					Index:    breakIdx,
					DataType: telem.Int64T,
				},
				cesium.Channel{
					Key:      coverSpare,
					Name:     "spare",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      coverIdle,
					Name:     "idle",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
			)).To(Succeed())
			breakWrite(ctx, db, 10, 11)
			breakWrite(ctx, db, 20, 21, 22)
			// The close of a writer reaps, so it takes dead segments while the log
			// waits.
			idle = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{coverIdle},
				Clock:    coverIdle,
				Start:    100 * telem.SecondTS,
			}))
			// A Drop returns once the log covers it, so no mark of an earlier frame
			// is left to write when the hold arms.
			Expect(db.Drop(ctx, []cesium.ChannelKey{coverSpare})).To(Succeed())
			h.armed.Store(true)
		})

		// durable reads a crash image of disk that keeps the held frame.
		durable := func(ctx context.Context) ([]int64, error) {
			GinkgoHelper()
			re, closeRe := breakOpen(ctx, MustSucceed(disk.Crash(fault.CrashConfig{})))
			defer closeRe()
			return breakRead(ctx, re)
		}

		// cutImages expects every sample of want in each crash image of disk with the
		// held frame cut.
		cutImages := func(ctx context.Context, want []int64) {
			GinkgoHelper()
			var failures []string
			for runKey := range uint64(8) {
				image := MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:            runKey,
					EntrySurvivalRate: 0.5,
					DataSurvivalRate:  0.5,
				}))
				h.cut(image)
				re, closeRe := breakOpen(ctx, image)
				got, err := breakRead(ctx, re)
				if err != nil || !slices.Equal(got, want) {
					failures = append(failures, fmt.Sprintf(
						"run key %d: samples %v, error %v", runKey, got, err,
					))
				}
				closeRe()
			}
			Expect(failures).To(BeEmpty())
		}

		DescribeTable("Should bring a key back with its files", func(
			ctx SpecContext,
			keys ...cesium.ChannelKey,
		) {
			dropped := make(chan error, 1)
			go func() { dropped <- db.Drop(ctx, keys) }()
			Eventually(h.held).WithTimeout(time.Minute).Should(BeClosed())
			Consistently(dropped).ShouldNot(Receive(), "a Drop before its cover")
			Expect(idle.Close()).To(Succeed())
			kept := MustSucceed(disk.Crash(fault.CrashConfig{}))
			re, closeRe := breakOpen(ctx, kept)
			// The Drop takes the samples, and a second Drop removes the key that its
			// removal did not reach.
			Expect(breakRead(ctx, re)).To(BeEmpty())
			Expect(re.Drop(ctx, keys)).To(Succeed())
			for _, k := range keys {
				_, err := re.RetrieveChannel(ctx, k)
				Expect(err).To(MatchError(cesium.ErrChannelNotFound))
				Expect(err).To(MatchError(ContainSubstring(
					"channel with key %d not found", k,
				)))
				Expect(kept.Exists(strconv.Itoa(int(k)))).To(BeFalse(), "key %d", k)
			}
			closeRe()
			cutImages(ctx, []int64{10, 11, 20, 21, 22})
			h.let()
			Eventually(dropped).Should(Receive(Succeed()))
		},
			Entry("in a Drop of the key alone", breakData),
			Entry("in a Drop of the key and its index", breakData, breakIdx),
		)

		It("Should keep the file of a segment that a burying Tombstone empties", func(
			ctx SpecContext,
		) {
			Expect(db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{breakData},
				(0 * telem.SecondTS).Range(15*telem.SecondTS),
			)).To(Succeed())
			Eventually(h.held).WithTimeout(time.Minute).Should(BeClosed())
			Expect(idle.Close()).To(Succeed())
			Expect(durable(ctx)).To(Equal([]int64{20, 21, 22}))
			cutImages(ctx, []int64{10, 11, 20, 21, 22})
		})
	})
