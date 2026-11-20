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
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("UniqueError", func() {
	It("Should name the holder and the space", func() {
		err := gorp.UniqueError{
			Space:  "namespace.item",
			Holder: gorp.NewTarget("item", uint32(7)),
		}
		Expect(err).To(MatchError(query.ErrUniqueViolation))
		Expect(err.Error()).
			To(Equal("[gorp] - item 7 already holds the value in namespace.item"))
	})
})

var _ = Describe("UniqueIndex delete checks", func() {
	var (
		db      *gorp.DB
		items   *gorp.Table[uint32, uniqueItem]
		folders *gorp.Table[string, uniqueFolder]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		folders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, uniqueFolder]{
			DB:      db,
			Name:    "folder",
			Indexes: []gorp.Index[string, uniqueFolder]{newFolderIndex()},
		}))
		items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
			DB:         db,
			Name:       "item",
			Indexes:    []gorp.Index[uint32, uniqueItem]{newItemIndex()},
			References: []gorp.Reference[uint32, uniqueItem]{newItemFolderRef()},
		}))
	})

	It("Should delete a folder whose item takes the folder's name at the root", func(
		ctx SpecContext,
	) {
		Expect(folders.NewCreate().
			Entry(&uniqueFolder{Key: "a", Name: "pt"}).
			Exec(ctx, db)).To(Succeed())
		Expect(items.NewCreate().
			Entry(&uniqueItem{Key: 1, Folder: "a", Name: "pt"}).
			Exec(ctx, db)).To(Succeed())
		Expect(folders.NewDelete().
			Where(gorp.MatchKeys[string, uniqueFolder]("a")).
			Exec(ctx, db)).To(Succeed())
		var item uniqueItem
		Expect(items.NewRetrieve().
			Where(gorp.MatchKeys[uint32, uniqueItem](1)).
			Entry(&item).
			Exec(ctx, db)).To(Succeed())
		Expect(item).To(Equal(uniqueItem{Key: 1, Name: "pt"}))
	})

	It("Should delete two folders when an item of one takes the other's name", func(
		ctx SpecContext,
	) {
		Expect(folders.NewCreate().
			Entries(&[]uniqueFolder{{Key: "x", Name: "x"}, {Key: "y", Name: "pt"}}).
			Exec(ctx, db)).To(Succeed())
		Expect(items.NewCreate().
			Entry(&uniqueItem{Key: 1, Folder: "x", Name: "pt"}).
			Exec(ctx, db)).To(Succeed())
		Expect(folders.NewDelete().
			Where(gorp.MatchKeys[string, uniqueFolder]("x", "y")).
			Exec(ctx, db)).To(Succeed())
		Expect(folders.NewRetrieve().Count(ctx, db)).To(Equal(0))
	})
})
