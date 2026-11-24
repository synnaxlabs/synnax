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
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = DescribeTableSubtree("Handoff", func(pointerLog bool) {
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
		var opts []cesium.Option
		if pointerLog {
			opts = append(
				opts,
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
			)
		}
		db = mustOpenDBOnFS(ctx, fs.NewMem(), opts...)
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

	It("Should keep the stored data of a channel that a handoff drops after a "+
		"skip and adds back", func(ctx SpecContext) {
		w := committedFirst(ctx, clock, data)
		keys := []cesium.ChannelKey{clock, data}
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(14, 15),
			telem.NewSeriesV[int64](5, 6),
		}), func(k cesium.ChannelKey) bool { return k == data }))
		w, p := MustSucceed2(db.Handoff(ctx, w, next(clock), nil, cesium.SameRun))
		Expect(p).To(BeNil())
		end, _ := MustSucceed2(w.Commit(ctx, nil))
		Expect(end).To(Equal(15*telem.SecondTS + 1))
		Expect(w.Sync(ctx)).To(Succeed())
		other := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{data},
			Clock:    clock,
			Start:    14 * telem.SecondTS,
		}))
		mustWrite(
			ctx,
			other,
			telem.UnaryFrame(data, telem.NewSeriesV[int64](50, 60)),
		)
		MustSucceed2(other.Commit(ctx, nil))
		Expect(other.Close()).To(Succeed())
		cfg := config(clock, data)
		cfg.Start = 15*telem.SecondTS + 1
		w, p = MustSucceed2(db.Handoff(ctx, w, cfg, nil, cesium.SameRun))
		Expect(p).To(BeNil())
		mustWrite(ctx, w, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(16),
			telem.NewSeriesV[int64](7),
		}))
		end, _ = MustSucceed2(w.Commit(ctx, nil))
		Expect(end).To(Equal(16*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		got := make([]int64, 0, 7)
		for _, s := range MustSucceed(
			db.Read(ctx, telem.TimeRangeMax, data),
		).Get(data).Series {
			got = append(got, s.Unmarshal[int64]()...)
		}
		Expect(got).To(Equal([]int64{1, 2, 3, 4, 50, 60, 7}))
	})

	// On the pointer log, the skipped key commits over the stamps of its own samples.
	if !pointerLog {
		DescribeTable(
			"Should keep a run that a skipped data channel held back across a handoff "+
				"that drops the channel",
			func(ctx SpecContext, run cesium.HandoffRun) {
				w := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
				keys := []cesium.ChannelKey{clock, data}
				mustWrite(ctx, w, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(10, 11),
					telem.NewSeriesV[int64](1, 2),
				}))
				MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(12, 13),
					telem.NewSeriesV[int64](3, 4),
				}), func(k cesium.ChannelKey) bool { return k == data }))
				next, p := MustSucceed2(db.Handoff(ctx, w, config(clock), nil, run))
				end, _ := MustSucceed2(next.Commit(ctx, nil))
				Expect(end).To(Equal(13*telem.SecondTS + 1))
				Expect(p).ToNot(BeNil())
				pending, committed := MustSucceed2(p.Commit(ctx))
				Expect(committed).To(BeTrue())
				Expect(pending).To(Equal(13*telem.SecondTS + 1))
				Expect(next.Close()).To(Succeed())
				Expect(stored(ctx, clock)).To(Equal(int64(4)))
				Expect(stored(ctx, data)).To(Equal(int64(2)))
			},
			Entry("a new run", cesium.NewRun),
			Entry("the same run", cesium.SameRun),
		)
	}

	It("Should return no pending samples once a commit covers a run that a skipped "+
		"channel held back", func(ctx SpecContext) {
		w := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		keys := []cesium.ChannelKey{clock, data}
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11),
			telem.NewSeriesV[int64](1, 2),
		}), func(k cesium.ChannelKey) bool { return k == data }))
		mustWrite(ctx, w, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(12, 13),
			telem.NewSeriesV[int64](3, 4),
		}))
		MustSucceed2(w.Commit(ctx, nil))
		cfg := config(clock)
		cfg.Start = 13*telem.SecondTS + 1
		next, p := MustSucceed2(db.Handoff(ctx, w, cfg, nil, cesium.NewRun))
		Expect(p).To(BeNil())
		Expect(next.Close()).To(Succeed())
	})

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
		if pointerLog {
			_, held = MustSucceed2(p.Commit(ctx))
			Expect(held).To(BeFalse(), "the pending waits until its stamps are durable")
			Expect(w.Sync(ctx)).To(Succeed())
		}
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

	// On the pointer log, the stretch of the clock goes on instead.
	if !pointerLog {
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
				[]telem.Series{
					telem.NewSeriesSecondsTSV(15),
					telem.NewSeriesV[int64](5),
				},
			))
			MustSucceed2(back.Commit(ctx, nil))
			Expect(back.Close()).To(Succeed())
			series := MustSucceed(
				db.Read(ctx, telem.TimeRangeMax, clock),
			).Get(clock).
				Series
			Expect(series).To(HaveLen(2))
			Expect(series[0].Alignment.DomainIndex()).
				ToNot(Equal(series[1].Alignment.DomainIndex()))
			Expect(series[0].Len() + series[1].Len()).To(Equal(int64(6)))
		})
	}

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

	It("Should let a writer claim a pending key after the stamps of its samples", func(
		ctx SpecContext,
	) {
		w, p := MustSucceed2(
			db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
		)
		after := config(data)
		after.Start = 13*telem.SecondTS + 1
		next := MustSucceed(db.OpenWriter(ctx, after))
		Expect(next.Close()).To(Succeed())
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		_, held := MustSucceed2(p.Commit(ctx))
		Expect(held).To(BeTrue())
		Expect(stored(ctx, data)).To(Equal(int64(4)))
	})

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

	if pointerLog {
		It("Should refuse a write at the last stamp of the run after a handoff", func(
			ctx SpecContext,
		) {
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
	}

	Describe("A writer of data alone, handed off with no channels", func() {
		// alone opens a writer of the data alone from second 10.
		alone := func(ctx SpecContext, unstored bool) *cesium.Writer {
			GinkgoHelper()
			cfg := config(data)
			cfg.Unstored = unstored
			return MustSucceed(db.OpenWriter(ctx, cfg))
		}
		values := func() cesium.Frame {
			return telem.UnaryFrame(data, telem.NewSeriesV[int64](1, 2, 3, 4))
		}
		// stamps commits the clock from second 10 to 13, and makes it durable.
		stamps := func(ctx SpecContext) {
			GinkgoHelper()
			Expect(committedFirst(ctx, clock).Close()).To(Succeed())
		}

		It("Should close the writer, and return no pending samples when it has none",
			func(ctx SpecContext) {
				w, p := MustSucceed2(db.Handoff(
					ctx,
					alone(ctx, false),
					cesium.WriterConfig{Clock: clock},
					nil,
					cesium.NewRun,
				))
				Expect(w).To(BeNil())
				Expect(p).To(BeNil())
				Expect(committedFirst(ctx, clock, data).Close()).To(Succeed())
				Expect(stored(ctx, data)).To(Equal(int64(4)))
			})

		It("Should return the uncommitted samples as pending samples that commit on "+
			"the stamps of the clock", func(ctx SpecContext) {
			stamps(ctx)
			first := alone(ctx, false)
			mustWrite(ctx, first, values())
			w, p := MustSucceed2(db.Handoff(
				ctx,
				first,
				cesium.WriterConfig{Clock: clock},
				nil,
				cesium.NewRun,
			))
			Expect(w).To(BeNil())
			Expect(stored(ctx, data)).To(BeZero())
			end, held := MustSucceed2(p.Commit(ctx))
			Expect(held).To(BeTrue())
			Expect(end).To(Equal(13*telem.SecondTS + 1))
			Expect(stored(ctx, data)).To(Equal(int64(4)))
		})

		It("Should let a writer claim the key after the stamps of the pending samples",
			func(ctx SpecContext) {
				stamps(ctx)
				first := alone(ctx, false)
				mustWrite(ctx, first, values())
				_, p := MustSucceed2(db.Handoff(
					ctx,
					first,
					cesium.WriterConfig{Clock: clock},
					nil,
					cesium.NewRun,
				))
				after := config(data)
				after.Start = 13*telem.SecondTS + 1
				Expect(MustSucceed(db.OpenWriter(ctx, after)).Close()).To(Succeed())
				_, held := MustSucceed2(p.Commit(ctx))
				Expect(held).To(BeTrue())
				Expect(stored(ctx, data)).To(Equal(int64(4)))
			})

		It("Should commit the pending samples through the writer that adopts them",
			func(ctx SpecContext) {
				stamps(ctx)
				first := alone(ctx, false)
				mustWrite(ctx, first, values())
				_, p := MustSucceed2(db.Handoff(
					ctx,
					first,
					cesium.WriterConfig{Clock: clock},
					nil,
					cesium.NewRun,
				))
				taker := MustSucceed(db.OpenWriter(ctx, next(clock)))
				w, _ := MustSucceed2(
					db.Handoff(ctx, taker, next(clock, data), p, cesium.NewRun),
				)
				mustWrite(ctx, w, telem.MultiFrame(
					[]cesium.ChannelKey{clock, data},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(14),
						telem.NewSeriesV[int64](5),
					},
				))
				Expect(
					MustSucceed2(w.Commit(ctx, nil)),
				).To(Equal(14*telem.SecondTS + 1))
				Expect(w.Close()).To(Succeed())
				Expect(stored(ctx, clock)).To(Equal(int64(5)))
				Expect(stored(ctx, data)).To(Equal(int64(5)))
			})

		It("Should return no pending samples for a writer that stores none", func(
			ctx SpecContext,
		) {
			stamps(ctx)
			first := alone(ctx, true)
			mustWrite(ctx, first, values())
			w, p := MustSucceed2(db.Handoff(
				ctx,
				first,
				cesium.WriterConfig{Clock: clock, Unstored: true},
				nil,
				cesium.NewRun,
			))
			Expect(w).To(BeNil())
			Expect(p).To(BeNil())
			Expect(stored(ctx, data)).To(BeZero())
		})

		It("Should refuse pending samples to adopt", func(ctx SpecContext) {
			other := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      other,
				Name:     "other",
				Index:    clock,
				DataType: telem.Int64T,
			})).To(Succeed())
			w, p := MustSucceed2(
				db.Handoff(ctx, openFirst(ctx), config(clock), nil, cesium.NewRun),
			)
			alone := MustSucceed(db.OpenWriter(ctx, config(other)))
			Expect(db.Handoff(
				ctx,
				alone,
				cesium.WriterConfig{Clock: clock},
				p,
				cesium.NewRun,
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("adopts no pending samples")),
			))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			Expect(stored(ctx, data)).To(BeZero(), "the refusal drops the pending")
		})

		It("Should refuse no channels for a writer of the clock", func(
			ctx SpecContext,
		) {
			Expect(db.Handoff(
				ctx,
				openFirst(ctx),
				cesium.WriterConfig{Clock: clock},
				nil,
				cesium.NewRun,
			)).Error().To(MatchError(validate.ErrValidation))
			Expect(stored(ctx, clock)).To(BeZero())
		})
	})
}, Entry("old layout", false), Entry("pointer log", true))

