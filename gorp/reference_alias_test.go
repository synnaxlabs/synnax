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
	"unsafe"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// aliasHolder holds one target, which its reference returns as a slice over the field.
type aliasHolder struct {
	Key    uint32
	Target uint32
}

func (h aliasHolder) GorpKey() uint32 { return h.Key }

// aliasTarget is an entry that aliasHolder points at.
type aliasTarget struct{ Key uint32 }

func (t aliasTarget) GorpKey() uint32 { return t.Key }

var _ = Describe("A reference whose Targets aliases its entry", func() {
	var (
		db      *gorp.DB
		targets *gorp.Table[uint32, aliasTarget]
	)
	openTable := func(ctx SpecContext) {
		GinkgoHelper()
		targets = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, aliasTarget]{
			DB:   db,
			Name: "alias_target",
		}))
		stored := []aliasTarget{{Key: 10}, {Key: 20}, {Key: 30}}
		Expect(targets.NewCreate().Entries(&stored).Exec(ctx, db)).To(Succeed())
		type config = gorp.KeyReferenceConfig[uint32, aliasHolder, uint32]
		ref := gorp.NewKeyReference(config{
			Name:     "target",
			Table:    "alias_target",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Use,
			Targets: func(h *aliasHolder) []uint32 {
				if h.Target == 0 {
					return nil
				}
				return unsafe.Slice(&h.Target, 1)
			},
		})
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, aliasHolder]{
			DB:         db,
			Name:       "alias_holder",
			References: []gorp.Reference[uint32, aliasHolder]{ref},
		}))
	}
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
	})
	// holdersOf returns the holders that the reference index records for target.
	holdersOf := func(ctx SpecContext, snap gorp.Snapshot, target uint32) []string {
		GinkgoHelper()
		deps := MustSucceed(targets.Dependents(ctx, snap, target))
		keys := make([]string, len(deps))
		for i, d := range deps {
			keys[i] = d.Holder.Key
		}
		return keys
	}

	It("Should keep the index unchanged when a written entry changes in place", func(
		ctx SpecContext,
	) {
		openTable(ctx)
		h := aliasHolder{Key: 1, Target: 10}
		Expect(gorp.NewCreate[uint32, aliasHolder]().Entry(&h).Exec(ctx, db)).
			To(Succeed())
		h.Target = 20
		Expect(holdersOf(ctx, db, 10)).To(Equal([]string{"1"}))
		Expect(holdersOf(ctx, db, 20)).To(BeEmpty())
	})

	It("Should keep a staged write unchanged when its entry changes in place", func(
		ctx SpecContext,
	) {
		openTable(ctx)
		tx := DeferClose(db.OpenTx())
		h := aliasHolder{Key: 1, Target: 10}
		Expect(gorp.NewCreate[uint32, aliasHolder]().Entry(&h).Exec(ctx, tx)).
			To(Succeed())
		h.Target = 20
		Expect(holdersOf(ctx, tx, 10)).To(Equal([]string{"1"}))
		Expect(holdersOf(ctx, tx, 20)).To(BeEmpty())
	})

	It("Should fill each entry's own target from a scan that reuses one entry", func(
		ctx SpecContext,
	) {
		stored := []aliasHolder{{Key: 1, Target: 10}, {Key: 2, Target: 20}, {Key: 3}}
		Expect(gorp.NewCreate[uint32, aliasHolder]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		openTable(ctx)
		Expect(holdersOf(ctx, db, 10)).To(Equal([]string{"1"}))
		Expect(holdersOf(ctx, db, 20)).To(Equal([]string{"2"}))
		// A write replaces the targets that the scan recorded for the entry.
		moved := aliasHolder{Key: 1, Target: 30}
		Expect(gorp.NewCreate[uint32, aliasHolder]().Entry(&moved).Exec(ctx, db)).
			To(Succeed())
		Expect(holdersOf(ctx, db, 10)).To(BeEmpty())
		Expect(holdersOf(ctx, db, 20)).To(Equal([]string{"2"}))
		Expect(holdersOf(ctx, db, 30)).To(Equal([]string{"1"}))
	})
})
