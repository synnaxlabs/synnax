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
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

var _ = Describe("Check: a rewrite whose Replace frame fault 5d cuts", func() {
	It("Should keep the old file, and read every live sample through it", func(
		ctx SpecContext,
	) {
		const n = 1 << 19
		disk := MustSucceed(fault.New(fault.Config{}))
		h := newFrameHold()
		fake := timetestutil.NewClock(time.Unix(0, 0))
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(frameHoldFS{FS: disk, h: h}),
			cesium.WithPointerLog(),
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
				Key: breakIdx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key: breakData, Name: "data", Index: breakIdx, DataType: telem.Int64T,
			},
			cesium.Channel{
				Key:      coverSpare,
				Name:     "spare",
				IsIndex:  true,
				DataType: telem.TimestampT,
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
		// The second write seals the segment of the first.
		write(0, values)
		write(1<<22, []int64{-1})
		// The delete leaves the segment above the dead share, and its rewrite takes
		// three jobs of 1 MiB, each of which waits a second of the clock.
		Expect(db.DeleteTimeRange(
			ctx,
			[]cesium.ChannelKey{breakData},
			stamp(0).Range(stamp(1<<17)),
		)).To(Succeed())
		Eventually(fake.Pending).Should(Equal(1))
		old := breakSegments(disk)[0]
		// A Drop returns once the log covers it, so the next frame is the Replace.
		Expect(db.Drop(ctx, []cesium.ChannelKey{coverSpare})).To(Succeed())
		h.armed.Store(true)
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
		}).Should(BeTrue())
		Consistently(func() []string { return breakSegments(disk) }).
			Should(ContainElement(old), "the old file before the cover of the Replace")
		want := slices.Concat(values[1<<17:], []int64{-1})
		kept := MustSucceed(disk.Crash(fault.CrashConfig{}))
		re, closeRe := breakOpen(ctx, kept)
		Expect(breakRead(ctx, re)).To(Equal(want))
		closeRe()
		Expect(breakSegments(kept)).ToNot(ContainElement(old))
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
				failures = append(failures, fmt.Sprintf("run key %d: %v", runKey, err))
			case !slices.Equal(got, want):
				failures = append(failures, fmt.Sprintf(
					"run key %d: %d samples, %d wanted", runKey, len(got), len(want),
				))
			case !slices.Contains(breakSegments(image), old):
				failures = append(failures, fmt.Sprintf(
					"run key %d: no old file %s", runKey, old,
				))
			}
		}
		Expect(failures).To(BeEmpty())
	})
})
