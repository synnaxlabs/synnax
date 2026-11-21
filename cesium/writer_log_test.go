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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xerrors "github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer on the pointer log", func() {
	var (
		fs        xfs.FS
		db        *cesium.DB
		stop      func()
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
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
		fs = xfs.NewMem()
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
	})
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
	openWriter := func(ctx context.Context, start telem.TimeStamp) *cesium.Writer {
		GinkgoHelper()
		return MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    start,
		}))
	}

	It("Should stamp written series with one alignment and time range", func(
		ctx SpecContext,
	) {
		w := openWriter(ctx, 10*telem.SecondTS)
		written, skipped := MustSucceed2(w.Write(ctx, frame(10, 11, 12), nil))
		Expect(skipped).To(BeFalse())
		stamps, values := written.Get(idx).Series[0], written.Get(data).Series[0]
		Expect(stamps.Alignment).To(Equal(values.Alignment))
		Expect(stamps.Alignment.SampleIndex()).To(BeZero())
		want := (10 * telem.SecondTS).Range(12*telem.SecondTS + 1)
		Expect(stamps.TimeRange).To(Equal(want))
		Expect(values.TimeRange).To(Equal(want))
		next, _ := MustSucceed2(w.Write(ctx, frame(13), nil))
		Expect(
			next.Get(idx).Series[0].Alignment,
		).To(Equal(stamps.Alignment.AddSamples(3)))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
	})

	It("Should keep the channel list and the commits of a synced writer across a "+
		"reopen", func(ctx SpecContext) {
		w := openWriter(ctx, 10*telem.SecondTS)
		MustSucceed2(w.Write(ctx, frame(10, 11), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Sync()).To(Succeed())
		Expect(w.Close()).To(Succeed())
		reopen(ctx)
		Expect(MustSucceed(db.RetrieveChannel(ctx, data)).Index).To(Equal(idx))
		Expect(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    10 * telem.SecondTS,
		})).Error().To(MatchError(validate.ErrValidation))
		w = openWriter(ctx, 12*telem.SecondTS)
		MustSucceed2(w.Write(ctx, frame(12), nil))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(12*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
	})

	Describe("Reads", func() {
		read := func(ctx context.Context, key cesium.ChannelKey) telem.MultiSeries {
			GinkgoHelper()
			return MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key)
		}
		It("Should read the committed samples of each key with their alignment", func(
			ctx SpecContext,
		) {
			w := openWriter(ctx, 10*telem.SecondTS)
			written, _ := MustSucceed2(w.Write(ctx, frame(10, 11, 12), nil))
			MustSucceed2(w.Commit(ctx, nil))
			values := read(ctx, data)
			Expect(values.Series).To(HaveLen(1))
			Expect(values.Series[0]).To(telem.MatchSeriesDataV[int64](10, 11, 12))
			Expect(values.Series[0].Alignment).To(
				Equal(written.Get(data).Series[0].Alignment),
			)
			Expect(read(ctx, idx).Series[0]).To(
				telem.MatchSeriesData(telem.NewSeriesSecondsTSV(10, 11, 12)),
			)
			Expect(w.Close()).To(Succeed())
		})
		It("Should not read samples that are not committed", func(ctx SpecContext) {
			w := openWriter(ctx, 10*telem.SecondTS)
			MustSucceed2(w.Write(ctx, frame(10, 11), nil))
			Expect(read(ctx, data).Series).To(BeEmpty())
			Expect(w.Close()).To(Succeed())
		})
		It("Should read the commits of a synced writer after a reopen", func(
			ctx SpecContext,
		) {
			w := openWriter(ctx, 10*telem.SecondTS)
			MustSucceed2(w.Write(ctx, frame(10, 11), nil))
			MustSucceed2(w.Commit(ctx, nil))
			MustSucceed2(w.Write(ctx, frame(12), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			reopen(ctx)
			values := read(ctx, data)
			Expect(values.Series).To(HaveLen(1))
			Expect(values.Series[0]).To(telem.MatchSeriesDataV[int64](10, 11, 12))
		})
		It("Should read the stretches of two writers in time order", func(
			ctx SpecContext,
		) {
			for _, start := range []telem.TimeStamp{20, 10} {
				w := openWriter(ctx, start*telem.SecondTS)
				MustSucceed2(w.Write(ctx, frame(start, start+1), nil))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
			}
			values := read(ctx, data)
			Expect(values.Series).To(HaveLen(2))
			Expect(values.Series[0]).To(telem.MatchSeriesDataV[int64](10, 11))
			Expect(values.Series[1]).To(telem.MatchSeriesDataV[int64](20, 21))
			Expect(values.Series[0].Alignment.DomainIndex()).ToNot(
				Equal(values.Series[1].Alignment.DomainIndex()),
			)
		})
		It("Should read a channel of variable density", func(ctx SpecContext) {
			text := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: text, Name: "text", Index: idx, DataType: telem.StringT,
			})).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{idx, text},
				Start:    10 * telem.SecondTS,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(
				[]cesium.ChannelKey{idx, text},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11),
					telem.NewSeriesV("hello", "world"),
				},
			), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			Expect(read(ctx, text).Series[0]).To(
				telem.MatchSeriesDataV("hello", "world"),
			)
		})
	})

	Describe("Deletes", func() {
		commit := func(ctx context.Context) {
			GinkgoHelper()
			w := openWriter(ctx, 10*telem.SecondTS)
			MustSucceed2(w.Write(ctx, frame(10, 11), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
		}
		segments := func(key cesium.ChannelKey) []string {
			GinkgoHelper()
			ok := MustSucceed(fs.Exists(strconv.Itoa(int(key))))
			if !ok {
				return nil
			}
			var names []string
			for _, info := range MustSucceed(fs.List(strconv.Itoa(int(key)))) {
				if strings.HasSuffix(info.Name(), ".seg") {
					names = append(names, info.Name())
				}
			}
			return names
		}
		It("Should drop a key, its samples, and its files, across a reopen", func(
			ctx SpecContext,
		) {
			commit(ctx)
			Expect(segments(data)).To(HaveLen(1))
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			Expect(db.Drop([]cesium.ChannelKey{data})).To(Succeed())
			Expect(db.RetrieveChannel(ctx, data)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
			Expect(fs.Exists(strconv.Itoa(int(data)))).To(BeFalse())
			reopen(ctx)
			Expect(db.RetrieveChannel(ctx, data)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
			Expect(MustSucceed(db.RetrieveChannel(ctx, idx)).Key).To(Equal(idx))
		})
		It("Should keep an index that a channel outside the drop uses", func(
			ctx SpecContext,
		) {
			commit(ctx)
			Expect(db.Drop([]cesium.ChannelKey{idx})).To(Succeed())
			Expect(MustSucceed(db.RetrieveChannel(ctx, idx)).Key).To(Equal(idx))
			Expect(segments(idx)).To(HaveLen(1))
		})
		It("Should keep a key with samples and no record, and drop one with an intent",
			func(ctx SpecContext) {
				commit(ctx)
				Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
				reopen(ctx)
				kept, highest := MustSucceed2(db.Sweep(ctx, nil))
				Expect(kept).To(ConsistOf(cesium.KeptChannel{
					Key:  idx,
					Size: 16,
				}))
				Expect(highest).To(Equal(idx))
				Expect(segments(data)).To(BeEmpty())
				Expect(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{idx},
					Start:    20 * telem.SecondTS,
				})).Error().To(MatchError(cesium.ErrChannelNotFound))
			},
		)
		It("Should clear the intent of a key that keeps its record", func(
			ctx SpecContext,
		) {
			commit(ctx)
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			reopen(ctx)
			MustSucceed2(db.Sweep(ctx, []cesium.ChannelKey{idx, data}))
			reopen(ctx)
			kept, _ := MustSucceed2(db.Sweep(ctx, []cesium.ChannelKey{idx}))
			Expect(kept).To(ConsistOf(cesium.KeptChannel{
				Key:   data,
				Index: idx,
				Size:  16,
			}))
		})
		It("Should refuse a delete of a time range", func(ctx SpecContext) {
			Expect(db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{data},
				telem.TimeRangeMax,
			)).To(MatchError(xerrors.ErrUnsupported))
		})
	})

	It("Should refuse a stamp that does not rise", func(ctx SpecContext) {
		w := openWriter(ctx, 10*telem.SecondTS)
		MustSucceed2(w.Write(ctx, frame(10, 11), nil))
		Expect(w.Write(ctx, frame(11), nil)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("must be after")),
		))
		Expect(w.Close()).To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a series of another data type", func(ctx SpecContext) {
		w := openWriter(ctx, 10*telem.SecondTS)
		Expect(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10),
			telem.NewSeriesV[float32](1),
		}), nil)).Error().To(MatchError(ContainSubstring("invalid data type")))
		Expect(w.Close()).To(MatchError(validate.ErrValidation))
	})

	It("Should skip every key of a frame when the clock is skipped", func(
		ctx SpecContext,
	) {
		w := openWriter(ctx, 10*telem.SecondTS)
		written, skipped := MustSucceed2(w.Write(
			ctx,
			frame(10),
			func(k cesium.ChannelKey) bool { return k == idx },
		))
		Expect(skipped).To(BeTrue())
		Expect(written.Count()).To(BeZero())
		Expect(w.Close()).To(Succeed())
	})

	It("Should refuse a writer of two clocks, and a writer without its clock", func(
		ctx SpecContext,
	) {
		other := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: other, Name: "other", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data, other},
			Start:    10 * telem.SecondTS,
		})).Error().To(MatchError(xerrors.ErrUnsupported))
		Expect(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{data},
			Start:    10 * telem.SecondTS,
		})).Error().To(MatchError(xerrors.ErrUnsupported))
	})
})
