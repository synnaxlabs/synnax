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
	// run expects the stored samples of key to hold n samples in one run, and returns
	// the alignment of its first sample.
	run := func(ctx SpecContext, key cesium.ChannelKey, n int64) telem.Alignment {
		GinkgoHelper()
		series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key).Series
		Expect(series).ToNot(BeEmpty())
		bounds := series[0].AlignmentBounds()
		for _, s := range series[1:] {
			Expect(s.AlignmentBounds().Lower).To(Equal(bounds.Upper))
			bounds.Upper = s.AlignmentBounds().Upper
		}
		Expect(bounds.Lower.AddSamples(uint32(n))).To(Equal(bounds.Upper))
		return bounds.Lower
	}
	// committedFirst opens a writer of keys, which holds the clock, and commits the
	// samples of openFirst for them.
	committedFirst := func(ctx SpecContext, keys ...cesium.ChannelKey) *cesium.Writer {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, config(keys...)))
		series := []telem.Series{telem.NewSeriesSecondsTSV(10, 11, 12, 13)}
		if len(keys) > 1 {
			series = append(series, telem.NewSeriesV[int64](1, 2, 3, 4))
		}
		mustWrite(ctx, w, telem.MultiFrame(keys, series))
		MustSucceed2(w.Commit(ctx, nil))
		return w
	}
	// next is the config of keys that starts after the samples of committedFirst.
	next := func(keys ...cesium.ChannelKey) cesium.WriterConfig {
		cfg := config(keys...)
		cfg.Start = 13*telem.SecondTS + 1
		return cfg
	}
	// carry holds the specs of a handoff that goes on in the run of the clock, which
	// both layouts run.
	carry := func() {
		It("Should go on in the run of the clock after a handoff that drops a key",
			func(ctx SpecContext) {
				first := committedFirst(ctx, clock, data)
				w, p := MustSucceed2(
					db.Handoff(ctx, first, next(clock), nil, cesium.SameRun),
				)
				Expect(p).To(BeNil())
				end, _ := MustSucceed2(w.Commit(ctx, nil))
				Expect(end).To(Equal(13*telem.SecondTS + 1))
				mustWrite(ctx, w, telem.UnaryFrame(
					clock,
					telem.NewSeriesSecondsTSV(14, 15),
				))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
				run(ctx, clock, 6)
			})

		It("Should go on in the run of the clock after a handoff that adds a key, "+
			"with the alignments of the clock", func(ctx SpecContext) {
			first := committedFirst(ctx, clock)
			w, _ := MustSucceed2(
				db.Handoff(ctx, first, next(clock, data), nil, cesium.SameRun),
			)
			mustWrite(ctx, w, telem.MultiFrame(
				[]cesium.ChannelKey{clock, data},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(14, 15),
					telem.NewSeriesV[int64](5, 6),
				},
			))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			Expect(run(ctx, data, 2)).To(Equal(run(ctx, clock, 6).AddSamples(4)))
		})

		It(
			"Should start a new run after a handoff in a new run",
			func(ctx SpecContext) {
				first := committedFirst(ctx, clock)
				w, _ := MustSucceed2(
					db.Handoff(ctx, first, next(clock), nil, cesium.NewRun),
				)
				mustWrite(
					ctx,
					w,
					telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)),
				)
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
				series := MustSucceed(
					db.Read(ctx, telem.TimeRangeMax, clock),
				).Get(clock).
					Series
				Expect(series).To(HaveLen(2))
				Expect(series[1].Alignment.DomainIndex()).
					ToNot(Equal(series[0].Alignment.DomainIndex()))
			},
		)

		It(
			"Should refuse a handoff in the same run that starts in data another "+
				"writer stored",
			func(ctx SpecContext) {
				other := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{clock},
					Clock:    clock,
					Start:    30 * telem.SecondTS,
				}))
				mustWrite(
					ctx,
					other,
					telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(30, 31)),
				)
				MustSucceed2(other.Commit(ctx, nil))
				Expect(other.Close()).To(Succeed())
				cfg := config(clock)
				cfg.Start = 30 * telem.SecondTS
				Expect(
					db.Handoff(
						ctx,
						committedFirst(ctx, clock),
						cfg,
						nil,
						cesium.SameRun,
					),
				).
					Error().
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(
							"cannot open writer because there is already data in the "+
								"writer's time range",
						)),
					))
				Expect(stored(ctx, clock)).To(Equal(int64(6)))
			},
		)

		It("Should hand off in the same run before data that another writer stored",
			func(ctx SpecContext) {
				other := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{clock},
					Clock:    clock,
					Start:    30 * telem.SecondTS,
				}))
				mustWrite(ctx, other, telem.UnaryFrame(
					clock,
					telem.NewSeriesSecondsTSV(30, 31),
				))
				MustSucceed2(other.Commit(ctx, nil))
				Expect(other.Close()).To(Succeed())
				first := committedFirst(ctx, clock)
				w, _ := MustSucceed2(
					db.Handoff(ctx, first, next(clock), nil, cesium.SameRun),
				)
				Expect(w.Close()).To(Succeed())
			})

		It("Should refuse an unknown handoff run", func(ctx SpecContext) {
			Expect(db.Handoff(ctx, committedFirst(ctx, clock), next(clock), nil, 2)).
				Error().To(MatchError(ContainSubstring("unknown handoff run 2")))
		})

		It("Should refuse a handoff that changes whether the writer stores samples",
			func(ctx SpecContext) {
				cfg := next(clock)
				cfg.Unstored = true
				Expect(
					db.Handoff(
						ctx,
						committedFirst(ctx, clock),
						cfg,
						nil,
						cesium.NewRun,
					),
				).Error().
					To(MatchError(ContainSubstring("keeps whether it stores samples")))
				Expect(stored(ctx, clock)).To(Equal(int64(4)))
			})
	}
	carry()

	It("Should move the uncommitted clock samples into the new writer", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
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
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
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
			w, p := MustSucceed2(
				db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
			)
			back, none := MustSucceed2(
				db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun),
			)
			Expect(none).To(BeNil())
			Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
			Expect(stored(ctx, clock)).To(Equal(int64(4)))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
			Expect(back.Close()).To(Succeed())
		},
	)

	It("Should commit the pending samples alone when they come back after another "+
		"writer committed their stamps", func(ctx SpecContext) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
		mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
		MustSucceed2(w.Commit(ctx, nil))
		cfg := config(clock, data)
		cfg.Start = 14*telem.SecondTS + 1
		back, _ := MustSucceed2(db.Handoff(ctx, w, cfg, p, cesium.NewRun))
		Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(13*telem.SecondTS + 1))
		Expect(stored(ctx, clock)).To(Equal(int64(5)))
		Expect(stored(ctx, data)).To(Equal(int64(4)))
		Expect(back.Close()).To(Succeed())
	})

	It("Should start a new run when a handoff adopts samples on stamps that the "+
		"clock committed", func(ctx SpecContext) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
		mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
		MustSucceed2(w.Commit(ctx, nil))
		cfg := config(clock, data)
		cfg.Start = 14*telem.SecondTS + 1
		back, _ := MustSucceed2(db.Handoff(ctx, w, cfg, p, cesium.NewRun))
		MustSucceed2(back.Commit(ctx, nil))
		mustWrite(ctx, back, telem.MultiFrame(
			[]cesium.ChannelKey{clock, data},
			[]telem.Series{telem.NewSeriesSecondsTSV(15), telem.NewSeriesV[int64](5)},
		))
		MustSucceed2(back.Commit(ctx, nil))
		Expect(back.Close()).To(Succeed())
		series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, clock)).Get(clock).Series
		Expect(series).To(HaveLen(2))
		Expect(series[0].Alignment.DomainIndex()).
			ToNot(Equal(series[1].Alignment.DomainIndex()))
		Expect(series[0].Len() + series[1].Len()).To(Equal(int64(6)))
	})

	It("Should commit the clock samples that a writer handed back with no commit",
		func(ctx SpecContext) {
			w, p := MustSucceed2(
				db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
			)
			mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
			back, _ := MustSucceed2(
				db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun),
			)
			Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
			Expect(stored(ctx, clock)).To(Equal(int64(5)))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
			Expect(back.Close()).To(Succeed())
		},
	)

	It("Should drop the pending samples on Close", func(ctx SpecContext) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
		Expect(p.Close()).To(Succeed())
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(stored(ctx, clock)).To(Equal(int64(4)))
		Expect(stored(ctx, data)).To(BeZero())
	})

	It("Should refuse a first write before the last stamp of the moved run", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
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
		w, p := MustSucceed2(db.Handoff(ctx, first, cfg, nil, cesium.NewRun))
		Expect(p).To(BeNil())
		Expect(stored(ctx, data)).To(Equal(int64(4)))
		Expect(w.Close()).To(Succeed())
	})

	It("Should refuse a writer with nothing to move that starts in stored data", func(
		ctx SpecContext,
	) {
		first := openFirst(ctx)
		MustSucceed2(first.Commit(ctx, nil))
		Expect(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun)).Error().
			To(MatchError(ContainSubstring("already data in the writer's time range")))
		Expect(stored(ctx, clock)).To(Equal(int64(4)))
	})

	It("Should refuse a handoff to a writer that does not write the clock", func(
		ctx SpecContext,
	) {
		first := openFirst(ctx)
		Expect(db.Handoff(ctx, first, config(data), nil, cesium.NewRun)).Error().
			To(MatchError(validate.ErrValidation))
		Expect(first.Write(ctx, telem.Frame[cesium.ChannelKey]{}, nil)).Error().
			To(MatchError(cesium.ErrWriterClosed))
	})

	It("Should refuse a pending sample whose key the new writer does not write", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
		Expect(db.Handoff(ctx, w, config(clock), p, cesium.NewRun)).Error().
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
		carry()

		It("Should refuse a write at the last stamp of the run after a handoff",
			func(ctx SpecContext) {
				first := committedFirst(ctx, clock)
				w, _ := MustSucceed2(
					db.Handoff(ctx, first, next(clock), nil, cesium.SameRun),
				)
				Expect(w.Write(
					ctx,
					telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(13)),
					nil,
				)).Error().To(MatchError(ContainSubstring("must be after")))
				Expect(w.Close()).To(MatchError(ContainSubstring("must be after")))
				Expect(stored(ctx, clock)).To(Equal(int64(4)))
			})

		It("Should refuse to move uncommitted samples, and drop them", func(
			ctx SpecContext,
		) {
			Expect(
				db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
			).Error().
				To(
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
			w, p := MustSucceed2(db.Handoff(ctx, first, cfg, nil, cesium.NewRun))
			Expect(p).To(BeNil())
			mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(14)))
			Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
			Expect(w.Close()).To(Succeed())
			Expect(stored(ctx, clock)).To(Equal(int64(5)))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
		})
	})
})
