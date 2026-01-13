// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"slices"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		tx gorp.Tx
		w  policy.Writer
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx, false)
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Create", func() {
		It("Should create a policy with auto-generated UUID", func() {
			p := &policy.Policy{
				Name:    "test-policy",
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create a policy with provided UUID", func() {
			key := uuid.New()
			p := &policy.Policy{
				Key:     key,
				Name:    "test-policy",
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).To(Equal(key))
		})

		It("Should create a policy with multiple objects", func() {
			p := &policy.Policy{
				Name: "multi-object-policy",
				Objects: []ontology.ID{
					{Type: "channel", Key: "ch1"},
					{Type: "channel", Key: "ch2"},
					{Type: "workspace", Key: "ws1"},
				},
				Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create a policy with ActionAll wildcard", func() {
			p := &policy.Policy{
				Name:    "wildcard-policy",
				Objects: []ontology.ID{{Type: "channel"}},
				Actions: access.AllActions,
			}
			Expect(w.Create(ctx, p)).To(Succeed())
		})

		It("Should define policy in ontology", func() {
			p := &policy.Policy{
				Name:    "ontology-test",
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())

			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(policy.OntologyID(p.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.ID.Key).To(Equal(p.Key.String()))
			Expect(res.Name).To(Equal(p.Name))
		})

		It("Should create an internal policy when allowInternal is true", func() {
			internalWriter := svc.NewWriter(tx, true)
			p := &policy.Policy{
				Name:     "internal-policy",
				Objects:  []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: true,
			}
			Expect(internalWriter.Create(ctx, p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should fail to create an internal policy when allowInternal is false", func() {
			p := &policy.Policy{
				Name:     "internal-policy",
				Objects:  []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: true,
			}
			err := w.Create(ctx, p)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot create internal policy"))
		})
	})

	Describe("Delete", func() {
		var policies []policy.Policy
		BeforeEach(func() {
			policies = []policy.Policy{
				{
					Name:    "policy-1",
					Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
				{
					Name:    "policy-2",
					Objects: []ontology.ID{{Type: "workspace", Key: "ws1"}},
					Actions: []access.Action{access.ActionUpdate},
				},
			}
			for i := range policies {
				Expect(w.Create(ctx, &policies[i])).To(Succeed())
			}
		})

		It("Should delete a single policy", func() {
			Expect(w.Delete(ctx, policies[0].Key)).To(Succeed())

			var p policy.Policy
			err := svc.NewRetrieve().WhereKeys(policies[0].Key).Entry(&p).Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should delete multiple policies", func() {
			Expect(w.Delete(ctx, policies[0].Key, policies[1].Key)).To(Succeed())

			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereKeys(policies[0].Key, policies[1].Key).
				Entries(&ps).
				Exec(ctx, tx)).To(MatchError(query.NotFound))
			Expect(ps).To(BeEmpty())
		})
	})

	Describe("SetOnRole", func() {
		var (
			r        *role.Role
			policies []policy.Policy
		)
		BeforeEach(func() {
			rw := roleSvc.NewWriter(tx, true)
			r = &role.Role{
				Name:        "test-role",
				Description: "Test role",
			}
			Expect(rw.Create(ctx, r)).To(Succeed())

			policies = []policy.Policy{
				{
					Name:    "policy-1",
					Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
				{
					Name:    "policy-2",
					Objects: []ontology.ID{{Type: "workspace", Key: "ws1"}},
					Actions: []access.Action{access.ActionUpdate},
				},
			}
			for i := range policies {
				Expect(w.Create(ctx, &policies[i])).To(Succeed())
			}
		})

		It("Should set policies for a role", func() {
			policyKeys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(w.SetOnRole(ctx, r.Key, policyKeys...)).To(Succeed())

			var children []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(role.OntologyID(r.Key)).
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(policy.OntologyType).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(HaveLen(2))
		})

		It("Should attach single policy to role", func() {
			Expect(w.SetOnRole(ctx, r.Key, policies[0].Key)).To(Succeed())

			var children []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(role.OntologyID(r.Key)).
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(policy.OntologyType).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID.Key).To(Equal(policies[0].Key.String()))
		})
	})
})

var _ = Describe("Retriever", func() {
	var (
		tx       gorp.Tx
		w        policy.Writer
		policies []policy.Policy
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx, false)
		policies = []policy.Policy{
			{
				Name:    "alpha-policy",
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			},
			{
				Name:    "beta-policy",
				Objects: []ontology.ID{{Type: "workspace", Key: "ws1"}},
				Actions: []access.Action{access.ActionDelete},
			},
			{
				Name:    "gamma-policy",
				Objects: []ontology.ID{{Type: "user", Key: "u1"}},
				Actions: []access.Action{access.ActionUpdate},
			},
		}
		for i := range policies {
			Expect(w.Create(ctx, &policies[i])).To(Succeed())
		}
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("WhereKeys", func() {
		It("Should retrieve a single policy by key", func() {
			var p policy.Policy
			Expect(svc.NewRetrieve().
				WhereKeys(policies[0].Key).
				Entry(&p).
				Exec(ctx, tx)).To(Succeed())
			Expect(p.Key).To(Equal(policies[0].Key))
			Expect(p.Name).To(Equal(policies[0].Name))
		})

		It("Should retrieve multiple policies by keys", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereKeys(policies[0].Key, policies[1].Key).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})

		It("Should return empty when key not found", func() {
			var p policy.Policy
			err := svc.NewRetrieve().
				WhereKeys(uuid.New()).
				Entry(&p).
				Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})
	})

	Describe("WhereNames", func() {
		It("Should retrieve a policy by name", func() {
			var p policy.Policy
			Expect(svc.NewRetrieve().
				WhereNames("alpha-policy").
				Entry(&p).
				Exec(ctx, tx)).To(Succeed())
			Expect(p.Name).To(Equal("alpha-policy"))
		})

		It("Should retrieve multiple policies by names", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereNames("alpha-policy", "beta-policy").
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})
	})

	Describe("WhereSubjects", func() {
		var (
			r        *role.Role
			subject1 ontology.ID
			subject2 ontology.ID
		)
		BeforeEach(func() {
			rw := roleSvc.NewWriter(tx, true)
			r = &role.Role{
				Name:        "test-role",
				Description: "Test role for subject queries",
			}
			Expect(rw.Create(ctx, r)).To(Succeed())
			Expect(w.SetOnRole(ctx, r.Key, policies[0].Key, policies[1].Key)).To(Succeed())

			subject1 = ontology.ID{Type: "user", Key: uuid.New().String()}
			subject2 = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject1)).To(Succeed())
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject2)).To(Succeed())
			Expect(rw.AssignRole(ctx, subject1, r.Key)).To(Succeed())
		})

		It("Should retrieve policies for a subject via role", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereSubjects(subject1).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
			keys := []uuid.UUID{ps[0].Key, ps[1].Key}
			Expect(keys).To(ContainElements(policies[0].Key, policies[1].Key))
		})

		It("Should return empty when subject has no roles", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereSubjects(subject2).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(BeEmpty())
		})

		It("Should retrieve policies for multiple subjects", func() {
			rw := roleSvc.NewWriter(tx, true)
			r2 := &role.Role{
				Name:        "test-role-2",
				Description: "Second test role",
			}
			Expect(rw.Create(ctx, r2)).To(Succeed())
			Expect(w.SetOnRole(ctx, r2.Key, policies[2].Key)).To(Succeed())
			Expect(rw.AssignRole(ctx, subject2, r2.Key)).To(Succeed())

			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereSubjects(subject1, subject2).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(3))
		})
	})

	Describe("Limit and Offset", func() {
		It("Should limit results", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Limit(2).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})

		It("Should apply offset", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Offset(1).
				Limit(2).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})

		It("Should handle offset beyond results", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Offset(10).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(BeEmpty())
		})
	})

	Describe("WhereInternal", func() {
		var internalPolicy, regularPolicy policy.Policy
		BeforeEach(func() {
			internalWriter := svc.NewWriter(tx, true)
			internalPolicy = policy.Policy{
				Name:     "internal-policy",
				Objects:  []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: true,
			}
			regularPolicy = policy.Policy{
				Name:     "regular-policy",
				Objects:  []ontology.ID{{Type: "channel", Key: "ch2"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: false,
			}
			Expect(internalWriter.Create(ctx, &internalPolicy)).To(Succeed())
			Expect(internalWriter.Create(ctx, &regularPolicy)).To(Succeed())
		})

		It("Should retrieve only internal policies", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereInternal(true).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			for _, p := range ps {
				Expect(p.Internal).To(BeTrue())
			}
			Expect(ps).To(ContainElement(HaveField("Key", internalPolicy.Key)))
		})

		It("Should retrieve only non-internal policies", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereInternal(false).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			for _, p := range ps {
				Expect(p.Internal).To(BeFalse())
			}
			Expect(ps).To(ContainElement(HaveField("Key", regularPolicy.Key)))
		})
	})
})

