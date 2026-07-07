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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	retrieve := func(ctx context.Context, key panel.Key) panel.Panel {
		var res panel.Panel
		Expect(svc.NewRetrieve().Where(panel.MatchKeys(key)).Entry(&res).Exec(ctx, tx)).
			To(Succeed())
		return res
	}

	Describe("Create", func() {
		It("Should assign a key when the panel's key is nil", func(ctx SpecContext) {
			p := panel.Panel{Name: "test", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should default a freshly created panel to a single empty leaf", func(ctx SpecContext) {
			p := panel.Panel{Name: "test", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			res := retrieve(ctx, p.Key)
			leaf := MustBeOk(asLeaf(res.Root))
			Expect(leaf.Tabs).To(BeEmpty())
		})

		It("Should preserve a caller-provided tree", func(ctx SpecContext) {
			key := uuid.New()
			p := panel.Panel{Name: "test", Root: leafNode(tab(key)), Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			res := retrieve(ctx, p.Key)
			leaf := MustBeOk(asLeaf(res.Root))
			Expect(leaf.Tabs).To(HaveLen(1))
			Expect(leaf.Tabs[0].Key()).To(Equal(key))
		})

		DescribeTable("Should reject a caller-provided tree that violates invariants",
			func(ctx SpecContext, root func() panel.Node, expected string) {
				p := panel.Panel{Name: "invalid", Root: root(), Parent: &parentID}
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
					return panel.Tab{Variant: panel.TabResource{
						TabBase:  panel.TabBase{Key: uuid.New()},
						Resource: resource,
					}}
				}
				return splitNode(spatial.DirectionX, 0.5,
					leafNode(duplicated()), leafNode(duplicated()))
			}, "duplicate resource in panel tree"),
			Entry("split size out of bounds", func() panel.Node {
				return splitNode(spatial.DirectionX, 1.5,
					leafNode(tab(uuid.New())), leafNode(tab(uuid.New())))
			}, "split size must be in [0, 1]"),
			Entry("nil child node", func() panel.Node {
				return splitNode(spatial.DirectionX, 0.5,
					leafNode(tab(uuid.New())), panel.Node{})
			}, "node has no variant"),
		)

		It("Should register the panel as an ontology resource", func(ctx SpecContext) {
			p := panel.Panel{Name: "test", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			var resource ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(panel.OntologyID(p.Key)).
				Entry(&resource).
				Exec(ctx, tx)).To(Succeed())
			Expect(resource.ID).To(Equal(panel.OntologyID(p.Key)))
			Expect(resource.Name).To(Equal("test"))
		})

		It("Should parent the panel to the provided parent", func(ctx SpecContext) {
			p := panel.Panel{Name: "with-parent", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: parentID,
				Type: ontology.RelationshipTypeParentOf,
				To:   panel.OntologyID(p.Key),
			})).To(BeTrue())
		})

		It("Should create a panel with no parent relationship when parent is absent", func(ctx SpecContext) {
			p := panel.Panel{Name: "draft"}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: parentID,
				Type: ontology.RelationshipTypeParentOf,
				To:   panel.OntologyID(p.Key),
			})).To(BeFalse())
		})

		It("Should still register the resource when parent is absent", func(ctx SpecContext) {
			p := panel.Panel{Name: "draft"}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(otg.NewRetrieve().
				WhereIDs(panel.OntologyID(p.Key)).
				Entry(&ontology.Resource{}).
				Exec(ctx, tx)).To(Succeed())
		})

		It("Should update an existing panel when called with an existing key", func(ctx SpecContext) {
			key := uuid.New()
			first := panel.Panel{Key: key, Name: "first", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &first)).To(Succeed())
			second := panel.Panel{Key: key, Name: "second", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &second)).To(Succeed())
			Expect(retrieve(ctx, key).Name).To(Equal("second"))
		})
	})

	Describe("CreateMany", func() {
		It("Should create every panel with keys assigned and trees defaulted", func(ctx SpecContext) {
			ps := []panel.Panel{
				{Name: "a", Parent: &parentID},
				{Name: "b", Parent: &parentID},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &ps)).To(Succeed())
			for _, p := range ps {
				Expect(p.Key).ToNot(Equal(uuid.Nil))
				leaf := MustBeOk(asLeaf(retrieve(ctx, p.Key).Root))
				Expect(leaf.Tabs).To(BeEmpty())
			}
		})

		It("Should parent each panel to its own parent", func(ctx SpecContext) {
			other := MustSucceed(node.Group.CreateOrRetrieve(ctx, "other-parent", ontology.RootID)).
				OntologyID()
			ps := []panel.Panel{
				{Name: "a", Parent: &parentID},
				{Name: "b", Parent: &other},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &ps)).To(Succeed())
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: parentID,
				Type: ontology.RelationshipTypeParentOf,
				To:   panel.OntologyID(ps[0].Key),
			})).To(BeTrue())
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: other,
				Type: ontology.RelationshipTypeParentOf,
				To:   panel.OntologyID(ps[1].Key),
			})).To(BeTrue())
		})

		It("Should reject the batch when any panel's tree is invalid", func(ctx SpecContext) {
			key := uuid.New()
			ps := []panel.Panel{
				{Name: "ok", Parent: &parentID},
				{Name: "bad", Parent: &parentID, Root: splitNode(
					spatial.DirectionX, 0.5, leafNode(tab(key)), leafNode(tab(key)),
				)},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &ps)).
				To(MatchError(ContainSubstring("duplicate tab key in panel tree")))
		})
	})

	Describe("Delete", func() {
		It("Should delete a panel so it is no longer retrievable", func(ctx SpecContext) {
			p := panel.Panel{Name: "to-delete", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, p.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(panel.MatchKeys(p.Key)).Entry(&panel.Panel{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should remove the panel's ontology resource", func(ctx SpecContext) {
			p := panel.Panel{Name: "to-delete", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, p.Key)).To(Succeed())
			Expect(otg.NewRetrieve().WhereIDs(panel.OntologyID(p.Key)).
				Entry(&ontology.Resource{}).Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple panels in one call", func(ctx SpecContext) {
			a := panel.Panel{Name: "a", Parent: &parentID}
			b := panel.Panel{Name: "b", Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(panel.MatchKeys(a.Key, b.Key)).
				Entries(&[]panel.Panel{}).Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Dispatch", func() {
		create := func(ctx context.Context, root panel.Node) panel.Key {
			p := panel.Panel{Name: "test", Root: root, Parent: &parentID}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			return p.Key
		}

		It("Should apply a Rename action and persist it", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d1", []panel.Action{
				panel.NewRenameAction(panel.RenamePayload{Name: "renamed"}),
			})).To(Succeed())
			Expect(retrieve(ctx, key).Name).To(Equal("renamed"))
		})

		It("Should apply an InsertTab action and persist the tab", func(ctx SpecContext) {
			tabKey := uuid.New()
			key := create(ctx, leafNode())
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d1", []panel.Action{
				panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(tabKey), TargetLeaf: new(int32(1))}),
			})).To(Succeed())
			leaf := MustBeOk(asLeaf(retrieve(ctx, key).Root))
			Expect(leaf.Tabs).To(HaveLen(1))
			Expect(leaf.Tabs[0].Key()).To(Equal(tabKey))
		})

		It("Should apply a multi-action batch atomically", func(ctx SpecContext) {
			tabKey := uuid.New()
			key := create(ctx, leafNode())
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d1", []panel.Action{
				panel.NewRenameAction(panel.RenamePayload{Name: "batched"}),
				panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(tabKey), TargetLeaf: new(int32(1))}),
			})).To(Succeed())
			res := retrieve(ctx, key)
			Expect(res.Name).To(Equal("batched"))
			Expect(MustBeOk(asLeaf(res.Root)).Tabs).To(HaveLen(1))
		})

		It("Should apply no action when one in the batch is rejected", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d1", []panel.Action{
				panel.NewRenameAction(panel.RenamePayload{Name: "after"}),
				panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: new(int32(99))}),
			})).Error().To(MatchError(ContainSubstring("invalid node path")))
			Expect(retrieve(ctx, key).Name).To(Equal("test"))
		})

		It("Should upsert rather than duplicate when inserting a tab with an existing key", func(ctx SpecContext) {
			tabKey := uuid.New()
			key := create(ctx, leafNode(viewTab(tabKey, "selector")))
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d1", []panel.Action{
				panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(tabKey), TargetLeaf: new(int32(1))}),
			})).To(Succeed())
			tabs := MustBeOk(asLeaf(retrieve(ctx, key).Root)).Tabs
			Expect(tabs).To(HaveLen(1))
			Expect(tabs[0].Variant).To(Equal(panel.TabResource{
				TabBase:  panel.TabBase{Key: tabKey},
				Resource: tabResource(tabKey),
			}))
		})

		It("Should notify subscribers with the dispatched action on success", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			rec := &Recorder[panel.Key, panel.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "client-xyz", []panel.Action{
				panel.NewRenameAction(panel.RenamePayload{Name: "broadcast"}),
			})).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(key))
			Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
			Expect(seen[0].Actions).To(HaveLen(1))
			Expect(seen[0].Actions[0].Type).To(Equal(panel.ActionTypeRename))
		})

		It("Should assign strictly monotonic Seq across successive dispatches", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			rec := &Recorder[panel.Key, panel.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			for _, name := range []string{"a", "b", "c"} {
				Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d", []panel.Action{
					panel.NewRenameAction(panel.RenamePayload{Name: name}),
				})).To(Succeed())
			}
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(3))
			Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
			Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
		})

		It("Should not notify subscribers when the reducer rejects the action", func(ctx SpecContext) {
			key := create(ctx, leafNode())
			rec := &Recorder[panel.Key, panel.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(svc.NewWriter(tx).Dispatch(ctx, key, "d1", []panel.Action{
				panel.NewInsertTabAction(panel.InsertTabPayload{Tab: tab(uuid.New()), TargetLeaf: new(int32(99))}),
			})).Error().To(MatchError(ContainSubstring("invalid node path")))
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})
})
