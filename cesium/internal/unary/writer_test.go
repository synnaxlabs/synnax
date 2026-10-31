// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func filterDataFiles(l []os.FileInfo) []os.FileInfo {
	return lo.Filter(l, func(item os.FileInfo, _ int) bool {
		return item.Name() != "counter.domain" && item.Name() != "index.domain" &&
			item.Name() != "tombstone.domain" &&
			item.Name() != "meta.json"
	})
}

var _ = Describe("Writer Behavior", Ordered, func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			Describe("Index", func() {
				var (
					db *unary.DB
					fs fs.FS
				)
				BeforeEach(func(ctx SpecContext) {
					fs = openFS()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Name:     "Conrad",
							Key:      2,
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
				})
				Specify("Happy Path", func(ctx SpecContext) {
					w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					}))
					Expect(t.Occurred()).To(BeTrue())
					v := MustSucceed(
						w.Write(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15)),
					)
					Expect(v).To(Equal(alignment.Leading(1, 0)))
					Expect(
						MustSucceed(
							w.Write(telem.NewSeriesSecondsTSV(16, 17, 18, 19, 20, 21)),
						),
					).To(Equal(alignment.Leading(1, 6)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(21*telem.SecondTS + 1))
					t = MustSucceed(w.Close())
					Expect(t.Occurred()).To(BeTrue())
					Expect(db.LeadingControlState()).To(BeNil())
				})
				Specify("WriteCursor", func(ctx SpecContext) {
					By(
						"Reserving a fresh leading domain when nothing controls the channel",
					)
					Expect(db.WriteCursor(0)).To(Equal(alignment.Leading(1, 0)))
					Expect(db.WriteCursor(0)).To(Equal(alignment.Leading(2, 0)))
					w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "leading"},
					}))
					Expect(t.Occurred()).To(BeTrue())

					By("Tracking the controlling writer's next write position")
					Expect(MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10)))).
						To(Equal(alignment.Leading(3, 0)))
					Expect(
						db.WriteCursor(10 * telem.SecondTS),
					).To(Equal(alignment.Leading(3, 1)))
					Expect(MustSucceed(w.Write(telem.NewSeriesSecondsTSV(11, 12)))).
						To(Equal(alignment.Leading(3, 1)))
					Expect(
						db.WriteCursor(10 * telem.SecondTS),
					).To(Equal(alignment.Leading(3, 3)))

					t = MustSucceed(w.Close())
					Expect(t.Occurred()).To(BeTrue())
				})
				Specify(
					"WriteCursor only tracks the region covering the timestamp",
					func(ctx SpecContext) {
						w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							End:     15 * telem.SecondTS,
							Subject: control.Subject{Key: "bounded"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12)))

						By("Tracking the bounded region for a timestamp inside it")
						Expect(
							db.WriteCursor(11 * telem.SecondTS),
						).To(Equal(alignment.Leading(1, 3)))

						By(
							"Reserving a fresh domain for a timestamp the region does not cover",
						)
						Expect(
							db.WriteCursor(20 * telem.SecondTS),
						).To(Equal(alignment.Leading(2, 0)))
						t = MustSucceed(w.Close())
						Expect(t.Occurred()).To(BeTrue())
					},
				)
				Describe("EndDomain", func() {
					open := func(
						ctx SpecContext,
						key string,
						authority control.Authority,
					) *unary.Writer {
						GinkgoHelper()
						w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Subject:   control.Subject{Key: key},
							Authority: authority,
						}))
						return w
					}
					openBounded := func(start telem.TimeStamp) unary.WriterConfig {
						return unary.WriterConfig{
							Start:     start,
							Subject:   control.Subject{Key: "backfill"},
							Authority: 1,
							Bounded:   new(true),
						}
					}
					It(
						"Should start a new domain in a new alignment domain",
						func(ctx SpecContext) {
							w := open(ctx, "live", 2)
							first := MustSucceed(
								w.Write(telem.NewSeriesSecondsTSV(10, 11)),
							)
							MustSucceed(w.Commit(ctx))
							Expect(w.EndDomain(ctx)).To(Succeed())
							w.SetDomainStart(20 * telem.SecondTS)
							second := MustSucceed(
								w.Write(telem.NewSeriesSecondsTSV(20, 21)),
							)
							Expect(
								second.DomainIndex(),
							).To(BeNumerically(">", first.DomainIndex()))
							Expect(second.SampleIndex()).To(BeZero())
							Expect(db.WriteCursor(20 * telem.SecondTS)).
								To(Equal(second.AddSamples(2)))
							Expect(MustSucceed(w.Commit(ctx))).
								To(Equal(21*telem.SecondTS + 1))
							MustSucceed(w.Close())
							Expect(MustSucceed(db.HasDataFor(
								ctx,
								(12 * telem.SecondTS).Range(20*telem.SecondTS),
							))).To(BeFalse())
						},
					)
					It(
						"Should hold the time between the domains until the next write",
						func(ctx SpecContext) {
							w := open(ctx, "live", 2)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11)))
							MustSucceed(w.Commit(ctx))
							Expect(w.EndDomain(ctx)).To(Succeed())
							Expect(db.OpenWriter(ctx, openBounded(15*telem.SecondTS))).
								Error().To(MatchError(control.ErrUnauthorized))
							w.SetDomainStart(20 * telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(20)))
							MustSucceed(w.Commit(ctx))
							backfill, _ := MustSucceed2(
								db.OpenWriter(ctx, openBounded(15*telem.SecondTS)),
							)
							MustSucceed(
								backfill.Write(telem.NewSeriesSecondsTSV(15, 16)),
							)
							MustSucceed(backfill.Commit(ctx))
							MustSucceed(backfill.Write(telem.NewSeriesSecondsTSV(21)))
							Expect(backfill.Commit(ctx)).Error().To(And(
								MatchError(validate.ErrValidation),
								MatchError(
									ContainSubstring("cannot extend control region"),
								),
							))
							MustSucceed(backfill.Close())
							MustSucceed(w.Close())
						},
					)
					It(
						"Should do nothing when the domain holds no data",
						func(ctx SpecContext) {
							w := open(ctx, "live", 2)
							Expect(w.EndDomain(ctx)).To(Succeed())
							Expect(db.WriteCursor(10 * telem.SecondTS)).
								To(Equal(alignment.Leading(1, 0)))
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11)))
							MustSucceed(w.Commit(ctx))
							MustSucceed(w.Close())
						},
					)
					It(
						"Should fail while another writer holds control",
						func(ctx SpecContext) {
							lo := open(ctx, "lo", 1)
							hi := open(ctx, "hi", 2)
							MustSucceed(hi.Write(telem.NewSeriesSecondsTSV(10, 11)))
							MustSucceed(hi.Commit(ctx))
							Expect(
								lo.EndDomain(ctx),
							).To(MatchError(control.ErrUnauthorized))
							MustSucceed(hi.Write(telem.NewSeriesSecondsTSV(12)))
							Expect(MustSucceed(hi.Commit(ctx))).
								To(Equal(12*telem.SecondTS + 1))
							MustSucceed(hi.Close())
							MustSucceed(lo.Close())
						},
					)
					It("Should fail on a closed writer", func(ctx SpecContext) {
						w := open(ctx, "live", 2)
						MustSucceed(w.Close())
						Expect(w.EndDomain(ctx)).To(MatchError(unary.ErrWriterClosed))
					})
				})

				Describe("Control region start", func() {
					It(
						"Should move forward to the first sample",
						func(ctx SpecContext) {
							w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:     10 * telem.SecondTS,
								Subject:   control.Subject{Key: "live"},
								Authority: 2,
							}))
							w.SetDomainStart(20 * telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(20)))
							Expect(db.Delete(
								ctx,
								(12 * telem.SecondTS).Range(15*telem.SecondTS),
							)).To(Succeed())
							Expect(db.Delete(
								ctx,
								(20 * telem.SecondTS).Range(21*telem.SecondTS),
							)).To(MatchError(control.ErrUnauthorized))
							MustSucceed(w.Close())
						},
					)
					It(
						"Should stay at the start of a domain the writer did not open",
						func(ctx SpecContext) {
							lo, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:     10 * telem.SecondTS,
								Subject:   control.Subject{Key: "lo"},
								Authority: 2,
							}))
							MustSucceed(lo.Write(telem.NewSeriesSecondsTSV(10, 11)))
							hi, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:     10 * telem.SecondTS,
								Subject:   control.Subject{Key: "hi"},
								Authority: 3,
							}))
							hi.SetDomainStart(20 * telem.SecondTS)
							MustSucceed(hi.Write(telem.NewSeriesSecondsTSV(20)))
							Expect(db.Delete(
								ctx,
								(10 * telem.SecondTS).Range(11*telem.SecondTS),
							)).To(MatchError(control.ErrUnauthorized))
							MustSucceed(hi.Close())
							MustSucceed(lo.Close())
						},
					)
				})

				Describe("Bounded", func() {
					openLive := func(
						ctx SpecContext,
						start telem.TimeStamp,
						authority control.Authority,
					) *unary.Writer {
						GinkgoHelper()
						w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     start,
							Subject:   control.Subject{Key: "live"},
							Authority: authority,
						}))
						return w
					}
					boundedConfig := func(start telem.TimeStamp) unary.WriterConfig {
						return unary.WriterConfig{
							Start:     start,
							Subject:   control.Subject{Key: "backfill"},
							Authority: 1,
							Bounded:   new(true),
						}
					}
					openBounded := func(
						ctx SpecContext,
						start telem.TimeStamp,
					) *unary.Writer {
						GinkgoHelper()
						w, _ := MustSucceed2(db.OpenWriter(ctx, boundedConfig(start)))
						return w
					}
					It("Should write behind a live writer", func(ctx SpecContext) {
						live := openLive(ctx, 10*telem.SecondTS, 2)
						MustSucceed(live.Write(telem.NewSeriesSecondsTSV(10, 11)))
						Expect(MustSucceed(live.Commit(ctx))).
							To(Equal(11*telem.SecondTS + 1))
						w := openBounded(ctx, 1*telem.SecondTS)
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2, 3)))
						Expect(MustSucceed(w.Commit(ctx))).
							To(Equal(3*telem.SecondTS + 1))
						MustSucceed(live.Write(telem.NewSeriesSecondsTSV(12)))
						Expect(MustSucceed(live.Commit(ctx))).
							To(Equal(12*telem.SecondTS + 1))
						MustSucceed(w.Close())
						MustSucceed(live.Close())
						Expect(MustSucceed(db.HasDataFor(
							ctx,
							(1 * telem.SecondTS).Range(3*telem.SecondTS),
						))).To(BeTrue())
					})
					It("Should let a live writer open after it", func(ctx SpecContext) {
						w := openBounded(ctx, 1*telem.SecondTS)
						live := openLive(ctx, 10*telem.SecondTS, 1)
						MustSucceed(live.Write(telem.NewSeriesSecondsTSV(10)))
						MustSucceed(live.Commit(ctx))
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2)))
						MustSucceed(w.Commit(ctx))
						MustSucceed(w.Close())
						MustSucceed(live.Close())
					})
					It(
						"Should run next to another bounded writer",
						func(ctx SpecContext) {
							w1 := openBounded(ctx, 1*telem.SecondTS)
							cfg := boundedConfig(5 * telem.SecondTS)
							cfg.Subject.Key = "other"
							w2, _ := MustSucceed2(db.OpenWriter(ctx, cfg))
							MustSucceed(w2.Write(telem.NewSeriesSecondsTSV(5, 6)))
							MustSucceed(w2.Commit(ctx))
							MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(1, 2)))
							MustSucceed(w1.Commit(ctx))
							MustSucceed(w1.Close())
							MustSucceed(w2.Close())
						},
					)
					DescribeTable(
						"Should fail to open where a live writer holds the start",
						func(ctx SpecContext, authority control.Authority) {
							live := openLive(ctx, 10*telem.SecondTS, 2)
							cfg := boundedConfig(15 * telem.SecondTS)
							cfg.Authority = authority
							Expect(db.OpenWriter(ctx, cfg)).Error().To(And(
								MatchError(control.ErrUnauthorized),
								MatchError(ContainSubstring(
									"overlaps with a controlled region",
								)),
							))
							MustSucceed(live.Write(telem.NewSeriesSecondsTSV(10)))
							MustSucceed(live.Commit(ctx))
							MustSucceed(live.Close())
						},
						Entry("with less authority", control.Authority(1)),
						Entry("with equal authority", control.Authority(2)),
						Entry("with more authority", control.AuthorityAbsolute),
					)
					It(
						"Should fail to open with no start next to a live writer",
						func(ctx SpecContext) {
							live := openLive(ctx, 10*telem.SecondTS, 2)
							Expect(db.OpenWriter(ctx, boundedConfig(0))).Error().
								To(MatchError(control.ErrUnauthorized))
							MustSucceed(live.Close())
						},
					)
					It(
						"Should fail a commit that reaches a live writer",
						func(ctx SpecContext) {
							live := openLive(ctx, 10*telem.SecondTS, 2)
							w := openBounded(ctx, 1*telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2, 10)))
							Expect(w.Commit(ctx)).Error().To(And(
								MatchError(validate.ErrValidation),
								MatchError(
									ContainSubstring("cannot extend control region"),
								),
							))
							MustSucceed(w.Close())
							MustSucceed(live.Write(telem.NewSeriesSecondsTSV(10, 11)))
							Expect(MustSucceed(live.Commit(ctx))).
								To(Equal(11*telem.SecondTS + 1))
							MustSucceed(live.Close())
						},
					)
					It(
						"Should commit up to the start of a live writer",
						func(ctx SpecContext) {
							live := openLive(ctx, 10*telem.SecondTS, 2)
							w := openBounded(ctx, 1*telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1)))
							Expect(
								w.CommitWithEnd(ctx, 10*telem.SecondTS),
							).To(Succeed())
							MustSucceed(w.Close())
							MustSucceed(live.Close())
						},
					)
					It(
						"Should fail a commit that reaches a stored domain",
						func(ctx SpecContext) {
							Expect(unary.Write(
								ctx,
								db,
								10*telem.SecondTS,
								telem.NewSeriesSecondsTSV(10, 11),
							)).To(Succeed())
							w := openBounded(ctx, 1*telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2, 12)))
							Expect(w.Commit(ctx)).Error().To(And(
								MatchError(domain.ErrWriteConflict),
								MatchError(
									ContainSubstring("overlaps with existing data"),
								),
							))
							MustSucceed(w.Close())
						},
					)
					It(
						"Should give way to a live writer that moves back over uncommitted time",
						func(ctx SpecContext) {
							live := openLive(ctx, 10*telem.SecondTS, 2)
							w := openBounded(ctx, 1*telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2, 8)))
							live.SetDomainStart(5 * telem.SecondTS)
							MustSucceed(live.Write(telem.NewSeriesSecondsTSV(5, 6)))
							Expect(MustSucceed(live.Commit(ctx))).
								To(Equal(6*telem.SecondTS + 1))
							Expect(w.Commit(ctx)).Error().
								To(MatchError(validate.ErrValidation))
							MustSucceed(w.Close())
							MustSucceed(live.Close())
						},
					)
					It(
						"Should move back to a first sample before its start",
						func(ctx SpecContext) {
							w, _ := MustSucceed2(db.OpenWriter(ctx, boundedConfig(0)))
							w.SetDomainStart(1 * telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2)))
							MustSucceed(w.Commit(ctx))
							live := openLive(ctx, 10*telem.SecondTS, 2)
							MustSucceed(w.Close())
							MustSucceed(live.Close())
						},
					)
					It(
						"Should protect the time it has committed",
						func(ctx SpecContext) {
							w := openBounded(ctx, 1*telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1, 2, 3)))
							MustSucceed(w.Commit(ctx))
							Expect(db.Delete(
								ctx,
								(2 * telem.SecondTS).Range(3*telem.SecondTS),
							)).To(MatchError(control.ErrUnauthorized))
							MustSucceed(w.Close())
							Expect(db.Delete(
								ctx,
								(2 * telem.SecondTS).Range(3*telem.SecondTS),
							)).To(Succeed())
						},
					)
					It("Should report no control transfers", func(ctx SpecContext) {
						w, t := MustSucceed2(db.OpenWriter(
							ctx,
							boundedConfig(1*telem.SecondTS),
						))
						Expect(t.Occurred()).To(BeFalse())
						Expect(w.SetAuthority(control.AuthorityAbsolute).Occurred()).
							To(BeFalse())
						Expect(MustSucceed(w.Close()).Occurred()).To(BeFalse())
					})
					It(
						"Should be absent from the control state",
						func(ctx SpecContext) {
							w := openBounded(ctx, 1*telem.SecondTS)
							Expect(db.LeadingControlState()).To(BeNil())
							live := openLive(ctx, 10*telem.SecondTS, 2)
							Expect(
								db.LeadingControlState().Subject.Key,
							).To(Equal("live"))
							MustSucceed(w.Close())
							MustSucceed(live.Close())
						},
					)
				})
				Describe("Domain start", func() {
					It(
						"Should claim from the open time when Start is zero",
						func(ctx SpecContext) {
							Expect(unary.Write(
								ctx,
								db,
								1*telem.SecondTS,
								telem.NewSeriesSecondsTSV(1, 2, 3),
							)).To(Succeed())
							w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Subject: control.Subject{Key: "foo"},
							}))
							Expect(db.Delete(
								ctx,
								(1 * telem.SecondTS).Range(4*telem.SecondTS),
							)).To(Succeed())
							MustSucceed(w.Close())
						},
					)
					It(
						"Should not check stored data at open when Start is zero",
						func(ctx SpecContext) {
							now := telem.Now()
							Expect(unary.Write(
								ctx,
								db,
								now.Add(-telem.Hour),
								telem.NewSeriesV(
									now.Add(-telem.Hour),
									now.Add(telem.Hour),
								),
							)).To(Succeed())
							w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Subject: control.Subject{Key: "foo"},
							}))
							first := now.Add(2 * telem.Hour)
							w.SetDomainStart(first)
							MustSucceed(w.Write(telem.NewSeriesV(first)))
							Expect(MustSucceed(w.Commit(ctx))).To(Equal(first + 1))
							MustSucceed(w.Close())
						},
					)
					It(
						"Should fail when the domain start reaches another region",
						func(ctx SpecContext) {
							w1, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:   1 * telem.SecondTS,
								End:     3 * telem.SecondTS,
								Subject: control.Subject{Key: "foo"},
							}))
							w2, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:   10 * telem.SecondTS,
								Subject: control.Subject{Key: "bar"},
							}))
							w2.SetDomainStart(2 * telem.SecondTS)
							Expect(
								w2.Write(telem.NewSeriesSecondsTSV(2, 3)),
							).Error().To(And(
								MatchError(validate.ErrValidation),
								MatchError(
									ContainSubstring("cannot extend control region"),
								),
							))
							MustSucceed(w2.Close())
							MustSucceed(w1.Close())
						},
					)
					It(
						"Should not delete a domain that starts before the claim",
						func(ctx SpecContext) {
							w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:   10 * telem.SecondTS,
								Subject: control.Subject{Key: "foo"},
							}))
							w.SetDomainStart(5 * telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(5, 6, 7)))
							MustSucceed(w.Commit(ctx))
							Expect(db.Delete(
								ctx,
								(5 * telem.SecondTS).Range(7*telem.SecondTS),
							)).To(MatchError(control.ErrUnauthorized))
							Expect(db.Delete(
								ctx,
								(1 * telem.SecondTS).Range(5*telem.SecondTS),
							)).To(Succeed())
							MustSucceed(w.Close())
							Expect(db.Delete(
								ctx,
								(5 * telem.SecondTS).Range(7*telem.SecondTS),
							)).To(Succeed())
						},
					)
					It(
						"Should start the domain where SetDomainStart sets it",
						func(ctx SpecContext) {
							w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:   10 * telem.SecondTS,
								Subject: control.Subject{Key: "foo"},
							}))
							w.SetDomainStart(5 * telem.SecondTS)
							MustSucceed(w.Write(telem.NewSeriesSecondsTSV(5, 6, 7)))
							Expect(
								MustSucceed(w.Commit(ctx)),
							).To(Equal(7*telem.SecondTS + 1))
							MustSucceed(w.Close())
							Expect(MustSucceed(db.HasDataFor(
								ctx,
								(5 * telem.SecondTS).Range(6*telem.SecondTS),
							))).To(BeTrue())
						},
					)
					It(
						"Should fail the first write when stored data holds the domain start",
						func(ctx SpecContext) {
							Expect(unary.Write(
								ctx,
								db,
								10*telem.SecondTS,
								telem.NewSeriesSecondsTSV(10, 11, 12, 13),
							)).To(Succeed())
							w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:   20 * telem.SecondTS,
								Subject: control.Subject{Key: "foo"},
							}))
							w.SetDomainStart(12 * telem.SecondTS)
							Expect(
								w.Write(telem.NewSeriesSecondsTSV(12, 20)),
							).Error().To(And(
								MatchError(domain.ErrWriteConflict),
								MatchError(ContainSubstring("cannot start writer")),
							))
							MustSucceed(w.Close())
						},
					)
				})
				Describe("Open", func() {
					It(
						"Should not allow opening a writer if the specified end timestamp is before the start",
						func(ctx SpecContext) {
							_, _, err := db.OpenWriter(
								ctx,
								unary.WriterConfig{
									Start:   10,
									End:     3,
									Subject: control.Subject{Key: "foo"},
								},
							)
							Expect(
								err,
							).To(MatchError(ContainSubstring("end timestamp must be after")))
						},
					)

					It(
						"Should not allow opening a writer without a subject key",
						func(ctx SpecContext) {
							_, _, err := db.OpenWriter(
								ctx,
								unary.WriterConfig{Start: 3, End: 10},
							)
							Expect(
								err,
							).To(MatchError(ContainSubstring("subject.key: required")))
						},
					)
				})
			})
			Describe("Channel Indexed", func() {
				var (
					dataDB  *unary.DB
					dataFS  fs.FS
					indexDB *unary.DB
					indexFS fs.FS
					index   = GenerateChannelKey()
					data    = GenerateChannelKey()
				)
				BeforeEach(func(ctx SpecContext) {
					fs := openFS()
					indexFS = MustSucceed(fs.Sub("index"))
					indexDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        indexFS,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      index,
							Name:     "Cayley",
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						FileSize: telem.Size(
							10*telem.TimestampT.Density(),
						) * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					dataFS = MustSucceed(fs.Sub("data"))
					dataDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        dataFS,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      data,
							Name:     "Maxwell",
							DataType: telem.Int64T,
							Index:    index,
						},
						FileSize: telem.Size(
							5*telem.Int64T.Density(),
						) * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					dataDB.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(dataDB.Close()).To(Succeed())
					Expect(indexDB.Close()).To(Succeed())
				})
				Specify("Happy Path", func(ctx SpecContext) {
					Expect(
						unary.Write(
							ctx,
							indexDB,
							10*telem.SecondTS,
							telem.NewSeriesSecondsTSV(
								10,
								11,
								12,
								13,
								14,
								15,
								16,
								17,
								18,
								19,
								20,
							),
						),
					).To(Succeed())
					w, t := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					}))
					By("Taking control of the DB")
					Expect(
						dataDB.LeadingControlState().Subject,
					).To(Equal(control.Subject{Key: "foo"}))
					Expect(t.Occurred()).To(BeTrue())
					Expect(
						MustSucceed(
							w.Write(
								telem.NewSeries(
									[]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
								),
							),
						),
					).To(Equal(alignment.Leading(1, 0)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(20*telem.SecondTS + 1))
					t = MustSucceed(w.Close())
					Expect(t.Occurred()).To(BeTrue())
					By("Releasing control of the DB")
					Expect(dataDB.LeadingControlState()).To(BeNil())
				})
				Specify(
					"Commit end after a control handoff",
					func(ctx SpecContext) {
						Expect(unary.Write(
							ctx,
							indexDB,
							10*telem.SecondTS,
							telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15),
						)).To(Succeed())
						w1, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.AuthorityAbsolute - 1,
							Subject:   control.Subject{Key: "foo"},
						}))
						MustSucceed(w1.Write(telem.NewSeries([]int64{0, 1})))
						Expect(
							MustSucceed(w1.Commit(ctx)),
						).To(Equal(11*telem.SecondTS + 1))
						w2, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
							Start:     12 * telem.SecondTS,
							Authority: control.AuthorityAbsolute,
							Subject:   control.Subject{Key: "bar"},
						}))
						MustSucceed(w2.Write(telem.NewSeries([]int64{2, 3})))
						Expect(
							MustSucceed(w2.Commit(ctx)),
						).To(Equal(13*telem.SecondTS + 1))
						MustSucceed(w2.Close())
						MustSucceed(w1.Close())
					},
				)
				Specify("Open Writer domain overlap", func(ctx SpecContext) {
					Expect(
						unary.Write(
							ctx,
							indexDB,
							10*telem.SecondTS,
							telem.NewSeriesSecondsTSV(
								10,
								11,
								12,
								13,
								14,
								15,
								16,
								17,
								18,
								19,
								20,
							),
						),
					).To(Succeed())
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					}))
					Expect(
						MustSucceed(w.Write(telem.NewSeries([]int64{0, 1, 2, 3, 4}))),
					).To(Equal(alignment.Leading(1, 0)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(14*telem.SecondTS + 1))
					MustSucceed(w.Close())

					Expect(dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   12 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					})).Error().To(And(
						MatchError(ContainSubstring("channel [Maxwell]<%d>", data)),
						MatchError(ContainSubstring("overlaps")),
					))
				})
				Describe("Auto file switch", func() {
					Specify(
						"File cutoff on commit with no preset end",
						func(ctx SpecContext) {
							w1, _ := MustSucceed2(
								indexDB.OpenWriter(
									ctx,
									unary.WriterConfig{
										Start:   10 * telem.SecondTS,
										Subject: control.Subject{Key: "foo"},
									},
								),
							)
							w2, _ := MustSucceed2(
								dataDB.OpenWriter(
									ctx,
									unary.WriterConfig{
										Start:   10 * telem.SecondTS,
										Subject: control.Subject{Key: "moo"},
									},
								),
							)

							By("Writing telemetry")
							MustSucceed(
								w1.Write(
									telem.NewSeriesSecondsTSV(
										10,
										11,
										13,
										14,
										15,
										16,
										17,
									),
								),
							)
							MustSucceed(
								w2.Write(
									telem.NewSeriesV[int64](
										100,
										101,
										103,
										104,
										105,
										106,
										107,
									),
								),
							)

							By(
								"Asserting that both writers have still written to one file",
							)
							l := filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(1))
							Expect(
								l[0].Size(),
							).To(Equal(int64(7 * telem.Int64T.Density())))

							l = filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(1))
							Expect(
								l[0].Size(),
							).To(Equal(int64(7 * telem.Int64T.Density())))

							By("Committing the data")
							Expect(w1.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))
							Expect(w2.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))

							By("Writing more telemetry")
							MustSucceed(
								w1.Write(
									telem.NewSeriesSecondsTSV(23, 29, 31, 37, 41, 43),
								),
							)
							MustSucceed(
								w2.Write(
									telem.NewSeriesV[int64](
										203,
										209,
										301,
										307,
										401,
										403,
									),
								),
							)

							By(
								"Asserting that the second writer is now writing to a different file",
							)
							l = filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(1))
							Expect(
								l[0].Size(),
							).To(Equal(int64(13 * telem.Int64T.Density())))

							l = filterDataFiles(MustSucceed(dataFS.List("")))
							Expect(l).To(HaveLen(2))
							Expect(
								[]int64{l[0].Size(), l[1].Size()},
							).To(ConsistOf(int64(7*telem.Int64T.Density()), int64(6*telem.Int64T.Density())))

							By("Committing the data")
							Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
							Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

							By("Writing more telemetry")
							MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(47, 53)))

							By(
								"Asserting that the first writer is now writing to a different file",
							)
							l = filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(2))
							Expect(
								[]int64{l[0].Size(), l[1].Size()},
							).To(ConsistOf(int64(13*telem.Int64T.Density()), int64(2*telem.Int64T.Density())))

							Expect(w1.Commit(ctx)).To(Equal(53*telem.SecondTS + 1))

							MustSucceed(w1.Close())
							MustSucceed(w2.Close())

							By("Asserting that the data is correct", func() {
								i, _ := dataDB.OpenIterator(
									unary.IteratorConfig{Bounds: telem.TimeRangeMax},
								)
								Expect(i.SeekFirst(ctx)).To(BeTrue())
								Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
								f := i.Value()
								Expect(
									f.SeriesAt(0),
								).To(telem.MatchSeriesDataV[int64](100, 101, 103, 104, 105, 106, 107))
								Expect(
									f.SeriesAt(0).TimeRange,
								).To(Equal((10 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
								Expect(
									f.SeriesAt(1),
								).To(telem.MatchSeriesDataV[int64](203, 209, 301, 307, 401, 403))
								Expect(
									f.SeriesAt(1).TimeRange,
								).To(Equal((17*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
								Expect(i.Close()).To(Succeed())
							})
						},
					)
					Specify(
						"Special case where the file exceeds limit by 1",
						func(ctx SpecContext) {
							w1, _ := MustSucceed2(
								indexDB.OpenWriter(
									ctx,
									unary.WriterConfig{
										Start:   10 * telem.SecondTS,
										Subject: control.Subject{Key: "foo"},
									},
								),
							)
							w2, _ := MustSucceed2(
								dataDB.OpenWriter(
									ctx,
									unary.WriterConfig{
										Start:   10 * telem.SecondTS,
										Subject: control.Subject{Key: "moo"},
									},
								),
							)

							By("Writing telemetry")
							MustSucceed(
								w1.Write(
									telem.NewSeriesSecondsTSV(
										10,
										11,
										13,
										14,
										15,
										16,
										17,
									),
								),
							)
							MustSucceed(
								w2.Write(
									telem.NewSeriesV[int64](
										100,
										101,
										103,
										104,
										105,
										106,
										107,
									),
								),
							)

							By(
								"Asserting that both writers have still written to one file",
							)
							l := filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(1))
							Expect(
								l[0].Size(),
							).To(Equal(int64(7 * telem.Int64T.Density())))

							l = filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(1))
							Expect(
								l[0].Size(),
							).To(Equal(int64(7 * telem.Int64T.Density())))

							By("Committing the data")
							Expect(w1.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))
							Expect(w2.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))

							By("Writing more telemetry")
							MustSucceed(
								w1.Write(
									telem.NewSeriesSecondsTSV(23, 29, 31, 37, 41, 43),
								),
							)
							MustSucceed(
								w2.Write(
									telem.NewSeriesV[int64](
										203,
										209,
										301,
										307,
										401,
										403,
									),
								),
							)

							By(
								"Asserting that the second writer is now writing to a different file",
							)
							l = filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(1))
							Expect(
								l[0].Size(),
							).To(Equal(int64(13 * telem.Int64T.Density())))

							l = filterDataFiles(MustSucceed(dataFS.List("")))
							Expect(l).To(HaveLen(2))
							Expect(
								[]int64{l[0].Size(), l[1].Size()},
							).To(ConsistOf(int64(7*telem.Int64T.Density()), int64(6*telem.Int64T.Density())))

							By("Committing the data")
							Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
							Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

							By("Writing more telemetry")
							MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(47, 53)))

							By(
								"Asserting that the first writer is now writing to a different file",
							)
							l = filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(2))
							Expect(
								[]int64{l[0].Size(), l[1].Size()},
							).To(ConsistOf(int64(13*telem.Int64T.Density()), int64(2*telem.Int64T.Density())))

							Expect(w1.Commit(ctx)).To(Equal(53*telem.SecondTS + 1))

							MustSucceed(w1.Close())
							MustSucceed(w2.Close())

							By("Asserting that the data is correct", func() {
								i, _ := dataDB.OpenIterator(
									unary.IterRange(telem.TimeRangeMax),
								)
								Expect(i.SeekFirst(ctx)).To(BeTrue())
								Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
								f := i.Value()
								Expect(f.SeriesAt(0).Len()).To(Equal(int64(7)))
								Expect(
									f.SeriesAt(0).TimeRange,
								).To(Equal((10 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
								Expect(f.SeriesAt(1).Len()).To(Equal(int64(6)))
								Expect(
									f.SeriesAt(1).TimeRange,
								).To(Equal((17*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
								Expect(
									f.Get(data).Data(),
								).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 203, 209, 301, 307, 401, 403).Data))
								Expect(i.Close()).To(Succeed())
							})
						},
					)
					Specify(
						"File cutoff on commit with preset end",
						func(ctx SpecContext) {
							w1, _ := MustSucceed2(
								indexDB.OpenWriter(
									ctx,
									unary.WriterConfig{
										Start:   10 * telem.SecondTS,
										End:     100 * telem.SecondTS,
										Subject: control.Subject{Key: "foo"},
									},
								),
							)
							w2, _ := MustSucceed2(
								dataDB.OpenWriter(
									ctx,
									unary.WriterConfig{
										Start:   10 * telem.SecondTS,
										End:     100 * telem.SecondTS,
										Subject: control.Subject{Key: "moo"},
									},
								),
							)

							By("Writing telemetry")
							MustSucceed(
								w1.Write(
									telem.NewSeriesSecondsTSV(
										10,
										11,
										13,
										14,
										15,
										16,
										17,
										18,
										20,
										23,
										25,
									),
								),
							)
							MustSucceed(
								w2.Write(
									telem.NewSeriesV[int64](
										100,
										101,
										103,
										104,
										105,
										106,
										107,
										108,
										200,
										203,
										205,
									),
								),
							)

							By("Committing the data")
							Expect(w1.Commit(ctx)).To(Equal(25*telem.SecondTS + 1))
							Expect(w2.Commit(ctx)).To(Equal(25*telem.SecondTS + 1))

							By("Writing more telemetry")
							MustSucceed(
								w1.Write(telem.NewSeriesSecondsTSV(29, 31, 37, 41, 43)),
							)
							MustSucceed(
								w2.Write(
									telem.NewSeriesV[int64](209, 301, 307, 401, 403),
								),
							)

							By("Committing the data")
							Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
							Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

							By(
								"Asserting that both writers are now writing to a different file",
							)
							l := filterDataFiles(MustSucceed(indexFS.List("")))
							Expect(l).To(HaveLen(2))
							Expect(
								[]int64{l[0].Size(), l[1].Size()},
							).To(ConsistOf(int64(11*telem.Int64T.Density()), int64(5*telem.Int64T.Density())))

							l = filterDataFiles(MustSucceed(dataFS.List("")))
							Expect(l).To(HaveLen(3))
							Expect(
								[]int64{l[0].Size(), l[1].Size(), l[2].Size()},
							).To(ConsistOf(int64(11*telem.Int64T.Density()), int64(5*telem.Int64T.Density()), int64(0)))

							MustSucceed(w1.Close())
							MustSucceed(w2.Close())

							By("Asserting that the data is correct", func() {
								i, _ := dataDB.OpenIterator(
									unary.IterRange(telem.TimeRangeMax),
								)
								Expect(i.SeekFirst(ctx)).To(BeTrue())
								Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
								f := i.Value()
								Expect(f.SeriesAt(0).Len()).To(Equal(int64(11)))
								Expect(f.SeriesAt(1).Len()).To(Equal(int64(5)))
								Expect(
									f.Get(data).Data(),
								).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 108, 200, 203, 205, 209, 301, 307, 401, 403).Data))
								Expect(
									f.SeriesAt(0).TimeRange,
								).To(Equal((10 * telem.SecondTS).Range(25*telem.SecondTS + 1)))
								Expect(
									f.SeriesAt(1).TimeRange,
								).To(Equal((25*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
								Expect(i.Close()).To(Succeed())
							})
						},
					)

					Specify("Just enough data to switch files", func(ctx SpecContext) {
						w1, _ := MustSucceed2(
							indexDB.OpenWriter(
								ctx,
								unary.WriterConfig{
									Start:   10 * telem.SecondTS,
									End:     100 * telem.SecondTS,
									Subject: control.Subject{Key: "foo"},
								},
							),
						)
						w2, _ := MustSucceed2(
							dataDB.OpenWriter(
								ctx,
								unary.WriterConfig{
									Start:   10 * telem.SecondTS,
									End:     100 * telem.SecondTS,
									Subject: control.Subject{Key: "moo"},
								},
							),
						)

						By("Writing telemetry")
						MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(10)))
						MustSucceed(w2.Write(telem.NewSeriesV[int64](100)))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(10*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(10*telem.SecondTS + 1))

						By("Writing more telemetry")
						MustSucceed(
							w1.Write(
								telem.NewSeriesSecondsTSV(
									11,
									12,
									13,
									14,
									15,
									16,
									17,
									18,
									19,
								),
							),
						)
						MustSucceed(
							w2.Write(
								telem.NewSeriesV[int64](
									101,
									102,
									103,
									104,
									105,
									106,
									107,
									108,
									109,
								),
							),
						)

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(19*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(19*telem.SecondTS + 1))

						By("Asserting that both writers have only written to one file")
						l := filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect(
							[]int64{l[0].Size(), l[1].Size()},
						).To(ConsistOf(int64(10*telem.Int64T.Density()), int64(0)))

						l = filterDataFiles(MustSucceed(dataFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect(
							[]int64{l[0].Size(), l[1].Size()},
						).To(ConsistOf(int64(10*telem.Int64T.Density()), int64(0)))

						MustSucceed(w1.Close())
						MustSucceed(w2.Close())

						By("Asserting that the data is correct", func() {
							i, _ := dataDB.OpenIterator(
								unary.IteratorConfig{Bounds: telem.TimeRangeMax},
							)
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.SeriesAt(0).Len()).To(Equal(int64(10)))
							Expect(
								f.SeriesAt(0).Data,
							).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106, 107, 108, 109).Data))
							Expect(
								f.SeriesAt(0).TimeRange,
							).To(Equal((10 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
							Expect(i.Close()).To(Succeed())
						})
					})
				})
			})

			Describe("Control", func() {
				var (
					db *unary.DB
					fs fs.FS
				)
				BeforeEach(func(ctx SpecContext) {
					fs = openFS()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Name:     "Frederick",
							Key:      2,
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
				})
				Describe("Index", func() {
					Specify("Control Handoff", func(ctx SpecContext) {
						w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.AuthorityAbsolute - 1,
							Subject:   control.Subject{Key: "foo"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(
							MustSucceed(
								w1.Write(telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4, 5)),
							),
						).To(Equal(alignment.Leading(1, 0)))
						w2, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.AuthorityAbsolute,
							Subject:   control.Subject{Key: "bar"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(
							MustSucceed(
								w2.Write(telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10, 11)),
							),
						).To(Equal(alignment.Leading(1, 6)))
						a, err := w1.Write(
							telem.NewSeriesSecondsTSV(12, 13, 14, 15, 16, 17),
						)
						Expect(err).To(MatchError(control.ErrUnauthorized))
						Expect(a).To(Equal(telem.Alignment(0)))
						_, err = w1.Commit(ctx)
						Expect(err).To(MatchError(control.ErrUnauthorized))
						t = MustSucceed(w2.Close())
						Expect(t.Occurred()).To(BeTrue())
						Expect(
							MustSucceed(
								w1.Write(
									telem.NewSeriesSecondsTSV(12, 13, 14, 15, 16, 17),
								),
							),
						).To(Equal(alignment.Leading(1, 12)))
						Expect(
							MustSucceed(w1.Commit(ctx)),
						).To(Equal(17*telem.SecondTS + 1))
						t = MustSucceed(w1.Close())
						Expect(t.Occurred()).To(BeTrue())
					})
					Specify(
						"Domain start of a region another writer has written",
						func(ctx SpecContext) {
							w1, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:     10 * telem.SecondTS,
								Authority: control.AuthorityAbsolute - 1,
								Subject:   control.Subject{Key: "foo"},
							}))
							MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(10, 11)))
							w2, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:     20 * telem.SecondTS,
								Authority: control.AuthorityAbsolute,
								Subject:   control.Subject{Key: "bar"},
							}))
							w2.SetDomainStart(20 * telem.SecondTS)
							MustSucceed(w2.Write(telem.NewSeriesSecondsTSV(12, 13)))
							Expect(
								MustSucceed(w2.Commit(ctx)),
							).To(Equal(13*telem.SecondTS + 1))
							MustSucceed(w2.Close())
							MustSucceed(w1.Close())
							Expect(MustSucceed(db.HasDataFor(
								ctx,
								(10 * telem.SecondTS).Range(11*telem.SecondTS),
							))).To(BeTrue())
						},
					)
				})
				Describe("ErrOnUnauthorizedOpen", func() {
					It(
						"Should return an error if the write does not acquire control",
						func(ctx SpecContext) {
							w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:                 10 * telem.SecondTS,
								Authority:             control.AuthorityAbsolute,
								Subject:               control.Subject{Key: "foo"},
								ErrOnUnauthorizedOpen: new(true),
							}))
							Expect(t.Occurred()).To(BeTrue())
							w2, t, err := db.OpenWriter(ctx, unary.WriterConfig{
								Start:                 10 * telem.SecondTS,
								Authority:             control.AuthorityAbsolute - 1,
								Subject:               control.Subject{Key: "bar"},
								ErrOnUnauthorizedOpen: new(true),
							})
							Expect(t.Occurred()).To(BeFalse())
							Expect(err).To(MatchError(control.ErrUnauthorized))
							Expect(w2).To(BeNil())
							t = MustSucceed(w1.Close())
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsRelease()).To(BeTrue())
						},
					)
				})
				Describe("Control Subjects", func() {
					It(
						"Should return an error when attempting to open a writer with a duplicate control subject key",
						func(ctx SpecContext) {
							w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
								Start:   10 * telem.SecondTS,
								Subject: control.Subject{Key: "foo"},
							}))
							Expect(t.Occurred()).To(BeTrue())
							w2, t, err := db.OpenWriter(ctx, unary.WriterConfig{
								Start:   10 * telem.SecondTS,
								Subject: control.Subject{Key: "foo"},
							})
							Expect(t.Occurred()).To(BeFalse())
							Expect(
								err,
							).To(MatchError(ContainSubstring("already registered in the region")))
							Expect(w2).To(BeNil())
							MustSucceed(w1.Close())
						},
					)
				})
			})

			Describe("Close", Ordered, func() {
				var (
					db  *unary.DB
					fs  fs.FS
					key = GenerateChannelKey()
				)
				BeforeEach(func(ctx SpecContext) {
					fs = openFS()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      key,
							Name:     "gauss",
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
				})
				It(
					"Should not allow operations on a closed writer",
					func(ctx SpecContext) {
						w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							Subject: control.Subject{Key: "foo"},
						},
						))
						Expect(t.Occurred()).To(BeTrue())
						MustSucceed(w.Close())
						Expect(w.Commit(ctx)).Error().To(SatisfyAll(
							MatchError(unary.ErrWriterClosed),
							MatchError(ContainSubstring("channel [gauss]<%d>", key)),
						),
						)
						Expect(w.Write(telem.Series{Data: []byte{1, 2, 3}})).
							Error().To(MatchError(unary.ErrWriterClosed))
						MustSucceed(w.Close())
					},
				)
				It(
					"Should not open a writer on a closed database",
					func(ctx SpecContext) {
						Expect(db.Close()).To(Succeed())
						_, _, err := db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							Subject: control.Subject{Key: "foo"},
						},
						)
						Expect(err).To(MatchError(unary.ErrDBClosed))
						Expect(
							err,
						).To(MatchError(ContainSubstring("channel [gauss]<%d>", key)))
					},
				)
				It("Should not write on a closed database", func(ctx SpecContext) {
					Expect(db.Close()).To(Succeed())
					Expect(
						unary.Write(ctx, db, 0, telem.NewSeriesV[int64](0, 1, 2)),
					).To(MatchError(unary.ErrDBClosed))
				})
			})
		})
	}
})
