// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"context"
	"iter"
	"slices"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ontology", func() {
	Describe("ResourceID", func() {
		It(
			"Should build a resource ID with the project resource type and the key",
			func() {
				key := uuid.New()
				id := project.ResourceID(key)
				Expect(id.Type).To(Equal(resource.TypeProject))
				Expect(id.Key).To(Equal(key.String()))
			},
		)
	})
	Describe("ResourceIDs", func() {
		It("Should map a slice of keys to resource IDs", func() {
			a, b := uuid.New(), uuid.New()
			Expect(project.ResourceIDs([]project.Key{a, b})).
				To(ConsistOf(project.ResourceID(a), project.ResourceID(b)))
		})
	})
	Describe("ResourceIDsFromProjects", func() {
		It("Should map a slice of projects to resource IDs", func() {
			a := project.Project{Key: uuid.New(), Name: "a"}
			b := project.Project{Key: uuid.New(), Name: "b"}
			Expect(project.ResourceIDsFromProjects([]project.Project{a, b})).
				To(ConsistOf(a.ResourceID(), b.ResourceID()))
		})
	})
	Describe("Project.ResourceID", func() {
		It("Should return a resource ID populated from the project's key", func() {
			p := project.Project{Key: uuid.New(), Name: "x"}
			Expect(p.ResourceID()).To(Equal(p.ResourceID()))
		})
	})
	Describe("KeyFromResourceID", func() {
		It("Should return the key of a project ID", func() {
			key := uuid.New()
			Expect(project.KeyFromResourceID(project.ResourceID(key))).To(Equal(key))
		})
		DescribeTable("Should reject an ID that is not a project",
			func(id resource.ID, msg string) {
				Expect(project.KeyFromResourceID(id)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("a group",
				resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()},
				`must be a project, got "group"`),
			Entry("a zero ID", resource.ID{}, `must be a project, got ""`),
			Entry("a project whose key is not a UUID",
				resource.ID{Type: resource.TypeProject, Key: "not-a-uuid"},
				`invalid project key "not-a-uuid"`),
			Entry("a project with an empty key",
				resource.ID{Type: resource.TypeProject},
				`invalid project key ""`),
		)
	})
	Describe("Type", func() {
		It("Should return the project resource type", func() {
			Expect(svc.Type()).To(Equal(resource.TypeProject))
		})
	})
	Describe("RetrieveResource", func() {
		It("Should retrieve a project as an ontology resource", func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			p := project.Project{Key: uuid.New(), Name: "resource"}
			Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
			resource := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(resource.ID).To(Equal(p.ResourceID()))
			Expect(resource.Name).To(Equal("resource"))
		})
		It(
			"Should return an error when the key is not a valid UUID",
			func(ctx SpecContext) {
				Expect(svc.RetrieveResource(ctx, "not-a-uuid", nil)).Error().
					To(MatchError(ContainSubstring("invalid uuid")))
			},
		)
		It(
			"Should return query.ErrNotFound when no project has the given key",
			func(ctx SpecContext) {
				Expect(svc.RetrieveResource(ctx, uuid.New().String(), nil)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("OnChange", func() {
		It(
			"Should publish set and delete changes translated into ontology changes",
			func(ctx SpecContext) {
				var (
					mu      sync.Mutex
					changes []ontology.Change
				)
				disconnect := svc.OnChange(
					func(_ context.Context, it iter.Seq[ontology.Change]) {
						mu.Lock()
						defer mu.Unlock()
						changes = append(changes, slices.Collect(it)...)
					},
				)
				DeferCleanup(disconnect)

				p := project.Project{Key: uuid.New(), Name: "observed"}
				Expect(writer.Create(ctx, &p)).To(Succeed())
				expectedID := p.ResourceID().String()

				Eventually(func(g Gomega) {
					mu.Lock()
					defer mu.Unlock()
					idx := slices.IndexFunc(changes, func(c ontology.Change) bool {
						return c.Key == expectedID && c.Variant == xchange.VariantSet
					})
					g.Expect(idx).ToNot(Equal(-1))
					g.Expect(changes[idx].Value.Name).To(Equal("observed"))
				}).Should(Succeed())

				Expect(writer.Delete(ctx, p.Key)).To(Succeed())

				Eventually(func(g Gomega) {
					mu.Lock()
					defer mu.Unlock()
					g.Expect(changes).To(ContainElement(SatisfyAll(
						HaveField("Key", expectedID),
						HaveField("Variant", xchange.VariantDelete),
					)))
				}).Should(Succeed())
			},
		)
	})
	Describe("Search", func() {
		It(
			"Should return the projects whose name matches the term",
			func(ctx SpecContext) {
				a := project.Project{Key: uuid.New(), Name: "turbopump test"}
				b := project.Project{Key: uuid.New(), Name: "turbopump"}
				Expect(writer.Create(ctx, &a)).To(Succeed())
				Expect(writer.Create(ctx, &b)).To(Succeed())
				q := search.Query[resource.ID]{Term: "turbopump", Limit: 10}
				Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b.ResourceID(), a.ResourceID()}))
				q.Limit = 1
				Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b.ResourceID()}))
			},
		)
	})
})
