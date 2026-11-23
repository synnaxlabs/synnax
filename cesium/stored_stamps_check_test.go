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
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// clockSyncFS fails each Sync of a segment file under dir with EIO while failing holds.
type clockSyncFS struct {
	xfs.FS
	dir     string
	failing *atomic.Bool
}

func (f clockSyncFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasPrefix(name, f.dir+"/") ||
		!strings.HasSuffix(name, ".seg") {
		return file, err
	}
	return segSyncFile{File: file, failing: f.failing}, nil
}

func (f clockSyncFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != f.dir {
		return sub, err
	}
	return segSyncFS{FS: sub, failing: f.failing}, nil
}

var _ = Describe("Check: data written against stored stamps", func() {
	var (
		fs                xfs.FS
		failing           *atomic.Bool
		db                *cesium.DB
		stop              func()
		idx, data         cesium.ChannelKey
		first, second     telem.Alignment
		clockKeys, onData []cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithPointerLog(),
		))
		stop = StartRun(db.Run)
	}
	// clock writes one stretch of stamps, at seconds, to the clock alone, and returns
	// the alignment of its first stamp.
	clock := func(ctx context.Context, seconds ...telem.TimeStamp) telem.Alignment {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: clockKeys,
			Start:    seconds[0] * telem.SecondTS,
		}))
		fr, _ := MustSucceed2(w.Write(ctx, telem.MultiFrame(
			clockKeys,
			[]telem.Series{telem.NewSeriesSecondsTSV(seconds...)},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		return fr.Get(idx).Series[0].Alignment
	}
	writer := func(ctx context.Context, at telem.TimeStamp) *cesium.Writer {
		GinkgoHelper()
		return MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: onData,
			Start:    at * telem.SecondTS,
		}))
	}
	frame := func(values ...int64) cesium.Frame {
		return telem.MultiFrame(onData, []telem.Series{telem.NewSeries(values)})
	}
	// read returns the series of key in tr.
	read := func(
		ctx context.Context,
		key cesium.ChannelKey,
		tr telem.TimeRange,
	) telem.MultiSeries {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, tr, key)).Get(key)
	}
	int64s := func(s telem.Series) []int64 {
		out := make([]int64, 0, s.Len())
		for i := 0; i+8 <= len(s.Data); i += 8 {
			out = append(out, int64(binary.LittleEndian.Uint64(s.Data[i:])))
		}
		return out
	}
	span := func(from, to telem.TimeStamp) telem.TimeRange {
		return telem.TimeRange{Start: from * telem.SecondTS, End: to * telem.SecondTS}
	}
	BeforeEach(func(ctx SpecContext) {
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		clockKeys, onData = []cesium.ChannelKey{idx}, []cesium.ChannelKey{data}
		failing = &atomic.Bool{}
		fs = clockSyncFS{
			FS:      xfs.NewMem(),
			dir:     strconv.FormatUint(uint64(idx), 10),
			failing: failing,
		}
		open(ctx)
		DeferCleanup(func() {
			stop()
			Expect(db.Close()).To(Succeed())
		})
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		first = clock(ctx, 10, 11, 12, 13)
		second = clock(ctx, 20, 21, 22, 23)
		Expect(second.DomainIndex()).ToNot(Equal(first.DomainIndex()))
	})

	It("Should stream and store the alignment of the stamp at the start of the data",
		func(ctx SpecContext) {
			w := writer(ctx, 11)
			fr, _ := MustSucceed2(w.Write(ctx, frame(1, 2), nil))
			Expect(fr.Get(data).Series[0].Alignment).
				To(Equal(telem.NewAlignment(first.DomainIndex(), 1)))
			Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(12*telem.SecondTS + 1))
			Expect(w.Close()).To(Succeed())
			series := read(ctx, data, span(10, 14)).Series
			Expect(series).To(HaveLen(1))
			Expect(int64s(series[0])).To(Equal([]int64{1, 2}))
			Expect(series[0].Alignment).
				To(Equal(telem.NewAlignment(first.DomainIndex(), 1)))
			Expect(series[0].TimeRange).To(Equal(telem.TimeRange{
				Start: 11 * telem.SecondTS,
				End:   12*telem.SecondTS + 1,
			}))
			stamps := read(ctx, idx, span(11, 13)).Series
			Expect(stamps).To(HaveLen(1))
			Expect(stamps[0].Alignment).To(Equal(series[0].Alignment))
		})

	It("Should commit data across two stretches of its clock as one series for each, "+
		"with the alignment of its stamps", func(ctx SpecContext) {
		w := writer(ctx, 12)
		MustSucceed2(w.Write(ctx, frame(1, 2, 3, 4), nil))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(21*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		series := read(ctx, data, telem.TimeRangeMax).Series
		Expect(series).To(HaveLen(2))
		Expect(int64s(series[0])).To(Equal([]int64{1, 2}))
		Expect(series[0].Alignment).
			To(Equal(telem.NewAlignment(first.DomainIndex(), 2)))
		Expect(int64s(series[1])).To(Equal([]int64{3, 4}))
		Expect(series[1].Alignment).To(Equal(second))
	})

	It("Should keep data against stored stamps across a reopen", func(ctx SpecContext) {
		w := writer(ctx, 11)
		MustSucceed2(w.Write(ctx, frame(1, 2, 3), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
		series := read(ctx, data, telem.TimeRangeMax).Series
		Expect(series).To(HaveLen(1))
		Expect(int64s(series[0])).To(Equal([]int64{1, 2, 3}))
		Expect(series[0].Alignment).
			To(Equal(telem.NewAlignment(first.DomainIndex(), 1)))
	})

	It("Should refuse data that starts between two stored stamps", func(
		ctx SpecContext,
	) {
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: onData,
			Start:    11*telem.SecondTS + 500*telem.MillisecondTS,
		}))
		_, _, err := w.Write(ctx, frame(1), nil)
		if err == nil {
			_, _, err = w.Commit(ctx, nil)
		}
		Expect(err).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("does not exist in the index")),
		))
		Expect(w.Close()).To(MatchError(validate.ErrValidation))
		Expect(read(ctx, data, telem.TimeRangeMax).Series).To(BeEmpty())
	})

	It("Should refuse a commit of more samples than the stored stamps after its start",
		func(ctx SpecContext) {
			w := writer(ctx, 22)
			_, _, err := w.Write(ctx, frame(1, 2, 3), nil)
			if err == nil {
				_, _, err = w.Commit(ctx, nil)
			}
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(w.Close()).To(MatchError(validate.ErrValidation))
			Expect(read(ctx, data, telem.TimeRangeMax).Series).To(BeEmpty())
		})

	It("Should refuse a delete of the stamps that an open writer of data writes "+
		"against, and take it once the writer closes", func(ctx SpecContext) {
		w := writer(ctx, 11)
		MustSucceed2(w.Write(ctx, frame(1), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(db.DeleteTimeRange(ctx, clockKeys, span(12, 14))).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(fmt.Sprintf(
				"delete of key %d from %d to %d reaches the claim of",
				idx,
				12*telem.SecondTS,
				14*telem.SecondTS,
			))),
		))
		MustSucceed2(w.Write(ctx, frame(2, 3), nil))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		Expect(db.DeleteTimeRange(ctx, onData, span(12, 14))).To(Succeed())
		Expect(db.DeleteTimeRange(ctx, clockKeys, span(12, 14))).To(Succeed())
		series := read(ctx, data, telem.TimeRangeMax).Series
		Expect(series).To(HaveLen(1))
		Expect(int64s(series[0])).To(Equal([]int64{1}))
	})

	It("Should hold the sync of data until its stamps are durable, and fail it when a "+
		"failed sync of the clock pulls them back", func(ctx SpecContext) {
		c := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: clockKeys,
			Start:    30 * telem.SecondTS,
		}))
		MustSucceed2(c.Write(ctx, telem.MultiFrame(
			clockKeys,
			[]telem.Series{telem.NewSeriesSecondsTSV(30, 31)},
		), nil))
		MustSucceed2(c.Commit(ctx, nil))
		w := writer(ctx, 30)
		MustSucceed2(w.Write(ctx, frame(1, 2), nil))
		MustSucceed2(w.Commit(ctx, nil))
		synced := make(chan error, 1)
		go func() { synced <- w.Sync() }()
		Consistently(synced, 200*time.Millisecond).ShouldNot(Receive())
		failing.Store(true)
		Expect(c.Sync()).To(MatchError(syscall.EIO))
		failing.Store(false)
		Eventually(synced).Should(Receive(HaveOccurred()))
		Expect(c.Close()).To(MatchError(syscall.EIO))
		Expect(w.Close()).To(HaveOccurred())
		Expect(read(ctx, data, span(30, 32)).Series).To(BeEmpty())
	})
})
