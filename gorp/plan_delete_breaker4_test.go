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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// b4Formula inlines each channel it reads, in the order the delete gives them.
type b4Formula struct {
	Key      string
	Channels []uint32
	Text     string
}

func (f b4Formula) GorpKey() string { return f.Key }

// b4Pin watches channels through a Weak reference and plots them through a Detach
// reference.
type b4Pin struct {
	Key     string
	Watched []uint32
	Plotted []uint32
}

func (p b4Pin) GorpKey() string { return p.Key }

// b4Note detaches the guards it names.
type b4Note struct {
	Key    string
	Guards []string
}

func (n b4Note) GorpKey() string { return n.Key }

// b4Lock refuses the delete of its guard.
type b4Lock struct {
	Key   string
	Guard string
}

func (l b4Lock) GorpKey() string { return l.Key }

var errB4Rollback = errors.New("rolled back")

var _ = Describe("Planned delete breaker 4", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, b2Channel]
		tasks    *gorp.Table[string, b2Task]
		plots    *gorp.Table[string, b2Plot]
		statuses *gorp.Table[string, b2Status]
		guards   *gorp.Table[string, b3Guard]
		formulas *gorp.Table[string, b4Formula]
		pins     *gorp.Table[string, b4Pin]
		notes    *gorp.Table[string, b4Note]
		locks    *gorp.Table[string, b4Lock]
		onDetach func(ctx context.Context, tx gorp.Tx, t b2Task) error
	)
	BeforeEach(func(ctx SpecContext) {
		onDetach = nil
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
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
					Targets: func(s *b2Status) []uint32 {
						if s.Channel == 0 {
							return nil
						}
						return []uint32{s.Channel}
					},
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
					Targets: func(g *b3Guard) []uint32 {
						if g.Channel == 0 {
							return nil
						}
						return []uint32{g.Channel}
					},
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
		formulas = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b4Formula]{
			DB:   db,
			Name: "formula",
			References: []gorp.Reference[string, b4Formula]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b4Formula, uint32]{
					Name:     "channels",
					Table:    "channel",
					OnDelete: gorp.Inline,
					Kind:     gorp.Use,
					Targets: func(f *b4Formula) []uint32 {
						return slices.Clone(f.Channels)
					},
					Inline: func(
						_ context.Context,
						_ gorp.Tx,
						f *b4Formula,
						k uint32,
					) error {
						f.Text += fmt.Sprintf("%d;", k)
						f.Channels = b2Without(f.Channels, k)
						return nil
					},
				}),
			},
		}))
		pins = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b4Pin]{
			DB:   db,
			Name: "pin",
			References: []gorp.Reference[string, b4Pin]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b4Pin, uint32]{
					Name:     "watched",
					Table:    "channel",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(p *b4Pin) []uint32 { return slices.Clone(p.Watched) },
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b4Pin, uint32]{
					Name:     "plotted",
					Table:    "channel",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(p *b4Pin) []uint32 { return slices.Clone(p.Plotted) },
					Detach: func(p *b4Pin, k uint32) {
						p.Plotted = b2Without(p.Plotted, k)
					},
				}),
			},
		}))
		notes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b4Note]{
			DB:   db,
			Name: "note",
			References: []gorp.Reference[string, b4Note]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b4Note, string]{
					Name:     "guards",
					Table:    "guard",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(n *b4Note) []string { return slices.Clone(n.Guards) },
					Detach: func(n *b4Note, k string) {
						n.Guards = slices.DeleteFunc(n.Guards, func(g string) bool {
							return g == k
						})
					},
				}),
			},
		}))
		locks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b4Lock]{
			DB:   db,
			Name: "lock",
			References: []gorp.Reference[string, b4Lock]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b4Lock, string]{
					Name:     "guard",
					Table:    "guard",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets:  func(l *b4Lock) []string { return []string{l.Guard} },
				}),
			},
		}))
		for k := uint32(1); k <= 3; k++ {
			Expect(channels.NewCreate().Entry(&b2Channel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
	})
	// state returns every entry of every table, as tx sees them.
	state := func(ctx SpecContext, tx gorp.Tx) string {
		GinkgoHelper()
		var (
			cs []b2Channel
			ts []b2Task
			ps []b2Plot
			ss []b2Status
			gs []b3Guard
			fs []b4Formula
			pn []b4Pin
			ns []b4Note
			ls []b4Lock
		)
		Expect(channels.NewRetrieve().Entries(&cs).Exec(ctx, tx)).To(Succeed())
		Expect(tasks.NewRetrieve().Entries(&ts).Exec(ctx, tx)).To(Succeed())
		Expect(plots.NewRetrieve().Entries(&ps).Exec(ctx, tx)).To(Succeed())
		Expect(statuses.NewRetrieve().Entries(&ss).Exec(ctx, tx)).To(Succeed())
		Expect(guards.NewRetrieve().Entries(&gs).Exec(ctx, tx)).To(Succeed())
		Expect(formulas.NewRetrieve().Entries(&fs).Exec(ctx, tx)).To(Succeed())
		Expect(pins.NewRetrieve().Entries(&pn).Exec(ctx, tx)).To(Succeed())
		Expect(notes.NewRetrieve().Entries(&ns).Exec(ctx, tx)).To(Succeed())
		Expect(locks.NewRetrieve().Entries(&ls).Exec(ctx, tx)).To(Succeed())
		return fmt.Sprint(cs, ts, ps, ss, gs, fs, pn, ns, ls)
	}
	// outcome runs f in a transaction that it rolls back, and returns the error of f
	// and the state f leaves.
	outcome := func(ctx SpecContext, f func(tx gorp.Tx) error) string {
		GinkgoHelper()
		var got string
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			err := f(tx)
			got = fmt.Sprintf("err=%v state=%s", err, state(ctx, tx))
			return errB4Rollback
		})).To(MatchError(errB4Rollback))
		return got
	}
	// fresh runs between, then deletes the channels with deleted with a walk of its
	// own.
	fresh := func(
		ctx SpecContext,
		deleted []uint32,
		between func(tx gorp.Tx) error,
	) string {
		GinkgoHelper()
		return outcome(ctx, func(tx gorp.Tx) error {
			if err := between(tx); err != nil {
				return err
			}
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, b2Channel](deleted...)).
				Exec(ctx, tx)
		})
	}
	// planned plans the delete of the channels with planned, runs between, then deletes
	// the channels with deleted through Delete.Planned.
	planned := func(
		ctx SpecContext,
		plannedKeys, deleted []uint32,
		between func(tx gorp.Tx) error,
	) string {
		GinkgoHelper()
		return outcome(ctx, func(tx gorp.Tx) error {
			effects, err := channels.PlanDelete(ctx, tx, plannedKeys)
			if err != nil {
				return err
			}
			if err = between(tx); err != nil {
				return err
			}
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, b2Channel](deleted...)).
				Planned(effects).
				Exec(ctx, tx)
		})
	}
	// same expects the planned delete of channel 1 to end as the fresh delete does.
	same := func(ctx SpecContext, between func(tx gorp.Tx) error) {
		GinkgoHelper()
		one := []uint32{1}
		Expect(planned(ctx, one, one, between)).To(Equal(fresh(ctx, one, between)))
	}
	none := func(gorp.Tx) error { return nil }
	// rolledBack runs f in a transaction that it rolls back, and returns the error of
	// f.
	rolledBack := func(ctx SpecContext, f func(tx gorp.Tx) error) error {
		GinkgoHelper()
		var got error
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			got = f(tx)
			return errB4Rollback
		})).To(MatchError(errB4Rollback))
		return got
	}
	freshDelete := func(
		ctx SpecContext,
		deleted []uint32,
		between func(tx gorp.Tx) error,
	) error {
		GinkgoHelper()
		return rolledBack(ctx, func(tx gorp.Tx) error {
			if err := between(tx); err != nil {
				return err
			}
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, b2Channel](deleted...)).
				Exec(ctx, tx)
		})
	}
	plannedDelete := func(
		ctx SpecContext,
		plannedKeys, deleted []uint32,
		between func(tx gorp.Tx) error,
	) error {
		GinkgoHelper()
		return rolledBack(ctx, func(tx gorp.Tx) error {
			effects, err := channels.PlanDelete(ctx, tx, plannedKeys)
			if err != nil {
				return err
			}
			if err = between(tx); err != nil {
				return err
			}
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, b2Channel](deleted...)).
				Planned(effects).
				Exec(ctx, tx)
		})
	}
	create := func(ctx SpecContext, entries ...any) {
		GinkgoHelper()
		for _, e := range entries {
			var err error
			switch e := e.(type) {
			case b2Task:
				err = tasks.NewCreate().Entry(&e).Exec(ctx, db)
			case b2Plot:
				err = plots.NewCreate().Entry(&e).Exec(ctx, db)
			case b2Status:
				err = statuses.NewCreate().Entry(&e).Exec(ctx, db)
			case b3Guard:
				err = guards.NewCreate().Entry(&e).Exec(ctx, db)
			case b4Formula:
				err = formulas.NewCreate().Entry(&e).Exec(ctx, db)
			case b4Pin:
				err = pins.NewCreate().Entry(&e).Exec(ctx, db)
			case b4Note:
				err = notes.NewCreate().Entry(&e).Exec(ctx, db)
			case b4Lock:
				err = locks.NewCreate().Entry(&e).Exec(ctx, db)
			case b2Channel:
				err = channels.NewCreate().Entry(&e).Exec(ctx, db)
			default:
				Fail(fmt.Sprintf("no table for %T", e))
			}
			Expect(err).To(Succeed())
		}
	}
	deleteTask := func(ctx SpecContext, k string) func(tx gorp.Tx) error {
		return func(tx gorp.Tx) error {
			return tasks.NewDelete().
				Where(gorp.MatchKeys[string, b2Task](k)).
				Exec(ctx, tx)
		}
	}
	wrapDeleteTask := func(ctx SpecContext, k string) func(tx gorp.Tx) error {
		return func(tx gorp.Tx) error {
			return gorp.WrapWriter[string, b2Task](tx).Delete(ctx, k)
		}
	}

	Describe("Findings", func() {
		It("Should fail like a fresh walk when the OnRewrite of one planned holder "+
			"deletes another planned holder", func(ctx SpecContext) {
			create(
				ctx,
				b2Task{Key: "a", Plotted: []uint32{1}},
				b2Task{Key: "b", Plotted: []uint32{1}},
			)
			onDetach = func(ctx context.Context, tx gorp.Tx, t b2Task) error {
				other := "a"
				if t.Key == "a" {
					other = "b"
				}
				return gorp.WrapWriter[string, b2Task](tx).Delete(ctx, other)
			}
			one := []uint32{1}
			Expect(freshDelete(ctx, one, none)).To(MatchError(query.ErrNotFound))
			Expect(plannedDelete(ctx, one, one, none)).To(MatchError(query.ErrNotFound))
		})

		It("Should leave a holder that a hook writes again during the delete as a "+
			"fresh walk does", func(ctx SpecContext) {
			create(
				ctx,
				b2Task{Key: "d", Plotted: []uint32{1}},
				b2Task{Key: "e", Plotted: []uint32{1}},
			)
			onDetach = func(ctx context.Context, tx gorp.Tx, t b2Task) error {
				if t.Key != "e" {
					return nil
				}
				return gorp.WrapWriter[string, b2Task](tx).
					Set(ctx, b2Task{Key: "d", Plotted: []uint32{1}})
			}
			one := []uint32{1}
			between := wrapDeleteTask(ctx, "d")
			plottedByD := func(tx gorp.Tx) string {
				var d b2Task
				Expect(tasks.NewRetrieve().
					Where(gorp.MatchKeys[string, b2Task]("d")).
					Entry(&d).
					Exec(ctx, tx)).To(Succeed())
				return fmt.Sprint(d.Plotted)
			}
			var fresh string
			Expect(rolledBack(ctx, func(tx gorp.Tx) error {
				Expect(between(tx)).To(Succeed())
				Expect(channels.NewDelete().
					Where(gorp.MatchKeys[uint32, b2Channel](1)).
					Exec(ctx, tx)).To(Succeed())
				fresh = plottedByD(tx)
				return nil
			})).To(Succeed())
			Expect(fresh).To(Equal("[1]"))
			plannedResults := set.New[string]()
			for range 32 {
				Expect(rolledBack(ctx, func(tx gorp.Tx) error {
					effects := MustSucceed(channels.PlanDelete(ctx, tx, one))
					Expect(between(tx)).To(Succeed())
					Expect(channels.NewDelete().
						Where(gorp.MatchKeys[uint32, b2Channel](1)).
						Planned(effects).
						Exec(ctx, tx)).To(Succeed())
					plannedResults.Add(plottedByD(tx))
					return nil
				})).To(Succeed())
			}
			Expect(plannedResults.Slice()).To(ConsistOf(fresh))
		})

		It("Should inline the targets of a holder in the order a fresh walk does",
			func(ctx SpecContext) {
				create(
					ctx,
					b2Channel{Key: 9},
					b2Channel{Key: 10},
					b4Formula{Key: "f", Channels: []uint32{9, 10}},
				)
				keys := []uint32{9, 10}
				text := func(planned bool) string {
					var got string
					Expect(rolledBack(ctx, func(tx gorp.Tx) error {
						del := channels.NewDelete().
							Where(gorp.MatchKeys[uint32, b2Channel](keys...))
						if planned {
							del = del.Planned(
								MustSucceed(channels.PlanDelete(ctx, tx, keys)),
							)
						}
						Expect(del.Exec(ctx, tx)).To(Succeed())
						var f b4Formula
						Expect(formulas.NewRetrieve().
							Where(gorp.MatchKeys[string, b4Formula]("f")).
							Entry(&f).
							Exec(ctx, tx)).To(Succeed())
						got = f.Text
						return nil
					})).To(Succeed())
					return got
				}
				Expect(text(false)).To(Equal("9;10;"))
				Expect(text(true)).To(Equal("9;10;"))
			})

		It("Should refuse with the same Restrict holder as a fresh walk", func(
			ctx SpecContext,
		) {
			create(
				ctx,
				b2Task{Key: "t", Reads: []uint32{1}},
				b3Guard{Key: "g", Channel: 1},
			)
			one := []uint32{1}
			restrict := gorp.RestrictError{
				Target:    gorp.NewTarget("channel", uint32(1)),
				Holder:    gorp.Target{Table: "task", Key: "t"},
				Reference: "reads",
			}
			Expect(freshDelete(ctx, one, none)).To(MatchError(restrict))
			Expect(plannedDelete(ctx, one, one, none)).To(MatchError(restrict))
		})

		// D-173 as amended: the walk's order is fixed, so one delete names the same
		// Restrict holder each time it runs, fresh or planned.
		holderNamed := func(err error) string {
			GinkgoHelper()
			var re gorp.RestrictError
			Expect(errors.As(err, &re)).To(BeTrue(), "no RestrictError in %v", err)
			return re.Holder.Key
		}
		namedEachRun := func(ctx SpecContext, between func(gorp.Tx) error) []string {
			GinkgoHelper()
			one := []uint32{1}
			named := set.New[string]()
			for range 16 {
				named.Add(holderNamed(freshDelete(ctx, one, between)))
				named.Add(holderNamed(plannedDelete(ctx, one, one, between)))
			}
			return named.Slice()
		}

		It("Should name the same Restrict holder each run with more holders than the "+
			"index keeps in order", func(ctx SpecContext) {
			for i := range 40 {
				create(ctx, b2Task{Key: fmt.Sprintf("t%02d", i), Reads: []uint32{1}})
			}
			Expect(namedEachRun(ctx, none)).To(HaveLen(1))
		})

		It("Should name the same Restrict holder each run after a write in the "+
			"transaction", func(ctx SpecContext) {
			create(ctx,
				b2Task{Key: "r1", Reads: []uint32{1}},
				b2Task{Key: "r2", Reads: []uint32{1}},
			)
			staged := func(tx gorp.Tx) error {
				return tasks.NewCreate().
					Entry(&b2Task{Key: "r3", Reads: []uint32{2}}).
					Exec(ctx, tx)
			}
			Expect(namedEachRun(ctx, staged)).To(HaveLen(1))
		})

		It("Should delete like a fresh walk once the table of a planned Restrict "+
			"holder closes", func(ctx SpecContext) {
			create(ctx, b3Guard{Key: "g", Channel: 1})
			one := []uint32{1}
			Expect(plannedDelete(ctx, one, one, func(gorp.Tx) error {
				return guards.Close()
			})).To(Succeed())
			Expect(freshDelete(ctx, one, none)).To(Succeed())
		})
	})

	Describe("Cases that hold", func() {
		It("Should match a fresh delete for a deleted holder in Detaches and Weak",
			func(ctx SpecContext) {
				create(ctx, b4Pin{Key: "p", Watched: []uint32{1}, Plotted: []uint32{1}})
				same(ctx, func(tx gorp.Tx) error {
					return gorp.WrapWriter[string, b4Pin](tx).Delete(ctx, "p")
				})
			})

		It("Should match a fresh delete for a Restrict holder of a Cascade holder "+
			"at depth 2 deleted after the plan", func(ctx SpecContext) {
			create(
				ctx,
				b2Status{Key: "s", Channel: 1},
				b3Guard{Key: "g", Status: "s"},
				b4Lock{Key: "l", Guard: "g"},
				b4Note{Key: "n", Guards: []string{"g"}},
			)
			same(ctx, func(tx gorp.Tx) error {
				return gorp.WrapWriter[string, b4Lock](tx).Delete(ctx, "l")
			})
			same(ctx, func(tx gorp.Tx) error {
				return locks.NewDelete().
					Where(gorp.MatchKeys[string, b4Lock]("l")).
					Exec(ctx, tx)
			})
			same(ctx, func(tx gorp.Tx) error {
				if err := gorp.WrapWriter[string, b4Lock](tx).
					Delete(ctx, "l"); err != nil {
					return err
				}
				return gorp.WrapWriter[string, b4Note](tx).Delete(ctx, "n")
			})
		})

		It("Should match a fresh delete for a Restrict holder that a cascade also "+
			"reaches, deleted after the plan", func(ctx SpecContext) {
			create(
				ctx,
				b2Status{Key: "s", Channel: 1},
				b3Guard{Key: "g", Channel: 1, Status: "s"},
			)
			same(ctx, func(tx gorp.Tx) error {
				return gorp.WrapWriter[string, b3Guard](tx).Delete(ctx, "g")
			})
		})

		It("Should match a fresh delete for a Detach holder that holds a Detach "+
			"holder through a Cascade", func(ctx SpecContext) {
			create(
				ctx,
				b2Task{Key: "d", Plotted: []uint32{1}},
				b2Plot{Key: "p", Task: "d", Plotted: []uint32{1}},
			)
			same(ctx, deleteTask(ctx, "d"))
			same(ctx, wrapDeleteTask(ctx, "d"))
			same(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[string, b2Task]().
					Where(gorp.MatchKeys[string, b2Task]("d")).
					Exec(ctx, tx)
			})
			same(ctx, func(tx gorp.Tx) error {
				return tasks.NewUncheckedWriter(tx).Delete(ctx, "d")
			})
		})

		It("Should match a fresh delete for a deleted Detach holder whose hook "+
			"writes a Restrict holder", func(ctx SpecContext) {
			create(ctx, b2Task{Key: "p", Plotted: []uint32{1}})
			onDetach = func(ctx context.Context, tx gorp.Tx, _ b2Task) error {
				return guards.NewCreate().
					Entry(&b3Guard{Key: "late", Channel: 1}).
					Exec(ctx, tx)
			}
			same(ctx, wrapDeleteTask(ctx, "p"))
		})

		It("Should match a fresh delete for a deleted Inline holder", func(
			ctx SpecContext,
		) {
			create(ctx, b4Formula{Key: "f", Channels: []uint32{1}})
			same(ctx, func(tx gorp.Tx) error {
				return gorp.WrapWriter[string, b4Formula](tx).Delete(ctx, "f")
			})
		})

		It("Should match a fresh delete for a channel deleted and created again "+
			"after the plan", func(ctx SpecContext) {
			create(
				ctx,
				b2Task{Key: "p", Plotted: []uint32{1}},
				b2Status{Key: "s", Channel: 1},
				b4Pin{Key: "w", Watched: []uint32{1}},
			)
			same(ctx, func(tx gorp.Tx) error {
				if err := gorp.WrapWriter[uint32, b2Channel](tx).
					Delete(ctx, 1); err != nil {
					return err
				}
				return channels.NewCreate().Entry(&b2Channel{Key: 1}).Exec(ctx, tx)
			})
		})

		It("Should match a fresh delete for keys in another order with duplicates "+
			"and a missing key", func(ctx SpecContext) {
			create(
				ctx,
				b2Status{Key: "s1", Channel: 1},
				b2Status{Key: "s2", Channel: 2},
				b2Task{Key: "p", Plotted: []uint32{1, 2, 3}},
			)
			Expect(planned(ctx, []uint32{1, 2, 7}, []uint32{2, 1, 1, 7}, none)).
				To(Equal(fresh(ctx, []uint32{2, 1, 1, 7}, none)))
		})

		It("Should match a fresh delete for a holder touched twice after the plan",
			func(ctx SpecContext) {
				create(
					ctx,
					b2Task{Key: "t", Reads: []uint32{1}},
					b2Task{Key: "p", Plotted: []uint32{1}},
				)
				same(ctx, func(tx gorp.Tx) error {
					w := gorp.WrapWriter[string, b2Task](tx)
					if err := w.Set(
						ctx,
						b2Task{Key: "t", Reads: []uint32{1, 2}},
					); err != nil {
						return err
					}
					if err := w.Set(
						ctx,
						b2Task{Key: "p", Plotted: []uint32{1}},
					); err != nil {
						return err
					}
					return w.Delete(ctx, "t", "p")
				})
				same(ctx, func(tx gorp.Tx) error {
					w := gorp.WrapWriter[string, b2Task](tx)
					if err := w.Delete(ctx, "t"); err != nil {
						return err
					}
					return w.Set(ctx, b2Task{Key: "t"})
				})
			})

		It("Should match a fresh delete for one of its own keys deleted after the "+
			"plan", func(ctx SpecContext) {
			create(
				ctx,
				b2Status{Key: "s2", Channel: 2},
				b2Task{Key: "p", Plotted: []uint32{1, 2}},
			)
			keys := []uint32{1, 2}
			for _, between := range []func(tx gorp.Tx) error{
				func(tx gorp.Tx) error {
					return channels.NewDelete().
						Where(gorp.MatchKeys[uint32, b2Channel](2)).
						Exec(ctx, tx)
				},
				func(tx gorp.Tx) error {
					return gorp.WrapWriter[uint32, b2Channel](tx).Delete(ctx, 2)
				},
			} {
				Expect(planned(ctx, keys, keys, between)).
					To(Equal(fresh(ctx, keys, between)))
			}
		})

		It("Should match a fresh delete for a Detach holder deleted and written "+
			"again raw after the plan", func(ctx SpecContext) {
			create(ctx, b2Task{Key: "p", Plotted: []uint32{1}})
			same(ctx, func(tx gorp.Tx) error {
				if err := wrapDeleteTask(ctx, "p")(tx); err != nil {
					return err
				}
				codec := key.NewCodec[string](key.Prefix[b2Task]())
				raw, err := msgpack.Codec.Encode(b2Task{Key: "p", Plotted: []uint32{1}})
				if err != nil {
					return err
				}
				return tx.Set(ctx, bytes.Clone(codec.Encode("p")), raw)
			})
		})

		It("Should walk again for a plan from a snapshot", func(ctx SpecContext) {
			create(ctx, b2Task{Key: "t", Reads: []uint32{1}})
			var effects gorp.DeleteEffects
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
				var err error
				effects, err = channels.PlanDelete(ctx, snap, []uint32{1})
				return err
			})).To(Succeed())
			Expect(effects.Restricts).To(HaveLen(1))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := wrapDeleteTask(ctx, "t")(tx); err != nil {
					return err
				}
				if err := guards.NewCreate().
					Entry(&b3Guard{Key: "late", Channel: 1}).
					Exec(ctx, tx); err != nil {
					return err
				}
				return channels.NewDelete().
					Where(gorp.MatchKeys[uint32, b2Channel](1)).
					Planned(effects).
					Exec(ctx, tx)
			})).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("channel", uint32(1)),
				Holder:    gorp.Target{Table: "guard", Key: "late"},
				Reference: "channel",
			}))
		})

		It("Should match a fresh delete for a Restrict holder that a guard of the "+
			"delete removes", func(ctx SpecContext) {
			create(ctx, b2Task{Key: "t", Reads: []uint32{1}})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				effects := MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1}))
				return channels.NewDelete().
					Where(gorp.MatchKeys[uint32, b2Channel](1)).
					Guard(func(gctx gorp.Context, _ b2Channel) error {
						return wrapDeleteTask(ctx, "t")(tx)
					}).
					Planned(effects).
					Exec(ctx, tx)
			})).To(Succeed())
		})
	})
})

