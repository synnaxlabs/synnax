// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// b3Guard refuses the delete of its channel and is deleted with its status.
type b3Guard struct {
	Key     string
	Channel uint32
	Status  string
}

func (g b3Guard) GorpKey() string { return g.Key }

// b3Label inlines its channel when the channel is deleted.
type b3Label struct {
	Key     string
	Channel uint32
}

func (l b3Label) GorpKey() string { return l.Key }

var errB3Rollback = errors.New("rolled back")

var _ = Describe("Planned delete breaker 3", func() {
	var (
		db          *gorp.DB
		channels    *gorp.Table[uint32, b2Channel]
		tasks       *gorp.Table[string, b2Task]
		plots       *gorp.Table[string, b2Plot]
		statuses    *gorp.Table[string, b2Status]
		guards      *gorp.Table[string, b3Guard]
		onDetach    func(ctx context.Context, tx gorp.Tx, t b2Task) error
		onTargetSet func(ctx context.Context, tx gorp.Tx, t b2Task) error
	)
	BeforeEach(func(ctx SpecContext) {
		onDetach, onTargetSet = nil, nil
		db = DeferClose(gorp.Wrap(
			memkv.New(),
			gorp.WithCodec(msgpack.Codec),
			gorp.WithMaxDeleteSize(6),
		))
		StartRun(db.Run)
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, b2Channel]{
			DB:     db,
			Name:   "channel",
			Issued: func(uint32) bool { return true },
		}))
		tasks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b2Task]{
			DB:   db,
			Name: "task",
			References: []gorp.Reference[string, b2Task]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Task, uint32]{
					Name:     "reads",
					Table:    "channel",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets:  func(t *b2Task) []uint32 { return slices.Clone(t.Reads) },
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Task, uint32]{
					Name:     "plotted",
					Table:    "channel",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(t *b2Task) []uint32 { return slices.Clone(t.Plotted) },
					Detach: func(t *b2Task, k uint32) {
						t.Plotted = b2Without(t.Plotted, k)
					},
				}),
			},
			OnRewrite: func(ctx context.Context, tx gorp.Tx, t b2Task) error {
				if onDetach == nil {
					return nil
				}
				return onDetach(ctx, tx, t)
			},
			OnTargetSet: map[string]func(context.Context, gorp.Tx, b2Task) error{
				"channel": func(ctx context.Context, tx gorp.Tx, t b2Task) error {
					if onTargetSet == nil {
						return nil
					}
					return onTargetSet(ctx, tx, t)
				},
			},
		}))
		plots = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b2Plot]{
			DB:   db,
			Name: "plot",
			References: []gorp.Reference[string, b2Plot]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Plot, string]{
					Name:     "task",
					Table:    "task",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets: func(p *b2Plot) []string {
						if p.Task == "" {
							return nil
						}
						return []string{p.Task}
					},
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Plot, uint32]{
					Name:     "plotted",
					Table:    "channel",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(p *b2Plot) []uint32 { return slices.Clone(p.Plotted) },
					Detach: func(p *b2Plot, k uint32) {
						p.Plotted = b2Without(p.Plotted, k)
					},
				}),
			},
		}))
		statuses = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b2Status]{
			DB:   db,
			Name: "status",
			References: []gorp.Reference[string, b2Status]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Status, uint32]{
					Name:     "channel",
					Table:    "channel",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets:  func(s *b2Status) []uint32 { return []uint32{s.Channel} },
				}),
			},
		}))
		guards = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b3Guard]{
			DB:   db,
			Name: "guard",
			References: []gorp.Reference[string, b3Guard]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b3Guard, uint32]{
					Name:     "channel",
					Table:    "channel",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets:  func(g *b3Guard) []uint32 { return []uint32{g.Channel} },
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b3Guard, string]{
					Name:     "status",
					Table:    "status",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets: func(g *b3Guard) []string {
						if g.Status == "" {
							return nil
						}
						return []string{g.Status}
					},
				}),
			},
		}))
		for k := uint32(1); k <= 3; k++ {
			Expect(channels.NewCreate().Entry(&b2Channel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
	})
	// plan plans the delete of the channels with planned in tx, runs between, then
	// deletes the channels with deleted through Delete.Planned.
	plan := func(
		ctx SpecContext,
		tx gorp.Tx,
		planned, deleted []uint32,
		between func(tx gorp.Tx) error,
	) error {
		effects, err := channels.PlanDelete(ctx, tx, planned)
		if err != nil {
			return err
		}
		if between != nil {
			if err = between(tx); err != nil {
				return err
			}
		}
		return channels.NewDelete().
			Where(gorp.MatchKeys[uint32, b2Channel](deleted...)).
			Planned(effects).
			Exec(ctx, tx)
	}
	planThenDelete := func(
		ctx SpecContext,
		between func(tx gorp.Tx) error,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return plan(ctx, tx, []uint32{1}, []uint32{1}, between)
		})
	}
	// rolledBack runs f in a transaction that it rolls back, and returns the error of
	// f.
	rolledBack := func(ctx SpecContext, f func(tx gorp.Tx) error) error {
		GinkgoHelper()
		var got error
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			got = f(tx)
			return errB3Rollback
		})).To(MatchError(errB3Rollback))
		return got
	}
	// freshDelete runs between, then deletes channel 1 with a walk of its own.
	freshDelete := func(ctx SpecContext, between func(tx gorp.Tx) error) error {
		GinkgoHelper()
		return rolledBack(ctx, func(tx gorp.Tx) error {
			if err := between(tx); err != nil {
				return err
			}
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, b2Channel](1)).
				Exec(ctx, tx)
		})
	}
	plannedDelete := func(ctx SpecContext, between func(tx gorp.Tx) error) error {
		GinkgoHelper()
		return rolledBack(ctx, func(tx gorp.Tx) error {
			return plan(ctx, tx, []uint32{1}, []uint32{1}, between)
		})
	}
	channelExists := func(ctx SpecContext, k uint32) bool {
		GinkgoHelper()
		return MustSucceed(channels.NewRetrieve().
			Where(gorp.MatchKeys[uint32, b2Channel](k)).
			Exists(ctx, db))
	}
	statusExists := func(ctx SpecContext, k string) bool {
		GinkgoHelper()
		return MustSucceed(statuses.NewRetrieve().
			Where(gorp.MatchKeys[string, b2Status](k)).
			Exists(ctx, db))
	}
	retrieveTask := func(ctx SpecContext, k string) b2Task {
		GinkgoHelper()
		var t b2Task
		Expect(tasks.NewRetrieve().
			Where(gorp.MatchKeys[string, b2Task](k)).
			Entry(&t).
			Exec(ctx, db)).To(Succeed())
		return t
	}
	retrievePlot := func(ctx SpecContext, k string) b2Plot {
		GinkgoHelper()
		var p b2Plot
		Expect(plots.NewRetrieve().
			Where(gorp.MatchKeys[string, b2Plot](k)).
			Entry(&p).
			Exec(ctx, db)).To(Succeed())
		return p
	}
	readsRestrict := func(holder string) gorp.RestrictError {
		return gorp.RestrictError{
			Target:    gorp.NewTarget("channel", uint32(1)),
			Holder:    gorp.Target{Table: "task", Key: holder},
			Reference: "reads",
		}
	}
	createTask := func(ctx SpecContext, t b2Task) func(tx gorp.Tx) error {
		return func(tx gorp.Tx) error {
			return tasks.NewCreate().Entry(&t).Exec(ctx, tx)
		}
	}

	Describe("Findings", func() {
		It("Should refuse the delete of a channel with a plan made for another "+
			"channel", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "t", Reads: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return plan(ctx, tx, []uint32{2}, []uint32{1}, nil)
			})).To(MatchError(readsRestrict("t")))
			Expect(channelExists(ctx, 1)).To(BeTrue())
		})

		It("Should keep the holders of a planned channel that the delete keeps",
			func(ctx SpecContext) {
				Expect(statuses.NewCreate().
					Entry(&b2Status{Key: "s2", Channel: 2}).
					Exec(ctx, db)).To(Succeed())
				Expect(plots.NewCreate().
					Entry(&b2Plot{Key: "p", Plotted: []uint32{2}}).
					Exec(ctx, db)).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return plan(ctx, tx, []uint32{1, 2}, []uint32{1}, nil)
				})).To(Succeed())
				Expect(channelExists(ctx, 2)).To(BeTrue())
				Expect(statusExists(ctx, "s2")).To(BeTrue())
				Expect(retrievePlot(ctx, "p").Plotted).To(Equal([]uint32{2}))
			})

		It("Should refuse like a fresh walk once a delete with no table drops a "+
			"cascade path after the plan", func(ctx SpecContext) {
			Expect(statuses.NewCreate().
				Entry(&b2Status{Key: "s", Channel: 1}).
				Exec(ctx, db)).To(Succeed())
			Expect(guards.NewCreate().
				Entry(&b3Guard{Key: "g", Channel: 1, Status: "s"}).
				Exec(ctx, db)).To(Succeed())
			dropStatus := func(tx gorp.Tx) error {
				return gorp.NewDelete[string, b2Status]().
					Where(gorp.MatchKeys[string, b2Status]("s")).
					Exec(ctx, tx)
			}
			restrict := gorp.RestrictError{
				Target:    gorp.NewTarget("channel", uint32(1)),
				Holder:    gorp.Target{Table: "guard", Key: "g"},
				Reference: "channel",
			}
			Expect(freshDelete(ctx, dropStatus)).To(MatchError(restrict))
			Expect(plannedDelete(ctx, dropStatus)).To(MatchError(restrict))
		})

		It("Should fail like a fresh walk when the Inline of a planned holder finds "+
			"nothing", func(ctx SpecContext) {
			labels := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b3Label]{
				DB:   db,
				Name: "label",
				References: []gorp.Reference[string, b3Label]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, b3Label, uint32]{
							Name:     "channel",
							Table:    "channel",
							OnDelete: gorp.Inline,
							Kind:     gorp.Use,
							Targets: func(l *b3Label) []uint32 {
								if l.Channel == 0 {
									return nil
								}
								return []uint32{l.Channel}
							},
							Inline: func(context.Context, gorp.Tx, *b3Label, uint32) error {
								return query.ErrNotFound
							},
						},
					),
				},
			}))
			Expect(labels.NewCreate().
				Entry(&b3Label{Key: "l", Channel: 1}).
				Exec(ctx, db)).To(Succeed())
			none := func(gorp.Tx) error { return nil }
			Expect(freshDelete(ctx, none)).To(MatchError(query.ErrNotFound))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return plan(ctx, tx, []uint32{1}, []uint32{1}, nil)
			})).To(MatchError(query.ErrNotFound))
			Expect(channelExists(ctx, 1)).To(BeTrue())
		})

		It("Should fail like a fresh walk when the OnRewrite of a planned holder "+
			"finds nothing", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "p", Plotted: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			onDetach = func(context.Context, gorp.Tx, b2Task) error {
				return query.ErrNotFound
			}
			none := func(gorp.Tx) error { return nil }
			Expect(freshDelete(ctx, none)).To(MatchError(query.ErrNotFound))
			Expect(plannedDelete(ctx, none)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Cases that hold", func() {
		It("Should refuse with a Restrict holder that WrapWriter writes after the plan",
			func(ctx SpecContext) {
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return gorp.WrapWriter[string, b2Task](tx).
						Set(ctx, b2Task{Key: "late", Reads: []uint32{1}})
				})).To(MatchError(readsRestrict("late")))
			})

		It("Should refuse with a Restrict holder that a create with no table writes "+
			"after the plan", func(ctx SpecContext) {
			Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[string, b2Task]().
					Entry(&b2Task{Key: "late", Reads: []uint32{1}}).
					Exec(ctx, tx)
			})).To(MatchError(readsRestrict("late")))
		})

		It(
			"Should refuse with a Restrict holder that an unchecked writer writes after "+
				"the plan",
			func(ctx SpecContext) {
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return tasks.NewUncheckedWriter(tx).
						Set(ctx, b2Task{Key: "late", Reads: []uint32{1}})
				})).To(MatchError(readsRestrict("late")))
			},
		)

		It("Should refuse with a Restrict link that an update adds after the plan",
			func(ctx SpecContext) {
				Expect(tasks.NewCreate().
					Entry(&b2Task{Key: "free"}).
					Exec(ctx, db)).To(Succeed())
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return tasks.NewUpdate().
						Where(gorp.MatchKeys[string, b2Task]("free")).
						Change(func(_ gorp.Context, t b2Task) b2Task {
							t.Reads = []uint32{1}
							return t
						}).
						Exec(ctx, tx)
				})).To(MatchError(readsRestrict("free")))
			})

		It("Should keep a Cascade holder that an update repoints after the plan",
			func(ctx SpecContext) {
				Expect(statuses.NewCreate().
					Entry(&b2Status{Key: "s", Channel: 1}).
					Exec(ctx, db)).To(Succeed())
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return statuses.NewUpdate().
						Where(gorp.MatchKeys[string, b2Status]("s")).
						ChangeErr(func(_ gorp.Context, s b2Status) (b2Status, error) {
							s.Channel = 2
							return s, nil
						}).
						Exec(ctx, tx)
				})).To(Succeed())
				Expect(statusExists(ctx, "s")).To(BeTrue())
			})

		It("Should detach a holder that another delete detaches after the plan",
			func(ctx SpecContext) {
				Expect(plots.NewCreate().
					Entry(&b2Plot{Key: "p", Plotted: []uint32{1, 2}}).
					Exec(ctx, db)).To(Succeed())
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return channels.NewDelete().
						Where(gorp.MatchKeys[uint32, b2Channel](2)).
						Exec(ctx, tx)
				})).To(Succeed())
				Expect(retrievePlot(ctx, "p").Plotted).To(BeEmpty())
			})

		It("Should refuse with a holder that an OnRewrite hook of another delete "+
			"writes after the plan", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "p2", Plotted: []uint32{2}}).
				Exec(ctx, db)).To(Succeed())
			onDetach = func(ctx context.Context, tx gorp.Tx, _ b2Task) error {
				return tasks.NewCreate().
					Entry(&b2Task{Key: "late", Reads: []uint32{1}}).
					Exec(ctx, tx)
			}
			Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
				return channels.NewDelete().
					Where(gorp.MatchKeys[uint32, b2Channel](2)).
					Exec(ctx, tx)
			})).To(MatchError(readsRestrict("late")))
		})

		It("Should refuse with a holder that an OnTargetSet hook writes after the plan",
			func(ctx SpecContext) {
				Expect(tasks.NewCreate().
					Entry(&b2Task{Key: "w", Plotted: []uint32{2}}).
					Exec(ctx, db)).To(Succeed())
				onTargetSet = func(ctx context.Context, tx gorp.Tx, _ b2Task) error {
					return guards.NewCreate().
						Entry(&b3Guard{Key: "late", Channel: 1}).
						Exec(ctx, tx)
				}
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return channels.NewCreate().
						Entry(&b2Channel{Key: 2}).
						Exec(ctx, tx)
				})).To(MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("channel", uint32(1)),
					Holder:    gorp.Target{Table: "guard", Key: "late"},
					Reference: "channel",
				}))
			})

		It("Should refuse with a holder that a failed hook wrote after the plan",
			func(ctx SpecContext) {
				Expect(tasks.NewCreate().
					Entry(&b2Task{Key: "w", Plotted: []uint32{2}}).
					Exec(ctx, db)).To(Succeed())
				hookErr := errors.New("hook failed")
				onTargetSet = func(ctx context.Context, tx gorp.Tx, _ b2Task) error {
					if err := tasks.NewCreate().
						Entry(&b2Task{Key: "late", Reads: []uint32{1}}).
						Exec(ctx, tx); err != nil {
						return err
					}
					return hookErr
				}
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					Expect(channels.NewCreate().
						Entry(&b2Channel{Key: 2}).
						Exec(ctx, tx)).To(MatchError(hookErr))
					return nil
				})).To(MatchError(readsRestrict("late")))
			})

		It("Should apply the plan after a write that failed its check", func(
			ctx SpecContext,
		) {
			Expect(statuses.NewCreate().
				Entry(&b2Status{Key: "s", Channel: 1}).
				Exec(ctx, db)).To(Succeed())
			Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
				Expect(tasks.NewCreate().
					Entry(&b2Task{Key: "late", Reads: []uint32{1, 9}}).
					Exec(ctx, tx),
				).To(MatchError(ContainSubstring("channel 9 does not exist")))
				return nil
			})).To(Succeed())
			Expect(statusExists(ctx, "s")).To(BeFalse())
			Expect(MustSucceed(tasks.NewRetrieve().
				Where(gorp.MatchKeys[string, b2Task]("late")).
				Exists(ctx, db))).To(BeFalse())
		})

		It("Should detach a holder in a second table that the transaction writes "+
			"after the plan", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "t", Plotted: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
				return plots.NewCreate().
					Entry(&b2Plot{Key: "late", Plotted: []uint32{1, 2}}).
					Exec(ctx, tx)
			})).To(Succeed())
			Expect(retrievePlot(ctx, "late").Plotted).To(Equal([]uint32{2}))
			Expect(retrieveTask(ctx, "t").Plotted).To(BeEmpty())
		})

		It("Should walk again for the first of two plans after a write that the "+
			"second saw", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				first := MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1}))
				Expect(createTask(ctx, b2Task{Key: "late", Reads: []uint32{1}})(tx)).
					To(Succeed())
				second := MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1}))
				Expect(second.Restricts).To(HaveLen(1))
				return channels.NewDelete().
					Where(gorp.MatchKeys[uint32, b2Channel](1)).
					Planned(first).
					Exec(ctx, tx)
			})).To(MatchError(readsRestrict("late")))
		})

		It("Should refuse with a plan used again after the channel comes back with a "+
			"holder", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				effects := MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1}))
				del := channels.NewDelete().
					Where(gorp.MatchKeys[uint32, b2Channel](1)).
					Planned(effects)
				Expect(del.Exec(ctx, tx)).To(Succeed())
				Expect(channels.NewCreate().
					Entry(&b2Channel{Key: 1}).
					Exec(ctx, tx)).To(Succeed())
				Expect(createTask(ctx, b2Task{Key: "late", Reads: []uint32{1}})(tx)).
					To(Succeed())
				return del.Exec(ctx, tx)
			})).To(MatchError(readsRestrict("late")))
		})

		It("Should allow a Restrict holder that WrapWriter deletes after the plan",
			func(ctx SpecContext) {
				Expect(tasks.NewCreate().
					Entry(&b2Task{Key: "t", Reads: []uint32{1}}).
					Exec(ctx, db)).To(Succeed())
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					return gorp.WrapWriter[string, b2Task](tx).Delete(ctx, "t")
				})).To(Succeed())
				Expect(channelExists(ctx, 1)).To(BeFalse())
			})

		It("Should refuse with a Restrict holder deleted and written again after "+
			"the plan", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "t", Reads: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
				if err := tasks.NewDelete().
					Where(gorp.MatchKeys[string, b2Task]("t")).
					Exec(ctx, tx); err != nil {
					return err
				}
				return createTask(ctx, b2Task{Key: "t", Reads: []uint32{1}})(tx)
			})).To(MatchError(readsRestrict("t")))
		})

		It("Should fail on MaxDeleteSize when the walk again grows past it",
			func(ctx SpecContext) {
				Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
					for _, k := range []string{"a", "b", "c", "d", "e", "f"} {
						if err := statuses.NewCreate().
							Entry(&b2Status{Key: k, Channel: 1}).
							Exec(ctx, tx); err != nil {
							return err
						}
					}
					return nil
				})).To(MatchError(ContainSubstring("would change more than 6 entries")))
				Expect(channelExists(ctx, 1)).To(BeTrue())
			})

		It("Should handle a channel that the transaction creates after the plan",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return plan(
						ctx,
						tx,
						[]uint32{1, 4},
						[]uint32{1, 4},
						func(tx gorp.Tx) error {
							if err := channels.NewCreate().
								Entry(&b2Channel{Key: 4}).
								Exec(ctx, tx); err != nil {
								return err
							}
							return statuses.NewCreate().
								Entry(&b2Status{Key: "s4", Channel: 4}).
								Exec(ctx, tx)
						},
					)
				})).To(Succeed())
				Expect(statusExists(ctx, "s4")).To(BeFalse())
			})

		It("Should match a fresh delete for a Restrict holder that a raw set writes "+
			"after the plan", func(ctx SpecContext) {
			rawHolder := func(tx gorp.Tx) error {
				codec := key.NewCodec[string](key.Prefix[b2Task]())
				raw, err := msgpack.Codec.Encode(b2Task{Key: "raw", Reads: []uint32{1}})
				if err != nil {
					return err
				}
				return tx.Set(ctx, bytes.Clone(codec.Encode("raw")), raw)
			}
			Expect(fmt.Sprint(plannedDelete(ctx, rawHolder))).
				To(Equal(fmt.Sprint(freshDelete(ctx, rawHolder))))
		})

		It("Should match a fresh delete for a Restrict holder that a raw set writes "+
			"again after a typed delete", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "t", Reads: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			again := func(tx gorp.Tx) error {
				if err := tasks.NewDelete().
					Where(gorp.MatchKeys[string, b2Task]("t")).
					Exec(ctx, tx); err != nil {
					return err
				}
				codec := key.NewCodec[string](key.Prefix[b2Task]())
				raw, err := msgpack.Codec.Encode(b2Task{Key: "t", Reads: []uint32{1}})
				if err != nil {
					return err
				}
				return tx.Set(ctx, bytes.Clone(codec.Encode("t")), raw)
			}
			Expect(fmt.Sprint(plannedDelete(ctx, again))).
				To(Equal(fmt.Sprint(freshDelete(ctx, again))))
		})

		It("Should match a fresh delete when the detach of the delete itself writes "+
			"a holder", func(ctx SpecContext) {
			Expect(tasks.NewCreate().
				Entry(&b2Task{Key: "p", Plotted: []uint32{1}}).
				Exec(ctx, db)).To(Succeed())
			onDetach = func(ctx context.Context, tx gorp.Tx, _ b2Task) error {
				return guards.NewCreate().
					Entry(&b3Guard{Key: "late", Channel: 1}).
					Exec(ctx, tx)
			}
			none := func(gorp.Tx) error { return nil }
			Expect(fmt.Sprint(plannedDelete(ctx, none))).
				To(Equal(fmt.Sprint(freshDelete(ctx, none))))
		})
	})
})
