// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Group", Ordered, func() {
	var (
		db  *gorp.DB
		svc *group.Service
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
		}))
	})

	create := func(ctx context.Context, name string, parent *group.Group) group.Group {
		GinkgoHelper()
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}

	retrieve := func(ctx context.Context, key group.Key) group.Group {
		GinkgoHelper()
		var g group.Group
		Expect(svc.NewRetrieve().
			Where(group.MatchKeys(key)).
			Entry(&g).
			Exec(ctx, db)).To(Succeed())
		return g
	}

	Describe("Create", func() {
		It("Should create a top-level group", func(ctx SpecContext) {
			g := create(ctx, "test1", nil)
			Expect(g.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, g.Key)).To(Equal(g))
		})

		It("Should keep a key that the caller provides", func(ctx SpecContext) {
			key := uuid.New()
			g := group.Group{Key: key, Name: "test2", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			Expect(retrieve(ctx, key).Name).To(Equal("test2"))
		})

		It("Should give a nested group the scope of its parent", func(ctx SpecContext) {
			parent := create(ctx, "parent", nil)
			child := group.Group{Name: "child", Parent: new(parent.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &child)
			})).To(Succeed())
			Expect(retrieve(ctx, child.Key).Scope).To(Equal(resource.TypeChannel))
		})

		It(
			"Should reject a scope that differs from the parent's",
			func(ctx SpecContext) {
				parent := create(ctx, "parent", nil)
				child := group.Group{
					Name:   "child",
					Scope:  resource.TypeDevice,
					Parent: new(parent.ResourceID()),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &child)
				})).To(MatchError(
					"parent: a device group cannot sit under a channel parent: " +
						"validation error",
				))
			},
		)

		It(
			"Should reject a parent that is not a group, project, or rack",
			func(ctx SpecContext) {
				g := group.Group{
					Name:   "bad parent",
					Parent: &resource.ID{Type: resource.TypeChannel, Key: "1"},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &g)
				})).To(MatchError(
					ContainSubstring(
						`must be a group, project, or rack, got "channel"`,
					),
				))
			},
		)

		It("Should reject a parent group that does not exist", func(ctx SpecContext) {
			missing := uuid.New()
			g := group.Group{Name: "orphan", Parent: new(group.ResourceID(missing))}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(MatchError(
				ContainSubstring("group " + missing.String() + " does not exist"),
			))
		})

		It("Should refuse a group whose key exists", func(ctx SpecContext) {
			parent := create(ctx, "parent", nil)
			g := create(ctx, "child", &parent)
			again := group.Group{
				Key:   g.Key,
				Name:  "refused",
				Scope: resource.TypeDevice,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &again)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("group %s already exists", g.Key)),
			))
			Expect(retrieve(ctx, g.Key)).To(Equal(g))
		})

		It("Should reject a group with an empty name", func(ctx SpecContext) {
			g := group.Group{Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should reject a group with no scope and no parent", func(ctx SpecContext) {
			g := group.Group{Name: "no scope"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(MatchError("scope: invalid scope: "))
		})
	})

	Describe("CreateOrRetrieve", func() {
		It("Should create a top-level group once", func(ctx SpecContext) {
			a := MustSucceed(svc.CreateOrRetrieve(ctx, "Once", resource.TypeDevice))
			b := MustSucceed(svc.CreateOrRetrieve(ctx, "Once", resource.TypeDevice))
			Expect(b).To(Equal(a))
			Expect(a.Scope).To(Equal(resource.TypeDevice))
			Expect(a.Parent).To(BeNil())
		})

		It(
			"Should keep groups of the same name in different scopes apart",
			func(ctx SpecContext) {
				a := MustSucceed(
					svc.CreateOrRetrieve(ctx, "Twice", resource.TypeDevice),
				)
				b := MustSucceed(
					svc.CreateOrRetrieve(ctx, "Twice", resource.TypeProject),
				)
				Expect(b.Key).ToNot(Equal(a.Key))
			},
		)
	})

	Describe("Retrieve", func() {
		It("Should retrieve multiple groups by keys", func(ctx SpecContext) {
			g1 := create(ctx, "multi1", nil)
			g2 := create(ctx, "multi2", nil)
			var ret []group.Group
			Expect(svc.NewRetrieve().
				Where(group.MatchKeys(g1.Key, g2.Key)).
				Entries(&ret).
				Exec(ctx, db)).To(Succeed())
			Expect(ret).To(ConsistOf(g1, g2))
		})

		It("Should retrieve a group by its name", func(ctx SpecContext) {
			created := create(ctx, "name-test", nil)
			var g group.Group
			Expect(svc.NewRetrieve().
				Where(group.MatchNames(created.Name)).
				Entry(&g).
				Exec(ctx, db)).To(Succeed())
			Expect(g).To(Equal(created))
		})

		It("Should filter groups by whether they have a parent", func(ctx SpecContext) {
			parent := create(ctx, "nested-filter", nil)
			child := create(ctx, "nested-filter-child", &parent)
			var nested, top []group.Group
			Expect(svc.NewRetrieve().
				Where(group.MatchKeys(parent.Key, child.Key)).
				Where(group.MatchHasParent(true)).
				Entries(&nested).
				Exec(ctx, db)).To(Succeed())
			Expect(nested).To(ConsistOf(child))
			Expect(svc.NewRetrieve().
				Where(group.MatchKeys(parent.Key, child.Key)).
				Where(group.MatchHasParent(false)).
				Entries(&top).
				Exec(ctx, db)).To(Succeed())
			Expect(top).To(ConsistOf(parent))
		})
	})

	Describe("Ancestors", func() {
		It(
			"Should return every group above the given groups once",
			func(ctx SpecContext) {
				root := create(ctx, "root", nil)
				mid := create(ctx, "mid", &root)
				a := create(ctx, "a", &mid)
				b := create(ctx, "b", &mid)
				Expect(svc.Ancestors(ctx, db, a.Key, b.Key)).
					To(ConsistOf(mid, root))
			},
		)

		It("Should return nothing for a top-level group", func(ctx SpecContext) {
			g := create(ctx, "lonely", nil)
			Expect(svc.Ancestors(ctx, db, g.Key)).To(BeEmpty())
		})
	})

	Describe("TopParent", func() {
		It("Should return nil when the top group has no parent", func(ctx SpecContext) {
			root := create(ctx, "top-root", nil)
			leaf := create(ctx, "top-leaf", &root)
			Expect(group.TopParent(ctx, db, leaf.Key)).To(BeNil())
		})

		It(
			"Should return a validation error for a missing group",
			func(ctx SpecContext) {
				key := uuid.New()
				Expect(group.TopParent(ctx, db, key)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("group "+key.String()+" does not exist"),
					),
				))
			},
		)
	})

	Describe("Children", func() {
		It("Should return the groups that sit in a group", func(ctx SpecContext) {
			parent := create(ctx, "children-parent", nil)
			a := create(ctx, "children-a", &parent)
			b := create(ctx, "children-b", &parent)
			Expect(svc.Children(ctx, db, parent.Key)).
				To(ConsistOf(a.ResourceID(), b.ResourceID()))
		})
	})

	It("Should read the groups around a group through a snapshot", func(
		ctx SpecContext,
	) {
		root := create(ctx, "snapshot-root", nil)
		mid := create(ctx, "snapshot-mid", &root)
		leaf := create(ctx, "snapshot-leaf", &mid)
		var (
			ancestors []group.Group
			children  []resource.ID
			top       *resource.ID
		)
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			if ancestors, err = svc.Ancestors(ctx, snap, leaf.Key); err != nil {
				return err
			}
			if children, err = svc.Children(ctx, snap, mid.Key); err != nil {
				return err
			}
			top, err = group.TopParent(ctx, snap, leaf.Key)
			return err
		})).To(Succeed())
		Expect(ancestors).To(ConsistOf(mid, root))
		Expect(children).To(ConsistOf(leaf.ResourceID()))
		Expect(top).To(BeNil())
	})

	Describe("SetParent", func() {
		It("Should move groups under a new parent", func(ctx SpecContext) {
			parent := create(ctx, "new-parent", nil)
			a := create(ctx, "moved-a", nil)
			b := create(ctx, "moved-b", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					SetParent(ctx, new(parent.ResourceID()), a.Key, b.Key)
			})).
				To(Succeed())
			Expect(retrieve(ctx, a.Key).Parent).To(Equal(new(parent.ResourceID())))
			Expect(retrieve(ctx, b.Key).Parent).To(Equal(new(parent.ResourceID())))
		})

		It("Should move a group to the top level", func(ctx SpecContext) {
			parent := create(ctx, "old-parent", nil)
			child := create(ctx, "promoted", &parent)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetParent(ctx, nil, child.Key)
			})).To(Succeed())
			Expect(retrieve(ctx, child.Key).Parent).To(BeNil())
		})

		It("Should reject a parent of a different scope", func(ctx SpecContext) {
			parent := group.Group{Name: "device parent", Scope: resource.TypeDevice}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &parent)
			})).To(Succeed())
			g := create(ctx, "channel child", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetParent(ctx, new(parent.ResourceID()), g.Key)
			})).To(MatchError(
				"parent: a channel group cannot sit under a device parent: " +
					"validation error",
			))
		})

		It("Should reject a move that forms a cycle", func(ctx SpecContext) {
			root := create(ctx, "cycle-root", nil)
			child := create(ctx, "cycle-child", &root)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					SetParent(ctx, new(child.ResourceID()), root.Key)
			})).
				To(MatchError(ContainSubstring("would form a cycle")))
		})
	})

	Describe("Rename", func() {
		It("Should rename a group", func(ctx SpecContext) {
			created := create(ctx, "original-name", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, created.Key, "renamed")
			})).To(Succeed())
			Expect(retrieve(ctx, created.Key).Name).To(Equal("renamed"))
		})

		It("Should reject an empty name", func(ctx SpecContext) {
			created := create(ctx, "keep-name", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, created.Key, "")
			})).
				To(MatchError(ContainSubstring("name: required")))
		})
	})

	Describe("Delete", func() {
		It(
			"Should not delete a group that holds a group it does not name",
			func(ctx SpecContext) {
				parent := create(ctx, "parent-to-keep", nil)
				create(ctx, "child-blocking-delete", &parent)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, parent.Key)
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot delete group parent-to-keep because it is not empty",
					)),
				))
				Expect(retrieve(ctx, parent.Key).Name).To(Equal("parent-to-keep"))
			},
		)

		It("Should delete an empty group", func(ctx SpecContext) {
			created := create(ctx, "to-delete", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, created.Key)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(group.MatchKeys(created.Key)).
				Entry(new(group.Group)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})

		It(
			"Should delete a chain of groups when the delete names all of them",
			func(ctx SpecContext) {
				root := create(ctx, "root-nested", nil)
				level1 := create(ctx, "level1-nested", &root)
				level2 := create(ctx, "level2-nested", &level1)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						Delete(ctx, root.Key, level2.Key, level1.Key)
				})).To(Succeed())
				var groups []group.Group
				Expect(svc.NewRetrieve().
					Where(group.MatchKeys(root.Key, level1.Key, level2.Key)).
					Entries(&groups).
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
				Expect(groups).To(BeEmpty())
			},
		)
	})

	Describe("BuiltinDeleteMigration", func() {
		It("Should delete every group with the builtin scope", func(ctx SpecContext) {
			builtin := group.Group{
				Name:  "builtin-to-delete",
				Scope: resource.TypeBuiltin,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &builtin)
			})).To(Succeed())
			kept := create(ctx, "kept-by-builtin-delete", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.BuiltinDeleteMigration().
					Run(ctx, tx, alamos.Instrumentation{})
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(group.MatchScopes(resource.TypeBuiltin)).
				Exists(ctx, db)).To(BeFalse())
			Expect(retrieve(ctx, kept.Key).Name).To(Equal("kept-by-builtin-delete"))
		})
	})

	Describe("Changes", func() {
		It("Should return the group that a commit creates", func(ctx SpecContext) {
			var names []string
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
					for ch := range svc.Changes(ctx, c) {
						names = append(names, ch.Value.Name)
					}
				}),
			)
			g := group.Group{Name: "observe-test", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			Expect(names).To(Equal([]string{"observe-test"}))
		})
	})

	Describe("Search", func() {
		var tx gorp.Tx
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		createIn := func(ctx SpecContext, name string) resource.ID {
			GinkgoHelper()
			g := group.Group{Name: name, Scope: resource.TypeChannel}
			Expect(svc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
			return g.ResourceID()
		}

		It(
			"Should return the matching groups, closest match first",
			func(ctx SpecContext) {
				a := createIn(ctx, "turbopump test")
				b := createIn(ctx, "turbopump")
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b, a}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b}))
			},
		)

		It(
			"Should not match a staged group outside its transaction",
			func(ctx SpecContext) {
				createIn(ctx, "injector")
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(BeEmpty())
			},
		)
	})
})
