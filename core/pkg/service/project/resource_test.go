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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each project by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := project.Project{Name: "resource-project"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{p.ResourceID()})).
			To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeProject, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid project key "pump"`)),
		))
	})

	It("Should leave out an ID that names no project", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := project.Project{Name: "listed-project"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		missing := resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, p.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should list each project by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := project.Project{Name: "listed-project"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: p.ResourceID(), Name: p.Name},
		))
	})
})

var _ = Describe("Resource", func() {
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
	Describe("Search", func() {
		It(
			"Should return the projects whose name matches the term",
			func(ctx SpecContext) {
				a := project.Project{Key: uuid.New(), Name: "turbopump test"}
				b := project.Project{Key: uuid.New(), Name: "turbopump"}
				Expect(writer.Create(ctx, &a)).To(Succeed())
				Expect(writer.Create(ctx, &b)).To(Succeed())
				q := search.Query[resource.ID]{Term: "turbopump", Limit: 10}
				Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b.ResourceID(), a.ResourceID()}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b.ResourceID()}))
			},
		)
	})
})