var _ = Describe("Handoff on the pointer log", func() {
	var (
		fsys        fs.FS
		db          *cesium.DB
		stop        func()
		clock, data cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = DeferClose(openDBOnFS(ctx, fsys, cesium.WithPointerLog()))
		stop = StartRun(db.Run)
	}
	BeforeEach(func(ctx SpecContext) {
		fsys = fs.NewMem()
		open(ctx)
		clock, data = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: clock, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     "data",
				Index:    clock,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
	})
	config := func(keys ...cesium.ChannelKey) cesium.WriterConfig {
		return cesium.WriterConfig{
			Channels: keys,
			Clock:    clock,
			Start:    10 * telem.SecondTS,
		}
	}
	// both returns a frame of the clock at seconds and of data with values from 1.
	both := func(seconds ...telem.TimeStamp) cesium.Frame {
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s) - 9
		}
		return telem.MultiFrame(
			[]cesium.ChannelKey{clock, data},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(seconds...),
				telem.NewSeriesV(values...),
			},
		)
	}
	stored := func(ctx SpecContext) []int64 {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, clock, data))
		out := make([]int64, 0, fr.Get(data).Len())
		for _, s := range fr.Get(data).Series {
			out = append(out, s.Unmarshal[int64]()...)
		}
		Expect(fr.Get(clock).Len()).To(Equal(int64(len(out))))
		return out
	}

	DescribeTable("Should go on with the stretch of the clock across a handoff that "+
		"carries its uncommitted samples", func(ctx SpecContext, run cesium.HandoffRun) {
		w := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		before := mustWrite(ctx, w, both(10, 11)).Get(clock).Series[0].Alignment
		next, p := MustSucceed2(db.Handoff(ctx, w, config(clock), nil, run))
		after := mustWrite(ctx, next, telem.UnaryFrame(
			clock,
			telem.NewSeriesSecondsTSV(12),
		)).Get(clock).Series[0].Alignment
		Expect(after).To(Equal(before.AddSamples(2)), "the clock's stretch goes on")
		Expect(MustSucceed2(next.Commit(ctx, nil))).To(Equal(12*telem.SecondTS + 1))
		stamps := MustSucceed(db.Read(ctx, telem.TimeRangeMax, clock)).Get(clock).Series
		Expect(stamps).To(HaveLen(1))
		Expect(next.Close()).To(Succeed())
		Expect(p.Close()).To(Succeed())
	},
		Entry("in a new run, as the old layout does", cesium.NewRun),
		Entry("in the same run", cesium.SameRun),
	)

	It("Should commit the samples written after pending samples that come back on "+
		"committed stamps", func(ctx SpecContext) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11, 12, 13))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun))
		MustSucceed2(w.Commit(ctx, nil))
		back, _ := MustSucceed2(
			db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun),
		)
		mustWrite(ctx, back, both(14))
		Expect(MustSucceed2(back.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
		Expect(stored(ctx)).To(Equal([]int64{1, 2, 3, 4, 5}))
		Expect(back.Close()).To(Succeed())
	})

	It("Should commit the pending samples of each run that they come back to", func(
		ctx SpecContext,
	) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11, 12, 13))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun))
		// The next run starts a new stretch, so the two runs do not touch.
		Expect(w.EndStretch(ctx)).To(Succeed())
		w, _ = MustSucceed2(db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun))
		mustWrite(ctx, w, both(14))
		w, p = MustSucceed2(db.Handoff(ctx, w, config(clock), nil, cesium.NewRun))
		w, _ = MustSucceed2(db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
		Expect(stored(ctx)).To(Equal([]int64{1, 2, 3, 4, 5}))
		Expect(w.Close()).To(Succeed())
	})

	It("Should refuse a stamp before the pending samples that come back after the "+
		"stretch ended", func(ctx SpecContext) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11, 12, 13))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun))
		Expect(w.EndStretch(ctx)).To(Succeed())
		back, _ := MustSucceed2(
			db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun),
		)
		Expect(back.Write(ctx, both(12), nil)).Error().
			To(MatchError(ContainSubstring("timestamp 1970-01-01T00:00:12Z at sample 0 must be after")))
		Expect(back.Close()).To(MatchError(ContainSubstring("must be after")))
	})

	It("Should log the commits of pending samples that wait for durable stamps "+
		"through the writer that adopts them", func(ctx SpecContext) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11, 12, 13))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun))
		MustSucceed2(w.Commit(ctx, nil))
		_, held := MustSucceed2(p.Commit(ctx))
		Expect(held).To(BeFalse(), "the stamps are not durable")
		back, _ := MustSucceed2(
			db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun),
		)
		Expect(back.Close()).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
		Expect(stored(ctx)).To(Equal([]int64{1, 2, 3, 4}))
	})

	It("Should write the keys of pending samples that a reopened writer adopts", func(
		ctx SpecContext,
	) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11, 12, 13))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun))
		MustSucceed2(w.Commit(ctx, nil))
		reopen := config(clock)
		reopen.Start = 13*telem.SecondTS + 1
		w, _ = MustSucceed2(db.Handoff(ctx, w, reopen, nil, cesium.NewRun))
		w, _ = MustSucceed2(db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun))
		mustWrite(ctx, w, both(14))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(14*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		Expect(stored(ctx)).To(Equal([]int64{1, 2, 3, 4, 5}))
	})

	// seconds returns the stamp of each stored sample of data, in seconds.
	seconds := func(ctx SpecContext) []int64 {
		GinkgoHelper()
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, clock, data))
		out := make([]int64, 0, fr.Get(data).Len())
		for _, s := range fr.Get(data).Series {
			for i := range s.Len() {
				at := s.Alignment.AddSamples(uint32(i))
				for _, c := range fr.Get(clock).Series {
					if b := c.AlignmentBounds(); b.Contains(at) {
						t := c.Unmarshal[telem.TimeStamp]()[at.SampleIndex()-
							b.Lower.SampleIndex()]
						out = append(out, int64(t/telem.SecondTS))
					}
				}
			}
		}
		return out
	}

	DescribeTable("Should stamp the samples of a key that a handoff adds or brings "+
		"back with the stamps they were written at", func(
		ctx SpecContext,
		adopt bool,
		run cesium.HandoffRun,
	) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, run))
		mustWrite(ctx, w, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(12)))
		if !adopt {
			Expect(p.Close()).To(Succeed())
			p = nil
		}
		w, _ = MustSucceed2(db.Handoff(ctx, w, config(clock, data), p, run))
		mustWrite(ctx, w, both(13))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		expected := []int64{13}
		if adopt {
			expected = []int64{10, 11, 13}
		}
		Expect(seconds(ctx)).To(Equal(expected))
	},
		Entry("an added key, in a new run", false, cesium.NewRun),
		Entry("an added key, in the same run", false, cesium.SameRun),
		Entry("an adopted key, in a new run", true, cesium.NewRun),
		Entry("an adopted key, in the same run", true, cesium.SameRun),
	)

	It("Should commit a key that a write skips only over the stamps of its samples",
		func(ctx SpecContext) {
			w := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
			mustWrite(ctx, w, both(10, 11))
			_, skipped := MustSucceed2(w.Write(
				ctx,
				both(12, 13),
				func(k cesium.ChannelKey) bool { return k == data },
			))
			Expect(skipped).To(BeTrue())
			w, p := MustSucceed2(db.Handoff(ctx, w, config(clock), nil, cesium.NewRun))
			after := config(data)
			after.Start = 12 * telem.SecondTS
			next := MustSucceed(db.OpenWriter(ctx, after))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			end, held := MustSucceed2(p.Commit(ctx))
			Expect(held).To(BeTrue())
			Expect(end).To(Equal(11*telem.SecondTS + 1))
			Expect(next.Close()).To(Succeed())
			Expect(seconds(ctx)).To(Equal([]int64{10, 11}))
		})

	It("Should write a key again after a handoff brings back its samples that a "+
		"write skipped", func(ctx SpecContext) {
		w := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, w, both(10, 11))
		MustSucceed2(w.Write(
			ctx,
			both(12, 13),
			func(k cesium.ChannelKey) bool { return k == data },
		))
		w, p := MustSucceed2(db.Handoff(ctx, w, config(clock), nil, cesium.NewRun))
		w, _ = MustSucceed2(db.Handoff(ctx, w, config(clock, data), p, cesium.NewRun))
		_, skipped := MustSucceed2(w.Write(ctx, both(14), nil))
		Expect(skipped).To(BeFalse())
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(seconds(ctx)).To(Equal([]int64{10, 11, 14}))
	})

	It("Should go on in the stretch of a writer that stores no samples, with no "+
		"pending samples", func(ctx SpecContext) {
		cfg := config(clock, data)
		cfg.Unstored = true
		w := MustSucceed(db.OpenWriter(ctx, cfg))
		before := mustWrite(ctx, w, both(10, 11)).Get(clock).Series[0].Alignment
		cfg = config(clock)
		cfg.Unstored = true
		next, p := MustSucceed2(db.Handoff(ctx, w, cfg, nil, cesium.SameRun))
		Expect(p).To(BeNil())
		after := mustWrite(ctx, next, telem.UnaryFrame(
			clock,
			telem.NewSeriesSecondsTSV(12),
		)).Get(clock).Series[0].Alignment
		Expect(after).To(Equal(before.AddSamples(2)))
		Expect(MustSucceed2(next.Commit(ctx, nil))).To(Equal(12*telem.SecondTS + 1))
		Expect(next.Close()).To(Succeed())
		Expect(stored(ctx)).To(BeEmpty())
	})

	It("Should refuse a writer that stores no samples to adopt pending samples", func(
		ctx SpecContext,
	) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock, data)))
		mustWrite(ctx, first, both(10, 11))
		w, p := MustSucceed2(db.Handoff(ctx, first, config(clock), nil, cesium.NewRun))
		cfg := config(clock)
		cfg.Unstored = true
		stream := MustSucceed(db.OpenWriter(ctx, cfg))
		mustWrite(ctx, stream, telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(20)))
		cfg = config(clock, data)
		cfg.Unstored = true
		Expect(db.Handoff(ctx, stream, cfg, p, cesium.NewRun)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("adopts no pending samples")),
		))
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(11*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, clock, data))
		Expect(fr.Get(clock).Len()).To(Equal(int64(2)))
		Expect(fr.Get(data).Len()).To(BeZero(), "the refused handoff drops the pending")
	})

	It("Should return no pending samples for a key that the writer added with no "+
		"sample since", func(ctx SpecContext) {
		first := MustSucceed(db.OpenWriter(ctx, config(clock)))
		mustWrite(
			ctx,
			first,
			telem.UnaryFrame(clock, telem.NewSeriesSecondsTSV(10, 11)),
		)
		w, p := MustSucceed2(
			db.Handoff(ctx, first, config(clock, data), nil, cesium.NewRun),
		)
		Expect(p).To(BeNil())
		w, p = MustSucceed2(db.Handoff(ctx, w, config(clock), nil, cesium.NewRun))
		Expect(p).To(BeNil())
		Expect(MustSucceed2(w.Commit(ctx, nil))).To(Equal(11*telem.SecondTS + 1))
		Expect(w.Close()).To(Succeed())
	})
})
