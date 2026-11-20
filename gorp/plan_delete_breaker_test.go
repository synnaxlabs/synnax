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
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/hookkv"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// bkGroup is deleted with any of its calcs.
type bkGroup struct {
	Key   string
	Calcs []string
}

func (g bkGroup) GorpKey() string { return g.Key }

// bkPanel is deleted with its widget.
type bkPanel struct {
	Key    string
	Widget string
}

func (p bkPanel) GorpKey() string { return p.Key }

// bkMixed detaches channels and is deleted with its calc.
type bkMixed struct {
	Key     string
	Plotted []uint32
	Calc    string
}

func (m bkMixed) GorpKey() string { return m.Key }

// bkNode is deleted with its next node and restricts the delete of its blocker.
type bkNode struct {
	Key     string
	Next    string
	Blocker string
}

func (n bkNode) GorpKey() string { return n.Key }

// bkInliner inlines its channel.
type bkInliner struct {
	Key     string
	Channel uint32
	Copied  uint32
}

func (i bkInliner) GorpKey() string { return i.Key }

// bkLoose is the entry of a table outside the registry.
type bkLoose struct{ Key uint32 }

func (l bkLoose) GorpKey() uint32 { return l.Key }

func bkNonEmpty(keys ...string) []string {
	return slices.DeleteFunc(keys, func(k string) bool { return k == "" })
}

