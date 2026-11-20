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
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type pdChannel struct{ Key uint32 }

func (c pdChannel) GorpKey() uint32 { return c.Key }

// pdCalc is deleted with its output channel.
type pdCalc struct {
	Key    string
	Output uint32
}

func (c pdCalc) GorpKey() string { return c.Key }

// pdWidget is deleted with its calc, and holds a channel through a Restrict reference.
type pdWidget struct {
	Key     string
	Calc    string
	Channel uint32
}

func (w pdWidget) GorpKey() string { return w.Key }

// pdHolder holds channels through every other OnDelete.
type pdHolder struct {
	Key     string
	Plotted []uint32
	Tasks   []uint32
	Symbol  uint32
}

func (h pdHolder) GorpKey() string { return h.Key }

const pdTaskTag uint8 = 3

func pdChannelRef[E gorp.Entry[string]](
	name string,
	onDelete gorp.OnDelete,
	targets func(e *E) []uint32,
	tag uint8,
) gorp.Reference[string, E] {
	cfg := gorp.KeyReferenceConfig[string, E, uint32]{
		Name:     name,
		Table:    "channel",
		OnDelete: onDelete,
		Kind:     gorp.Use,
		Tag:      tag,
		Targets: func(e *E) []uint32 {
			return slices.DeleteFunc(targets(e), func(k uint32) bool { return k == 0 })
		},
	}
	if onDelete == gorp.Detach {
		cfg.Detach = func(*E, uint32) {}
	}
	return gorp.NewKeyReference(cfg)
}

