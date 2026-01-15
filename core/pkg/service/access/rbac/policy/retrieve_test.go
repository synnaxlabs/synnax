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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Retrieve", func() {
	var (
		w        policy.Writer
		policies []policy.Policy
	)
	BeforeEach(func() {
		w = svc.NewWriter(tx)
		policies = []policy.Policy{
			{
				Name:   "alpha-policy",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
			},
			{
				Name:   "beta-policy",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "workspace", Key: "ws1"}},
					Actions: []access.Action{access.ActionDelete},
				},
			},
			{
				Name:   "gamma-policy",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "user", Key: "u1"}},
					Actions: []access.Action{access.ActionUpdate},
				},
			},
		}
		for i := range policies {
			Expect(w.Create(ctx, &policies[i])).To(Succeed())
		}
	})

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
			Expect(svc.NewRetrieve().
				WhereKeys(uuid.New()).
				Entry(&p).
				Exec(ctx, tx)).To(MatchError(query.NotFound))
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
		It("Should return empty when name not found", func() {
			var p policy.Policy
			Expect(svc.NewRetrieve().
				WhereNames("nonexistent-policy").
				Entry(&p).
				Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
		It("Should return empty when multiple names not found", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereNames("nonexistent-policy", "nonexistent-policy-2").
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(BeEmpty())
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
			Expect(w.SetOnRole(ctx, r.Key, policies[0].Key, policies[1].Key)).
				To(Succeed())

			subject1 = ontology.ID{Type: "user", Key: uuid.New().String()}
			subject2 = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject1)).To(Succeed())
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject2)).To(Succeed())
			Expect(rw.AssignRole(ctx, subject1, r.Key)).To(Succeed())
		})

		It("Should retrieve policies for a subject via role", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereSubject(subject1).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
			keys := []uuid.UUID{ps[0].Key, ps[1].Key}
			Expect(keys).To(ContainElements(policies[0].Key, policies[1].Key))
		})
		It("Should return empty when subject has no roles", func() {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				WhereSubject(subject2).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(BeEmpty())
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
})
