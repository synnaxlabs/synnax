// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	retrieve := func(ctx context.Context, tx gorp.Tx, key panel.Key) panel.Panel {
		GinkgoHelper()
		var res panel.Panel
		Expect(svc.NewRetrieve().Where(panel.MatchKeys(key)).Entry(&res).Exec(ctx, tx)).
			To(Succeed())
		return res
	}

	Describe("Create", func() {
		It("Should assign a key when the panel's key is nil", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			p := panel.Panel{Name: "test", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil()))
		})

		It(
			"Should default a freshly created panel to a single empty leaf",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				p := panel.Panel{Name: "test", Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
				res := retrieve(ctx, tx, p.Key)
				leaf := MustBeOk(asLeaf(res.Root))
				Expect(leaf.Tabs).To(BeEmpty())
			},
		)

		It("Should preserve a caller-provided tree", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			key := uuid.New()
			p := panel.Panel{Name: "test", Root: leafNode(tab(key)), Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			res := retrieve(ctx, tx, p.Key)
			leaf := MustBeOk(asLeaf(res.Root))
			Expect(leaf.Tabs).To(HaveLen(1))
			Expect(leaf.Tabs[0].Key()).To(Equal(key))
		})

		DescribeTable("Should reject a caller-provided tree that violates invariants",
			func(ctx SpecContext, root func() panel.Node, expected string) {
				tx := DeferClose(db.OpenTx())
				p := panel.Panel{Name: "invalid", Root: root(), Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &p)).
					To(MatchError(ContainSubstring(expected)))
			},
			Entry("duplicate tab keys", func() panel.Node {
				key := uuid.New()
				return splitNode(spatial.DirectionX, 0.5,
					leafNode(tab(key)), leafNode(tab(key)))
			}, "duplicate tab key in panel tree"),
			Entry("duplicate resources", func() panel.Node {
				resource := tabResource(uuid.New())
				duplicated := func() panel.Tab {
					return panel.Tab{Variant: panel.ResourceTab{
						Key:      uuid.New(),
						Resource: resource,
					}}
				}
				return splitNode(spatial.DirectionX, 0.5,
					leafNode(duplicated()), leafNode(duplicated()))
			}, "duplicate resource in panel tree"),
			Entry("resource tab type outside the registry", func() panel.Node {
				return leafNode(panel.Tab{Variant: panel.ResourceTab{
					Key:      uuid.New(),
					Resource: resource.ID{Type: "user", Key: uuid.New().String()},
				}})
			}, "resource tab cannot display a user"),
			Entry("split size out of bounds", func() panel.Node {
				return splitNode(spatial.DirectionX, 1.5,
					leafNode(tab(uuid.New())), leafNode(tab(uuid.New())))
			}, "split size must be in [0, 1]"),
			Entry("nil child node", func() panel.Node {
				return splitNode(spatial.DirectionX, 0.5,
					leafNode(tab(uuid.New())), panel.Node{})
			}, "node has no variant"),
		)

		It(
			"Should reject a panel with no project",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				p := panel.Panel{Name: "draft"}
				Expect(
					svc.NewWriter(tx).Create(ctx, &p),
				).To(MatchError(ContainSubstring(
					"project: must belong to a project",
				)))
			},
		)

		It("Should reject a panel whose project does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			missing := uuid.New()
			p := panel.Panel{Name: "dangling", Project: &missing}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(MatchError(
				ContainSubstring(
					"project: project " + missing.String() + " does not exist",
				),
			))
		})

		It("Should refuse a panel whose key exists", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			other := project.Project{Name: "ignored"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			stored := panel.Panel{Name: "first", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &stored)).To(Succeed())
			again := panel.Panel{Key: stored.Key, Name: "second", Project: &other.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("panel %s already exists", stored.Key)),
			))
			Expect(retrieve(ctx, tx, stored.Key)).To(Equal(stored))
		})

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[panel.Key, panel.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := panel.Panel{Name: "broadcast", Project: &proj.Key}
				Expect(
					svc.NewWriter(tx).WithDispatchKey("dk-1").Create(ctx, &v),
				).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Key))
				Expect(seen[0].DispatchKey).To(Equal("dk-1"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(panel.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[panel.Key, panel.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := panel.Panel{Name: "discarded", Project: &proj.Key}
				Expect(svc.NewWriter(uncommitted).Create(ctx, &v)).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})

	Describe("CreateMany", func() {
		It(
			"Should create every panel with keys assigned and trees defaulted",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				ps := []panel.Panel{
					{Name: "a", Project: &proj.Key},
					{Name: "b", Project: &proj.Key},
				}
				Expect(svc.NewWriter(tx).CreateMany(ctx, &ps)).To(Succeed())
				for _, p := range ps {
					Expect(p.Key).ToNot(Equal(uuid.Nil()))
					leaf := MustBeOk(asLeaf(retrieve(ctx, tx, p.Key).Root))
					Expect(leaf.Tabs).To(BeEmpty())
				}
			},
		)

		It("Should give each panel its own project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			other := project.Project{Name: "other-project"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			ps := []panel.Panel{
				{Name: "a", Project: &proj.Key},
				{Name: "b", Project: &other.Key},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &ps)).To(Succeed())
			Expect(retrieve(ctx, tx, ps[0].Key).Project).To(HaveValue(Equal(proj.Key)))
			Expect(retrieve(ctx, tx, ps[1].Key).Project).To(HaveValue(Equal(other.Key)))
		})

		It(
			"Should reject the batch when any panel's tree is invalid",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				key := uuid.New()
				ps := []panel.Panel{
					{Name: "ok", Project: &proj.Key},
					{Name: "bad", Project: &proj.Key, Root: splitNode(
						spatial.DirectionX, 0.5, leafNode(tab(key)), leafNode(tab(key)),
					)},
				}
				Expect(svc.NewWriter(tx).CreateMany(ctx, &ps)).
					To(MatchError(ContainSubstring("duplicate tab key in panel tree")))
			},
		)
	})

	Describe("Apply of a project move", func() {
		It("Should move the panel into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			other := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			p := panel.Panel{Name: "moved", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, p.Key, setProject(other.Key))).
				To(BeEmpty())
			Expect(retrieve(ctx, tx, p.Key).Project).To(HaveValue(Equal(other.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			p := panel.Panel{Name: "stuck", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, p.Key, setProject(uuid.New()))).
				Error().To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Delete", func() {
		var tx gorp.Tx
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It(
			"Should delete a panel so it is no longer retrievable",
			func(ctx SpecContext) {
				p := panel.Panel{Name: "to-delete", Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
				Expect(svc.NewWriter(tx).Delete(ctx, p.Key)).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(panel.MatchKeys(p.Key)).
						Entry(&panel.Panel{}).
						Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
			},
		)

		It("Should delete multiple panels in one call", func(ctx SpecContext) {
			a := panel.Panel{Name: "a", Project: &proj.Key}
			b := panel.Panel{Name: "b", Project: &proj.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(panel.MatchKeys(a.Key, b.Key)).
				Entries(&[]panel.Panel{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Dispatch", func() {
		create := func(ctx context.Context, root panel.Node) panel.Key {
			GinkgoHelper()
			p := panel.Panel{Name: "test", Root: root, Project: &proj.Key}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &p)
			})).To(Succeed())
			return p.Key
		}

		It(
			"Should reject an action that leaves the panel invalid",
			func(ctx SpecContext) {
				key := create(ctx, leafNode())
				Expect(dispatch(ctx, key, "d1", []panel.Action{
					panel.NewSetAction(
						panel.SetPayload{Path: []string{"name"}, Value: ""},
					),
				})).To(MatchError(ContainSubstring("name: required")))
				Expect(retrieve(ctx, db, key).Name).To(Equal("test"))
			},
		)

		It("Should reject an action with no payload", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			Expect(dispatch(ctx, key, "d1", []panel.Action{
				{Type: panel.ActionTypeSet},
			})).To(MatchError(union.ErrMissingPayload))
			Expect(retrieve(ctx, db, key).Name).To(Equal("test"))
		})

		It(
			"Should reject a create without changing the panel it names",
			func(ctx SpecContext) {
				target := create(ctx, leafNode())
				other := create(ctx, leafNode())
				Expect(dispatch(ctx, target, "d1", []panel.Action{
					panel.NewCreateAction(panel.CreatePayload{
						Panel: panel.Panel{
							Key:     other,
							Name:    "overwritten",
							Root:    leafNode(),
							Project: &proj.Key,
						},
					}),
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("only the Core dispatches a panel create"),
					),
				))
				Expect(retrieve(ctx, db, other).Name).To(Equal("test"))
			},
		)

		It("Should apply a set of the name and persist it", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			Expect(dispatch(ctx, key, "d1", []panel.Action{
				panel.NewSetAction(
					panel.SetPayload{Path: []string{"name"}, Value: "renamed"},
				),
			})).To(Succeed())
			Expect(retrieve(ctx, db, key).Name).To(Equal("renamed"))
		})

		It(
			"Should apply an InsertTabs action and persist the tab",
			func(ctx SpecContext) {
				tabKey := uuid.New()
				key := create(ctx, leafNode())
				Expect(dispatch(ctx, key, "d1", []panel.Action{
					panel.NewInsertTabsAction(
						panel.InsertTabsPayload{
							Tabs:       []panel.Tab{tab(tabKey)},
							TargetLeaf: new(int32(1)),
						},
					),
				})).To(Succeed())
				leaf := MustBeOk(asLeaf(retrieve(ctx, db, key).Root))
				Expect(leaf.Tabs).To(HaveLen(1))
				Expect(leaf.Tabs[0].Key()).To(Equal(tabKey))
			},
		)

		It("Should apply a multi-action batch atomically", func(ctx SpecContext) {
			tabKey := uuid.New()
			key := create(ctx, leafNode())
			Expect(dispatch(ctx, key, "d1", []panel.Action{
				panel.NewSetAction(
					panel.SetPayload{Path: []string{"name"}, Value: "batched"},
				),
				panel.NewInsertTabsAction(
					panel.InsertTabsPayload{
						Tabs:       []panel.Tab{tab(tabKey)},
						TargetLeaf: new(int32(1)),
					},
				),
			})).To(Succeed())
			res := retrieve(ctx, db, key)
			Expect(res.Name).To(Equal("batched"))
			Expect(MustBeOk(asLeaf(res.Root)).Tabs).To(HaveLen(1))
		})

		It(
			"Should apply no action when one in the batch is rejected",
			func(ctx SpecContext) {
				key := create(ctx, leafNode())
				Expect(dispatch(ctx, key, "d1", []panel.Action{
					panel.NewSetAction(
						panel.SetPayload{Path: []string{"name"}, Value: "after"},
					),
					panel.NewMoveTabAction(
						panel.MoveTabPayload{
							Key:        uuid.New(),
							TargetLeaf: 1,
						},
					),
				})).Error().To(MatchError(ContainSubstring("tab not found in tree")))
				Expect(retrieve(ctx, db, key).Name).To(Equal("test"))
			},
		)

		It(
			"Should upsert rather than duplicate when inserting a tab with an existing key",
			func(ctx SpecContext) {
				tabKey := uuid.New()
				key := create(ctx, leafNode(viewTab(tabKey, "selector")))
				Expect(dispatch(ctx, key, "d1", []panel.Action{
					panel.NewInsertTabsAction(
						panel.InsertTabsPayload{
							Tabs:       []panel.Tab{tab(tabKey)},
							TargetLeaf: new(int32(1)),
						},
					),
				})).To(Succeed())
				tabs := MustBeOk(asLeaf(retrieve(ctx, db, key).Root)).Tabs
				Expect(tabs).To(HaveLen(1))
				Expect(tabs[0].Variant).To(Equal(panel.ResourceTab{
					Key:      tabKey,
					Resource: tabResource(tabKey),
				}))
			},
		)

		It(
			"Should notify subscribers with the dispatched action on success",
			func(ctx SpecContext) {
				key := create(ctx, leafNode())
				rec := &Recorder[panel.Key, panel.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(
					dispatch(ctx, key, "client-xyz", []panel.Action{
						panel.NewSetAction(
							panel.SetPayload{
								Path:  []string{"name"},
								Value: "broadcast",
							},
						),
					}),
				).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(key))
				Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(panel.ActionTypeSet))
			},
		)

		It(
			"Should assign strictly monotonic Seq across successive dispatches",
			func(ctx SpecContext) {
				key := create(ctx, leafNode())
				rec := &Recorder[panel.Key, panel.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				for _, name := range []string{"a", "b", "c"} {
					Expect(dispatch(ctx, key, "d", []panel.Action{
						panel.NewSetAction(
							panel.SetPayload{Path: []string{"name"}, Value: name},
						),
					})).To(Succeed())
				}
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(3))
				Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
				Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
			},
		)

		It(
			"Should not notify subscribers when the reducer rejects the action",
			func(ctx SpecContext) {
				key := create(ctx, leafNode())
				rec := &Recorder[panel.Key, panel.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(dispatch(ctx, key, "d1", []panel.Action{
					panel.NewMoveTabAction(
						panel.MoveTabPayload{
							Key:        uuid.New(),
							TargetLeaf: 1,
						},
					),
				})).Error().To(MatchError(ContainSubstring("tab not found in tree")))
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})
})
