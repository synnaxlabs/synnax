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

var _ = Describe("UniqueIndex.Holder break r2", func() {
	var (
		db    *gorp.DB
		index *gorp.UniqueIndex[uint32, uniqueItem, uniqueSibling]
		items *gorp.Table[uint32, uniqueItem]
		pt    = uniqueSibling{Name: "pt"}
		ox    = uniqueSibling{Name: "ox"}
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		index = newItemIndex()
		items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
			DB:      db,
			Name:    "item",
			Indexes: []gorp.Index[uint32, uniqueItem]{index},
		}))
	})
	holder := func(ctx SpecContext, tx gorp.Tx, v uniqueSibling) (uint32, bool) {
		GinkgoHelper()
		return MustSucceed2(index.Holder(ctx, tx, v))
	}

	It("Should find no holder of a missing value", func(ctx SpecContext) {
		_, found := holder(ctx, db, pt)
		Expect(found).To(BeFalse())
	})

	It("Should find no holder of a value deleted in the transaction", func(
		ctx SpecContext,
	) {
		Expect(items.NewCreate().Entry(&uniqueItem{Key: 1, Name: "pt"}).Exec(ctx, db)).
			To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			Expect(items.NewDelete().
				Where(gorp.MatchKeys[uint32, uniqueItem](1)).
				Exec(ctx, tx)).To(Succeed())
			_, found := holder(ctx, tx, pt)
			Expect(found).To(BeFalse())
			k, found := holder(ctx, db, pt)
			Expect(found).To(BeTrue())
			Expect(k).To(Equal(uint32(1)))
			return nil
		})).To(Succeed())
	})

	It("Should follow a value that changes in the transaction", func(ctx SpecContext) {
		Expect(items.NewCreate().Entry(&uniqueItem{Key: 1, Name: "pt"}).Exec(ctx, db)).
			To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			Expect(items.NewCreate().
				Entry(&uniqueItem{Key: 1, Name: "ox"}).
				Exec(ctx, tx)).To(Succeed())
			_, found := holder(ctx, tx, pt)
			Expect(found).To(BeFalse())
			k, found := holder(ctx, tx, ox)
			Expect(found).To(BeTrue())
			Expect(k).To(Equal(uint32(1)))
			_, found = holder(ctx, db, ox)
			Expect(found).To(BeFalse())
			return nil
		})).To(Succeed())
	})

	It("Should find no holder of an exempt entry", func(ctx SpecContext) {
		Expect(items.NewCreate().
			Entry(&uniqueItem{Key: 1, Name: "pt", Exempt: true}).
			Exec(ctx, db)).To(Succeed())
		_, found := holder(ctx, db, pt)
		Expect(found).To(BeFalse())
	})

	It("Should find the other holder of a stored duplicate after one goes", func(
		ctx SpecContext,
	) {
		Expect(gorp.WrapWriter[uint32, uniqueItem](db).Set(
			ctx,
			uniqueItem{Key: 1, Name: "pt"},
			uniqueItem{Key: 2, Name: "pt"},
		)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			first, found := holder(ctx, tx, pt)
			Expect(found).To(BeTrue())
			Expect(items.NewDelete().
				Where(gorp.MatchKeys[uint32, uniqueItem](first)).
				Exec(ctx, tx)).To(Succeed())
			second, found := holder(ctx, tx, pt)
			Expect(found).To(BeTrue())
			Expect(second).ToNot(Equal(first))
			return nil
		})).To(Succeed())
	})
})