var _ = Describe("PlanDelete breaker", func() {
	var (
		db       *gorp.DB
		hooked   *hookkv.DB
		channels *gorp.Table[uint32, pdChannel]
		calcs    *gorp.Table[string, pdCalc]
		widgets  *gorp.Table[string, pdWidget]
		holders  *gorp.Table[string, pdHolder]
		groups   *gorp.Table[string, bkGroup]
		panels   *gorp.Table[string, bkPanel]
		mixeds   *gorp.Table[string, bkMixed]
		nodes    *gorp.Table[string, bkNode]
		inliners *gorp.Table[string, bkInliner]
		open     = func(ctx SpecContext, opts ...gorp.Option) {
			GinkgoHelper()
			hooked = hookkv.Wrap(memkv.New())
			db = DeferClose(gorp.Wrap(
				hooked,
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
							Targets: func(w *pdWidget) []string {
								return bkNonEmpty(w.Calc)
							},
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
			groups = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, bkGroup]{
				DB:   db,
				Name: "group",
				References: []gorp.Reference[string, bkGroup]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, bkGroup, string]{
							Name:     "calcs",
							Table:    "calc",
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets: func(g *bkGroup) []string {
								return bkNonEmpty(slices.Clone(g.Calcs)...)
							},
						},
					),
				},
			}))
			panels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, bkPanel]{
				DB:   db,
				Name: "panel",
				References: []gorp.Reference[string, bkPanel]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, bkPanel, string]{
							Name:     "widget",
							Table:    "widget",
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets:  func(p *bkPanel) []string { return bkNonEmpty(p.Widget) },
						},
					),
				},
			}))
			mixeds = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, bkMixed]{
				DB:   db,
				Name: "mixed",
				References: []gorp.Reference[string, bkMixed]{
					pdChannelRef("plotted", gorp.Detach, func(m *bkMixed) []uint32 {
						return slices.Clone(m.Plotted)
					}, 0),
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, bkMixed, string]{
							Name:     "calc",
							Table:    "calc",
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets:  func(m *bkMixed) []string { return bkNonEmpty(m.Calc) },
						},
					),
				},
			}))
			nodes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, bkNode]{
				DB:   db,
				Name: "node",
				References: []gorp.Reference[string, bkNode]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, bkNode, string]{
							Name:     "next",
							Table:    "node",
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets:  func(n *bkNode) []string { return bkNonEmpty(n.Next) },
						},
					),
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, bkNode, string]{
							Name:     "blocker",
							Table:    "node",
							OnDelete: gorp.Restrict,
							Kind:     gorp.Use,
							Targets:  func(n *bkNode) []string { return bkNonEmpty(n.Blocker) },
						},
					),
				},
			}))
			inliners = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, bkInliner]{
				DB:   db,
				Name: "inliner",
				References: []gorp.Reference[string, bkInliner]{
					gorp.NewKeyReference(
						gorp.KeyReferenceConfig[string, bkInliner, uint32]{
							Name:     "channel",
							Table:    "channel",
							OnDelete: gorp.Inline,
							Kind:     gorp.Use,
							Targets: func(i *bkInliner) []uint32 {
								if i.Channel == 0 {
									return nil
								}
								return []uint32{i.Channel}
							},
							Inline: func(
								ctx context.Context,
								tx gorp.Tx,
								i *bkInliner,
								target uint32,
							) error {
								var c pdChannel
								if err := channels.NewRetrieve().
									Where(gorp.MatchKeys[uint32, pdChannel](target)).
									Entry(&c).
									Exec(ctx, tx); err != nil {
									return err
								}
								i.Copied, i.Channel = c.Key, 0
								return nil
							},
						},
					),
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
				case bkGroup:
					Expect(groups.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				case bkPanel:
					Expect(panels.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				case bkMixed:
					Expect(mixeds.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				case bkNode:
					Expect(nodes.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				case bkInliner:
					Expect(inliners.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				default:
					Fail("unknown fixture")
				}
			}
		}
		holdersOf = func(effects []gorp.DeleteEffect) []gorp.Target {
			out := make([]gorp.Target, len(effects))
			for i, e := range effects {
				out[i] = e.Holder
			}
			return out
		}
		causeOf = func(effects []gorp.DeleteEffect, holder gorp.Target) gorp.Target {
			GinkgoHelper()
			for _, e := range effects {
				if e.Holder == holder {
					return e.Target
				}
			}
			Fail("no effect on " + holder.Table + " " + holder.Key)
			return gorp.Target{}
		}
		deleteChannels = func(ctx SpecContext, tx gorp.Tx, keys ...uint32) error {
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, pdChannel](keys...)).
				Exec(ctx, tx)
		}
		channel = func(k uint32) gorp.Target { return gorp.NewTarget("channel", k) }
		target  = func(table, k string) gorp.Target { return gorp.NewTarget(table, k) }
	)

	Describe("Findings", func() {
		It("Should not fail the size limit for requested keys that do not exist", func(
			ctx SpecContext,
		) {
			open(ctx, gorp.WithMaxDeleteSize(2))
			keys := []uint32{1, 2, 98, 99}
			tx := db.OpenTx()
			Expect(deleteChannels(ctx, tx, keys...)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(channels.PlanDelete(ctx, db, keys)).Error().ToNot(HaveOccurred())
		})

		It("Should list no Weak effect for a key that the delete does not remove", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(ctx, db, pdHolder{Key: "h", Symbol: 3})
			Expect(deleteChannels(ctx, db, 3)).To(Succeed())
			Expect(channels.NewRetrieve().
				Where(gorp.MatchKeys[uint32, pdChannel](3)).
				Exists(ctx, db)).To(BeFalse())
			Expect(MustSucceed(channels.PlanDelete(ctx, db, []uint32{3})).Weak).
				To(BeEmpty())
		})

		It("Should list no Restrict for a key that the delete does not remove", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(ctx, db, pdHolder{Key: "h", Tasks: []uint32{3}})
			tx := db.OpenTx()
			Expect(channels.NewUncheckedWriter(tx).Delete(ctx, 3)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(deleteChannels(ctx, db, 3)).To(Succeed())
			Expect(MustSucceed(channels.PlanDelete(ctx, db, []uint32{3})).Restricts).
				To(BeEmpty())
		})

		It("Should name the delete in the size error as the delete does", func(
			ctx SpecContext,
		) {
			open(ctx, gorp.WithMaxDeleteSize(1))
			create(ctx, db, pdCalc{Key: "c", Output: 1})
			deleteErr := deleteChannels(ctx, db, 1, 1)
			Expect(deleteErr).To(HaveOccurred())
			_, planErr := channels.PlanDelete(ctx, db, []uint32{1, 1})
			Expect(planErr).To(MatchError(deleteErr.Error()))
		})

		It("Should give the same first cause to each plan of one state", func(
			ctx SpecContext,
		) {
			open(ctx)
			tx := DeferClose(db.OpenTx())
			create(
				ctx,
				tx,
				pdCalc{Key: "a", Output: 1},
				pdCalc{Key: "b", Output: 1},
				bkGroup{Key: "g", Calcs: []string{"a", "b"}},
			)
			causes := set.New[gorp.Target]()
			for range 64 {
				effects := MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1}))
				causes.Add(causeOf(effects.Cascades, target("group", "g")))
			}
			Expect(causes.Slice()).To(HaveLen(1))
		})

		It("Should give the same first cause to each plan of committed state", func(
			ctx SpecContext,
		) {
			open(ctx)
			keys := make([]string, 40)
			for i := range keys {
				keys[i] = "c" + string(rune('A'+i))
				create(ctx, db, pdCalc{Key: keys[i], Output: 1})
			}
			create(ctx, db, bkGroup{Key: "g", Calcs: keys})
			causes := set.New[gorp.Target]()
			for range 64 {
				effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1}))
				causes.Add(causeOf(effects.Cascades, target("group", "g")))
			}
			Expect(causes.Slice()).To(HaveLen(1))
		})

		It("Should not list a holder from a commit that fails", func(ctx SpecContext) {
			open(ctx)
			held, release := make(chan struct{}), make(chan struct{})
			hook := func(func() error) error {
				close(held)
				<-release
				return errors.New("disk full")
			}
			committed := make(chan error, 1)
			go func() {
				defer GinkgoRecover()
				tx := db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				Expect(calcs.NewCreate().
					Entry(&pdCalc{Key: "phantom", Output: 1}).
					Exec(ctx, tx)).To(Succeed())
				committed <- tx.Commit(hookkv.WithHook(ctx, hook))
			}()
			Eventually(held).Should(BeClosed())
			effects, err := channels.PlanDelete(ctx, db, []uint32{1})
			close(release)
			Expect(<-committed).To(MatchError(ContainSubstring("disk full")))
			Expect(err).ToNot(HaveOccurred())
			Expect(calcs.NewRetrieve().
				Where(gorp.MatchKeys[string, pdCalc]("phantom")).
				Exists(ctx, db)).To(BeFalse())
			Expect(effects.Cascades).To(BeEmpty())
		})
	})

	Describe("Cases that hold", func() {
		It("Should match the delete through three levels and a diamond", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(
				ctx,
				db,
				pdCalc{Key: "a", Output: 1},
				pdCalc{Key: "b", Output: 1},
				pdWidget{Key: "w", Calc: "a", Channel: 2},
				bkPanel{Key: "p", Widget: "w"},
				bkGroup{Key: "g", Calcs: []string{"a", "b"}},
				bkGroup{Key: "keep"},
			)
			effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1}))
			Expect(holdersOf(effects.Cascades)).To(Equal([]gorp.Target{
				target("calc", "a"),
				target("calc", "b"),
				target("group", "g"),
				target("panel", "p"),
				target("widget", "w"),
			}))
			Expect(effects.Restricts).To(BeEmpty())
			Expect(deleteChannels(ctx, db, 1)).To(Succeed())
			Expect(calcs.NewRetrieve().Exists(ctx, db)).To(BeFalse())
			Expect(widgets.NewRetrieve().Exists(ctx, db)).To(BeFalse())
			Expect(panels.NewRetrieve().Exists(ctx, db)).To(BeFalse())
			Expect(groups.NewRetrieve().
				Where(gorp.MatchKeys[string, bkGroup]("g")).
				Exists(ctx, db)).To(BeFalse())
			Expect(groups.NewRetrieve().
				Where(gorp.MatchKeys[string, bkGroup]("keep")).
				Exists(ctx, db)).To(BeTrue())
		})

		It("Should follow a cycle of cascades once", func(ctx SpecContext) {
			open(ctx)
			create(ctx, db, bkNode{Key: "b"}, bkNode{Key: "a", Next: "b"})
			create(ctx, db, bkNode{Key: "b", Next: "a"})
			effects := MustSucceed(nodes.PlanDelete(ctx, db, []string{"a"}))
			Expect(holdersOf(effects.Cascades)).To(Equal([]gorp.Target{
				target("node", "b"),
			}))
			Expect(nodes.NewDelete().
				Where(gorp.MatchKeys[string, bkNode]("a")).
				Exec(ctx, db)).To(Succeed())
			Expect(nodes.NewRetrieve().Exists(ctx, db)).To(BeFalse())
		})

		It("Should list a holder that is detached and cascaded only as a cascade", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(
				ctx,
				db,
				pdCalc{Key: "c", Output: 2},
				bkMixed{Key: "m", Plotted: []uint32{1}, Calc: "c"},
				bkMixed{Key: "n", Plotted: []uint32{1, 2}},
			)
			effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1, 2}))
			Expect(holdersOf(effects.Cascades)).To(Equal([]gorp.Target{
				target("calc", "c"),
				target("mixed", "m"),
			}))
			Expect(effects.Detaches).To(HaveLen(2))
			Expect(holdersOf(effects.Detaches)).To(HaveEach(target("mixed", "n")))
		})

		It("Should list a widget restricted through one target only while it is not "+
			"cascaded", func(ctx SpecContext) {
			open(ctx)
			create(
				ctx,
				db,
				pdCalc{Key: "c", Output: 1},
				pdWidget{Key: "w", Calc: "c", Channel: 2},
			)
			Expect(MustSucceed(channels.PlanDelete(ctx, db, []uint32{2, 1})).Restricts).
				To(BeEmpty())
			Expect(deleteChannels(ctx, db, 2, 1)).To(Succeed())
		})

		It("Should list a holder that detaches one target and restricts another in "+
			"both lists", func(ctx SpecContext) {
			open(ctx)
			create(
				ctx,
				db,
				pdHolder{Key: "h", Plotted: []uint32{1}, Tasks: []uint32{2}},
			)
			effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1, 2}))
			Expect(holdersOf(effects.Detaches)).To(Equal([]gorp.Target{
				target("holder", "h"),
			}))
			Expect(effects.Restricts).To(HaveLen(1))
			Expect(effects.Restricts[0].Target).To(Equal(channel(2)))
			Expect(deleteChannels(ctx, db, 1, 2)).To(MatchError(ContainSubstring(
				"cannot delete channel 2 because holder h references it",
			)))
		})

		It("Should list an Inline holder as a detach unless a cascade deletes it", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(
				ctx,
				db,
				bkInliner{Key: "i", Channel: 1},
				bkInliner{Key: "j", Channel: 2},
			)
			effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1, 2}))
			Expect(holdersOf(effects.Detaches)).To(Equal([]gorp.Target{
				target("inliner", "i"),
				target("inliner", "j"),
			}))
			Expect(deleteChannels(ctx, db, 1, 2)).To(Succeed())
			var i bkInliner
			Expect(inliners.NewRetrieve().
				Where(gorp.MatchKeys[string, bkInliner]("i")).
				Entry(&i).
				Exec(ctx, db)).To(Succeed())
			Expect(i.Copied).To(Equal(uint32(1)))
		})

		It("Should plan nothing for keys that do not exist and hold nothing", func(
			ctx SpecContext,
		) {
			open(ctx)
			effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{77}))
			Expect(effects.Cascades).To(BeEmpty())
			Expect(effects.Weak).To(BeEmpty())
			effects = MustSucceed(channels.PlanDelete(ctx, db, nil))
			Expect(effects.Cascades).To(BeEmpty())
			Expect(effects.Detaches).To(BeEmpty())
			Expect(effects.Restricts).To(BeEmpty())
			Expect(effects.Weak).To(BeEmpty())
		})

		It("Should list each effect once for duplicate requested keys", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(
				ctx,
				db,
				pdCalc{Key: "c", Output: 1},
				pdHolder{Key: "h", Plotted: []uint32{1}, Tasks: []uint32{1}, Symbol: 1},
			)
			effects := MustSucceed(channels.PlanDelete(ctx, db, []uint32{1, 1, 1}))
			Expect(effects.Cascades).To(HaveLen(1))
			Expect(effects.Detaches).To(HaveLen(1))
			Expect(effects.Restricts).To(HaveLen(1))
			Expect(effects.Weak).To(HaveLen(1))
		})

		It("Should not list a requested key that holds another requested key", func(
			ctx SpecContext,
		) {
			open(ctx)
			create(ctx, db, bkNode{Key: "r"}, bkNode{Key: "s", Blocker: "r"})
			effects := MustSucceed(nodes.PlanDelete(ctx, db, []string{"r", "s"}))
			Expect(effects.Restricts).To(BeEmpty())
			Expect(nodes.NewDelete().
				Where(gorp.MatchKeys[string, bkNode]("r", "s")).
				Exec(ctx, db)).To(Succeed())
			Expect(MustSucceed(nodes.PlanDelete(ctx, db, []string{"r"})).Restricts).
				To(BeEmpty())
		})

		DescribeTable("Should list restricted holders up to the limit",
			func(ctx SpecContext, n int, tooLarge bool) {
				open(ctx, gorp.WithMaxDeleteSize(3))
				for i := range n {
					create(ctx, db, pdHolder{
						Key:   string(rune('a' + i)),
						Tasks: []uint32{1},
					})
				}
				effects, err := channels.PlanDelete(ctx, db, []uint32{1})
				if tooLarge {
					Expect(err).To(MatchError(ContainSubstring(
						"deleting channel 1 would change more than 3 entries",
					)))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(effects.Restricts).To(HaveLen(n))
				}
				Expect(deleteChannels(ctx, db, 1)).To(MatchError(ContainSubstring(
					"cannot delete channel 1 because holder a references it",
				)))
			},
			Entry("one under", 2, false),
			Entry("at", 3, false),
			Entry("one over", 4, true),
		)

		It("Should leave the delete unchanged after a plan in the same transaction",
			func(ctx SpecContext) {
				open(ctx)
				create(
					ctx,
					db,
					pdCalc{Key: "c", Output: 1},
					pdHolder{Key: "h", Plotted: []uint32{1}, Tasks: []uint32{2}},
				)
				tx := DeferClose(db.OpenTx())
				MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1, 2}))
				Expect(deleteChannels(ctx, tx, 2)).To(MatchError(ContainSubstring(
					"cannot delete channel 2 because holder h references it",
				)))
				Expect(deleteChannels(ctx, tx, 1)).To(Succeed())
				MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1}))
				Expect(calcs.NewRetrieve().Exists(ctx, tx)).To(BeFalse())
				var h pdHolder
				Expect(holders.NewRetrieve().
					Where(gorp.MatchKeys[string, pdHolder]("h")).
					Entry(&h).
					Exec(ctx, tx)).To(Succeed())
				Expect(h.Tasks).To(Equal([]uint32{2}))
			})

		It("Should not list a holder whose delete is staged", func(ctx SpecContext) {
			open(ctx)
			create(ctx, db, pdHolder{Key: "h", Tasks: []uint32{1}})
			tx := DeferClose(db.OpenTx())
			Expect(holders.NewDelete().
				Where(gorp.MatchKeys[string, pdHolder]("h")).
				Exec(ctx, tx)).To(Succeed())
			Expect(MustSucceed(channels.PlanDelete(ctx, tx, []uint32{1})).Restricts).
				To(BeEmpty())
			Expect(MustSucceed(channels.PlanDelete(ctx, db, []uint32{1})).Restricts).
				To(HaveLen(1))
		})

		It("Should plan through a snapshot", func(ctx SpecContext) {
			open(ctx)
			create(ctx, db, pdHolder{Key: "h", Tasks: []uint32{1}})
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
				effects, err := channels.PlanDelete(ctx, snap, []uint32{1})
				Expect(effects.Restricts).To(HaveLen(1))
				return err
			})).To(Succeed())
		})

		It("Should plan nothing for a table outside the registry", func(
			ctx SpecContext,
		) {
			open(ctx)
			t := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, bkLoose]{
				DB: db,
			}))
			Expect(t.NewCreate().Entry(&bkLoose{Key: 1}).Exec(ctx, db)).To(Succeed())
			Expect(MustSucceed(t.PlanDelete(ctx, db, []uint32{1}))).
				To(Equal(gorp.DeleteEffects{}))
		})

		It("Should return the RestrictError from the delete when restrictions pass "+
			"the limit", func(ctx SpecContext) {
			open(ctx, gorp.WithMaxDeleteSize(2))
			create(
				ctx,
				db,
				pdHolder{Key: "a", Tasks: []uint32{1}},
				pdHolder{Key: "b", Tasks: []uint32{1}},
				pdHolder{Key: "c", Tasks: []uint32{1}},
			)
			Expect(deleteChannels(ctx, db, 1)).To(MatchError(gorp.RestrictError{
				Target:    channel(1),
				Holder:    target("holder", "a"),
				Reference: "tasks",
			}))
		})
	})
})
