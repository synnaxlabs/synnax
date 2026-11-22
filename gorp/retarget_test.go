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
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

type rtChannel struct{ Key uint32 }

func (c rtChannel) GorpKey() uint32 { return c.Key }

type rtSensor struct{ Key uint32 }

func (s rtSensor) GorpKey() uint32 { return s.Key }

type rtPlot struct {
	Key    string
	X      uint32
	Ys     []uint32
	Spare  *uint32
	Frozen uint32
	Sensor uint32
}

func (p rtPlot) GorpKey() string { return p.Key }

type rtOwner struct {
	Key     string
	Channel uint32
}

func (o rtOwner) GorpKey() string { return o.Key }

// rtMove returns the value of k in to, or k when to does not move it.
func rtMove(to map[uint32]uint32, k uint32) uint32 {
	if t, ok := to[k]; ok {
		return t
	}
	return k
}

// rtScalarRef returns a reference through the scalar field of a plot that field
// returns, which holds no target while it is zero.
func rtScalarRef(
	name, table string,
	field func(p *rtPlot) *uint32,
	retarget bool,
) gorp.Reference[string, rtPlot] {
	cfg := gorp.KeyReferenceConfig[string, rtPlot, uint32]{
		Name:     name,
		Table:    table,
		OnDelete: gorp.Detach,
		Kind:     gorp.Use,
		Targets: func(p *rtPlot) []uint32 {
			if k := *field(p); k != 0 {
				return []uint32{k}
			}
			return nil
		},
		Detach: func(p *rtPlot, _ uint32) { *field(p) = 0 },
	}
	if retarget {
		cfg.Retarget = func(p *rtPlot, to map[uint32]uint32) {
			*field(p) = rtMove(to, *field(p))
		}
	}
	return gorp.NewKeyReference(cfg)
}

func rtYsRef() gorp.Reference[string, rtPlot] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[string, rtPlot, uint32]{
		Name:     "ys",
		Table:    "channel",
		OnDelete: gorp.Detach,
		Kind:     gorp.Use,
		Targets:  func(p *rtPlot) []uint32 { return p.Ys },
		Detach: func(p *rtPlot, t uint32) {
			p.Ys = slices.DeleteFunc(p.Ys, func(k uint32) bool { return k == t })
		},
		Retarget: func(p *rtPlot, to map[uint32]uint32) {
			for i, k := range p.Ys {
				p.Ys[i] = rtMove(to, k)
			}
		},
	})
}

func rtSpareRef() gorp.Reference[string, rtPlot] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[string, rtPlot, uint32]{
		Name:     "spare",
		Table:    "channel",
		OnDelete: gorp.Weak,
		Kind:     gorp.Use,
		Targets: func(p *rtPlot) []uint32 {
			if p.Spare == nil {
				return nil
			}
			return []uint32{*p.Spare}
		},
		Retarget: func(p *rtPlot, to map[uint32]uint32) {
			if p.Spare != nil {
				t := rtMove(to, *p.Spare)
				p.Spare = &t
			}
		},
	})
}

func rtPlotRefs() []gorp.Reference[string, rtPlot] {
	return []gorp.Reference[string, rtPlot]{
		rtScalarRef("x", "channel", func(p *rtPlot) *uint32 { return &p.X }, true),
		rtYsRef(),
		rtSpareRef(),
		rtScalarRef(
			"frozen",
			"channel",
			func(p *rtPlot) *uint32 { return &p.Frozen },
			false,
		),
		rtScalarRef(
			"sensor",
			"sensor",
			func(p *rtPlot) *uint32 { return &p.Sensor },
			true,
		),
	}
}

