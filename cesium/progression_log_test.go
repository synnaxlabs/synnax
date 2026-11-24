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
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Progressions on the pointer log", func() {
	var (
		mem       xfs.FS
		fs        countFS
		rate      telem.Size
		db        *cesium.DB
		stop      func()
		logs      *observer.ObservedLogs
		idx, data cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		core, observed := observer.New(zapcore.DebugLevel)
		logs = observed
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithGCConfig(cesium.GCConfig{Rate: rate}),
			cesium.WithInstrumentation(alamos.New("cesium", alamos.WithLogger(logger))),
		))
		stop = StartRun(db.Run)
	}
	reopen := func(ctx context.Context) {
		GinkgoHelper()
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
	}
	BeforeEach(func(ctx SpecContext) {
		mem, rate = xfs.NewMem(), 1<<30
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		fs = countFS{
			FS:      mem,
			counted: strconv.FormatUint(uint64(idx), 10),
			reads:   &atomic.Int64{},
		}
	})
	JustBeforeEach(func(ctx SpecContext) {
		open(ctx)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	// files returns the sizes of the segment files of key.
	files := func(key cesium.ChannelKey) func() []int64 {
		return func() []int64 {
			var out []int64
			for _, e := range MustSucceed(mem.List(strconv.FormatUint(uint64(key), 10))) {
				if strings.HasSuffix(e.Name(), ".seg") {
					out = append(out, e.Size())
				}
			}
			return out
		}
	}
	// write writes stamps to the clock and their positions to data, per stamps a write,
	// then a last sample far later, whose writer seals the segments of the first.
	write := func(ctx context.Context, stamps []telem.TimeStamp, per int) {
		GinkgoHelper()
		keys := []cesium.ChannelKey{idx, data}
		w := MustSucceed(db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: stamps[0]},
		))
		for i := 0; i < len(stamps); i += per {
			s := stamps[i:min(i+per, len(stamps))]
			values := make([]int64, len(s))
			for j := range values {
				values[j] = int64(i + j)
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(
				keys,
				[]telem.Series{telem.NewSeries(s), telem.NewSeries(values)},
			), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
		last := stamps[len(stamps)-1] + telem.HourTS
		w = MustSucceed(
			db.OpenWriter(ctx, cesium.WriterConfig{Channels: keys, Start: last}),
		)
		MustSucceed2(w.Write(ctx, telem.MultiFrame(
			keys,
			[]telem.Series{
				telem.NewSeries([]telem.TimeStamp{last}),
				telem.NewSeries([]int64{-1}),
			},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	// read returns the stamps of the clock and the values of data in tr, with the
	// alignment of each series of data.
	read := func(ctx context.Context, tr telem.TimeRange) (
		stamps []telem.TimeStamp,
		values []int64,
		alignments []telem.Alignment,
	) {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, tr, idx, data))
		for _, s := range fr.Get(idx).Series {
			for i := range int(s.Len()) {
				stamps = append(stamps, s.ValueAt[telem.TimeStamp](i))
			}
		}
		for _, s := range fr.Get(data).Series {
			for i := range int(s.Len()) {
				values = append(values, s.ValueAt[int64](i))
			}
			alignments = append(alignments, s.Alignment)
		}
		return stamps, values, alignments
	}
	// driver returns the stamps of writes of 125 stamps 40 µs apart, with a gap before
	// each write.
	driver := func(writes int) []telem.TimeStamp {
		r := rand.New(rand.NewPCG(1, 2))
		out := make([]telem.TimeStamp, 0, 125*writes)
		x := telem.SecondTS
		for range writes {
			x += telem.TimeStamp(40_000 + 1 + r.Int64N(40_000))
			for range 125 {
				out = append(out, x)
				x += 40_000
			}
		}
		return out
	}
	// encoded reports whether the clock holds a segment file of one block and its
	// trailer, which only version 2 gives a clock of the size of these specs.
	encoded := func() bool { return slices.Contains(files(idx)(), int64(65_540)) }

	Describe("A regular clock", func() {
		var stamps []telem.TimeStamp
		// The rate holds off each encoding until a reopen raises it.
		BeforeEach(func() { stamps, rate = driver(400), 1 })

		It("Should encode, and read, search, and cut as version 1 did", func(
			ctx SpecContext,
		) {
			write(ctx, stamps, 125)
			whole := telem.TimeRangeMax
			mid := stamps[1_000].Range(stamps[30_000])
			s1, v1, a1 := read(ctx, whole)
			s2, v2, a2 := read(ctx, mid)
			Expect(encoded()).To(BeFalse())
			rate = 1 << 30
			reopen(ctx)
			Eventually(encoded).Should(BeTrue())
			s, v, a := read(ctx, whole)
			Expect(s).To(Equal(s1))
			Expect(v).To(Equal(v1))
			Expect(a).To(Equal(a1))
			s, v, a = read(ctx, mid)
			Expect(s).To(Equal(s2))
			Expect(v).To(Equal(v2))
			Expect(a).To(Equal(a2))
			Expect(v2[0]).To(BeEquivalentTo(1_000))
			cut := stamps[20_010].Range(stamps[20_050])
			Expect(
				db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, cut),
			).To(Succeed())
			_, v, _ = read(ctx, stamps[20_000].Range(stamps[20_100]))
			want := make([]int64, 0, 60)
			for i := 20_000; i < 20_100; i++ {
				if i < 20_010 || i >= 20_050 {
					want = append(want, int64(i))
				}
			}
			Expect(v).To(Equal(want))
			reopen(ctx)
			_, v, _ = read(ctx, stamps[20_000].Range(stamps[20_100]))
			Expect(v).To(Equal(want))
		})

		It("Should encode a clock with a dead middle range, and read it as before",
			func(ctx SpecContext) {
				write(ctx, stamps, 125)
				dead := stamps[10_000].Range(stamps[30_000])
				Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data, idx}, dead)).
					To(Succeed())
				s1, v1, a1 := read(ctx, telem.TimeRangeMax)
				Expect(v1).To(HaveLen(len(stamps) - 20_000 + 1))
				rate = 1 << 30
				reopen(ctx)
				Eventually(encoded).Should(BeTrue())
				s, v, a := read(ctx, telem.TimeRangeMax)
				Expect(s).To(Equal(s1))
				Expect(v).To(Equal(v1))
				Expect(a).To(Equal(a1))
				_, v, _ = read(ctx, stamps[9_990].Range(stamps[30_010]))
				Expect(v).To(HaveLen(20))
				Expect(v[10]).To(BeEquivalentTo(30_000))
			})
	})

	It("Should keep an irregular clock at version 1, and read it no more after a "+
		"reopen", func(ctx SpecContext) {
		r := rand.New(rand.NewPCG(3, 4))
		stamps := make([]telem.TimeStamp, 40_000)
		x := telem.SecondTS
		for i := range stamps {
			x += telem.TimeStamp(1 + r.Int64N(1_000_000))
			stamps[i] = x
		}
		write(ctx, stamps, 1_000)
		Eventually(func() int {
			return logs.FilterMessage("clock segment kept at version 1").Len()
		}).Should(Equal(1))
		before := files(idx)()
		Expect(before).To(ConsistOf(int64(320_016), int64(8)), "the try left no file")
		reopen(ctx)
		fs.reads.Store(0)
		Consistently(fs.reads.Load).Should(BeZero())
		Expect(files(idx)()).To(Equal(before))
	})
})
