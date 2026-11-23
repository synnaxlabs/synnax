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
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Handoff", func() {
	var (
		db          *cesium.DB
		clock, data cesium.ChannelKey
	)
	create := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      clock,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     "data",
				Index:    clock,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, fs.NewMem())
		clock, data = GenerateChannelKey(), GenerateChannelKey()
		create(ctx)
	})
	config := func(keys ...cesium.ChannelKey) cesium.WriterConfig {
		return cesium.WriterConfig{
			Channels: keys,
			Clock:    clock,
			Start:    10 * telem.SecondTS,
		}
	}
	// openFirst opens a writer of the clock and the data, and writes four samples at
	// seconds 10 to 13 with no commit.
	openFirst := func(ctx SpecContext) *cesium.Writer {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, w, telem.MultiFrame(
			[]cesium.ChannelKey{clock, data},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12, 13),
				telem.NewSeriesV[int64](1, 2, 3, 4),
			},
		))
		return w
	}
	stored := func(ctx SpecContext, key cesium.ChannelKey) int64 {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key).Len()
	}

	It("Should move the uncommitted clock samples into the new writer", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
		Expect(p).ToNot(BeNil())
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
		Expect(stored(ctx, clock)).To(Equal(int64(4)))
		Expect(stored(ctx, data)).To(BeZero())
		Expect(w.Close()).To(Succeed())
		Expect(p.Close()).To(Succeed())
	})

	It("Should commit the pending samples once the clock holds their stamps", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
		_, held := MustSucceed2(p.Commit(ctx))
		Expect(held).To(BeFalse())
		Expect(stored(ctx, data)).To(BeZero())
		MustSucceed2(w.Commit(ctx, nil))
		end, held := MustSucceed2(p.Commit(ctx))
		Expect(held).To(BeTrue())
		Expect(end).To(Equal(13*telem.SecondTS + 1))
		Expect(stored(ctx, data)).To(Equal(int64(4)))
		Expect(w.Close()).To(Succeed())
	})

	It("Should commit the clock and then the pending samples when they come back",
		func(ctx SpecContext) {
			w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
			back, none := MustSucceed2(db.Handoff(ctx, w, config(clock, data), p))
			Expect(none).To(BeNil())
			Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
			Expect(stored(ctx, clock)).To(Equal(int64(4)))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
			Expect(back.Close()).To(Succeed())
		},
	)

	It("Should commit the pending samples alone when they come back after another "+
		"writer committed their stamps", func(ctx SpecContext) {
		w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
		mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
		MustSucceed2(w.Commit(ctx, nil))
		cfg := config(clock, data)
		cfg.Start = 14*telem.SecondTS + 1
		back, _ := MustSucceed2(db.Handoff(ctx, w, cfg, p))
		Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
		Expect(stored(ctx, clock)).To(Equal(int64(5)))
		Expect(stored(ctx, data)).To(Equal(int64(4)))
		Expect(back.Close()).To(Succeed())
	})

	It("Should commit the clock samples that a writer handed back with no commit",
		func(ctx SpecContext) {
			w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
			mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
			back, _ := MustSucceed2(db.Handoff(ctx, w, config(clock, data), p))
			Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
			Expect(stored(ctx, clock)).To(Equal(int64(5)))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
			Expect(back.Close()).To(Succeed())
		},
	)

	It("Should drop the pending samples on Close", func(ctx SpecContext) {
		w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
		Expect(p.Close()).To(Succeed())
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(stored(ctx, clock)).To(Equal(int64(4)))
		Expect(stored(ctx, data)).To(BeZero())
	})

	It("Should refuse a first write before the last stamp of the moved run", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
		Expect(w.Write(
			ctx,
			telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(12)),
			nil,
		)).Error().To(MatchError(ContainSubstring("index out of order")))
		Expect(w.Close()).To(MatchError(ContainSubstring("index out of order")))
		Expect(p.Close()).To(Succeed())
	})

	It("Should return no pending samples when the writer committed them", func(
		ctx SpecContext,
	) {
		first := openFirst(ctx)
		MustSucceed2(first.Commit(ctx, nil))
		cfg := config(clock)
		cfg.Start = 13*telem.SecondTS + 1
		w, p := MustSucceed2(db.Handoff(ctx, first, cfg, nil))
		Expect(p).To(BeNil())
		Expect(stored(ctx, data)).To(Equal(int64(4)))
		Expect(w.Close()).To(Succeed())
	})

	It("Should refuse a writer with nothing to move that starts in stored data", func(
		ctx SpecContext,
	) {
		first := openFirst(ctx)
		MustSucceed2(first.Commit(ctx, nil))
		Expect(db.Handoff(ctx, first, config(clock), nil)).Error().
			To(MatchError(ContainSubstring("already data in the writer's time range")))
		Expect(stored(ctx, clock)).To(Equal(int64(4)))
	})

	It("Should refuse a handoff to a writer that does not write the clock", func(
		ctx SpecContext,
	) {
		first := openFirst(ctx)
		Expect(db.Handoff(ctx, first, config(data), nil)).Error().
			To(MatchError(validate.ErrValidation))
		Expect(first.Write(ctx, telem.Frame[cesium.ChannelKey]{}, nil)).Error().
			To(MatchError(cesium.ErrWriterClosed))
	})

	It("Should refuse a pending sample whose key the new writer does not write", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(db.Handoff(ctx, openFirst(ctx), config(clock), nil))
		Expect(db.Handoff(ctx, w, config(clock), p)).Error().
			To(MatchError(ContainSubstring("the writer lacks a key of the pending")))
	})

	Describe("On the pointer log layout", func() {
		BeforeEach(func(ctx SpecContext) {
			db = mustOpenDBOnFS(
				ctx,
				fs.NewMem(),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
			)
			create(ctx)
		})

		It("Should refuse to move uncommitted samples, and drop them", func(
			ctx SpecContext,
		) {
			Expect(db.Handoff(ctx, openFirst(ctx), config(clock), nil)).Error().To(
				SatisfyAll(
					MatchError(errors.ErrUnsupported),
					MatchError(ContainSubstring(
						"dropped the uncommitted samples of clock",
					)),
				),
			)
			Expect(stored(ctx, clock)).To(BeZero())
			Expect(stored(ctx, data)).To(BeZero())
		})

		It("Should open the next writer when there is nothing to move", func(
			ctx SpecContext,
		) {
			first := openFirst(ctx)
			MustSucceed2(first.Commit(ctx, nil))
			cfg := config(clock)
			cfg.Start = 13*telem.SecondTS + 1
			w, p := MustSucceed2(db.Handoff(ctx, first, cfg, nil))
			Expect(p).To(BeNil())
			mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
			Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
			Expect(w.Close()).To(Succeed())
			Expect(stored(ctx, clock)).To(Equal(int64(5)))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
		})
	})
})
