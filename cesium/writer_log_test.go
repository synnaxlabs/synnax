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