// openPlannedDeleteBench opens a channel table with 1 channel that cascades statuses
// statuses hold, and n further channels with no holders.
func openPlannedDeleteBench(
	b *testing.B,
	statusCount, channelCount int,
) (*gorp.DB, *gorp.Table[uint32, b2Channel]) {
	b.Helper()
	db := gorp.Wrap(
		memkv.New(),
		gorp.WithCodec(msgpack.Codec),
		gorp.WithMaxDeleteSize(1<<30),
	)
	ctx := b.Context()
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- db.Run(runCtx) }()
	channels, err := gorp.OpenTable(ctx, gorp.TableConfig[uint32, b2Channel]{
		DB:     db,
		Name:   "channel",
		Issued: func(uint32) bool { return true },
	})
	if err != nil {
		b.Fatal(err)
	}
	statuses := openB4Statuses(b, db)
	b.Cleanup(func() {
		cancel()
		if err := errors.Join(
			statuses.Close(),
			channels.Close(),
			<-done,
			db.Close(),
		); err != nil {
			b.Error(err)
		}
	})
	cs := make([]b2Channel, channelCount+1)
	for i := range cs {
		cs[i] = b2Channel{Key: uint32(i + 1)}
	}
	if err = channels.NewCreate().Entries(&cs).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	ss := make([]b2Status, statusCount)
	for i := range ss {
		ss[i] = b2Status{Key: fmt.Sprintf("s%07d", i), Channel: 1}
	}
	if err = statuses.NewCreate().Entries(&ss).Exec(ctx, db); err != nil {
		b.Fatal(err)
	}
	return db, channels
}

