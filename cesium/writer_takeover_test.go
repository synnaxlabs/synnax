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
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Writer take over on the pointer log", func() {
	type sample struct {
		at   telem.TimeStamp
		size int
	}
	var (
		fs               xfs.FS
		db               *cesium.DB
		logs             *observer.ObservedLogs
		stop             func()
		idx, fixed, text cesium.ChannelKey
		keys             []cesium.ChannelKey
		written          []sample
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		ins, observed := ObservedInstrumentation(zapcore.ErrorLevel)
		logs = observed
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithInstrumentation(ins),
		))
		stop = StartRun(db.Run)
	}
	BeforeEach(func(ctx SpecContext) {
		fs, written = xfs.NewMem(), nil
		open(ctx)
		idx, fixed, text = GenerateChannelKey(), GenerateChannelKey(),
			GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, fixed, text}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key: fixed, Name: "fixed", Index: idx, DataType: telem.Int64T,
			},
			cesium.Channel{
				Key: text, Name: "text", Index: idx, DataType: telem.StringT,
			},
		)).To(Succeed())
	})
	// textOf returns the variable sample at at, size bytes long.
	textOf := func(s sample) string {
		prefix := strconv.Itoa(int(s.at)) + ":"
		return prefix + strings.Repeat("x", s.size-len(prefix))
	}
	// session writes three samples a second apart from start in one writer, with
	// variable samples size bytes long, then commits and closes.
	session := func(ctx context.Context, start telem.TimeStamp, size int) {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    start,
		}))
		var (
			stamps []telem.TimeStamp
			values []int64
			texts  []string
		)
		for i := range 3 {
			s := sample{at: start + telem.TimeStamp(i)*telem.SecondTS, size: size}
			stamps, values = append(stamps, s.at), append(values, int64(s.at))
			texts, written = append(texts, textOf(s)), append(written, s)
		}
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
			telem.NewSeriesV(texts...),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	segments := func(key cesium.ChannelKey) []string {
		GinkgoHelper()
		var names []string
		for _, info := range MustSucceed(fs.List(strconv.Itoa(int(key)))) {
			if strings.HasSuffix(info.Name(), ".seg") {
				names = append(names, info.Name())
			}
		}
		return names
	}
	// expectWritten reads each written sample from index from on alone.
	expectWritten := func(ctx context.Context, from int) {
		GinkgoHelper()
		for _, s := range written[from:] {
			fr := MustSucceed(db.Read(ctx, s.at.Range(s.at+1), fixed, text))
			Expect(fr.Get(fixed).Series).To(HaveLen(1))
			Expect(fr.Get(fixed).Series[0]).To(MatchSeriesDataV(int64(s.at)))
			Expect(fr.Get(text).Series).To(HaveLen(1))
			Expect(fr.Get(text).Series[0]).To(MatchSeriesDataV(textOf(s)))
		}
	}

	It("Should append the sessions of one clock to one segment per key", func(
		ctx SpecContext,
	) {
		for i, start := range []telem.TimeStamp{10, 20, 30} {
			session(ctx, start*telem.SecondTS, 30_000+7*i)
		}
		for _, key := range keys {
			Expect(segments(key)).To(HaveLen(1))
		}
		expectWritten(ctx, 0)
		Expect(logs.Len()).To(BeZero())
	})

	It("Should read back the variable samples of each take over across a reopen",
		func(ctx SpecContext) {
			for i, start := range []telem.TimeStamp{10, 20, 30, 40} {
				session(ctx, start*telem.SecondTS, 21_000+4_099*i)
			}
			Expect(segments(text)).To(HaveLen(1))
			stop()
			Expect(db.Close()).To(Succeed())
			open(ctx)
			expectWritten(ctx, 0)
		})

	It("Should seal a segment whose tail fails its check, log the error, and start "+
		"a new segment", func(ctx SpecContext) {
		session(ctx, 10*telem.SecondTS, 100)
		names := segments(fixed)
		Expect(names).To(HaveLen(1))
		dir := MustSucceed(fs.Sub(strconv.Itoa(int(fixed))))
		f := MustSucceed(dir.Open(names[0], os.O_RDWR))
		last := MustSucceed(f.Stat()).Size() - 1
		b := make([]byte, 1)
		MustSucceed(f.ReadAt(b, last))
		b[0] ^= 0xFF
		MustSucceed(f.WriteAt(b, last))
		Expect(f.Close()).To(Succeed())
		session(ctx, 20*telem.SecondTS, 100)
		Expect(segments(fixed)).To(HaveLen(2))
		Expect(segments(idx)).To(HaveLen(1))
		Expect(segments(text)).To(HaveLen(1))
		sealed := logs.FilterMessage("sealed a held segment whose tail fails its check")
		Expect(sealed.Len()).To(Equal(1))
		Expect(sealed.All()[0].ContextMap()).To(HaveKeyWithValue(
			"key",
			uint32(fixed),
		))
		Expect(sealed.All()[0].ContextMap()["error"]).To(ContainSubstring("corrupt"))
		expectWritten(ctx, 3)
	})
})
