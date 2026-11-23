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
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// appliedHookDB calls during inside the OnApplied of each commit, after the writes of
// the commit are visible and before Gorp applies its index changes.
type appliedHookDB struct {
	kv.DB
	during func()
}

type appliedHookTx struct {
	kv.Tx
	db *appliedHookDB
}

func (d *appliedHookDB) OpenTx() kv.Tx {
	return appliedHookTx{Tx: d.DB.OpenTx(), db: d}
}

func (t appliedHookTx) Commit(ctx context.Context, opts ...any) error {
	during := t.db.during
	wrapped := make([]any, len(opts))
	for i, o := range opts {
		if applied, ok := o.(kv.OnApplied); ok && during != nil {
			o = kv.OnApplied(func() {
				during()
				applied()
			})
		}
		wrapped[i] = o
	}
	return t.Tx.Commit(ctx, wrapped...)
}

var _ = Describe("UniqueIndex Value snapshot check", func() {
	var (
		db        *gorp.DB
		hook      *appliedHookDB
		items     *gorp.Table[uint32, uniqueItem]
		folders   *gorp.Table[string, uniqueFolder]
		itemIndex *gorp.UniqueIndex[uint32, uniqueItem, uniqueSibling]
		openDB    = func(store kv.DB) {
			db = DeferClose(gorp.Wrap(store, gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
		}
		openFolders = func(ctx context.Context) {
			GinkgoHelper()
			folders = MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, uniqueFolder]{
					DB:      db,
					Name:    "folder",
					Indexes: []gorp.Index[string, uniqueFolder]{newFolderIndex()},
				}),
			)
		}
		openItems = func(
			ctx context.Context,
			onRewrite func(context.Context, gorp.Tx, uniqueItem) error,
		) {
			GinkgoHelper()
			itemIndex = newItemIndex()
			items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
				DB:         db,
				Name:       "item",
				Indexes:    []gorp.Index[uint32, uniqueItem]{itemIndex},
				References: []gorp.Reference[uint32, uniqueItem]{newItemFolderRef()},
				OnRewrite:  onRewrite,
			}))
		}
		createItem = func(ctx context.Context, tx gorp.Tx, i uniqueItem) error {
			return items.NewCreate().Entry(&i).Exec(ctx, tx)
		}
		createFolder = func(ctx context.Context, tx gorp.Tx, f uniqueFolder) error {
			return folders.NewCreate().Entry(&f).Exec(ctx, tx)
		}
		folderExists = func(ctx context.Context, snap gorp.Snapshot, k string) bool {
			GinkgoHelper()
			return MustSucceed(folders.NewRetrieve().
				Where(gorp.MatchKeys[string, uniqueFolder](k)).
				Exists(ctx, snap))
		}
	)

	Describe("Commit in flight", func() {
		BeforeEach(func(ctx SpecContext) {
			hook = &appliedHookDB{DB: memkv.New()}
			openDB(hook)
			openFolders(ctx)
			openItems(ctx, nil)
		})

		It("Should read the new value of an entry once a rename is visible", func(
			ctx SpecContext,
		) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			var (
				stored uniqueItem
				v      uniqueSibling
				held   bool
			)
			hook.during = func() {
				Expect(items.NewRetrieve().
					Where(gorp.MatchKeys[uint32, uniqueItem](1)).
					Entry(&stored).
					Exec(ctx, db)).To(Succeed())
				v, held = MustSucceed2(itemIndex.Value(ctx, db, 1))
			}
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "zz"})).To(Succeed())
			hook.during = nil
			Expect(stored.Name).To(Equal("zz"))
			Expect(held).To(BeTrue())
			Expect(v).To(Equal(uniqueSibling{Name: "zz"}))
		})

		It("Should report no value for a folder once its delete is visible", func(
			ctx SpecContext,
		) {
			Expect(createFolder(ctx, db, uniqueFolder{Key: "a", Name: "a"})).
				To(Succeed())
			var (
				exists bool
				held   bool
			)
			hook.during = func() {
				exists = folderExists(ctx, db, "a")
				_, held = MustSucceed2(
					itemIndex.ValueOf(ctx, db, gorp.NewTarget("folder", "a")),
				)
			}
			Expect(folders.NewDelete().
				Where(gorp.MatchKeys[string, uniqueFolder]("a")).
				Exec(ctx, db)).To(Succeed())
			hook.during = nil
			Expect(exists).To(BeFalse())
			Expect(held).To(BeFalse())
		})
	})

	Describe("Delete in progress", func() {
		It("Should see a folder that a delete leaves in place for OnRewrite", func(
			ctx SpecContext,
		) {
			openDB(memkv.New())
			openFolders(ctx)
			var (
				ran    bool
				exists bool
				v      uniqueSibling
				held   bool
			)
			openItems(ctx, func(ctx context.Context, tx gorp.Tx, _ uniqueItem) error {
				ran = true
				exists = folderExists(ctx, tx, "a")
				var err error
				v, held, err = itemIndex.ValueOf(ctx, tx, gorp.NewTarget("folder", "a"))
				return err
			})
			Expect(createFolder(ctx, db, uniqueFolder{Key: "a", Name: "a"})).
				To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Folder: "a", Name: "pt"})).
				To(Succeed())
			Expect(folders.NewDelete().
				Where(gorp.MatchKeys[string, uniqueFolder]("a")).
				Exec(ctx, db)).To(Succeed())
			Expect(ran).To(BeTrue())
			Expect(exists).To(BeTrue())
			Expect(held).To(BeTrue())
			Expect(v).To(Equal(uniqueSibling{Name: "a"}))
		})
	})
})
