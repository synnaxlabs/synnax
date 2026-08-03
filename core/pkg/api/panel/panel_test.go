// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// createPanel persists a panel parented to the suite parent group and returns it
// with its key populated. Writes commit immediately (nil tx) so the api enforcer
// can observe the new ontology resource.
func createPanel(ctx context.Context, name string) panel.Panel {
	p := panel.Panel{Name: name, Parent: &parentID}
	Expect(panelSvc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
	return p
}

func hasParent(ctx context.Context, parent ontology.ID, key panel.Key) bool {
	return MustSucceed(otg.RelationshipExists(ctx, nil, ontology.Relationship{
		From: parent,
		Type: ontology.RelationshipTypeParentOf,
		To:   panel.OntologyID(key),
	}))
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It("Should reject the request when the subject has no create policy", func(ctx SpecContext) {
			u := newUser(ctx)
			p := panel.Panel{Key: uuid.New(), Name: "no-policy"}
			Expect(apiSvc.Create(authedCtx(ctx, u), nil, CreateRequest{
				Panels: []panel.Panel{p},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should create the panels under the provided parent", func(ctx SpecContext) {
			u := newUser(ctx)
			p := panel.Panel{Key: uuid.New(), Name: "with-parent", Parent: &parentID}
			grant(ctx, u.OntologyID(), access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypePanel}, parentID)
			res := MustSucceed(apiSvc.Create(authedCtx(ctx, u), nil, CreateRequest{
				Panels: []panel.Panel{p},
			}))
			Expect(res.Panels).To(HaveLen(1))
			Expect(res.Panels[0].Key).To(Equal(p.Key))
			var got panel.Panel
			Expect(panelSvc.NewRetrieve().Where(panel.MatchKeys(p.Key)).
				Entry(&got).Exec(ctx, nil)).To(Succeed())
			Expect(got.Name).To(Equal("with-parent"))
			Expect(hasParent(ctx, parentID, p.Key)).To(BeTrue())
		})

		It("Should parent a parent-less panel to the creating user as a draft", func(ctx SpecContext) {
			u := newUser(ctx)
			p := panel.Panel{Key: uuid.New(), Name: "draft"}
			grant(ctx, u.OntologyID(), access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypePanel})
			Expect(apiSvc.Create(authedCtx(ctx, u), nil, CreateRequest{
				Panels: []panel.Panel{p},
			})).Error().To(Succeed())
			Expect(hasParent(ctx, u.OntologyID(), p.Key)).To(BeTrue())
		})
	})

	Describe("Retrieve", func() {
		It("Should return the panels matching the requested keys", func(ctx SpecContext) {
			p := createPanel(ctx, "retrievable")
			grant(ctx, author.OntologyID(), access.ActionRetrieve, p.OntologyID())
			res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys: []panel.Key{p.Key},
			}))
			Expect(res.Panels).To(HaveLen(1))
			Expect(res.Panels[0].Key).To(Equal(p.Key))
			Expect(res.Panels[0].Name).To(Equal("retrievable"))
		})

		It("Should reject the request when the subject cannot retrieve a matched panel", func(ctx SpecContext) {
			p := createPanel(ctx, "forbidden")
			Expect(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys: []panel.Key{p.Key},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should omit missing keys from a multi-key retrieve instead of failing", func(ctx SpecContext) {
			p := createPanel(ctx, "partial-survivor")
			missing := uuid.New()
			grant(ctx, author.OntologyID(), access.ActionRetrieve,
				panel.OntologyID(p.Key), panel.OntologyID(missing))
			res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys:                []panel.Key{p.Key, missing},
				IgnoreNotFoundError: true,
			}))
			Expect(res.Panels).To(HaveLen(1))
			Expect(res.Panels[0].Key).To(Equal(p.Key))
		})

		It("Should return an empty result when every requested key is missing", func(ctx SpecContext) {
			res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys:                []panel.Key{uuid.New()},
				IgnoreNotFoundError: true,
			}))
			Expect(res.Panels).To(BeEmpty())
		})

		It("Should honor the limit and offset bounds", func(ctx SpecContext) {
			a := createPanel(ctx, "page-a")
			b := createPanel(ctx, "page-b")
			grant(ctx, author.OntologyID(), access.ActionRetrieve,
				a.OntologyID(), b.OntologyID())
			res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys:   []panel.Key{a.Key, b.Key},
				Limit:  1,
				Offset: 1,
			}))
			Expect(res.Panels).To(HaveLen(1))
		})
	})

	Describe("Dispatch", func() {
		It("Should reject the request when the subject has no update policy", func(ctx SpecContext) {
			p := createPanel(ctx, "dispatch-denied")
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), nil, DispatchRequest{
				Key:         p.Key,
				DispatchKey: "sess-1",
				Actions:     []panel.Action{panel.NewRenameAction(panel.RenamePayload{Name: "x"})},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should apply the action sequence to the target panel", func(ctx SpecContext) {
			p := createPanel(ctx, "dispatch-ok")
			grant(ctx, author.OntologyID(), access.ActionUpdate, p.OntologyID())
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), nil, DispatchRequest{
				Key:         p.Key,
				DispatchKey: "sess-1",
				Actions: []panel.Action{
					panel.NewRenameAction(panel.RenamePayload{Name: "renamed"}),
				},
			})).Error().To(Succeed())
			var got panel.Panel
			Expect(panelSvc.NewRetrieve().Where(panel.MatchKeys(p.Key)).
				Entry(&got).Exec(ctx, nil)).To(Succeed())
			Expect(got.Name).To(Equal("renamed"))
		})
	})

	Describe("Delete", func() {
		It("Should reject the request when the subject has no delete policy", func(ctx SpecContext) {
			p := createPanel(ctx, "delete-denied")
			Expect(apiSvc.Delete(authedCtx(ctx, author), nil, DeleteRequest{
				Keys: []panel.Key{p.Key},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should delete the panels with the given keys", func(ctx SpecContext) {
			p := createPanel(ctx, "deletable")
			grant(ctx, author.OntologyID(), access.ActionDelete, p.OntologyID())
			Expect(apiSvc.Delete(authedCtx(ctx, author), nil, DeleteRequest{
				Keys: []panel.Key{p.Key},
			})).Error().To(Succeed())
			var got panel.Panel
			Expect(panelSvc.NewRetrieve().Where(panel.MatchKeys(p.Key)).
				Entry(&got).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
	})
})
