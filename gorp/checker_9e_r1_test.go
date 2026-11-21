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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// c9eTarget is an entry that c9eHolder points at.
type c9eTarget struct{ Key uint32 }

func (t c9eTarget) GorpKey() uint32 { return t.Key }

// c9eHolder points at a target through a reference that can retarget, and at another
// through one that cannot.
type c9eHolder struct {
	Key   string
	Moved uint32
	Fixed uint32
}

func (h c9eHolder) GorpKey() string { return h.Key }

var _ = Describe("Checker 9e r1: references to a table", func() {
	var (
		db      *gorp.DB
		targets *gorp.Table[uint32, c9eTarget]
		holders *gorp.Table[string, c9eHolder]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		targets = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, c9eTarget]{
			DB:     db,
			Name:   "c9e_target",
			Issued: func(uint32) bool { return true },
		}))
		holders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, c9eHolder]{
			DB:   db,
			Name: "c9e_holder",
			References: []gorp.Reference[string, c9eHolder]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, c9eHolder, uint32]{
					Name:     "moved",
					Table:    "c9e_target",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(h *c9eHolder) []uint32 { return []uint32{h.Moved} },
					Retarget: func(h *c9eHolder, to map[uint32]uint32) {
						if k, ok := to[h.Moved]; ok {
							h.Moved = k
						}
					},
				}),
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, c9eHolder, uint32]{
					Name:     "fixed",
					Table:    "c9e_target",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(h *c9eHolder) []uint32 { return []uint32{h.Fixed} },
				}),
			},
		}))
	})

	It("Should report which references can retarget", func() {
		Expect(db.References("c9e_target")).To(ConsistOf(
			SatisfyAll(
				HaveField("ID", gorp.ReferenceID{Table: "c9e_holder", Name: "moved"}),
				HaveField("Retargets", true),
			),
			SatisfyAll(
				HaveField("ID", gorp.ReferenceID{Table: "c9e_holder", Name: "fixed"}),
				HaveField("Retargets", false),
			),
		))
	})

	It("Should name the reference of each dependent", func(ctx SpecContext) {
		Expect(targets.NewCreate().
			Entries(&[]c9eTarget{{Key: 1}, {Key: 2}}).
			Exec(ctx, db)).To(Succeed())
		Expect(holders.NewCreate().
			Entry(&c9eHolder{Key: "h", Moved: 1, Fixed: 2}).
			Exec(ctx, db)).To(Succeed())
		deps := MustSucceed(targets.Dependents(ctx, db, 1))
		Expect(deps).To(HaveLen(1))
		Expect(deps[0].ReferenceID()).
			To(Equal(gorp.ReferenceID{Table: "c9e_holder", Name: "moved"}))
	})
})