var _ = Describe("PlanDelete", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, pdChannel]
		calcs    *gorp.Table[string, pdCalc]
		widgets  *gorp.Table[string, pdWidget]
		holders  *gorp.Table[string, pdHolder]
		open     = func(ctx SpecContext, opts ...gorp.Option) {
			GinkgoHelper()
			db = DeferClose(gorp.Wrap(
				memkv.New(),
				append(opts, gorp.WithCodec(msgpack.Codec))...,
			))
			StartRun(db.Run)
			channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, pdChannel]{
				DB:     db,
				Name:   "channel",
				Issued: func(uint32) bool { return true },
			}))
			calcs = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pdCalc]{
				DB:   db,
				Name: "calc",
				References: []gorp.Reference[string, pdCalc]{
					pdChannelRef("output", gorp.Cascade, func(c *pdCalc) []uint32 {
						return []uint32{c.Output}
					}, 0),
				},
			}))
			widgets = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pdWidget]{
				DB:   db,
				Name: "widget",
				References: []gorp.Reference[string, pdWidget]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, pdWidget, string]{
							Name:     "calc",
							Table:    "calc",
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets:  func(w *pdWidget) []string { return []string{w.Calc} },
						},
					),
					pdChannelRef("channel", gorp.Restrict, func(w *pdWidget) []uint32 {
						return []uint32{w.Channel}
					}, 0),
				},
			}))
			holders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pdHolder]{
				DB:   db,
				Name: "holder",
				References: []gorp.Reference[string, pdHolder]{
					pdChannelRef("plotted", gorp.Detach, func(h *pdHolder) []uint32 {
						return slices.Clone(h.Plotted)
					}, 0),
					pdChannelRef("tasks", gorp.Restrict, func(h *pdHolder) []uint32 {
						return slices.Clone(h.Tasks)
					}, pdTaskTag),
					pdChannelRef("symbol", gorp.Weak, func(h *pdHolder) []uint32 {
						return []uint32{h.Symbol}
					}, 0),
				},
			}))
			for k := uint32(1); k <= 3; k++ {
				Expect(channels.NewCreate().Entry(&pdChannel{Key: k}).Exec(ctx, db)).
					To(Succeed())
			}
		}
		create = func(ctx SpecContext, tx gorp.Tx, entries ...any) {
			GinkgoHelper()
			for _, e := range entries {
				switch e := e.(type) {
				case pdCalc:
					Expect(calcs.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				case pdWidget:
					Expect(widgets.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				case pdHolder:
					Expect(holders.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				default:
					Fail(fmt.Sprintf("no table holds %T", e))
				}
			}
		}
		effect = func(
			table, key, reference string,
			kind gorp.LinkKind,
			target gorp.Target,
		) gorp.DeleteEffect {
			return gorp.DeleteEffect{
				Holder:    gorp.Target{Table: table, Key: key},
				Reference: reference,
				Kind:      kind,
				Target:    target,
			}
		}
		channel = func(k uint32) gorp.Target { return gorp.NewTarget("channel", k) }
	)

	It("Should list each effect of a delete and write nothing", func(ctx SpecContext) {
		open(ctx)
		create(
			ctx,
			db,
			pdCalc{Key: "c", Output: 1},
			pdWidget{Key: "w", Calc: "c", Channel: 2},
			pdHolder{Key: "h", Plotted: []uint32{1, 2}, Tasks: []uint32{1}, Symbol: 1},
		)
		effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1}))
		Expect(effects.Cascades).To(Equal([]gorp.DeleteEffect{
			effect("calc", "c", "output", gorp.Use, channel(1)),
			effect("widget", "w", "calc", gorp.Place, gorp.NewTarget("calc", "c")),
		}))
		Expect(effects.Detaches).To(Equal([]gorp.DeleteEffect{
			effect("holder", "h", "plotted", gorp.Use, channel(1)),
		}))
		task := effect("holder", "h", "tasks", gorp.Use, channel(1))
		task.Tag = pdTaskTag
		Expect(effects.Restricts).To(Equal([]gorp.DeleteEffect{task}))
		Expect(effects.Weak).To(Equal([]gorp.DeleteEffect{
			effect("holder", "h", "symbol", gorp.Use, channel(1)),
		}))
		Expect(calcs.NewRetrieve().
			Where(gorp.MatchKeys[string, pdCalc]("c")).
			Exists(ctx, db)).To(BeTrue())
		Expect(channels.NewRetrieve().
			Where(gorp.MatchKeys[uint32, pdChannel](1)).
			Exists(ctx, db)).To(BeTrue())
	})

	It("Should leave out each holder that a cascade deletes", func(ctx SpecContext) {
		open(ctx)
		create(
			ctx,
			db,
			pdCalc{Key: "c", Output: 1},
			pdWidget{Key: "w", Calc: "c", Channel: 1},
		)
		effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1}))
		Expect(effects.Cascades).To(HaveLen(2))
		Expect(effects.Restricts).To(BeEmpty())
	})

	It("Should list a holder once for each target that it loses", func(
		ctx SpecContext,
	) {
		open(ctx)
		create(ctx, db, pdHolder{Key: "h", Plotted: []uint32{1, 2}})
		effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{2, 1}))
		Expect(effects.Detaches).To(Equal([]gorp.DeleteEffect{
			effect("holder", "h", "plotted", gorp.Use, channel(1)),
			effect("holder", "h", "plotted", gorp.Use, channel(2)),
		}))
	})

	It("Should list a holder that restricts two targets once", func(ctx SpecContext) {
		open(ctx)
		create(ctx, db, pdHolder{Key: "h", Tasks: []uint32{1, 2}})
		Expect(MustSucceed(channels.PlanDelete(ctx, db, []uint32{1, 2})).Restricts).
			To(HaveLen(1))
	})

	It("Should see the changes staged in a transaction", func(ctx SpecContext) {
		open(ctx)
		tx := DeferClose(db.OpenTx())
		create(ctx, tx, pdHolder{Key: "h", Tasks: []uint32{3}})
		Expect(MustSucceed(channels.PlanDelete(ctx, tx, []uint32{3})).Restricts).
			To(HaveLen(1))
		Expect(MustSucceed(channels.PlanDelete(ctx, db, []uint32{3})).Restricts).
			To(BeEmpty())
	})

	It("Should match what the delete does once the restrictions go", func(
		ctx SpecContext,
	) {
		open(ctx)
		create(
			ctx,
			db,
			pdCalc{Key: "c", Output: 1},
			pdWidget{Key: "w", Calc: "c", Channel: 2},
			pdHolder{Key: "h", Plotted: []uint32{1}, Symbol: 1},
		)
		effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1}))
		Expect(effects.Restricts).To(BeEmpty())
		Expect(channels.NewDelete().
			Where(gorp.MatchKeys[uint32, pdChannel](1)).
			Exec(ctx, db)).To(Succeed())
		for _, e := range effects.Cascades {
			Expect(e.Holder.Table).To(BeElementOf("calc", "widget"))
		}
		Expect(calcs.NewRetrieve().
			Where(gorp.MatchKeys[string, pdCalc]("c")).
			Exists(ctx, db)).To(BeFalse())
		Expect(widgets.NewRetrieve().
			Where(gorp.MatchKeys[string, pdWidget]("w")).
			Exists(ctx, db)).To(BeFalse())
		Expect(holders.NewRetrieve().
			Where(gorp.MatchKeys[string, pdHolder]("h")).
			Exists(ctx, db)).To(BeTrue())
	})

	It("Should fail as the delete does when the delete is too large", func(
		ctx SpecContext,
	) {
		open(ctx, gorp.WithMaxDeleteSize(2))
		create(
			ctx,
			db,
			pdCalc{Key: "c", Output: 1},
			pdWidget{Key: "w", Calc: "c", Channel: 2},
		)
		Expect(channels.PlanDelete(ctx, db, []uint32{1})).Error().
			To(MatchError(ContainSubstring(
				"deleting channel 1 would change more than 2 entries",
			)))
	})

	It(
		"Should fail with a validation error when more holders refuse the delete than "+
			"the limit",
		func(ctx SpecContext) {
			open(ctx, gorp.WithMaxDeleteSize(2))
			create(
				ctx,
				db,
				pdHolder{Key: "a", Tasks: []uint32{1}},
				pdHolder{Key: "b", Tasks: []uint32{1}},
				pdHolder{Key: "c", Tasks: []uint32{1}},
			)
			Expect(channels.PlanDelete(ctx, db, []uint32{1})).Error().
				To(MatchError(ContainSubstring(
					"more than 2 entries refuse deleting channel 1",
				)))
		},
	)
})

