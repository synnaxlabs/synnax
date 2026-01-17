// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Writer", func() {
	var (
		tx gorp.Tx
		w  role.Writer
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Create", func() {
		It("Should create a role with auto-generated UUID", func() {
			r := &role.Role{
				Name:        "admin",
				Description: "Administrator role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
			Expect(r.Name).To(Equal("admin"))
			Expect(r.Description).To(Equal("Administrator role"))
		})

		It("Should create a role with provided UUID", func() {
			key := uuid.New()
			r := &role.Role{
				Key:         key,
				Name:        "viewer",
				Description: "View-only role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).To(Equal(key))
			Expect(r.Name).To(Equal("viewer"))
			Expect(r.Description).To(Equal("View-only role"))
		})

		It("Should define role in ontology", func() {
			r := &role.Role{
				Name:        "engineer",
				Description: "Engineering role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(r.OntologyID()).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.ID.Key).To(Equal(r.Key.String()))
			Expect(res.Name).To(Equal(r.Name))
		})

		It("Should create relationship to Users group", func() {
			r := &role.Role{
				Name:        "operator",
				Description: "Operator role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())

			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(r.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes("group").
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(HaveLen(1))
			Expect(parents[0].ID.Key).To(Equal(svc.UsersGroup().OntologyID().Key))
		})
	})

	Describe("Delete", func() {
		var roles []role.Role
		BeforeEach(func() {
			roles = []role.Role{
				{Name: "role-1", Description: "First role"},
				{Name: "role-2", Description: "Second role"},
			}
			for i := range roles {
				Expect(w.Create(ctx, &roles[i])).To(Succeed())
			}
		})

		It("Should delete a role", func() {
			Expect(w.Delete(ctx, roles[0].Key)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(roles[0].Key).
				Entry(&role.Role{}).Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("AssignRole", func() {
		var (
			r       *role.Role
			subject ontology.ID
		)
		BeforeEach(func() {
			r = &role.Role{
				Name:        "test-role",
				Description: "Test role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		It("Should assign role to subject", func() {
			Expect(w.AssignRole(ctx, subject, r.Key)).To(Succeed())
			var parent ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(subject).
				ExcludeFieldData(true).
				TraverseTo(ontology.ParentsTraverser).
				ExcludeFieldData(true).
				WhereTypes(role.OntologyType).
				Entry(&parent).
				Exec(ctx, tx)).To(Succeed())
			Expect(parent.ID.Key).To(Equal(r.Key.String()))
		})

		It("Should be idempotent", func() {
			Expect(w.AssignRole(ctx, subject, r.Key)).To(Succeed())
			Expect(w.AssignRole(ctx, subject, r.Key)).To(Succeed())

			var parent ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(subject).
				ExcludeFieldData(true).
				TraverseTo(ontology.ParentsTraverser).
				ExcludeFieldData(true).
				WhereTypes(role.OntologyType).
				Entry(&parent).
				Exec(ctx, tx)).To(Succeed())
			Expect(parent.ID.Key).To(Equal(r.Key.String()))
		})
	})

	Describe("UnassignRole", func() {
		var (
			r       *role.Role
			subject ontology.ID
		)
		BeforeEach(func() {
			r = &role.Role{
				Name:        "test-role",
				Description: "Test role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
			Expect(w.AssignRole(ctx, subject, r.Key)).To(Succeed())
		})

		It("Should unassign role from subject", func() {
			Expect(w.UnassignRole(ctx, subject, r.Key)).To(Succeed())

			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(subject).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(role.OntologyType).
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(BeEmpty())
		})

		It("Should be idempotent", func() {
			Expect(w.UnassignRole(ctx, subject, r.Key)).To(Succeed())
			Expect(w.UnassignRole(ctx, subject, r.Key)).To(Succeed())

			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(subject).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(role.OntologyType).
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(BeEmpty())
		})
	})

})
