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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type uniqueSibling struct{ Parent, Name string }

type uniqueItem struct {
	Key    uint32
	Folder string
	Name   string
	// Exempt entries hold no value.
	Exempt bool
}

func (i uniqueItem) GorpKey() uint32 { return i.Key }

type uniqueFolder struct {
	Key    string
	Parent string
	Name   string
}

func (f uniqueFolder) GorpKey() string { return f.Key }

const uniqueSpaceName = "namespace.item"

func newItemIndex() *gorp.UniqueIndex[uint32, uniqueItem, uniqueSibling] {
	return gorp.NewUniqueIndex(
		"sibling",
		uniqueSpaceName,
		func(i *uniqueItem) (uniqueSibling, bool) {
			return uniqueSibling{Parent: i.Folder, Name: i.Name}, !i.Exempt
		},
	)
}

func newFolderIndex() *gorp.UniqueIndex[string, uniqueFolder, uniqueSibling] {
	return gorp.NewUniqueIndex(
		"sibling",
		uniqueSpaceName,
		func(f *uniqueFolder) (uniqueSibling, bool) {
			return uniqueSibling{Parent: f.Parent, Name: f.Name}, true
		},
	)
}

func newItemFolderRef() *gorp.KeyReference[uint32, uniqueItem, string] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[uint32, uniqueItem, string]{
		Name:     "folder",
		Table:    "folder",
		OnDelete: gorp.Detach,
		Kind:     gorp.Place,
		Targets: func(i *uniqueItem) []string {
			if i.Folder == "" {
				return nil
			}
			return []string{i.Folder}
		},
		Detach: func(i *uniqueItem, _ string) { i.Folder = "" },
	})
}

// expectClash expects err to be the UniqueError of a write of name at the top, whose
// value holder holds.
func expectClash(err error, holder gorp.Target, name string) {
	GinkgoHelper()
	Expect(err).To(MatchError(query.ErrUniqueViolation))
	var clash gorp.UniqueError
	Expect(errors.As(err, &clash)).To(BeTrue())
	Expect(clash).To(Equal(gorp.UniqueError{
		Space:  uniqueSpaceName,
		Holder: holder,
		Value:  uniqueSibling{Name: name},
	}))
}

