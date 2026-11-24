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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/index"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Data against stored stamps on the pointer log", func() {
	var (
		fs        xfs.FS
		db        *cesium.DB
		stop      func()
		idx, data cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(ctx, "", cesium.WithFS(fs), cesium.WithPointerLog()))
		stop = StartRun(db.Run)
	}
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	// writeClock writes one stretch of the clock with stamps at seconds.
	writeClock := func(ctx context.Context, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{idx},
			Start:    seconds[0] * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.UnaryFrame(
			idx,
			telem.NewSeriesSecondsTSV(seconds...),
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	openData := func(ctx context.Context, start telem.TimeStamp) *cesium.Writer {
		GinkgoHelper()
		return MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{data},
			Start:    start,
		}))
	}
	values := func(v ...int64) cesium.Frame {
		return telem.UnaryFrame(data, telem.NewSeriesV(v...))
	}
	read := func(ctx context.Context, key cesium.ChannelKey) []telem.Series {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key).Series
	}

	It("Should stream and store each sample with the alignment of its stamp", func(
		ctx SpecContext,
	) {
		writeClock(ctx, 10, 11, 12, 13)
		stamps := read(ctx, idx)[0]
		w := openData(ctx, 11*telem.SecondTS)
		written, skipped := MustSucceed2(w.Write(ctx, values(1, 2), nil))
		Expect(skipped).To(BeFalse())
		Expect(written.Get(data).Series).To(HaveLen(1))
		streamed := written.Get(data).Series[0]
		Expect(streamed.Alignment).To(Equal(stamps.Alignment.AddSamples(1)))
		Expect(streamed.TimeRange).To(Equal(telem.TimeRangeZero))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(12*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(1))
		Expect(stored[0]).To(MatchSeriesDataV[int64](1, 2))
		Expect(stored[0].Alignment).To(Equal(streamed.Alignment))
		Expect(stored[0].TimeRange).
			To(Equal((11 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
	})

	It("Should log one Commit for each run of stamps that a write reaches", func(
		ctx SpecContext,
	) {
		writeClock(ctx, 10)
		writeClock(ctx, 20, 21)
		runs := read(ctx, idx)
		Expect(runs).To(HaveLen(2))
		w := openData(ctx, 10*telem.SecondTS)
		written, _ := MustSucceed2(w.Write(ctx, values(1, 2, 3), nil))
		streamed := written.Get(data).Series
		Expect(streamed).To(HaveLen(2))
		Expect(streamed[0].Alignment).To(Equal(runs[0].Alignment))
		Expect(streamed[1].Alignment).To(Equal(runs[1].Alignment))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(21*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(2))
		Expect(stored[0]).To(MatchSeriesDataV[int64](1))
		Expect(stored[1]).To(MatchSeriesDataV[int64](2, 3))
		Expect(stored[0].Alignment).To(Equal(runs[0].Alignment))
		Expect(stored[1].Alignment).To(Equal(runs[1].Alignment))
	})

	It("Should commit no run of a Commit whose later run the state refuses", func(
		ctx SpecContext,
	) {
		writeClock(ctx, 10, 11)
		writeClock(ctx, 20, 21)
		bounded := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{data},
			Start:    20 * telem.SecondTS,
			Bounded:  true,
		}))
		w := openData(ctx, 10*telem.SecondTS)
		MustSucceed2(w.Write(ctx, values(1, 2, 3, 4), nil))
		Expect(w.Commit(ctx, nil)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("reaches the claim of another writer")),
		))
		Expect(read(ctx, data)).To(BeEmpty(), "the first run stays uncommitted")
		Expect(w.Close()).To(MatchError(ContainSubstring("reaches the claim")))
		Expect(bounded.Close()).To(Succeed())
	})

	It("Should go on in the run of stamps that the clock's writer extends", func(
		ctx SpecContext,
	) {
		clock := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{idx},
			Start:    10 * telem.SecondTS,
		}))
		stamps := func(seconds ...telem.TimeStamp) {
			GinkgoHelper()
			MustSucceed2(clock.Write(ctx, telem.UnaryFrame(
				idx,
				telem.NewSeriesSecondsTSV(seconds...),
			), nil))
			MustSucceed2(clock.Commit(ctx, nil))
		}
		stamps(10, 11)
		w := openData(ctx, 10*telem.SecondTS)
		written, _ := MustSucceed2(w.Write(ctx, values(1, 2), nil))
		first := written.Get(data).Series[0]
		stamps(12, 13)
		written, _ = MustSucceed2(w.Write(ctx, values(3, 4), nil))
		next := written.Get(data).Series[0]
		Expect(next.Alignment).To(Equal(first.Alignment.AddSamples(2)))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
		Expect(clock.Close()).To(Succeed())
		Expect(w.Close()).To(Succeed())
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(1))
		Expect(stored[0]).To(MatchSeriesDataV[int64](1, 2, 3, 4))
	})

	It("Should fail the first Commit, every later Commit, and Close when the start is "+
		"not a stamp", func(ctx SpecContext) {
		writeClock(ctx, 10, 11)
		w := openData(ctx, 10*telem.SecondTS+telem.SecondTS/2)
		written, _ := MustSucceed2(w.Write(ctx, values(1), nil))
		unstored := written.Get(data).Series[0].Alignment
		Expect(unstored.DomainIndex()).To(BeNumerically(">=", telem.LeadingDomain))
		Expect(unstored.SampleIndex()).To(BeZero())
		written, _ = MustSucceed2(w.Write(ctx, values(2, 3), nil))
		Expect(written.Get(data).Series[0].Alignment).To(Equal(unstored.AddSamples(1)),
			"one writer streams its samples with no stamp in one unstored number")
		missing := SatisfyAll(
			MatchError(index.ErrDiscontinuous),
			MatchError(ContainSubstring("does not exist in the index")),
		)
		Expect(w.Commit(ctx, nil)).Error().To(missing)
		Expect(w.Commit(ctx, nil)).Error().To(missing)
		Expect(w.Close()).To(missing)
		Expect(read(ctx, data)).To(BeEmpty())
	})

	It("Should fail the first Commit that reaches a sample past the last stamp, and "+
		"never store that sample once the clock holds its stamp", func(
		ctx SpecContext,
	) {
		writeClock(ctx, 10, 11)
		w := openData(ctx, 10*telem.SecondTS)
		written, _ := MustSucceed2(w.Write(ctx, values(1, 2, 3), nil))
		streamed := written.Get(data).Series
		Expect(streamed).To(HaveLen(2))
		Expect(streamed[0]).To(MatchSeriesDataV[int64](1, 2))
		Expect(streamed[1]).To(MatchSeriesDataV[int64](3))
		Expect(streamed[1].Alignment.DomainIndex()).
			To(BeNumerically(">=", telem.LeadingDomain))
		writeClock(ctx, 12)
		past := SatisfyAll(
			MatchError(index.ErrDiscontinuous),
			MatchError(ContainSubstring(
				"failed to resolve position 2 in continuous index of length 2",
			)),
		)
		Expect(w.Commit(ctx, nil)).Error().To(past)
		Expect(w.Commit(ctx, nil)).Error().To(past)
		Expect(w.Close()).To(past)
		Expect(read(ctx, data)).To(BeEmpty())
	})

	It("Should drop samples with no stamp at a Close with no Commit", func(
		ctx SpecContext,
	) {
		writeClock(ctx, 10)
		w := openData(ctx, 10*telem.SecondTS)
		MustSucceed2(w.Write(ctx, values(1, 2), nil))
		Expect(w.Close()).To(Succeed())
		Expect(read(ctx, data)).To(BeEmpty())
	})

	It("Should refuse a writer of data with no start", func(ctx SpecContext) {
		Expect(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{data},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"channel <%d>: writer needs a start to write the channel without its "+
					"index <%d>",
				data,
				idx,
			)),
		))
	})

	It("Should refuse a delete of the clock's range while a writer of data is open",
		func(ctx SpecContext) {
			writeClock(ctx, 10, 11, 12)
			w := openData(ctx, 10*telem.SecondTS)
			tr := (10 * telem.SecondTS).Range(12 * telem.SecondTS)
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{idx}, tr)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("reaches the claim of another writer")),
			))
			Expect(w.Close()).To(Succeed())
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{idx}, tr)).To(Succeed())
		})

	It("Should keep the data and its alignment across a reopen", func(
		ctx SpecContext,
	) {
		writeClock(ctx, 10, 11)
		w := openData(ctx, 10*telem.SecondTS)
		written, _ := MustSucceed2(w.Write(ctx, values(1, 2), nil))
		streamed := written.Get(data).Series[0]
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
		stored := read(ctx, data)
		Expect(stored).To(HaveLen(1))
		Expect(stored[0]).To(MatchSeriesDataV[int64](1, 2))
		Expect(stored[0].Alignment).To(Equal(streamed.Alignment))
	})
})

