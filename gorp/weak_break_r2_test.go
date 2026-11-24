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

// r2Node is deleted with the node it names as its parent.
type r2Node struct {
	Key    uint32
	Parent uint32
}

func (n r2Node) GorpKey() uint32 { return n.Key }

// r2Symbol holds a node through a Weak reference.
type r2Symbol struct {
	Key  string
	Node uint32
}

func (s r2Symbol) GorpKey() string { return s.Key }

var _ = Describe("Held break r2", func() {
	var (
		db      *gorp.DB
		nodes   *gorp.Table[uint32, r2Node]
		symbols *gorp.Table[string, r2Symbol]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		nodes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, r2Node]{
			DB:     db,
			Name:   "node",
			Issued: func(uint32) bool { return true },
			References: []gorp.Reference[uint32, r2Node]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[uint32, r2Node, uint32]{
					Name:     "parent",
					Table:    "node",
					OnDelete: gorp.Cascade,
					Kind:     gorp.Place,
					Targets: func(n *r2Node) []uint32 {
						if n.Parent == 0 {
							return nil
						}
						return []uint32{n.Parent}
					},
				}),
			},
		}))
		symbols = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, r2Symbol]{
			DB:   db,
			Name: "symbol",
			References: []gorp.Reference[string, r2Symbol]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, r2Symbol, uint32]{
					Name:     "node",
					Table:    "node",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(s *r2Symbol) []uint32 { return []uint32{s.Node} },
				}),
			},
		}))
	})

	DescribeTable(
		"Should call f with an entry of the table that a cascade removes and a Weak "+
			"reference holds",
		func(ctx SpecContext, planned bool) {
			Expect(nodes.NewCreate().
				Entries(&[]r2Node{{Key: 1}, {Key: 2, Parent: 1}}).
				Exec(ctx, db)).To(Succeed())
			Expect(symbols.NewCreate().
				Entry(&r2Symbol{Key: "s", Node: 2}).
				Exec(ctx, db)).To(Succeed())
			type call struct {
				key  uint32
				held int
			}
			var called []call
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				d := nodes.NewDelete().
					Where(gorp.MatchKeys[uint32, r2Node](1)).
					Held(func(
						_ context.Context,
						_ gorp.Tx,
						n r2Node,
						held []gorp.Dependent,
					) error {
						called = append(called, call{key: n.Key, held: len(held)})
						return nil
					})
				if planned {
					d = d.Planned(MustSucceed(nodes.PlanDelete(ctx, tx, []uint32{1})))
				}
				return d.Exec(ctx, tx)
			})).To(Succeed())
			Expect(nodes.NewRetrieve().Count(ctx, db)).To(Equal(0))
			Expect(called).To(Equal([]call{{key: 1}, {key: 2, held: 1}}),
				"the delete asks for node 1, and its cascade removes node 2, which s holds")
		},
		Entry("walking the references", false),
		Entry("from a plan", true),
	)
})
