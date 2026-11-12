// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Log", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			l := defaults.Fill(log.Log{
				Project: &proj.Key,
				Name:    "test",
				Channels: []log.ChannelEntry{
					{
						Channel: ch1,
						Color:   MustSucceed(color.FromHex("#ff0000")),
					},
				},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			Expect(l.Key).ToNot(Equal(uuid.Nil()))
		})

		It(
			"Should reject an explicit zero in a defaulted field and not fill it",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "zero"})
				l.TimeZone = ""
				Expect(svc.NewWriter(tx).Create(ctx, &l)).
					To(MatchError(ContainSubstring("invalid time_zone: ")))
			},
		)

		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				l := defaults.Fill(log.Log{Project: &proj.Key})
				Expect(svc.NewWriter(tx).Create(ctx, &l)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)

		DescribeTable("Should return a validation error for an out-of-range precision",
			func(ctx SpecContext, precision uint8, msg string) {
				tx := DeferClose(db.OpenTx())
				l := defaults.Fill(log.Log{
					Project:  &proj.Key,
					Name:     "test",
					Channels: []log.ChannelEntry{{Channel: ch5, Precision: &precision}},
				})
				Expect(svc.NewWriter(tx).Create(ctx, &l)).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("above maximum", uint8(18), "channels.0.precision: must be less"),
		)

		It("Should reject a channel of 0", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			l := defaults.Fill(log.Log{
				Project:  &proj.Key,
				Name:     "test",
				Channels: []log.ChannelEntry{{}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).
				To(MatchError(ContainSubstring("channels.0.channel")))
		})

		It("Should reject two entries for one channel", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			l := defaults.Fill(log.Log{
				Project:  &proj.Key,
				Name:     "test",
				Channels: []log.ChannelEntry{{Channel: 5}, {Channel: 5}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).
				To(MatchError(ContainSubstring("channels.1.channel")))
		})

		It("Should keep a channel precision of 0", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			l := defaults.Fill(log.Log{
				Project:  &proj.Key,
				Name:     "test",
				Channels: []log.ChannelEntry{{Channel: ch5, Precision: new(uint8(0))}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			var res log.Log
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Channels[0].Precision).To(HaveValue(BeZero()))
		})

		It(
			"Should reject a log with neither a project nor a range",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				l := defaults.Fill(log.Log{Name: "orphan"})
				Expect(
					svc.NewWriter(tx).Create(ctx, &l),
				).To(MatchError(ContainSubstring(
					"project: must belong to exactly one of a project and a range",
				)))
			},
		)

		It("Should reject a log whose project does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			missing := uuid.New()
			l := defaults.Fill(log.Log{Name: "dangling", Project: &missing})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(MatchError(
				ContainSubstring(
					"project: project " + missing.String() + " does not exist",
				),
			))
		})

		It("Should refuse a log whose key exists", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			stored := defaults.Fill(log.Log{Project: &proj.Key, Name: "first"})
			Expect(svc.NewWriter(tx).Create(ctx, &stored)).To(Succeed())
			again := defaults.Fill(
				log.Log{Project: &proj.Key, Key: stored.Key, Name: "second"},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("log %s already exists", stored.Key)),
			))
			var res log.Log
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(stored.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(stored))
		})

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[log.Key, log.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := defaults.Fill(log.Log{Project: &proj.Key, Name: "broadcast"})
				Expect(
					svc.NewWriter(tx).WithDispatchKey("dk-1").Create(ctx, &v),
				).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Key))
				Expect(seen[0].DispatchKey).To(Equal("dk-1"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(log.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[log.Key, log.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := defaults.Fill(log.Log{Project: &proj.Key, Name: "discarded"})
				Expect(
					svc.NewWriter(uncommitted).Create(ctx, &v),
				).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})
	Describe("CreateMany", func() {
		It("Should create multiple logs", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			logs := defaults.Fill([]log.Log{
				{Name: "log-1", Project: &proj.Key},
				{Name: "log-2", Project: &proj.Key},
			})
			Expect(svc.NewWriter(tx).CreateMany(ctx, &logs)).To(Succeed())

			var retrieved []log.Log
			Expect(svc.NewRetrieve().Where(log.MatchKeys(
				logs[0].Key,
				logs[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Apply of a project move", func() {
		It("Should move the log into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &dest)).To(Succeed())
			v := defaults.Fill(log.Log{Project: &proj.Key, Name: "moved"})
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(dest.Key))).
				To(BeEmpty())
			var res log.Log
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(v.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := defaults.Fill(log.Log{Project: &proj.Key, Name: "stuck"})
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(uuid.New()))).
				Error().To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Delete", func() {
		var tx gorp.Tx
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It("Should delete a Log so it is no longer retrievable", func(ctx SpecContext) {
			l := defaults.Fill(log.Log{Project: &proj.Key, Name: "to-delete"})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&log.Log{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should leave a deleted log out of its resources", func(ctx SpecContext) {
			l := defaults.Fill(log.Log{Project: &proj.Key, Name: "to-delete-resource"})
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{l.ResourceID()})).
				To(BeEmpty())
		})

		It("Should delete multiple logs in one call", func(ctx SpecContext) {
			a := defaults.Fill(log.Log{Project: &proj.Key, Name: "a"})
			b := defaults.Fill(log.Log{Project: &proj.Key, Name: "b"})
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(a.Key, b.Key)).
				Entries(&[]log.Log{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})
	Describe("Dispatch", func() {
		retrieve := func(ctx context.Context, key log.Key) log.Log {
			GinkgoHelper()
			var res log.Log
			Expect(
				svc.NewRetrieve().Where(log.MatchKeys(key)).Entry(&res).Exec(ctx, db),
			).
				To(Succeed())
			return res
		}
		entry := func(ch channel.Key, alias string) log.ChannelEntry {
			return log.ChannelEntry{
				Channel:  ch,
				Alias:    alias,
				Notation: notation.NotationStandard,
				TimestampConfig: log.TimestampConfig{
					Format:   telem.TimestampFormatPreciseDate,
					TimeZone: telem.TimeZoneLocal,
				},
			}
		}
		It(
			"Should reject a create without changing the log it names",
			func(ctx SpecContext) {
				target := defaults.Fill(log.Log{Project: &proj.Key, Name: "target"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &target)
				})).To(Succeed())
				other := defaults.Fill(log.Log{Project: &proj.Key, Name: "other"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(dispatch(ctx, target.Key, "d1", []log.Action{
					log.NewCreateAction(log.CreatePayload{
						Log: defaults.Fill(
							log.Log{Key: other.Key, Name: "overwritten"},
						),
					}),
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("only the Core dispatches a log create"),
					),
				))
				Expect(retrieve(ctx, other.Key).Name).To(Equal("other"))
			},
		)

		Describe("per-action semantics", func() {
			It("Should apply a set of the name", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "original"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetAction(
						log.SetPayload{Path: []string{"name"}, Value: "renamed"},
					),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).Name).To(Equal("renamed"))
			})

			It("Should reject an empty name", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "original"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetAction(log.SetPayload{Path: []string{"name"}, Value: ""}),
				})).To(MatchError(ContainSubstring("name: required")))
				Expect(retrieve(ctx, l.Key).Name).To(Equal("original"))
			})

			DescribeTable(
				"Should reject a path action that does not set the name",
				func(ctx SpecContext, action log.Action, msg string) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{action})).
						To(SatisfyAll(
							MatchError(validate.ErrValidation),
							MatchError(ContainSubstring(msg)),
						))
				},
				Entry(
					"a set of another field",
					log.NewSetAction(log.SetPayload{
						Path:  []string{"time_zone"},
						Value: "UTC",
					}),
					`unknown object member name "time_zone"`,
				),
				Entry(
					"a remove of the name",
					log.NewRemoveAction(log.RemovePayload{Key: "name"}),
					"[name]: the document requires a value",
				),
				Entry(
					"an insert into the name",
					log.NewInsertAction(log.InsertPayload{
						Path:  []string{"name"},
						Value: map[string]any{"key": "a"},
					}),
					"[name]: the value is not a list",
				),
				Entry(
					"a move in the name",
					log.NewMoveAction(
						log.MovePayload{Path: []string{"name"}, Key: "a"},
					),
					"[name]: the value is not a list",
				),
			)

			It(
				"Should append a default channel entry via AddChannel",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewAddChannelAction(log.AddChannelPayload{Channel: ch42}),
					})).To(Succeed())
					res := retrieve(ctx, l.Key)
					Expect(res.Channels).To(HaveLen(1))
					entry := res.Channels[0]
					Expect(entry.Channel).To(Equal(ch42))
					Expect(entry.Notation).To(Equal(notation.NotationStandard))
					Expect(entry.Precision).To(BeNil())
					Expect(
						entry.TimestampConfig.Format,
					).To(Equal(telem.TimestampFormatPreciseDate))
					Expect(
						entry.TimestampConfig.TimeZone,
					).To(Equal(telem.TimeZoneLocal))
				},
			)

			It("Should treat a duplicate AddChannel as a no-op", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewAddChannelAction(log.AddChannelPayload{Channel: ch42}),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: ch42}),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).Channels).To(HaveLen(1))
			})

			It("Should drop an entry via RemoveChannel", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewAddChannelAction(log.AddChannelPayload{Channel: ch1}),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: ch2}),
					log.NewRemoveChannelAction(log.RemoveChannelPayload{Channel: ch1}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.Channels[0].Channel).To(Equal(ch2))
			})

			It(
				"Should insert then replace an entry via SetChannelEntry",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetChannelEntryAction(log.SetChannelEntryPayload{
							Entry: entry(ch5, "first"),
						}),
						log.NewSetChannelEntryAction(log.SetChannelEntryPayload{
							Entry: entry(ch5, "second"),
						}),
					})).To(Succeed())
					res := retrieve(ctx, l.Key)
					Expect(res.Channels).To(HaveLen(1))
					Expect(res.Channels[0].Alias).To(Equal("second"))
				},
			)

			It(
				"Should set per-field config via the fine-grained channel setters",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{
						Project:  &proj.Key,
						Name:     "test",
						Channels: []log.ChannelEntry{{Channel: ch5}},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					red := color.Color{R: 255, G: 0, B: 0, A: 1}
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetChannelColorAction(
							log.SetChannelColorPayload{Channel: ch5, Color: red},
						),
						log.NewSetChannelNotationAction(
							log.SetChannelNotationPayload{
								Channel:  ch5,
								Notation: notation.NotationScientific,
							},
						),
						log.NewSetChannelPrecisionAction(
							log.SetChannelPrecisionPayload{
								Channel:   ch5,
								Precision: new(uint8(4)),
							},
						),
						log.NewSetChannelAliasAction(
							log.SetChannelAliasPayload{Channel: ch5, Alias: "volts"},
						),
						log.NewSetChannelTimestampFormatAction(
							log.SetChannelTimestampFormatPayload{
								Channel: ch5,
								Format:  telem.TimestampFormatTime,
							},
						),
						log.NewSetChannelTimestampTimeZoneAction(
							log.SetChannelTimestampTimeZonePayload{
								Channel:  ch5,
								TimeZone: telem.TimeZoneUTC,
							},
						),
					})).To(Succeed())
					entry := retrieve(ctx, l.Key).Channels[0]
					Expect(entry.Color).To(Equal(red))
					Expect(entry.Notation).To(Equal(notation.NotationScientific))
					Expect(entry.Precision).To(Equal(new(uint8(4))))
					Expect(entry.Alias).To(Equal("volts"))
					Expect(
						entry.TimestampConfig.Format,
					).To(Equal(telem.TimestampFormatTime))
					Expect(entry.TimestampConfig.TimeZone).To(Equal(telem.TimeZoneUTC))
				},
			)

			It(
				"Should treat a fine-grained channel setter as a no-op when the channel is absent",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{
						Project:  &proj.Key,
						Name:     "test",
						Channels: []log.ChannelEntry{{Channel: ch1}},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetChannelAliasAction(
							log.SetChannelAliasPayload{Channel: ch99, Alias: "ignored"},
						),
					})).To(Succeed())
					res := retrieve(ctx, l.Key)
					Expect(res.Channels).To(HaveLen(1))
					Expect(res.Channels[0].Alias).To(BeEmpty())
				},
			)

			It("Should clear a channel precision", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{
					Project: &proj.Key,
					Name:    "test",
					Channels: []log.ChannelEntry{
						{Channel: ch5, Precision: new(uint8(2))},
					},
				})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetChannelPrecisionAction(
						log.SetChannelPrecisionPayload{Channel: ch5},
					),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).Channels[0].Precision).To(BeNil())
			})

			DescribeTable("Should reject an out-of-range channel precision",
				func(ctx SpecContext, precision uint8) {
					l := defaults.Fill(log.Log{
						Project:  &proj.Key,
						Name:     "test",
						Channels: []log.ChannelEntry{{Channel: ch5}},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetChannelPrecisionAction(log.SetChannelPrecisionPayload{
							Channel:   ch5,
							Precision: &precision,
						}),
					})).Error().To(MatchError(ContainSubstring("channels.0.precision")))
				},
				Entry("above maximum", uint8(18)),
			)

			It(
				"Should repoint a channel in place via SwapChannel, keeping position and config",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{
						Project: &proj.Key,
						Name:    "test",
						Channels: []log.ChannelEntry{
							{Channel: ch1},
							{Channel: ch2, Alias: "kept", Precision: new(uint8(3))},
							{Channel: ch4},
						},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSwapChannelAction(
							log.SwapChannelPayload{From: ch2, To: ch5},
						),
					})).To(Succeed())
					res := retrieve(ctx, l.Key)
					Expect(res.Channels).To(HaveLen(3))
					Expect(res.Channels[1].Channel).To(Equal(ch5))
					Expect(res.Channels[1].Alias).To(Equal("kept"))
					Expect(res.Channels[1].Precision).To(Equal(new(uint8(3))))
				},
			)

			It(
				"Should treat SwapChannel as a no-op when the from channel is absent",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{
						Project:  &proj.Key,
						Name:     "test",
						Channels: []log.ChannelEntry{{Channel: ch1}},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSwapChannelAction(
							log.SwapChannelPayload{From: ch99, To: ch5},
						),
					})).To(Succeed())
					res := retrieve(ctx, l.Key)
					Expect(res.Channels).To(HaveLen(1))
					Expect(res.Channels[0].Channel).To(Equal(ch1))
				},
			)

			It(
				"Should treat SwapChannel as a no-op when the to channel has an entry",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{
						Project: &proj.Key,
						Name:    "test",
						Channels: []log.ChannelEntry{
							{Channel: ch1, Alias: "one"},
							{Channel: ch2, Alias: "two"},
						},
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSwapChannelAction(
							log.SwapChannelPayload{From: ch1, To: ch2},
						),
					})).To(Succeed())
					Expect(retrieve(ctx, l.Key).Channels).To(Equal(l.Channels))
				},
			)

			It(
				"Should set the timestamp precision via SetTimestampPrecision",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetTimestampPrecisionAction(
							log.SetTimestampPrecisionPayload{
								TimestampPrecision: 9,
							},
						),
					})).To(Succeed())
					Expect(retrieve(ctx, l.Key).TimestampPrecision).To(Equal(uint8(9)))
				},
			)

			DescribeTable("Should reject an out-of-range timestamp precision",
				func(ctx SpecContext, precision uint8) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetTimestampPrecisionAction(
							log.SetTimestampPrecisionPayload{
								TimestampPrecision: precision,
							},
						),
					})).Error().To(MatchError(ContainSubstring("timestamp_precision")))
				},
				Entry("above maximum", uint8(10)),
			)

			It("Should reject an action with no payload", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					{Type: log.ActionTypeSet},
				})).To(MatchError(union.ErrMissingPayload))
				Expect(retrieve(ctx, l.Key).Name).To(Equal("test"))
			})

			DescribeTable("Should reject an action that leaves the log invalid",
				func(ctx SpecContext, act func() log.Action, path string) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewAddChannelAction(log.AddChannelPayload{Channel: ch5}),
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d2", []log.Action{act()})).
						To(MatchError(ContainSubstring(path)))
					Expect(retrieve(ctx, l.Key).Channels).To(Equal([]log.ChannelEntry{
						entry(ch5, ""),
					}))
				},
				Entry(
					"empty timestamp format",
					func() log.Action {
						return log.NewSetChannelTimestampFormatAction(
							log.SetChannelTimestampFormatPayload{Channel: ch5},
						)
					},
					"channels.0.timestamp_config.format",
				),
				Entry(
					"empty time zone",
					func() log.Action {
						return log.NewSetChannelTimestampTimeZoneAction(
							log.SetChannelTimestampTimeZonePayload{Channel: ch5},
						)
					},
					"channels.0.timestamp_config.time_zone",
				),
				Entry(
					"entry with no timestamp",
					func() log.Action {
						return log.NewSetChannelEntryAction(log.SetChannelEntryPayload{
							Entry: log.ChannelEntry{Channel: ch5},
						})
					},
					"channels.0.timestamp_config",
				),
				Entry(
					"added channel of 0",
					func() log.Action {
						return log.NewAddChannelAction(
							log.AddChannelPayload{Channel: 0},
						)
					},
					"channels.1.channel",
				),
			)

			It("Should set the time zone via SetTimeZone", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).TimeZone).To(Equal(telem.TimeZoneLocal))
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetTimeZoneAction(
						log.SetTimeZonePayload{TimeZone: telem.TimeZoneUTC},
					),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).TimeZone).To(Equal(telem.TimeZoneUTC))
			})

			It("Should reject an invalid time zone", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetTimeZoneAction(log.SetTimeZonePayload{TimeZone: "Mars"}),
				})).To(MatchError(ContainSubstring("time_zone")))
			})

			It("Should toggle the display flags", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetChannelNamesHiddenAction(log.SetChannelNamesHiddenPayload{
						ChannelNamesHidden: true,
					}),
					log.NewSetTimestampsHiddenAction(
						log.SetTimestampsHiddenPayload{
							TimestampsHidden: true,
						},
					),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.ChannelNamesHidden).To(BeTrue())
				Expect(res.TimestampsHidden).To(BeTrue())
			})
		})

		Describe("atomicity and broadcast", func() {
			It("Should apply a multi-action batch atomically", func(ctx SpecContext) {
				l := defaults.Fill(log.Log{Project: &proj.Key, Name: "test"})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				Expect(dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetAction(
						log.SetPayload{Path: []string{"name"}, Value: "batched"},
					),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: ch10}),
					log.NewSetTimestampPrecisionAction(log.SetTimestampPrecisionPayload{
						TimestampPrecision: 2,
					}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Name).To(Equal("batched"))
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.TimestampPrecision).To(Equal(uint8(2)))
			})

			It(
				"Should not apply any action when one in the batch is rejected",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "before"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetAction(
							log.SetPayload{Path: []string{"name"}, Value: "after"},
						),
						log.NewSetTimestampPrecisionAction(
							log.SetTimestampPrecisionPayload{
								TimestampPrecision: 99,
							},
						),
					})).Error().To(MatchError(ContainSubstring("timestamp_precision")))
					Expect(retrieve(ctx, l.Key).Name).To(Equal("before"))
				},
			)

			It(
				"Should notify subscribers with the dispatched ScopedAction on success",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "observed"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					rec := &Recorder[log.Key, log.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					actions := []log.Action{
						log.NewSetAction(
							log.SetPayload{Path: []string{"name"}, Value: "broadcast"},
						),
						log.NewAddChannelAction(log.AddChannelPayload{Channel: ch5}),
					}
					Expect(
						dispatch(ctx, l.Key, "client-xyz", actions),
					).
						To(Succeed())
					seen := rec.Snapshot()
					Expect(seen).To(HaveLen(1))
					Expect(seen[0].Key).To(Equal(l.Key))
					Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
					Expect(seen[0].Actions).To(HaveLen(2))
					Expect(seen[0].Actions[0].Type).To(Equal(log.ActionTypeSet))
					Expect(seen[0].Actions[1].Type).To(Equal(log.ActionTypeAddChannel))
				},
			)

			It(
				"Should assign strictly monotonic Seq across successive dispatches",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "seq"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					rec := &Recorder[log.Key, log.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					for _, name := range []string{"a", "b", "c"} {
						Expect(dispatch(ctx, l.Key, "d", []log.Action{
							log.NewSetAction(
								log.SetPayload{Path: []string{"name"}, Value: name},
							),
						})).To(Succeed())
					}
					seen := rec.Snapshot()
					Expect(seen).To(HaveLen(3))
					Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
					Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
				},
			)

			It(
				"Should not notify subscribers when Reduce rejects the action",
				func(ctx SpecContext) {
					l := defaults.Fill(log.Log{Project: &proj.Key, Name: "rejected"})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &l)
					})).To(Succeed())
					rec := &Recorder[log.Key, log.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					Expect(dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetTimestampPrecisionAction(
							log.SetTimestampPrecisionPayload{
								TimestampPrecision: 99,
							},
						),
					})).Error().To(MatchError(ContainSubstring("timestamp_precision")))
					Expect(rec.Snapshot()).To(BeEmpty())
				},
			)
		})
	})
})
