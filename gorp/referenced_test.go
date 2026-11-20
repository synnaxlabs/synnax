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

type usedChannel struct {
	Key  uint32
	Name string
}

func (c usedChannel) GorpKey() uint32 { return c.Key }

type usedSymbol struct {
	Key      string
	Channels []uint32
}

func (h usedSymbol) GorpKey() string { return h.Key }

type usedTask usedSymbol

func (h usedTask) GorpKey() string { return h.Key }

type usedStatus usedSymbol

func (h usedStatus) GorpKey() string { return h.Key }

type usedNote struct {
	Key    string
	Target *gorp.Target
}

func (n usedNote) GorpKey() string { return n.Key }

const (
	usedReadTag  uint8 = 1
	usedWriteTag uint8 = 2
)

// openUsedHolder opens the table named name, whose entries hold channels in the field
// that channels returns, through one reference with tag and onDelete.
func openUsedHolder[E gorp.Entry[string]](
	ctx SpecContext,
	db *gorp.DB,
	name string,
	tag uint8,
	onDelete gorp.OnDelete,
	channels func(e *E) *[]uint32,
) *gorp.Table[string, E] {
	GinkgoHelper()
	ref := gorp.NewKeyReference(gorp.KeyReferenceConfig[string, E, uint32]{
		Name:     "channels",
		Table:    "channel",
		OnDelete: onDelete,
		Kind:     gorp.Use,
		Tag:      tag,
		Targets:  func(e *E) []uint32 { return *channels(e) },
		Detach: func(e *E, k uint32) {
			*channels(e) = slices.DeleteFunc(
				slices.Clone(*channels(e)),
				func(v uint32) bool { return v == k },
			)
		},
	})
	return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, E]{
		DB:         db,
		Name:       name,
		References: []gorp.Reference[string, E]{ref},
	}))
}

