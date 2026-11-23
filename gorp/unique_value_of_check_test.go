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

var _ = Describe("UniqueIndex ValueOf check", func() {
	var (
		db        *gorp.DB
		folders   *gorp.Table[string, uniqueFolder]
		itemIndex *gorp.UniqueIndex[uint32, uniqueItem, uniqueSibling]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		folders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, uniqueFolder]{
			DB:      db,
			Name:    "folder",
			Indexes: []gorp.Index[string, uniqueFolder]{newFolderIndex()},
		}))
		itemIndex = newItemIndex()
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
			DB:         db,
			Name:       "item",
			Indexes:    []gorp.Index[uint32, uniqueItem]{itemIndex},
			References: []gorp.Reference[uint32, uniqueItem]{newItemFolderRef()},
		}))
	})

	It("Should fail on a target in a table that no index of the space holds", func(
		ctx SpecContext,
	) {
		Expect(itemIndex.ValueOf(ctx, db, gorp.NewTarget("other", "d"))).Error().
			To(MatchError(ContainSubstring("other")))
	})

	It("Should see an entry of another table that the snapshot stages", func(
		ctx SpecContext,
	) {
		tx := DeferClose(db.OpenTx())
		f := uniqueFolder{Key: "s", Name: "staged"}
		Expect(folders.NewCreate().Entry(&f).Exec(ctx, tx)).To(Succeed())
		staged := gorp.NewTarget("folder", "s")
		v, held := MustSucceed2(itemIndex.ValueOf(ctx, tx, staged))
		Expect(held).To(BeTrue())
		Expect(v).To(Equal(uniqueSibling{Name: "staged"}))
		_, held = MustSucceed2(itemIndex.ValueOf(ctx, db, staged))
		Expect(held).To(BeFalse())
	})

	It("Should fail on a target key that the key type of its table cannot parse", func(
		ctx SpecContext,
	) {
		Expect(itemIndex.ValueOf(ctx, db, gorp.NewTarget("item", "pt"))).Error().
			To(MatchError(ContainSubstring(`[gorp] - invalid key "pt"`)))
	})
})