var _ = Describe("Data against stamps that parts hold", func() {
	It("Should place samples on parted stamps, and read them after a crash", func(
		ctx SpecContext,
	) {
		fs := xfs.NewMem()
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		idx, timed, data := GenerateChannelKey(), GenerateChannelKey(),
			GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      timed,
				Name:     "timed",
				Index:    idx,
				DataType: telem.Int64T,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		writeParted(ctx, db, fs, idx, timed, 2, 3)
		runs := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx)).Get(idx).Series
		Expect(runs).To(HaveLen(2))
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Clock:    idx,
			Channels: []cesium.ChannelKey{data},
			Start:    telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.UnaryFrame(
			data,
			telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6),
		), nil))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(6*telem.SecondTS + 1))
		Expect(w.Sync(ctx)).To(Succeed())
		crashed := xfs.NewMem()
		Expect(xfs.CopyDir(fs, crashed, "")).To(Succeed())
		Expect(w.Close()).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		db = partedDB(ctx, crashed)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		StartRun(db.Run)
		stored := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).Get(data).Series
		Expect(stored).To(HaveLen(2))
		Expect(stored[0]).To(MatchSeriesDataV[int64](1, 2, 3))
		Expect(stored[1]).To(MatchSeriesDataV[int64](4, 5, 6))
		Expect(stored[0].Alignment).To(Equal(runs[0].Alignment))
		Expect(stored[1].Alignment).To(Equal(runs[1].Alignment))
		Expect(stored[1].TimeRange).
			To(Equal((4 * telem.SecondTS).Range(6*telem.SecondTS + 1)))
	})
})
