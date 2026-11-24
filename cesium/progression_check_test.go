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
	"encoding/binary"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// block is the file size of one full block of a fixed-density segment and its trailer.
const block = 65536 + 4

// readout is what a read gives for one series.
type readout struct {
	Values    []int64
	Alignment telem.Alignment
	TimeRange telem.TimeRange
}

var _ = Describe("Check: version 2 progressions", func() {
	const (
		// n stamps of 8 bytes fill 4 blocks of a clock.
		n    = 1 << 15
		step = telem.MillisecondTS
		// far is the start of the write that seals the segments before it.
		far = 1 << 24
	)
	var (
		fs        xfs.FS
		fake      *timetestutil.Clock
		db        *cesium.DB
		stop      func() error
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
		logs      *observer.ObservedLogs
	)
	stamp := func(i int) telem.TimeStamp {
		return telem.SecondTS + telem.TimeStamp(i)*step
	}
	open := func(ctx context.Context) {
		GinkgoHelper()
		core, observed := observer.New(zapcore.DebugLevel)
		logs = observed
		ins := alamos.New("cesium", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithClock(fake),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithGCConfig(cesium.GCConfig{Rate: 1 << 20}),
			cesium.WithInstrumentation(ins),
		))
		stop = GoRun(ctx, db.Run)
	}
	closeDB := func() {
		GinkgoHelper()
		Expect(stop()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	}
	// write writes the stamps of positions at, and each position as an int64 on the
	// data channel, in one writer, and returns the alignment of the first sample.
	write := func(ctx context.Context, at ...int) telem.Alignment {
		GinkgoHelper()
		stamps := make([]telem.TimeStamp, len(at))
		values := make([]int64, len(at))
		for i, p := range at {
			stamps[i], values[i] = stamp(p), int64(p)
		}
		w := MustSucceed(db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: stamps[0]},
		))
		fr, _ := MustSucceed2(w.Write(ctx, telem.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(values)},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		return fr.Get(idx).Series[0].Alignment
	}
	// positions returns 0 to count - 1.
	positions := func(count int) []int {
		out := make([]int, count)
		for p := range out {
			out[p] = p
		}
		return out
	}
	// sizes returns the file size of each segment of key.
	sizes := func(key cesium.ChannelKey) map[string]int64 {
		GinkgoHelper()
		out := map[string]int64{}
		for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
			if strings.HasSuffix(info.Name(), ".seg") {
				out[info.Name()] = info.Size()
			}
		}
		return out
	}
	hasSize := func(key cesium.ChannelKey, size int64) bool {
		for _, s := range sizes(key) {
			if s == size {
				return true
			}
		}
		return false
	}
	// settle advances the clock of the rate limit until done returns true.
	settle := func(done func() bool) {
		GinkgoHelper()
		Eventually(func() bool {
			if fake.Pending() > 0 {
				fake.Advance(time.Second)
			}
			return done()
		}).Should(BeTrue())
	}
	read := func(
		ctx context.Context,
		tr telem.TimeRange,
	) map[cesium.ChannelKey][]readout {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, tr, keys...))
		out := map[cesium.ChannelKey][]readout{}
		for _, k := range keys {
			for _, s := range fr.Get(k).Series {
				r := readout{Alignment: s.Alignment, TimeRange: s.TimeRange}
				for i := 0; i+8 <= len(s.Data); i += 8 {
					r.Values = append(
						r.Values,
						int64(binary.LittleEndian.Uint64(s.Data[i:])),
					)
				}
				out[k] = append(out[k], r)
			}
		}
		return out
	}
	// ranges are the reads that a rewrite must not change: all, a block end of the
	// clock, the last stamp, and starts and ends between stamps.
	ranges := []telem.TimeRange{
		telem.TimeRangeMax,
		{Start: stamp(100), End: stamp(200)},
		{Start: stamp(8191), End: stamp(8193)},
		{Start: stamp(n - 1), End: stamp(n)},
		{Start: stamp(10) + 1, End: stamp(20) + 1},
		{Start: stamp(2999), End: stamp(3001)},
	}
	readAll := func(ctx context.Context) []map[cesium.ChannelKey][]readout {
		GinkgoHelper()
		out := make([]map[cesium.ChannelKey][]readout, len(ranges))
		for i, tr := range ranges {
			out[i] = read(ctx, tr)
		}
		return out
	}
	// expectAligned checks that each series of the data channel before far starts at
	// the alignment of its first position, counted from a0.
	expectAligned := func(ctx context.Context, a0 telem.Alignment) {
		GinkgoHelper()
		for _, s := range read(ctx, telem.TimeRangeMax)[data] {
			if s.Values[0] < far {
				Expect(s.Alignment).To(Equal(a0.AddSamples(uint32(s.Values[0]))))
			}
		}
	}
	values := func(ctx context.Context) []int64 {
		GinkgoHelper()
		var out []int64
		for _, s := range read(ctx, telem.TimeRangeMax)[data] {
			out = append(out, s.Values...)
		}
		return out
	}
	// start opens a new database and creates its clock and data channel. Each spec
	// calls it, so that Run ends with the spec and not with a setup node.
	start := func(ctx context.Context) {
		GinkgoHelper()
		fs = xfs.NewMem()
		fake = timetestutil.NewClock(time.Unix(0, 0))
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	}

	It("Should encode a sealed clock of one step in one block, and read every range "+
		"as version 1 did, across a reopen", func(ctx SpecContext) {
		start(ctx)
		a0 := write(ctx, positions(n)...)
		write(ctx, far)
		Expect(hasSize(idx, 4*block)).To(BeTrue())
		Eventually(fake.Pending).Should(Equal(1))
		before := readAll(ctx)
		settle(func() bool { return hasSize(idx, block) && !hasSize(idx, 4*block) })
		Expect(readAll(ctx)).To(Equal(before))
		expectAligned(ctx, a0)
		closeDB()
		open(ctx)
		Expect(readAll(ctx)).To(Equal(before))
		closeDB()
	})

	It("Should encode a clock with a dead range in the middle, and read and cut it as "+
		"version 1 did", func(ctx SpecContext) {
		start(ctx)
		a0 := write(ctx, positions(n)...)
		Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: stamp(1000),
			End:   stamp(3000),
		})).To(Succeed())
		before := readAll(ctx)
		write(ctx, far)
		settle(func() bool { return hasSize(idx, block) && !hasSize(idx, 4*block) })
		// The reads before the sealing write lack its series.
		after := readAll(ctx)
		for _, byKey := range after {
			for k, rs := range byKey {
				byKey[k] = slices.DeleteFunc(rs, func(r readout) bool {
					return r.TimeRange.Start >= stamp(far)
				})
			}
		}
		Expect(after).To(Equal(before))
		expectAligned(ctx, a0)
		want := values(ctx)
		cut := telem.TimeRange{Start: stamp(5000), End: stamp(5003)}
		Expect(db.DeleteTimeRange(ctx, keys, cut)).To(Succeed())
		var kept []int64
		for _, v := range want {
			if !cut.ContainsStamp(stamp(int(v))) {
				kept = append(kept, v)
			}
		}
		Expect(values(ctx)).To(Equal(kept))
		expectAligned(ctx, a0)
		closeDB()
	})

	It("Should keep a version 2 segment through a delete, and remove it once nothing "+
		"in it is live", func(ctx SpecContext) {
		start(ctx)
		write(ctx, positions(n)...)
		write(ctx, far)
		settle(func() bool { return hasSize(idx, block) && !hasSize(idx, 4*block) })
		encoded := sizes(idx)
		Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: stamp(0),
			End:   stamp(n - 100),
		})).To(Succeed())
		for range 20 {
			fake.Advance(time.Second)
			Expect(sizes(idx)).To(Equal(encoded))
		}
		Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: stamp(0),
			End:   stamp(n),
		})).To(Succeed())
		settle(func() bool { return !hasSize(idx, block) })
		closeDB()
	})

	It("Should pad the rewrite of a version 1 segment to whole blocks", func(
		ctx SpecContext,
	) {
		start(ctx)
		const m = 1 << 17
		a0 := write(ctx, positions(m)...)
		write(ctx, far)
		Expect(hasSize(data, 16*block)).To(BeTrue())
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, telem.TimeRange{
			Start: stamp(0),
			End:   stamp(100000),
		})).To(Succeed())
		settle(func() bool { return !hasSize(data, 16*block) })
		// The live 31,072 samples fill 3.8 blocks, so the rewrite pads to 4.
		Expect(hasSize(data, 4*block)).To(BeTrue())
		Expect(values(ctx)).To(Equal(func() []int64 {
			out := make([]int64, 0, m-100000+1)
			for p := 100000; p < m; p++ {
				out = append(out, int64(p))
			}
			return append(out, far)
		}()))
		expectAligned(ctx, a0)
		closeDB()
	})

	It("Should try a clock that progressions cannot halve once, and not read it again "+
		"after a reopen", func(ctx SpecContext) {
		start(ctx)
		rng := rand.New(rand.NewPCG(3, 4))
		at := make([]int, n)
		for i := 1; i < n; i++ {
			at[i] = at[i-1] + 1 + rng.IntN(1000)
		}
		write(ctx, at...)
		write(ctx, at[n-1]+far)
		kept := sizes(idx)
		const keep = "clock segment kept at version 1"
		settle(func() bool { return logs.FilterMessage(keep).Len() == 1 })
		Expect(sizes(idx)).To(Equal(kept))
		closeDB()
		open(ctx)
		Consistently(func() int {
			if fake.Pending() > 0 {
				fake.Advance(time.Second)
			}
			return logs.FilterMessage(keep).Len()
		}, 500*time.Millisecond).Should(BeZero())
		Expect(sizes(idx)).To(Equal(kept))
		closeDB()
	})
})