var _ = Describe("Retarget", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, rtChannel]
		plots    *gorp.Table[string, rtPlot]
		owners   *gorp.Table[string, rtOwner]
		// rewritten holds each entry that the plot table's OnRewrite got.
		rewritten []rtPlot
	)
	BeforeEach(func(ctx SpecContext) {
		rewritten = nil
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, rtChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(k uint32) bool { return k <= 10 },
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, rtSensor]{
			DB:   db,
			Name: "sensor",
		}))
		plots = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, rtPlot]{
			DB:         db,
			Name:       "plot",
			References: rtPlotRefs(),
			OnRewrite: func(_ context.Context, _ gorp.Tx, p rtPlot) error {
				rewritten = append(rewritten, p)
				return nil
			},
		}))
		owners = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, rtOwner]{
			DB:   db,
			Name: "owner",
			References: []gorp.Reference[string, rtOwner]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, rtOwner, uint32]{
					Name:     "channel",
					Table:    "channel",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Unique:   true,
					Targets:  func(o *rtOwner) []uint32 { return []uint32{o.Channel} },
					Retarget: func(o *rtOwner, to map[uint32]uint32) {
						o.Channel = rtMove(to, o.Channel)
					},
				}),
			},
		}))
		for k := uint32(1); k <= 4; k++ {
			Expect(channels.NewCreate().Entry(&rtChannel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
	})

	createPlot := func(ctx SpecContext, p rtPlot) {
		GinkgoHelper()
		Expect(plots.NewCreate().Entry(&p).Exec(ctx, db)).To(Succeed())
	}
	retrievePlot := func(ctx SpecContext, k string) rtPlot {
		GinkgoHelper()
		var p rtPlot
		Expect(plots.NewRetrieve().
			Where(gorp.MatchKeys[string, rtPlot](k)).
			Entry(&p).
			Exec(ctx, db)).To(Succeed())
		return p
	}
	dependents := func(ctx SpecContext, keys ...uint32) []gorp.Dependent {
		GinkgoHelper()
		var out []gorp.Dependent
		for _, deps := range MustSucceed(channels.DependentsEach(ctx, db, keys)) {
			out = append(out, deps...)
		}
		return out
	}
	retarget := func(
		ctx SpecContext,
		to map[uint32]uint32,
		holders []gorp.Dependent,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.Retarget(ctx, tx, to, holders)
		})
	}
	holder := func(key, reference string) gorp.Dependent {
		return gorp.Dependent{
			Holder:    gorp.Target{Table: "plot", Key: key},
			Reference: reference,
		}
	}

	It("Should move each target that the map holds in every named reference", func(
		ctx SpecContext,
	) {
		spare := uint32(2)
		createPlot(ctx, rtPlot{Key: "a", X: 1, Ys: []uint32{1, 2}, Spare: &spare})
		Expect(retarget(ctx, map[uint32]uint32{1: 3}, dependents(ctx, 1))).
			To(Succeed())
		p := retrievePlot(ctx, "a")
		Expect(p.X).To(Equal(uint32(3)))
		Expect(p.Ys).To(Equal([]uint32{3, 2}))
		Expect(*p.Spare).To(Equal(uint32(2)))
	})

	It("Should swap two keys with one map", func(ctx SpecContext) {
		createPlot(ctx, rtPlot{Key: "a", X: 1, Ys: []uint32{1, 2}})
		holders := dependents(ctx, 1, 2)
		Expect(holders).To(HaveLen(3))
		Expect(retarget(ctx, map[uint32]uint32{1: 2, 2: 1}, holders)).To(Succeed())
		p := retrievePlot(ctx, "a")
		Expect(p.X).To(Equal(uint32(2)))
		Expect(p.Ys).To(Equal([]uint32{2, 1}))
	})

	It("Should write a holder named through several references once", func(
		ctx SpecContext,
	) {
		createPlot(ctx, rtPlot{Key: "a", X: 1, Ys: []uint32{1}})
		Expect(retarget(ctx, map[uint32]uint32{1: 3}, dependents(ctx, 1))).
			To(Succeed())
		Expect(rewritten).To(HaveLen(1))
	})

	It("Should run OnRewrite with the entry as written", func(ctx SpecContext) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1, 4}})
		Expect(retarget(ctx, map[uint32]uint32{1: 3}, dependents(ctx, 1))).
			To(Succeed())
		Expect(rewritten).To(HaveLen(1))
		Expect(rewritten[0].Ys).To(Equal([]uint32{3, 4}))
	})

	It("Should not write a holder that holds no key of the map", func(
		ctx SpecContext,
	) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{2}})
		Expect(retarget(
			ctx,
			map[uint32]uint32{1: 3},
			[]gorp.Dependent{holder("a", "ys")},
		)).To(Succeed())
		Expect(rewritten).To(BeEmpty())
		Expect(retrievePlot(ctx, "a").Ys).To(Equal([]uint32{2}))
	})

	It("Should write nothing for an empty map", func(ctx SpecContext) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1}})
		Expect(retarget(ctx, nil, dependents(ctx, 1))).To(Succeed())
		Expect(rewritten).To(BeEmpty())
	})

	It("Should move the holder from the dependents of the old key to the new", func(
		ctx SpecContext,
	) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1}})
		Expect(retarget(ctx, map[uint32]uint32{1: 3}, dependents(ctx, 1))).
			To(Succeed())
		Expect(dependents(ctx, 1)).To(BeEmpty())
		Expect(dependents(ctx, 3)).To(Equal([]gorp.Dependent{{
			Holder:    gorp.Target{Table: "plot", Key: "a"},
			Reference: "ys",
			Kind:      gorp.Use,
		}}))
	})

	It("Should fail when a new target does not exist", func(ctx SpecContext) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1}})
		Expect(retarget(ctx, map[uint32]uint32{1: 9}, dependents(ctx, 1))).
			To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("channel 9 does not exist")),
			))
		Expect(retrievePlot(ctx, "a").Ys).To(Equal([]uint32{1}))
	})

	It("Should let a Weak reference take a deleted key that was issued", func(
		ctx SpecContext,
	) {
		spare := uint32(1)
		createPlot(ctx, rtPlot{Key: "a", Spare: &spare})
		Expect(retarget(ctx, map[uint32]uint32{1: 7}, dependents(ctx, 1))).
			To(Succeed())
		Expect(*retrievePlot(ctx, "a").Spare).To(Equal(uint32(7)))
	})

	It("Should fail when a new target of a Weak reference was never issued", func(
		ctx SpecContext,
	) {
		spare := uint32(1)
		createPlot(ctx, rtPlot{Key: "a", Spare: &spare})
		Expect(retarget(ctx, map[uint32]uint32{1: 50}, dependents(ctx, 1))).
			To(MatchError(ContainSubstring("channel 50 was never issued")))
	})

	It("Should keep a Unique reference unique", func(ctx SpecContext) {
		for _, o := range []rtOwner{{Key: "a", Channel: 1}, {Key: "b", Channel: 2}} {
			Expect(owners.NewCreate().Entry(&o).Exec(ctx, db)).To(Succeed())
		}
		Expect(retarget(ctx, map[uint32]uint32{1: 2}, dependents(ctx, 1))).
			To(MatchError(ContainSubstring("owner b already references channel 2")))
	})

	It("Should fail on a reference that has no Retarget", func(ctx SpecContext) {
		createPlot(ctx, rtPlot{Key: "a", Frozen: 1})
		Expect(retarget(ctx, map[uint32]uint32{1: 3}, dependents(ctx, 1))).
			To(MatchError(ContainSubstring(
				"reference frozen of table plot cannot retarget keys of table channel",
			)))
		Expect(retrievePlot(ctx, "a").Frozen).To(Equal(uint32(1)))
	})

	It("Should fail on a reference that points at another table", func(
		ctx SpecContext,
	) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1}})
		Expect(retarget(
			ctx,
			map[uint32]uint32{1: 3},
			[]gorp.Dependent{holder("a", "sensor")},
		)).To(MatchError(ContainSubstring(
			"reference sensor of table plot cannot retarget keys of table channel",
		)))
	})

	It("Should fail on a reference that the holder's table does not have", func(
		ctx SpecContext,
	) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1}})
		Expect(retarget(
			ctx,
			map[uint32]uint32{1: 3},
			[]gorp.Dependent{holder("a", "zs")},
		)).To(MatchError(ContainSubstring("table plot has no reference zs")))
	})

	It("Should fail on a holder that does not exist", func(ctx SpecContext) {
		Expect(retarget(
			ctx,
			map[uint32]uint32{1: 3},
			[]gorp.Dependent{holder("gone", "ys")},
		)).To(MatchError(query.ErrNotFound))
	})

	It("Should write no holder when a later holder fails", func(ctx SpecContext) {
		createPlot(ctx, rtPlot{Key: "a", Ys: []uint32{1}})
		createPlot(ctx, rtPlot{Key: "b", Frozen: 1})
		Expect(retarget(ctx, map[uint32]uint32{1: 3}, dependents(ctx, 1))).
			To(MatchError(ContainSubstring("cannot retarget")))
		Expect(retrievePlot(ctx, "a").Ys).To(Equal([]uint32{1}))
	})
})