var _ = Describe("FilterReferenced", func() {
	var (
		db       *gorp.DB
		channels *gorp.Table[uint32, usedChannel]
		symbols  *gorp.Table[string, usedSymbol]
		tasks    *gorp.Table[string, usedTask]
		statuses *gorp.Table[string, usedStatus]
		notes    *gorp.Table[string, usedNote]
		open     = func(ctx SpecContext) {
			GinkgoHelper()
			db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			channels = MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[uint32, usedChannel]{
					DB:     db,
					Name:   "channel",
					Issued: func(uint32) bool { return true },
				}),
			)
			symbols = openUsedHolder(
				ctx, db, "symbol", usedReadTag, gorp.Weak,
				func(e *usedSymbol) *[]uint32 { return &e.Channels },
			)
			tasks = openUsedHolder(
				ctx, db, "task", usedWriteTag, gorp.Restrict,
				func(e *usedTask) *[]uint32 { return &e.Channels },
			)
			statuses = openUsedHolder(
				ctx, db, "status", 0, gorp.Cascade,
				func(e *usedStatus) *[]uint32 { return &e.Channels },
			)
			notes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, usedNote]{
				DB:   db,
				Name: "note",
				References: []gorp.Reference[string, usedNote]{
					gorp.NewTargetReference(
						gorp.TargetReferenceConfig[string, usedNote]{
							Name:     "target",
							Tables:   []string{"channel"},
							OnDelete: gorp.Detach,
							Kind:     gorp.Use,
							Tag:      usedReadTag,
							Targets: func(n *usedNote) []gorp.Target {
								if n.Target == nil {
									return nil
								}
								return []gorp.Target{*n.Target}
							},
							Detach: func(n *usedNote, _ gorp.Target) { n.Target = nil },
						},
					),
				},
			}))
		}
		createChannels = func(ctx SpecContext, keys ...uint32) {
			GinkgoHelper()
			for _, k := range keys {
				Expect(channels.NewCreate().
					Entry(&usedChannel{Key: k}).
					Exec(ctx, db)).To(Succeed())
			}
		}
		hold = func(ctx SpecContext, tx gorp.Tx, table, key string, chs ...uint32) {
			GinkgoHelper()
			var err error
			switch table {
			case "symbol":
				e := usedSymbol{Key: key, Channels: chs}
				err = symbols.NewCreate().Entry(&e).Exec(ctx, tx)
			case "task":
				e := usedTask{Key: key, Channels: chs}
				err = tasks.NewCreate().Entry(&e).Exec(ctx, tx)
			default:
				e := usedStatus{Key: key, Channels: chs}
				err = statuses.NewCreate().Entry(&e).Exec(ctx, tx)
			}
			Expect(err).ToNot(HaveOccurred())
		}
		retrieve = func(
			ctx SpecContext,
			snap gorp.Snapshot,
			f gorp.Filter[uint32, usedChannel],
		) []uint32 {
			GinkgoHelper()
			var found []usedChannel
			Expect(channels.NewRetrieve().Where(f).Entries(&found).Exec(ctx, snap)).
				To(Succeed())
			keys := make([]uint32, len(found))
			for i, c := range found {
				keys[i] = c.Key
			}
			slices.Sort(keys)
			return keys
		}
		used = func(referenced bool) gorp.Filter[uint32, usedChannel] {
			return channels.FilterReferenced(referenced, usedReadTag, usedWriteTag)
		}
	)

	BeforeEach(func(ctx SpecContext) {
		open(ctx)
		createChannels(ctx, 1, 2, 3, 4)
	})

	It("Should match the channels that a listed tag holds", func(ctx SpecContext) {
		hold(ctx, db, "symbol", "a", 1)
		hold(ctx, db, "task", "b", 2)
		Expect(retrieve(ctx, db, used(true))).To(Equal([]uint32{1, 2}))
		Expect(retrieve(ctx, db, used(false))).To(Equal([]uint32{3, 4}))
	})

	It("Should not count a reference whose tag is not listed", func(ctx SpecContext) {
		hold(ctx, db, "status", "a", 1)
		Expect(retrieve(ctx, db, used(true))).To(BeEmpty())
		Expect(retrieve(ctx, db, used(false))).To(Equal([]uint32{1, 2, 3, 4}))
	})

	It("Should count the references of every listed tag", func(ctx SpecContext) {
		hold(ctx, db, "status", "a", 1)
		hold(ctx, db, "symbol", "b", 2)
		hold(ctx, db, "task", "c", 3)
		Expect(retrieve(ctx, db, channels.FilterReferenced(true, 0, usedReadTag))).
			To(Equal([]uint32{1, 2}))
	})

	It(
		"Should count a reference that holds targets of any table",
		func(ctx SpecContext) {
			target := gorp.NewTarget("channel", uint32(3))
			Expect(notes.NewCreate().
				Entry(&usedNote{Key: "n", Target: &target}).
				Exec(ctx, db)).To(Succeed())
			Expect(retrieve(ctx, db, used(true))).To(Equal([]uint32{3}))
		},
	)

	It("Should see the links staged in the transaction", func(ctx SpecContext) {
		hold(ctx, db, "symbol", "a", 1)
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		hold(ctx, tx, "symbol", "a", 2)
		hold(ctx, tx, "task", "b", 3)
		Expect(retrieve(ctx, tx, used(true))).To(Equal([]uint32{2, 3}))
		Expect(retrieve(ctx, tx, used(false))).To(Equal([]uint32{1, 4}))
	})

	It("Should free a channel when its last holder is deleted", func(ctx SpecContext) {
		hold(ctx, db, "symbol", "a", 1)
		hold(ctx, db, "task", "b", 1)
		Expect(symbols.NewDelete().
			Where(gorp.MatchKeys[string, usedSymbol]("a")).
			Exec(ctx, db)).To(Succeed())
		Expect(retrieve(ctx, db, used(true))).To(Equal([]uint32{1}))
		Expect(tasks.NewDelete().
			Where(gorp.MatchKeys[string, usedTask]("b")).
			Exec(ctx, db)).To(Succeed())
		Expect(retrieve(ctx, db, used(true))).To(BeEmpty())
	})

	It("Should match no entry for the dead key of a weak link", func(ctx SpecContext) {
		hold(ctx, db, "symbol", "a", 1, 2)
		Expect(channels.NewDelete().
			Where(gorp.MatchKeys[uint32, usedChannel](1)).
			Exec(ctx, db)).To(Succeed())
		Expect(retrieve(ctx, db, used(true))).To(Equal([]uint32{2}))
		Expect(retrieve(ctx, db, used(false))).To(Equal([]uint32{3, 4}))
	})

	It("Should never decode a channel that it leaves out", func(ctx SpecContext) {
		hold(ctx, db, "symbol", "a", 1)
		corrupt[uint32, usedChannel](ctx, db, 1)
		Expect(retrieve(ctx, db, used(false))).To(Equal([]uint32{2, 3, 4}))
	})

	It("Should compose with another filter", func(ctx SpecContext) {
		hold(ctx, db, "symbol", "a", 1)
		f := gorp.And(
			used(false),
			gorp.Match(func(_ gorp.Context, c *usedChannel) (bool, error) {
				return c.Key%2 == 0, nil
			}),
		)
		Expect(retrieve(ctx, db, f)).To(Equal([]uint32{2, 4}))
		Expect(retrieve(ctx, db, gorp.Not(used(false)))).To(Equal([]uint32{1}))
	})

	It("Should fail when a reference index failed to populate", func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		Expect(gorp.NewCreate[string, usedSymbol]().
			Entry(&usedSymbol{Key: "a", Channels: []uint32{1}}).
			Exec(ctx, db)).To(Succeed())
		corrupt[string, usedSymbol](ctx, db, "a")
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, usedChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(uint32) bool { return true },
		}))
		openUsedHolder(
			ctx, db, "symbol", usedReadTag, gorp.Weak,
			func(e *usedSymbol) *[]uint32 { return &e.Channels },
		)
		Expect(channels.NewRetrieve().
			Where(used(false)).
			Entries(&[]usedChannel{}).
			Exec(ctx, db)).To(MatchError(gorp.ErrIndexInvalid))
	})
})
