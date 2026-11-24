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
	"strings"
	"sync/atomic"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// segSyncFS fails each Sync of a segment file with EIO while failing holds.
type segSyncFS struct {
	xfs.FS
	failing *atomic.Bool
}

func (f segSyncFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, ".seg") {
		return file, err
	}
	return segSyncFile{File: file, failing: f.failing}, nil
}

func (f segSyncFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return segSyncFS{FS: sub, failing: f.failing}, nil
}

type segSyncFile struct {
	xfs.File
	failing *atomic.Bool
}

func (f segSyncFile) Sync() error {
	if f.failing.Load() {
		return syscall.EIO
	}
	return f.File.Sync()
}

var _ = Describe("Pointer log writer check", func() {
	var (
		fs        xfs.FS
		failing   *atomic.Bool
		db        *cesium.DB
		stop      func()
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	open := func(ctx context.Context, opts ...cesium.Option) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(ctx, "", append([]cesium.Option{
			cesium.WithFS(fs),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		}, opts...)...))
		stop = StartRun(db.Run)
	}
	reopen := func(ctx context.Context, opts ...cesium.Option) {
		GinkgoHelper()
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx, opts...)
	}
	start := func(ctx context.Context, opts ...cesium.Option) {
		GinkgoHelper()
		failing = &atomic.Bool{}
		fs = segSyncFS{FS: xfs.NewMem(), failing: failing}
		open(ctx, opts...)
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
	// frame holds the stamps of seconds on the clock, and each second as an int64 on
	// the data channel.
	frame := func(seconds ...telem.TimeStamp) cesium.Frame {
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s)
		}
		return telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		})
	}
	openWriter := func(ctx context.Context, at telem.TimeStamp) *cesium.Writer {
		GinkgoHelper()
		return MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    at * telem.SecondTS,
		}))
	}
	// values reads the data channel and returns each value.
	values := func(ctx context.Context) []int64 {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
		var out []int64
		for _, s := range fr.Get(data).Series {
			for i := 0; i+8 <= len(s.Data); i += 8 {
				out = append(out, int64(binary.LittleEndian.Uint64(s.Data[i:])))
			}
		}
		return out
	}

	It("Should open after a failed segment sync, with only the synced samples", func(
		ctx SpecContext,
	) {
		start(ctx, cesium.WithPointerLog())
		w := openWriter(ctx, 10)
		MustSucceed2(w.Write(ctx, frame(10, 11), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Sync()).To(Succeed())
		MustSucceed2(w.Write(ctx, frame(12, 13), nil))
		MustSucceed2(w.Commit(ctx, nil))
		failing.Store(true)
		Expect(w.Sync()).To(MatchError(syscall.EIO))
		failing.Store(false)
		Expect(w.Close()).To(MatchError(syscall.EIO))
		Expect(values(ctx)).To(Equal([]int64{10, 11}))
		reopen(ctx, cesium.WithPointerLog())
		Expect(values(ctx)).To(Equal([]int64{10, 11}))
	})

	DescribeTable("Should refuse a frame that leaves out a data channel of its clock",
		func(ctx SpecContext, opts []cesium.Option) {
			start(ctx, opts...)
			w := openWriter(ctx, 10)
			MustSucceed2(w.Write(ctx, frame(10, 11), nil))
			Expect(w.Write(ctx, telem.MultiFrame(
				[]cesium.ChannelKey{idx},
				[]telem.Series{telem.NewSeriesSecondsTSV(12, 13, 14)},
			), nil)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("is missing a series for channel")),
			))
			Expect(w.Close()).To(MatchError(validate.ErrValidation))
		},
		Entry("old layout", []cesium.Option(nil)),
		Entry("pointer log", []cesium.Option{cesium.WithPointerLog()}),
	)

	It("Should drop a clock with the key that it times, across a reopen", func(
		ctx SpecContext,
	) {
		start(ctx, cesium.WithPointerLog())
		w := openWriter(ctx, 10)
		MustSucceed2(w.Write(ctx, frame(10, 11), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(db.Intend(keys)).To(Succeed())
		Expect(db.Drop(ctx, keys)).To(Succeed())
		reopen(ctx, cesium.WithPointerLog())
		for _, k := range keys {
			Expect(db.RetrieveChannel(ctx, k)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
		}
	})

	DescribeTable(
		"Should refuse a create of a clock that a drop keeps for a key that it times",
		func(ctx SpecContext, opts []cesium.Option) {
			start(ctx, opts...)
			w := openWriter(ctx, 10)
			MustSucceed2(w.Write(ctx, frame(10, 11), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			Expect(db.Intend([]cesium.ChannelKey{idx})).To(Succeed())
			Expect(db.Drop(ctx, []cesium.ChannelKey{idx})).To(Succeed())
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: idx, Name: "again", IsIndex: true, DataType: telem.TimestampT,
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("already exists")),
			))
		},
		Entry("old layout", []cesium.Option(nil)),
		Entry("pointer log", []cesium.Option{cesium.WithPointerLog()}),
	)
})
