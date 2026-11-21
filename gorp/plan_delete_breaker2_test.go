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
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type b2Channel struct{ Key uint32 }

func (c b2Channel) GorpKey() uint32 { return c.Key }

// b2Task refuses the delete of the channels it reads and detaches the channels it
// plots.
type b2Task struct {
	Key     string
	Reads   []uint32
	Plotted []uint32
}

func (t b2Task) GorpKey() string { return t.Key }

// b2Plot is deleted with its task and detaches the channels it plots.
type b2Plot struct {
	Key     string
	Task    string
	Plotted []uint32
}

func (p b2Plot) GorpKey() string { return p.Key }

// b2Twin holds channels in two fields through two Detach references of one name.
type b2Twin struct {
	Key   string
	Left  []uint32
	Right []uint32
}

func (t b2Twin) GorpKey() string { return t.Key }

// b2Status is deleted with its channel.
type b2Status struct {
	Key     string
	Channel uint32
}

func (s b2Status) GorpKey() string { return s.Key }

func b2Without(keys []uint32, t uint32) []uint32 {
	return slices.DeleteFunc(keys, func(k uint32) bool { return k == t })
}

var _ = Describe("Planned delete breaker 2", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, b2Channel]
		tasks    *gorp.Table[string, b2Task]
		plots    *gorp.Table[string, b2Plot]
		twins    *gorp.Table[string, b2Twin]
		statuses *gorp.Table[string, b2Status]
	)
	BeforeEach(func(ctx SpecContext) {
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
					Targets:  func(p *b2Plot) []string { return []string{p.Task} },
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
		twins = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, b2Twin]{
			DB:   db,
			Name: "twin",
			References: []gorp.Reference[string, b2Twin]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Twin, uint32]{
					Name:     "channels",
					Table:    "channel",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(t *b2Twin) []uint32 { return slices.Clone(t.Left) },
					Detach: func(t *b2Twin, k uint32) {
						t.Left = b2Without(t.Left, k)
					},
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, b2Twin, uint32]{
					Name:     "channels",
					Table:    "channel",
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Targets:  func(t *b2Twin) []uint32 { return slices.Clone(t.Right) },
					Detach: func(t *b2Twin, k uint32) {
						t.Right = b2Without(t.Right, k)
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
		for k := uint32(1); k <= 2; k++ {
			Expect(channels.NewCreate().Entry(&b2Channel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
	})
	// planThenDelete plans the delete of channel 1 in one transaction, runs between,
	// then applies the plan through Delete.Planned.
	planThenDelete := func(
		ctx SpecContext,
		between func(tx gorp.Tx) error,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			effects, err := channels.PlanDelete(ctx, tx, []uint32{1})
			if err != nil {
				return err
			}
			if err = between(tx); err != nil {
				return err
			}
			return channels.NewDelete().
				Where(gorp.MatchKeys[uint32, b2Channel](1)).
				Planned(effects).
				Exec(ctx, tx)
		})
	}
	deleteTask := func(ctx SpecContext, k string) func(tx gorp.Tx) error {
		return func(tx gorp.Tx) error {
			return tasks.NewDelete().
				Where(gorp.MatchKeys[string, b2Task](k)).
				Exec(ctx, tx)
		}
	}
	channelExists := func(ctx SpecContext, k uint32) bool {
		GinkgoHelper()
		return MustSucceed(channels.NewRetrieve().
			Where(gorp.MatchKeys[uint32, b2Channel](k)).
			Exists(ctx, db))
	}

	It("Should delete once the caller removes a Restrict holder that also detaches "+
		"the channel", func(ctx SpecContext) {
		Expect(tasks.NewCreate().
			Entry(&b2Task{Key: "t", Reads: []uint32{1}, Plotted: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		Expect(planThenDelete(ctx, deleteTask(ctx, "t"))).To(Succeed())
		Expect(channelExists(ctx, 1)).To(BeFalse())
	})

	It("Should delete once removing the Restrict holder cascades a planned Detach "+
		"holder", func(ctx SpecContext) {
		Expect(tasks.NewCreate().
			Entry(&b2Task{Key: "t", Reads: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		Expect(plots.NewCreate().
			Entry(&b2Plot{Key: "p", Task: "t", Plotted: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		Expect(planThenDelete(ctx, deleteTask(ctx, "t"))).To(Succeed())
		Expect(channelExists(ctx, 1)).To(BeFalse())
		Expect(plots.NewRetrieve().
			Where(gorp.MatchKeys[string, b2Plot]("p")).
			Exists(ctx, db)).To(BeFalse())
	})

	It("Should refuse with a Restrict holder that the transaction writes after the "+
		"plan", func(ctx SpecContext) {
		Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
			return tasks.NewCreate().
				Entry(&b2Task{Key: "late", Reads: []uint32{1}}).
				Exec(ctx, tx)
		})).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("channel", uint32(1)),
			Holder:    gorp.Target{Table: "task", Key: "late"},
			Reference: "reads",
		}))
		Expect(channelExists(ctx, 1)).To(BeTrue())
	})

	It("Should detach a holder that the transaction writes after the plan", func(
		ctx SpecContext,
	) {
		Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
			return tasks.NewCreate().
				Entry(&b2Task{Key: "late", Plotted: []uint32{1, 2}}).
				Exec(ctx, tx)
		})).To(Succeed())
		var late b2Task
		Expect(tasks.NewRetrieve().
			Where(gorp.MatchKeys[string, b2Task]("late")).
			Entry(&late).
			Exec(ctx, db)).To(Succeed())
		Expect(late.Plotted).To(Equal([]uint32{2}))
	})

	It("Should delete a Cascade holder that the transaction writes after the plan",
		func(ctx SpecContext) {
			Expect(planThenDelete(ctx, func(tx gorp.Tx) error {
				return statuses.NewCreate().
					Entry(&b2Status{Key: "late", Channel: 1}).
					Exec(ctx, tx)
			})).To(Succeed())
			Expect(statuses.NewRetrieve().
				Where(gorp.MatchKeys[string, b2Status]("late")).
				Exists(ctx, db)).To(BeFalse())
		})

	It("Should detach the field that holds the channel when two references share a "+
		"name", func(ctx SpecContext) {
		Expect(twins.NewCreate().
			Entry(&b2Twin{Key: "w", Left: []uint32{2}, Right: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		Expect(planThenDelete(ctx, func(gorp.Tx) error { return nil })).To(Succeed())
		var w b2Twin
		Expect(twins.NewRetrieve().
			Where(gorp.MatchKeys[string, b2Twin]("w")).
			Entry(&w).
			Exec(ctx, db)).To(Succeed())
		Expect(w.Right).To(BeEmpty())
		Expect(w.Left).To(Equal([]uint32{2}))
	})
})
