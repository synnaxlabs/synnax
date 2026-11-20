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
	"bytes"
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// breakerItem has a Detach reference and a Cascade reference to a folder.
type breakerItem struct {
	Key    uint32
	Folder string
	Owner  string
	Name   string
}

func (i breakerItem) GorpKey() uint32 { return i.Key }

// breakerFailing is the entry of a table whose open fails.
type breakerFailing struct {
	Key  uint32
	Name string
}

func (f breakerFailing) GorpKey() uint32 { return f.Key }

// gatedKV holds the first iterator opened on a key prefix that ends with match.
type gatedKV struct {
	kv.DB
	match   []byte
	held    chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *gatedKV) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	if bytes.HasSuffix(opts.LowerBound, g.match) {
		g.once.Do(func() {
			close(g.held)
			<-g.release
		})
	}
	return g.DB.OpenIterator(opts)
}

// populationWaitCtx closes waiting the first time a population wait blocks on it.
type populationWaitCtx struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (c *populationWaitCtx) Done() <-chan struct{} {
	pcs := make([]uintptr, 16)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])
	for {
		f, more := frames.Next()
		if strings.Contains(f.Function, "(*population).wait") {
			c.once.Do(func() { close(c.waiting) })
		}
		if !more {
			break
		}
	}
	return c.Context.Done()
}

var _ = Describe("UniqueIndex breaker", func() {
	var (
		db      *gorp.DB
		items   *gorp.Table[uint32, uniqueItem]
		folders *gorp.Table[string, uniqueFolder]
		openDB  = func(store kv.DB) {
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
		openItems = func(ctx context.Context) {
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
		deleteFolder = func(ctx context.Context, tx gorp.Tx, k string) error {
			return folders.NewDelete().
				Where(gorp.MatchKeys[string, uniqueFolder](k)).
				Exec(ctx, tx)
		}
	)

	It(
		"Should delete a folder whose child takes the folder's own name",
		func(ctx SpecContext) {
			openDB(memkv.New())
			openFolders(ctx)
			openItems(ctx)
			Expect(
				createFolder(ctx, db, uniqueFolder{Key: "a", Name: "pt"}),
			).To(Succeed())
			Expect(createItem(ctx, db, uniqueItem{Key: 1, Folder: "a", Name: "pt"})).
				To(Succeed())
			Expect(deleteFolder(ctx, db, "a")).To(Succeed())
			var item uniqueItem
			Expect(items.NewRetrieve().
				Where(gorp.MatchKeys[uint32, uniqueItem](1)).
				Entry(&item).
				Exec(ctx, db)).To(Succeed())
			Expect(item.Folder).To(BeEmpty())
		},
	)

	It(
		"Should delete a folder whose detached item takes the value of a cascaded item",
		func(ctx SpecContext) {
			openDB(memkv.New())
			openFolders(ctx)
			breakers := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[uint32, breakerItem]{
					DB:   db,
					Name: "breaker",
					Indexes: []gorp.Index[uint32, breakerItem]{
						gorp.NewUniqueIndex(
							"sibling",
							uniqueSpaceName,
							func(i *breakerItem) (uniqueSibling, bool) {
								return uniqueSibling{
									Parent: i.Folder,
									Name:   i.Name,
								}, true
							},
						),
					},
					References: []gorp.Reference[uint32, breakerItem]{
						gorp.NewKeyReference(
							gorp.KeyReferenceConfig[uint32, breakerItem, string]{
								Name:     "folder",
								Table:    "folder",
								OnDelete: gorp.Detach,
								Kind:     gorp.Place,
								Targets: func(i *breakerItem) []string {
									if i.Folder == "" {
										return nil
									}
									return []string{i.Folder}
								},
								Detach: func(i *breakerItem, _ string) { i.Folder = "" },
							},
						),
						gorp.NewKeyReference(
							gorp.KeyReferenceConfig[uint32, breakerItem, string]{
								Name:     "owner",
								Table:    "folder",
								OnDelete: gorp.Cascade,
								Kind:     gorp.Use,
								Targets: func(i *breakerItem) []string {
									if i.Owner == "" {
										return nil
									}
									return []string{i.Owner}
								},
							},
						),
					},
				}),
			)
			Expect(
				createFolder(ctx, db, uniqueFolder{Key: "x", Name: "x"}),
			).To(Succeed())
			Expect(breakers.NewCreate().
				Entry(&breakerItem{Key: 1, Folder: "x", Name: "pt"}).
				Exec(ctx, db)).To(Succeed())
			Expect(breakers.NewCreate().
				Entry(&breakerItem{Key: 2, Owner: "x", Name: "pt"}).
				Exec(ctx, db)).To(Succeed())
			Expect(deleteFolder(ctx, db, "x")).To(Succeed())
			Expect(breakers.NewRetrieve().Count(ctx, db)).To(Equal(1))
		},
	)

	It(
		"Should open a space of a new type after its last member closes",
		func(ctx SpecContext) {
			openDB(memkv.New())
			openFolders(ctx)
			Expect(folders.Close()).To(Succeed())
			t, err := gorp.OpenTable(ctx, gorp.TableConfig[uint32, uniqueItem]{
				DB:   db,
				Name: "item",
				Indexes: []gorp.Index[uint32, uniqueItem]{
					gorp.NewUniqueIndex(
						"name",
						uniqueSpaceName,
						func(i *uniqueItem) (string, bool) { return i.Name, true },
					),
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Close()).To(Succeed())
		},
	)

	It(
		"Should not hang a write when a table in its space fails to open",
		func(ctx SpecContext) {
			gated := &gatedKV{
				DB:      memkv.New(),
				match:   []byte("breakerFailing"),
				held:    make(chan struct{}),
				release: make(chan struct{}),
			}
			openDB(gated)
			openFolders(ctx)
			sibling := func(f *breakerFailing) (uniqueSibling, bool) {
				return uniqueSibling{Name: f.Name}, true
			}
			opened := make(chan error, 1)
			go func() {
				// Two indexes of one table in one space: the second join fails.
				_, err := gorp.OpenTable(ctx, gorp.TableConfig[uint32, breakerFailing]{
					DB:   db,
					Name: "failing",
					Indexes: []gorp.Index[uint32, breakerFailing]{
						gorp.NewUniqueIndex("a", uniqueSpaceName, sibling),
						gorp.NewUniqueIndex("b", uniqueSpaceName, sibling),
					},
				})
				opened <- err
			}()
			// The failed open waits for its populate, which the gate holds, with index
			// "a"
			// still in the space.
			Eventually(gated.held).Should(BeClosed())
			timeout, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			wctx := &populationWaitCtx{Context: timeout, waiting: make(chan struct{})}
			written := make(chan error, 1)
			go func() {
				written <- createFolder(wctx, db, uniqueFolder{Key: "a", Name: "pt"})
			}()
			Eventually(wctx.waiting).Should(BeClosed())
			close(gated.release)
			Eventually(opened).Should(Receive(MatchError(ContainSubstring(
				"table failing already has an index",
			))))
			var err error
			Eventually(written).WithTimeout(5 * time.Second).Should(Receive(&err))
			Expect(err).ToNot(HaveOccurred())
		},
	)
})
