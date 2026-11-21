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

// r3Node is deleted with the node it names as its parent.
type r3Node struct {
	Key    uint32
	Parent uint32
}

func (n r3Node) GorpKey() uint32 { return n.Key }

// r3Mark holds a node through a Weak reference, and is deleted with the node it names
// as its owner.
type r3Mark struct {
	Key   string
	Node  uint32
	Owner uint32
}

func (m r3Mark) GorpKey() string { return m.Key }

var _ = Describe("Held break r3", func() {
	var (
		db    *gorp.DB
		nodes *gorp.Table[uint32, r3Node]
		marks *gorp.Table[string, r3Mark]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		nodes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, r3Node]{
			DB:     db,
			Name:   "r3_node",
			Issued: func(uint32) bool { return true },
			References: []gorp.Reference[uint32, r3Node]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[uint32, r3Node, uint32]{
					Name:     "parent",
					Table:    "r3_node",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets: func(n *r3Node) []uint32 {
						if n.Parent == 0 {
							return nil
						}
						return []uint32{n.Parent}
					},
				}),
			},
		}))
		marks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, r3Mark]{
			DB:   db,
			Name: "r3_mark",
			References: []gorp.Reference[string, r3Mark]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, r3Mark, uint32]{
					Name:     "node",
					Table:    "r3_node",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(m *r3Mark) []uint32 { return []uint32{m.Node} },
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, r3Mark, uint32]{
					Name:     "owner",
					Table:    "r3_node",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets: func(m *r3Mark) []uint32 {
						if m.Owner == 0 {
							return nil
						}
						return []uint32{m.Owner}
					},
				}),
			},
		}))
	})

	create := func(ctx SpecContext, ns []r3Node, ms []r3Mark) {
		GinkgoHelper()
		Expect(nodes.NewCreate().Entries(&ns).Exec(ctx, db)).To(Succeed())
		Expect(marks.NewCreate().Entries(&ms).Exec(ctx, db)).To(Succeed())
	}

	// deleteRoot deletes node 1 in one transaction. With planned, it plans the delete
	// first, then runs before, then applies the plan. With held, it records each entry
	// that Held passes to f.
	deleteRoot := func(
		ctx SpecContext,
		planned, held bool,
		before func(tx gorp.Tx) error,
	) ([]uint32, error) {
		var called []uint32
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			var effects gorp.DeleteEffects
			if planned {
				var err error
				if effects, err = nodes.PlanDelete(ctx, tx, []uint32{1}); err != nil {
					return err
				}
			}
			if before != nil {
				if err := before(tx); err != nil {
					return err
				}
			}
			d := nodes.NewDelete().Where(gorp.MatchKeys[uint32, r3Node](1))
			if held {
				d = d.Held(func(
					_ context.Context,
					_ gorp.Tx,
					n r3Node,
					_ []gorp.Dependent,
				) error {
					called = append(called, n.Key)
					return nil
				})
			}
			if planned {
				d = d.Planned(effects)
			}
			return d.Exec(ctx, tx)
		})
		return called, err
	}

	DescribeTable("Should call f for no cascaded entry whose only Weak holder the "+
		"same delete removes",
		func(ctx SpecContext, planned bool) {
			create(
				ctx,
				[]r3Node{{Key: 1}, {Key: 2, Parent: 1}},
				[]r3Mark{{Key: "m", Node: 2, Owner: 1}},
			)
			called := MustSucceed(deleteRoot(ctx, planned, true, nil))
			Expect(marks.NewRetrieve().Count(ctx, db)).To(Equal(0))
			Expect(called).To(BeEmpty(),
				"mark m goes in the same delete, so nothing holds node 2 after it, "+
					"and a plan of the same delete calls f for nothing")
		},
		Entry("walking the references", false),
		Entry("from a plan", true),
	)

	DescribeTable("Should apply a plan after the transaction removed a cascaded entry",
		func(ctx SpecContext, held bool) {
			create(
				ctx,
				[]r3Node{{Key: 1}, {Key: 2, Parent: 1}},
				[]r3Mark{{Key: "m", Node: 2}},
			)
			Expect(deleteRoot(ctx, true, held, func(tx gorp.Tx) error {
				return nodes.NewDelete().
					Where(gorp.MatchKeys[uint32, r3Node](2)).
					Exec(ctx, tx)
			})).Error().ToNot(HaveOccurred())
			Expect(nodes.NewRetrieve().Count(ctx, db)).To(Equal(0))
		},
		Entry("without Held", false),
		Entry("with Held", true),
	)

	It("Should call f with the cascaded entries in key order", func(ctx SpecContext) {
		create(
			ctx,
			[]r3Node{{Key: 1}, {Key: 9, Parent: 1}, {Key: 10, Parent: 1}},
			[]r3Mark{{Key: "a", Node: 9}, {Key: "b", Node: 10}},
		)
		Expect(deleteRoot(ctx, false, true, nil)).To(Equal([]uint32{9, 10}))
	})
})
