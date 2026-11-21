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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// hbNode is deleted with the node it names as its parent.
type hbNode struct {
	Key    uint32
	Parent uint32
}

func (n hbNode) GorpKey() uint32 { return n.Key }

// hbMark holds a node through a Weak reference, and is deleted with its owner.
type hbMark struct {
	Key   string
	Node  uint32
	Owner uint32
}

func (m hbMark) GorpKey() string { return m.Key }

// hbPlain is the entry of a table that has no name, references, or unique indexes.
type hbPlain struct{ Key uint32 }

func (p hbPlain) GorpKey() uint32 { return p.Key }

var _ = Describe("Held heal break", func() {
	var (
		db    *gorp.DB
		nodes *gorp.Table[uint32, hbNode]
		marks *gorp.Table[string, hbMark]
	)
	optional := func(k uint32) []uint32 {
		if k == 0 {
			return nil
		}
		return []uint32{k}
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		nodes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, hbNode]{
			DB:     db,
			Name:   "hb_node",
			Issued: func(uint32) bool { return true },
			References: []gorp.Reference[uint32, hbNode]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[uint32, hbNode, uint32]{
					Name:     "parent",
					Table:    "hb_node",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets:  func(n *hbNode) []uint32 { return optional(n.Parent) },
				}),
			},
		}))
		marks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, hbMark]{
			DB:   db,
			Name: "hb_mark",
			References: []gorp.Reference[string, hbMark]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, hbMark, uint32]{
					Name:     "node",
					Table:    "hb_node",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(m *hbMark) []uint32 { return optional(m.Node) },
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, hbMark, uint32]{
					Name:     "owner",
					Table:    "hb_node",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets:  func(m *hbMark) []uint32 { return optional(m.Owner) },
				}),
			},
		}))
	})

	It("Should pass nil holders for a cascaded entry whose only Weak holder the same "+
		"delete removes", func(ctx SpecContext) {
		ns := []hbNode{{Key: 1}, {Key: 2, Parent: 1}}
		Expect(nodes.NewCreate().Entries(&ns).Exec(ctx, db)).To(Succeed())
		Expect(marks.NewCreate().
			Entry(&hbMark{Key: "m", Node: 2, Owner: 1}).
			Exec(ctx, db)).To(Succeed())
		type call struct {
			key     uint32
			holders []gorp.Dependent
		}
		var calls []call
		Expect(nodes.NewDelete().
			Where(gorp.MatchKeys[uint32, hbNode](1)).
			Held(func(
				_ context.Context,
				_ gorp.Tx,
				n hbNode,
				holders []gorp.Dependent,
			) error {
				calls = append(calls, call{key: n.Key, holders: holders})
				return nil
			}).
			Exec(ctx, db)).To(Succeed())
		Expect(calls).To(HaveLen(2))
		Expect(calls[1].key).To(Equal(uint32(2)))
		Expect(calls[1].holders).To(BeNil(),
			"HeldFunc promises nil holders when none holds the entry, but mark m "+
				"goes in the same delete and f gets an empty, non-nil slice")
	})

	It("Should call f for each entry that a table outside the registry removes",
		func(ctx SpecContext) {
			plain := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, hbPlain]{
				DB: db,
			}))
			Expect(plain.NewCreate().Entry(&hbPlain{Key: 7}).Exec(ctx, db)).
				To(Succeed())
			var called []uint32
			Expect(plain.NewDelete().
				Where(gorp.MatchKeys[uint32, hbPlain](7)).
				Held(func(
					_ context.Context,
					_ gorp.Tx,
					p hbPlain,
					_ []gorp.Dependent,
				) error {
					called = append(called, p.Key)
					return nil
				}).
				Exec(ctx, db)).To(Succeed())
			Expect(plain.NewRetrieve().Count(ctx, db)).To(Equal(0))
			Expect(called).To(Equal([]uint32{7}),
				"Held promises a call for each entry the delete removes, and fails "+
					"only on a Delete that no Table built, but this Table skips f")
		})
})