// openB4Statuses opens a status table whose statuses cascade from their channel.
func openB4Statuses(b *testing.B, db *gorp.DB) *gorp.Table[string, b2Status] {
	b.Helper()
	statuses, err := gorp.OpenTable(b.Context(), gorp.TableConfig[string, b2Status]{
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
	})
	if err != nil {
		b.Fatal(err)
	}
	return statuses
}

// b4Mode is what one run of a planned delete benchmark does in its transaction.
type b4Mode string

const (
	// b4Writes runs only the writes between the plan and the delete.
	b4Writes b4Mode = "writes"
	// b4Fresh runs the writes, then a delete that walks the references itself.
	b4Fresh b4Mode = "fresh"
	// b4Plan runs PlanDelete, then the writes.
	b4Plan b4Mode = "plan"
	// b4Planned runs PlanDelete, then the writes, then a delete with the plan.
	b4Planned b4Mode = "planned"
)

// runPlannedDeleteBench runs mode for the delete of keys, with writes between the plan
// and the delete. Each run uses a transaction that it then discards.
func runPlannedDeleteBench(
	b *testing.B,
	db *gorp.DB,
	channels *gorp.Table[uint32, b2Channel],
	keys []uint32,
	writes func(tx gorp.Tx) error,
	mode b4Mode,
) {
	b.Helper()
	ctx := b.Context()
	b.ReportAllocs()
	for b.Loop() {
		tx := db.OpenTx()
		del := channels.NewDelete().Where(gorp.MatchKeys[uint32, b2Channel](keys...))
		var err error
		if mode == b4Plan || mode == b4Planned {
			var effects gorp.DeleteEffects
			if effects, err = channels.PlanDelete(ctx, tx, keys); err != nil {
				b.Fatal(err)
			}
			del = del.Planned(effects)
		}
		err = writes(tx)
		if err == nil && (mode == b4Fresh || mode == b4Planned) {
			err = del.Exec(ctx, tx)
		}
		if err = errors.Join(err, tx.Close()); err != nil {
			b.Fatal(err)
		}
	}
}

