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
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/uuid"
)

var _ = Describe("Writer", func() {
	var (
		tx gorp.Tx
		w  role.Writer
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx, true)
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
		})

		It("Should define role in ontology", func() {
			r := &role.Role{
				Name:        "engineer",
				Description: "Engineering role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(role.OntologyID(r.Key)).
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
				WhereIDs(role.OntologyID(r.Key)).
				TraverseTo(ontology.Parents).
				WhereTypes("group").
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).ToNot(BeEmpty())
		})

		It("Should create an internal role when allowInternal is true", func() {
			r := &role.Role{
				Name:        "builtin-role",
				Description: "A builtin role",
				Internal:    true,
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should fail to create an internal role when allowInternal is false", func() {
			restrictedWriter := svc.NewWriter(tx, false)
			r := &role.Role{
				Name:        "builtin-role",
				Description: "A builtin role",
				Internal:    true,
			}
			err := restrictedWriter.Create(ctx, r)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot create internal role"))
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

			var r role.Role
			err := svc.NewRetrieve().WhereKeys(roles[0].Key).Entry(&r).Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should delete an internal role when allowInternal is true", func() {
			r := &role.Role{
				Name:        "internal-to-delete",
				Description: "Internal role to delete",
				Internal:    true,
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(w.Delete(ctx, r.Key)).To(Succeed())

			var retrieved role.Role
			err := svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieved).Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should fail to delete an internal role when allowInternal is false", func() {
			r := &role.Role{
				Name:        "internal-protected",
				Description: "Internal role that cannot be deleted",
				Internal:    true,
			}
			Expect(w.Create(ctx, r)).To(Succeed())

			restrictedWriter := svc.NewWriter(tx, false)
			err := restrictedWriter.Delete(ctx, r.Key)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot delete builtin role"))
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

			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(subject).
				ExcludeFieldData(true).
				TraverseTo(ontology.Parents).
				ExcludeFieldData(true).
				WhereTypes(role.OntologyType).
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(HaveLen(1))
			Expect(parents[0].ID.Key).To(Equal(r.Key.String()))
		})

		It("Should be idempotent", func() {
			Expect(w.AssignRole(ctx, subject, r.Key)).To(Succeed())
			Expect(w.AssignRole(ctx, subject, r.Key)).To(Succeed())

			var parents []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(subject).
				ExcludeFieldData(true).
				TraverseTo(ontology.Parents).
				ExcludeFieldData(true).
				WhereTypes(role.OntologyType).
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(HaveLen(1))
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
				TraverseTo(ontology.Parents).
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
				TraverseTo(ontology.Parents).
				WhereTypes(role.OntologyType).
				Entries(&parents).
				Exec(ctx, tx)).To(Succeed())
			Expect(parents).To(BeEmpty())
		})
	})

})

var _ = Describe("Retrieve", func() {
	var (
		tx    gorp.Tx
		w     role.Writer
		roles []role.Role
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx, true)
		roles = []role.Role{
			{Name: "admin", Description: "Administrator"},
			{Name: "engineer", Description: "Engineer"},
			{Name: "viewer", Description: "Viewer"},
		}
		for i := range roles {
			Expect(w.Create(ctx, &roles[i])).To(Succeed())
		}
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("WhereKeys", func() {
		It("Should retrieve a single role by key", func() {
			var r role.Role
			Expect(svc.NewRetrieve().
				WhereKeys(roles[0].Key).
				Entry(&r).
				Exec(ctx, tx)).To(Succeed())
			Expect(r.Key).To(Equal(roles[0].Key))
			Expect(r.Name).To(Equal(roles[0].Name))
		})

		It("Should retrieve multiple roles by keys", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				WhereKeys(roles[0].Key, roles[1].Key).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(HaveLen(2))
		})

		It("Should return error when key not found", func() {
			var r role.Role
			err := svc.NewRetrieve().
				WhereKeys(uuid.New()).
				Entry(&r).
				Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})
	})

	Describe("WhereName", func() {
		It("Should retrieve a role by name", func() {
			var r role.Role
			Expect(svc.NewRetrieve().
				WhereName("admin").
				Entry(&r).
				Exec(ctx, tx)).To(Succeed())
			Expect(r.Name).To(Equal("admin"))
		})

		It("Should return error when name not found", func() {
			var r role.Role
			err := svc.NewRetrieve().
				WhereName("nonexistent").
				Entry(&r).
				Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})
	})

	Describe("Limit and Offset", func() {
		It("Should limit results", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				Limit(2).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(HaveLen(2))
		})

		It("Should apply offset", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				Offset(1).
				Limit(2).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(HaveLen(2))
		})

		It("Should handle offset beyond results", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				Offset(10).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(BeEmpty())
		})
	})

	Describe("WhereInternal", func() {
		var internalRole, regularRole role.Role
		BeforeEach(func() {
			internalRole = role.Role{
				Name:        "internal-role",
				Description: "An internal role",
				Internal:    true,
			}
			regularRole = role.Role{
				Name:        "regular-role",
				Description: "A regular role",
				Internal:    false,
			}
			Expect(w.Create(ctx, &internalRole)).To(Succeed())
			Expect(w.Create(ctx, &regularRole)).To(Succeed())
		})

		It("Should retrieve only internal roles", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				WhereInternal(true).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			for _, r := range rs {
				Expect(r.Internal).To(BeTrue())
			}
			Expect(rs).To(ContainElement(HaveField("Key", internalRole.Key)))
		})

		It("Should retrieve only non-internal roles", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				WhereInternal(false).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			for _, r := range rs {
				Expect(r.Internal).To(BeFalse())
			}
			Expect(rs).To(ContainElement(HaveField("Key", regularRole.Key)))
		})
	})
})

var _ = Describe("Ontology Integration", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Type", func() {
		It("Should return correct ontology type", func() {
			Expect(svc.Type()).To(Equal(role.OntologyType))
		})
	})

	Describe("Schema", func() {
		It("Should return a valid schema", func() {
			schema := svc.Schema()
			Expect(schema).ToNot(BeNil())
		})
	})

	Describe("RetrieveResource", func() {
		It("Should retrieve a role as an ontology resource", func() {
			w := svc.NewWriter(tx, true)
			r := &role.Role{
				Name:        "resource-test",
				Description: "Resource test role",
			}
			Expect(w.Create(ctx, r)).To(Succeed())

			res, err := svc.RetrieveResource(ctx, r.Key.String(), tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.ID.Key).To(Equal(r.Key.String()))
			Expect(res.Name).To(Equal(r.Name))
		})

		It("Should return error for invalid UUID", func() {
			_, err := svc.RetrieveResource(ctx, "invalid-uuid", tx)
			Expect(err).To(HaveOccurred())
		})

		It("Should return error for non-existent role", func() {
			_, err := svc.RetrieveResource(ctx, uuid.New().String(), tx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("OpenNexter", func() {
		It("Should iterate over all roles", func() {
			w := svc.NewWriter(tx, true)
			for i := 0; i < 3; i++ {
				r := &role.Role{
					Name:        "nexter-test",
					Description: "Nexter test role",
				}
				Expect(w.Create(ctx, r)).To(Succeed())
			}
			Expect(tx.Commit(ctx)).To(Succeed())

			nexter, closer := MustSucceed2(svc.OpenNexter(ctx))
			defer func() { Expect(closer.Close()).To(Succeed()) }()

			count := len(slices.Collect(nexter))
			Expect(count).To(BeNumerically(">=", 3))
		})
	})
})
