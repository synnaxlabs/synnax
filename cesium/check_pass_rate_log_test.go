// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

// This file needs 18e-2: GCConfig.Rate and the shared limiter of the background reads.

import (
	"bytes"
	"context"
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
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
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// oldBytes returns the bytes of the old files of keys on fs.
func oldBytes(fs xfs.FS, keys ...cesium.ChannelKey) int64 {
	GinkgoHelper()
	var n int64
	for _, key := range keys {
		for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
			name := info.Name()
			if strings.HasSuffix(name, ".domain") && name != "index.domain" &&
				name != "counter.domain" {
				n += info.Size()
			}
		}
	}
	return n
}

var _ = Describe("Check pass rate on the pointer log", func() {
	const (
		idx = oldIdx
		i32 = oldI32
		str = oldStr
	)
	var (
		keys = []cesium.ChannelKey{idx, i32, str}
		disk *fault.FS
		old  oldSamples
		fake *timetestutil.Clock
		logs *observer.ObservedLogs
	)
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		old = writeOld(ctx, disk)
		fake = timetestutil.NewClock(time.Unix(0, 0))
	})
	// advanced returns f, which first moves the fake clock on by d, so that a wait of
	// the pass on the clock ends.
	advanced := func(d time.Duration, f func() bool) func() bool {
		return func() bool {
			fake.Advance(d)
			return f()
		}
	}
	// open converts disk with gc and the fake clock, and runs the DB until the returned
	// close.
	open := func(ctx context.Context, gc cesium.GCConfig) (*cesium.DB, func() error) {
		GinkgoHelper()
		core, observed := observer.New(zapcore.DebugLevel)
		logs = observed
		ins := alamos.New("cesium", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(disk),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
			cesium.WithClock(fake),
			cesium.WithGCConfig(gc),
			cesium.WithInstrumentation(ins),
		))
		stop := GoRun(ctx, db.Run)
		return db, func() error { return errors.Join(stop(), db.Close()) }
	}
	// finished reports whether the pass gave each version 0 segment of head 1 its check
	// file.
	finished := func() bool {
		GinkgoHelper()
		segs := oldSegments(disk, 1)
		for _, key := range keys {
			if MustSucceed(checkFiles(disk, key))[key] != len(segs[key]) {
				return false
			}
		}
		return true
	}

	It("Should read each block of an old file at least twice under the rate of the "+
		"background reads", func(ctx SpecContext) {
		total := oldBytes(disk, keys...)
		// At a rate of half the old bytes a second, two reads of each block take at
		// least four seconds of the clock.
		db, closeDB := open(ctx, cesium.GCConfig{Rate: telem.Size(total / 2)})
		const step = 10 * time.Millisecond
		var waited time.Duration
		for {
			Eventually(func() bool { return finished() || fake.Pending() > 0 }).
				Should(BeTrue())
			if finished() {
				break
			}
			fake.Advance(step)
			waited += step
		}
		Expect(waited).To(BeNumerically(">=", 4*time.Second))
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(closeDB()).To(Succeed())
	})

	It("Should log a rise-check Tombstone that a writer's claim blocked, after "+
		"TryInterval", func(ctx SpecContext) {
		p := oldPointers(disk, idx)[0]
		zero(disk, oldName(idx, p), p.offset+16, 8)
		// A writer that starts before the stored data claims the range of the Tombstone
		// before the pass reaches it.
		db, closeDB := open(
			ctx,
			cesium.GCConfig{Rate: 1 << 30, TryInterval: time.Minute},
		)
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    1,
		}))
		held := func() bool {
			return logs.FilterMessage("a claim holds a cut of the check pass").Len() > 0
		}
		Eventually(advanced(time.Millisecond, held)).Should(BeTrue())
		blocked := old.bytes(nil)
		blocked[idx] = slices.Clone(blocked[idx])
		clear(blocked[idx][16:24])
		Consistently(dataOf).WithArguments(ctx, db, keys).
			Should(Equal(blocked), "the claim blocks the Tombstones")
		Expect(w.Close()).To(Succeed())
		want := old.without(2, int(p.size/8))
		Eventually(advanced(time.Minute, func() bool {
			got, err := dataOf(ctx, db, keys)
			return err == nil && maps.EqualFunc(got, want, bytes.Equal)
		})).Should(BeTrue())
		Expect(closeDB()).To(Succeed())
	})
})