var _ = Describe("Ontology Integration", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Type", func() {
		It("Should return correct ontology type", func() {
			Expect(svc.Type()).To(Equal(policy.OntologyType))
		})
	})

	Describe("Schema", func() {
		It("Should return a valid schema", func() {
			schema := svc.Schema()
			Expect(schema).ToNot(BeNil())
		})
	})

	Describe("RetrieveResource", func() {
		It("Should retrieve a policy as an ontology resource", func() {
			w := svc.NewWriter(tx, false)
			p := &policy.Policy{
				Name:    "resource-test",
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())

			res, err := svc.RetrieveResource(ctx, p.Key.String(), tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.ID.Key).To(Equal(p.Key.String()))
			Expect(res.Name).To(Equal(p.Name))
		})

		It("Should return error for invalid UUID", func() {
			_, err := svc.RetrieveResource(ctx, "invalid-uuid", tx)
			Expect(err).To(HaveOccurred())
		})

		It("Should return error for non-existent policy", func() {
			_, err := svc.RetrieveResource(ctx, uuid.New().String(), tx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("OpenNexter", func() {
		It("Should iterate over all policies", func() {
			w := svc.NewWriter(tx, false)
			for i := 0; i < 3; i++ {
				p := &policy.Policy{
					Name:    "nexter-test",
					Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(w.Create(ctx, p)).To(Succeed())
			}
			Expect(tx.Commit(ctx)).To(Succeed())

			nexter, closer := MustSucceed2(svc.OpenNexter(ctx))
			defer func() { Expect(closer.Close()).To(Succeed()) }()

			count := len(slices.Collect(nexter))
			Expect(count).To(BeNumerically(">=", 3))
		})
	})
})
