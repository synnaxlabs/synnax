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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xerrors "github.com/synnaxlabs/x/errors"
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
		opens            *segmentOpens
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
		opens, written = &segmentOpens{}, nil
		fs = segmentOpenFS{FS: xfs.NewMem(), opens: opens}
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

	It("Should close the segments it took over when a later take over fails", func(
		ctx SpecContext,
	) {
		session(ctx, 10*telem.SecondTS, 100)
		before := opens.open.Load()
		opens.err = xerrors.New("open failed")
		opens.resumes.Store(0)
		opens.failAt.Store(int64(len(keys)))
		Expect(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    20 * telem.SecondTS,
		})).Error().To(MatchError(opens.err))
		Expect(opens.open.Load()).To(Equal(before))
	})
})

// segmentOpens counts the open segment files of a segmentOpenFS, and the opens of
// existing segment files, the open numbered failAt of which fails with err.
type segmentOpens struct {
	open, resumes, failAt atomic.Int64
	err                   error
}

type segmentOpenFS struct {
	xfs.FS
	opens *segmentOpens
}

func (f segmentOpenFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return segmentOpenFS{FS: sub, opens: f.opens}, nil
}

func (f segmentOpenFS) Open(name string, flag int) (xfs.File, error) {
	if !strings.HasSuffix(name, ".seg") {
		return f.FS.Open(name, flag)
	}
	if flag&os.O_CREATE == 0 && f.opens.resumes.Add(1) == f.opens.failAt.Load() {
		return nil, f.opens.err
	}
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	f.opens.open.Add(1)
	return segmentOpenFile{File: file, opens: f.opens}, nil
}

type segmentOpenFile struct {
	xfs.File
	opens *segmentOpens
}

func (f segmentOpenFile) Close() error {
	f.opens.open.Add(-1)
	return f.File.Close()
}