var _ = Describe("UniqueIndex", func() {
	var (
		db      *gorp.DB
		items   *gorp.Table[uint32, uniqueItem]
		folders *gorp.Table[string, uniqueFolder]
		openDB  = func() {
			db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
		}
		openFolders = func(ctx SpecContext) {
			GinkgoHelper()
			folders = MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, uniqueFolder]{
					DB:      db,
					Name:    "folder",
					Indexes: []gorp.Index[string, uniqueFolder]{newFolderIndex()},
				}),
			)
		}
		openItems = func(ctx SpecContext) {
			GinkgoHelper()
			items = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
				DB:         db,
				Name:       "item",
				Indexes:    []gorp.Index[uint32, uniqueItem]{newItemIndex()},
				References: []gorp.Reference[uint32, uniqueItem]{newItemFolderRef()},
			}))
		}
		createItem = func(ctx context.Context, tx gorp.Tx, i uniqueItem) error {
			return items.NewCreate().Entry(&i).Exec(ctx, tx)
		}
		createFolder = func(ctx context.Context, tx gorp.Tx, f uniqueFolder) error {
			return folders.NewCreate().Entry(&f).Exec(ctx, tx)
		}
		itemTarget = func(k uint32) gorp.Target { return gorp.NewTarget("item", k) }
	)

	Describe("One table", func() {
		BeforeEach(func(ctx SpecContext) {
			openDB()
			openFolders(ctx)
			openItems(ctx)
		})

		It("Should refuse a second entry with the same value", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			expectClash(
				createItem(ctx, db, uniqueItem{Key: 2, Name: "pt"}),
				itemTarget(1),
				"pt",
			)
		})

		It("Should allow the same name under two parents", func(ctx SpecContext) {
			Expect(createFolder(ctx, db, uniqueFolder{Key: "a", Name: "a"})).
				To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Folder: "a", Name: "pt"})).
				To(Succeed())
		})

		It("Should let an entry write the value it holds", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
		})

		It("Should see a value staged in the same transaction", func(ctx SpecContext) {
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := createItem(
					ctx,
					tx,
					uniqueItem{Key: 1, Name: "pt"},
				); err != nil {
					return err
				}
				return createItem(ctx, tx, uniqueItem{Key: 2, Name: "pt"})
			})
			expectClash(err, itemTarget(1), "pt")
			Expect(items.NewRetrieve().Count(ctx, db)).To(BeZero())
		})

		It("Should free a value when its entry is deleted", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := items.NewDelete().
					Where(gorp.MatchKeys[uint32, uniqueItem](1)).
					Exec(ctx, tx); err != nil {
					return err
				}
				return createItem(ctx, tx, uniqueItem{Key: 2, Name: "pt"})
			})).To(Succeed())
			expectClash(
				createItem(ctx, db, uniqueItem{Key: 3, Name: "pt"}),
				itemTarget(2),
				"pt",
			)
		})

		It("Should free the old value of a changed entry", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(items.NewUpdate().
				Where(gorp.MatchKeys[uint32, uniqueItem](1)).
				Change(func(_ gorp.Context, i uniqueItem) uniqueItem {
					i.Name = "pt_2"
					return i
				}).
				Exec(ctx, db)).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Name: "pt"})).To(Succeed())
			expectClash(
				createItem(ctx, db, uniqueItem{Key: 3, Name: "pt_2"}),
				itemTarget(1),
				"pt_2",
			)
		})

		It("Should keep the value of a write that fails", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := createItem(
					ctx,
					tx,
					uniqueItem{Key: 1, Name: "pt"},
				); err != nil {
					return err
				}
				return errors.New("abort")
			})).To(MatchError("abort"))
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Name: "pt"})).To(Succeed())
		})

		It("Should never check an exempt entry", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Name: "pt", Exempt: true})).
				To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt", Exempt: true})).
				To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 3, Name: "pt"})).To(Succeed())
		})

		It("Should skip the check in an unchecked writer", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return items.NewUncheckedWriter(tx).
					Set(ctx, uniqueItem{Key: 2, Name: "pt"})
			})).To(Succeed())
		})

		It("Should fail one of two concurrent writes", func(ctx SpecContext) {
			var (
				wg   sync.WaitGroup
				errs = make([]error, 2)
			)
			for i := range errs {
				wg.Go(func() {
					errs[i] = db.WithTx(ctx, func(tx gorp.Tx) error {
						return createItem(ctx, tx, uniqueItem{
							Key:  uint32(i + 1),
							Name: "pt",
						})
					})
				})
			}
			wg.Wait()
			failed := 0
			for _, err := range errs {
				if err != nil {
					Expect(err).To(MatchError(query.ErrUniqueViolation))
					failed++
				}
			}
			Expect(failed).To(Equal(1))
			Expect(items.NewRetrieve().Count(ctx, db)).To(Equal(1))
		})
	})

	Describe("Two tables in one space", func() {
		BeforeEach(func(ctx SpecContext) {
			openDB()
			openFolders(ctx)
			openItems(ctx)
		})

		It("Should refuse an item that a folder's value holds", func(ctx SpecContext) {
			Expect(createFolder(ctx, db, uniqueFolder{Key: "a", Name: "pt"})).
				To(Succeed())
			expectClash(
				createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"}),
				gorp.NewTarget("folder", "a"),
				"pt",
			)
		})

		It("Should refuse a folder that an item's value holds", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			expectClash(
				createFolder(ctx, db, uniqueFolder{Key: "a", Name: "pt"}),
				itemTarget(1),
				"pt",
			)
		})

		It("Should see a value staged in the other table", func(ctx SpecContext) {
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := createFolder(
					ctx,
					tx,
					uniqueFolder{Key: "a", Name: "pt"},
				); err != nil {
					return err
				}
				return createItem(ctx, tx, uniqueItem{Key: 1, Name: "pt"})
			})
			expectClash(err, gorp.NewTarget("folder", "a"), "pt")
		})

		It("Should fail a detach onto a taken value", func(ctx SpecContext) {
			Expect(createFolder(ctx, db, uniqueFolder{Key: "a", Name: "a"})).
				To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Folder: "a", Name: "pt"})).
				To(Succeed())
			err := folders.NewDelete().
				Where(gorp.MatchKeys[string, uniqueFolder]("a")).
				Exec(ctx, db)
			expectClash(err, itemTarget(1), "pt")
			Expect(folders.NewRetrieve().Count(ctx, db)).To(Equal(1))
		})

		It("Should stop checking a table after it closes", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
			Expect(items.Close()).To(Succeed())
			Expect(createFolder(ctx, db, uniqueFolder{Key: "a", Name: "pt"})).
				To(Succeed())
		})
	})

	Describe("Stored duplicates", func() {
		BeforeEach(func(ctx SpecContext) {
			openDB()
			Expect(gorp.NewCreate[uint32, uniqueItem]().
				Entries(&[]uniqueItem{{Key: 1, Name: "pt"}, {Key: 2, Name: "pt"}}).
				Exec(ctx, db)).To(Succeed())
			openFolders(ctx)
			openItems(ctx)
		})

		It("Should keep each holder of a duplicate writable", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Name: "pt"})).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Name: "pt"})).To(Succeed())
		})

		It("Should refuse a move into the duplicate", func(ctx SpecContext) {
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Name: "pt_2"})).To(Succeed())
			expectClash(
				createItem(ctx, db, uniqueItem{Key: 2, Name: "pt"}),
				itemTarget(1),
				"pt",
			)
		})
	})

	Describe("Open", func() {
		BeforeEach(func() { openDB() })

		It("Should fail a space of another type", func(ctx SpecContext) {
			openFolders(ctx)
			Expect(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
				DB:   db,
				Name: "item",
				Indexes: []gorp.Index[uint32, uniqueItem]{
					gorp.NewUniqueIndex(
						"name",
						uniqueSpaceName,
						func(i *uniqueItem) (string, bool) { return i.Name, true },
					),
				},
			})).Error().To(MatchError(ContainSubstring(
				`space "namespace.item" holds values of another type`,
			)))
		})

		It("Should fail two indexes of one table in one space", func(ctx SpecContext) {
			Expect(gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
				DB:   db,
				Name: "item",
				Indexes: []gorp.Index[uint32, uniqueItem]{
					newItemIndex(),
					newItemIndex(),
				},
			})).Error().To(MatchError(ContainSubstring(
				`table item already has an index in space "namespace.item"`,
			)))
		})

		It("Should fail when a populate failed", func(ctx SpecContext) {
			Expect(gorp.NewCreate[uint32, uniqueItem]().
				Entry(&uniqueItem{Key: 1, Name: "pt"}).
				Exec(ctx, db)).To(Succeed())
			corrupt[uint32, uniqueItem](ctx, db, 1)
			openFolders(ctx)
			openItems(ctx)
			Expect(createItem(ctx, db, uniqueItem{Key: 2, Name: "pt"})).
				To(MatchError(gorp.ErrIndexInvalid))
		})
	})
})