var _ = Describe("Planned delete", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, pdChannel]
		calcs    *gorp.Table[string, pdCalc]
		holders  *gorp.Table[string, pdHolder]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, pdChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(uint32) bool { return true },
		}))
		calcs = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pdCalc]{
			DB:   db,
			Name: "calc",
			References: []gorp.Reference[string, pdCalc]{
				pdChannelRef("output", gorp.Cascade, func(c *pdCalc) []uint32 {
					return []uint32{c.Output}
				}, 0),
			},
		}))
		holders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pdHolder]{
			DB:   db,
			Name: "holder",
			References: []gorp.Reference[string, pdHolder]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, pdHolder, uint32]{
					Name:     "plotted",
					Table:    "channel",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(h *pdHolder) []uint32 { return h.Plotted },
					Detach: func(h *pdHolder, t uint32) {
						h.Plotted = slices.DeleteFunc(h.Plotted, func(k uint32) bool {
							return k == t
						})
					},
				}),
				pdChannelRef("tasks", gorp.Restrict, func(h *pdHolder) []uint32 {
					return slices.Clone(h.Tasks)
				}, 0),
			},
		}))
		for k := uint32(1); k <= 2; k++ {
			Expect(channels.NewCreate().Entry(&pdChannel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
		Expect(calcs.NewCreate().Entry(&pdCalc{Key: "c", Output: 1}).Exec(ctx, db)).
			To(Succeed())
		Expect(holders.NewCreate().
			Entry(&pdHolder{Key: "p", Plotted: []uint32{1, 2}}).
			Exec(ctx, db)).To(Succeed())
	})
	deletePlanned := func(ctx SpecContext, tx gorp.Tx, effects gorp.DeleteEffects) error {
		return channels.NewDelete().
			Where(gorp.MatchKeys[uint32, pdChannel](1)).
			Planned(effects).
			Exec(ctx, tx)
	}
	retrieveHolder := func(ctx SpecContext, k string) pdHolder {
		GinkgoHelper()
		var h pdHolder
		Expect(holders.NewRetrieve().
			Where(gorp.MatchKeys[string, pdHolder](k)).
			Entry(&h).
			Exec(ctx, db)).To(Succeed())
		return h
	}

	It("Should apply the planned cascades and detaches", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			effects, err := channels.PlanDelete(ctx, tx, []uint32{1})
			if err != nil {
				return err
			}
			return deletePlanned(ctx, tx, effects)
		})).To(Succeed())
		Expect(channels.NewRetrieve().
			Where(gorp.MatchKeys[uint32, pdChannel](1)).
			Exists(ctx, db)).To(BeFalse())
		Expect(calcs.NewRetrieve().
			Where(gorp.MatchKeys[string, pdCalc]("c")).
			Exists(ctx, db)).To(BeFalse())
		Expect(retrieveHolder(ctx, "p").Plotted).To(Equal([]uint32{2}))
	})

	It("Should apply only the plan, with no walk of its own", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return deletePlanned(ctx, tx, gorp.DeleteEffects{})
		})).To(Succeed())
		Expect(calcs.NewRetrieve().
			Where(gorp.MatchKeys[string, pdCalc]("c")).
			Exists(ctx, db)).To(BeTrue())
		Expect(retrieveHolder(ctx, "p").Plotted).To(Equal([]uint32{1, 2}))
	})

	It("Should refuse while a planned Restrict holder still exists", func(
		ctx SpecContext,
	) {
		Expect(holders.NewCreate().
			Entry(&pdHolder{Key: "t", Tasks: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			effects, err := channels.PlanDelete(ctx, tx, []uint32{1})
			if err != nil {
				return err
			}
			return deletePlanned(ctx, tx, effects)
		})).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("channel", uint32(1)),
			Holder:    gorp.Target{Table: "holder", Key: "t"},
			Reference: "tasks",
		}))
		Expect(channels.NewRetrieve().
			Where(gorp.MatchKeys[uint32, pdChannel](1)).
			Exists(ctx, db)).To(BeTrue())
	})

	It("Should delete once the caller removes each planned Restrict holder", func(
		ctx SpecContext,
	) {
		Expect(holders.NewCreate().
			Entry(&pdHolder{Key: "t", Tasks: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			effects, err := channels.PlanDelete(ctx, tx, []uint32{1})
			if err != nil {
				return err
			}
			if err = holders.NewDelete().
				Where(gorp.MatchKeys[string, pdHolder]("t")).
				Exec(ctx, tx); err != nil {
				return err
			}
			return deletePlanned(ctx, tx, effects)
		})).To(Succeed())
		Expect(channels.NewRetrieve().
			Where(gorp.MatchKeys[uint32, pdChannel](1)).
			Exists(ctx, db)).To(BeFalse())
	})
})