var b4Modes = []b4Mode{b4Writes, b4Fresh, b4Plan, b4Planned}

// BenchmarkPlannedDeleteCascades measures the delete of 1 channel with c Cascade
// holders after d deletes of records that the plan does not name. The cost of the
// planned delete itself is planned - plan, and of the fresh delete fresh - writes.
func BenchmarkPlannedDeleteCascades(b *testing.B) {
	for _, c := range []int{100, 1000, 4000} {
		db, channels := openPlannedDeleteBench(b, c, 0)
		for _, d := range []int{0, 1000, 4000} {
			junk := make([]string, d)
			for i := range junk {
				junk[i] = fmt.Sprintf("junk%07d", i)
			}
			writes := func(tx gorp.Tx) error {
				return gorp.WrapWriter[string, b4Note](tx).Delete(b.Context(), junk...)
			}
			for _, mode := range b4Modes {
				b.Run(fmt.Sprintf("c=%d/d=%d/%s", c, d, mode), func(b *testing.B) {
					runPlannedDeleteBench(b, db, channels, []uint32{1}, writes, mode)
				})
			}
		}
	}
}

// BenchmarkPlannedDeleteKeys measures the delete of n channels with no holders. The
// cost of the planned delete itself is planned - plan, and of the fresh delete fresh -
// writes.
func BenchmarkPlannedDeleteKeys(b *testing.B) {
	for _, n := range []int{100, 1000, 10000, 30000} {
		db, channels := openPlannedDeleteBench(b, 0, n)
		keys := make([]uint32, n)
		for i := range keys {
			keys[i] = uint32(i + 2)
		}
		writes := func(gorp.Tx) error { return nil }
		for _, mode := range b4Modes {
			b.Run(fmt.Sprintf("n=%d/%s", n, mode), func(b *testing.B) {
				runPlannedDeleteBench(b, db, channels, keys, writes, mode)
			})
		}
	}
}
