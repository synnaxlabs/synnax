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
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
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
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

// replaceArmFS arms h at the write of the frame of log 1 that holds the Replace of
// segment old, so h keeps that frame uncovered whatever frames come before it.
type replaceArmFS struct {
	xfs.FS
	h   *frameHold
	old *atomic.Uint64
	// id is the identity of the database, which the checksum of a frame takes.
	id uint64
}

// Sub keeps the arm on the root that Open gives the engine.
func (f replaceArmFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f replaceArmFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != log.Name(1) {
		return file, err
	}
	return replaceArmFile{File: file, fs: f}, nil
}

type replaceArmFile struct {
	xfs.File
	fs replaceArmFS
}

func (f replaceArmFile) WriteAt(p []byte, off int64) (int, error) {
	if old := f.fs.old.Load(); old != 0 && replaces(p, f.fs.id, off, old) {
		f.fs.h.armed.Store(true)
	}
	return f.File.WriteAt(p, off)
}

// replaces reports whether p is the frame at off of log 1 of the database of identity
// id, with a Replace of segment old in its entries.
func replaces(p []byte, id uint64, off int64, old uint64) bool {
	fr, err := log.DecodeFrame(p, id, 1, uint64(off))
	if err != nil {
		return false
	}
	var d log.Decoder
	b := fr.Entries
	for range fr.Count {
		n, err := d.Reset(b)
		if err != nil {
			return false
		}
		for {
			r, ok, err := d.Next()
			if err != nil || !ok {
				break
			}
			if rp, isReplace := r.(*log.Replace); isReplace && rp.Old == old {
				return true
			}
		}
		b = b[n:]
	}
	return false
}

var _ = Describe("Check: a rewrite whose Replace frame fault 5d cuts", func() {
	DescribeTable("Should keep the old file, and read every live sample through it",
		func(ctx SpecContext, key cesium.ChannelKey) {
			const n = 1 << 19
			disk := MustSucceed(fault.New(fault.Config{}))
			h := newFrameHold()
			var old atomic.Uint64
			fake := timetestutil.NewClock(time.Unix(0, 0))
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(replaceArmFS{
					FS:  frameHoldFS{FS: disk, h: h},
					h:   h,
					old: &old,
					// The identity of a DB is the first draw of its random source.
					id: rand.NewPCG(1, 2).Uint64(),
				}),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithClock(fake),
				cesium.WithGCConfig(cesium.GCConfig{Rate: 1 << 20}),
			))
			stop := StartRun(db.Run)
			DeferCleanup(func() {
				h.let()
				stop()
				Expect(db.Close()).To(Succeed())
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
			)).To(Succeed())
			stamp := func(i int) telem.TimeStamp {
				return telem.TimeStamp(i+1) * telem.SecondTS
			}
			write := func(first int, values []int64) {
				GinkgoHelper()
				stamps := make([]telem.TimeStamp, len(values))
				for i := range stamps {
					stamps[i] = stamp(first + i)
				}
				keys := []cesium.ChannelKey{breakIdx, breakData}
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Clock:    breakIdx,
					Start:    stamp(first),
				}))
				MustSucceed2(w.Write(ctx, telem.MultiFrame(
					keys,
					[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(values)},
				), nil))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
			}
			values := make([]int64, n)
			for i := range values {
				values[i] = int64(i)
			}
			// The second write seals the segments of the first. The delete leaves the
			// segment of the data key above the dead share, and the clock holds more
			// than one block of live stamps, so the compactor rewrites the first and
			// encodes the second.
			write(0, values)
			write(1<<22, []int64{-1})
			Expect(db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{breakData},
				stamp(0).Range(stamp(1<<17)),
			)).To(Succeed())
			oldFile := segmentNames(disk, key)[0]
			old.Store(MustSucceed(strconv.ParseUint(
				strings.TrimSuffix(oldFile, ".seg"), 16, 64,
			)))
			// Each job of a rewrite waits a second of the clock.
			Eventually(func() bool {
				if fake.Pending() > 0 {
					fake.Advance(time.Second)
				}
				select {
				case <-h.held:
					return true
				default:
					return false
				}
			}).WithTimeout(time.Minute).Should(BeTrue())
			Consistently(func() []string { return segmentNames(disk, key) }).
				Should(ContainElement(oldFile), "the old file before the cover")
			want := slices.Concat(values[1<<17:], []int64{-1})
			kept := MustSucceed(disk.Crash(fault.CrashConfig{}))
			re, closeRe := breakOpen(ctx, kept)
			Expect(breakRead(ctx, re)).To(Equal(want))
			closeRe()
			Expect(segmentNames(kept, key)).ToNot(ContainElement(oldFile))
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
				closeRe()
				switch {
				case err != nil:
					failures = append(failures, fmt.Sprintf(
						"run key %d: %v", runKey, err,
					))
				case !slices.Equal(got, want):
					failures = append(failures, fmt.Sprintf(
						"run key %d: %d samples, %d wanted",
						runKey,
						len(got),
						len(want),
					))
				case !slices.Contains(segmentNames(image, key), oldFile):
					failures = append(failures, fmt.Sprintf(
						"run key %d: no old file %s", runKey, oldFile,
					))
				}
			}
			Expect(failures).To(BeEmpty())
		},
		Entry("in the rewrite of a data key's segment", breakData),
		Entry("in the progression encoding of a clock's segment", breakIdx),
	)
})
