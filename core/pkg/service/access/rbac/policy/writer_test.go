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

var _ = Describe("Writer", func() {
	var w policy.Writer
	BeforeEach(func() {
		w = svc.NewWriter(tx)
	})
	Describe("Create", func() {
		It("Should create a policy with auto-generated UUID", func() {
			p := &policy.Policy{
				Name:   "test-policy",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create a policy with provided UUID", func() {
			key := uuid.New()
			p := &policy.Policy{
				Key:    key,
				Name:   "test-policy",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).To(Equal(key))
		})
		It("Should define policy in ontology", func() {
			p := &policy.Policy{
				Name:   "ontology-test",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(p.OntologyID()).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.ID.Key).To(Equal(p.Key.String()))
			Expect(res.Name).To(Equal(p.Name))
		})
	})
	Describe("Delete", func() {
		var policies []policy.Policy
		BeforeEach(func() {
			policies = []policy.Policy{
				{
					Name:   "policy-1",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
						Actions: []access.Action{access.ActionRetrieve},
					},
				},
				{
					Name:   "policy-2",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						IDs:     []ontology.ID{{Type: "workspace", Key: "ws1"}},
						Actions: []access.Action{access.ActionUpdate},
					},
				},
			}
			for i := range policies {
				Expect(w.Create(ctx, &policies[i])).To(Succeed())
			}
		})

		It("Should delete a single policy", func() {
			Expect(w.Delete(ctx, policies[0].Key)).To(Succeed())
			Expect(svc.
				NewRetrieve().
				WhereKeys(policies[0].Key).
				Entry(&policy.Policy{}).
				Exec(ctx, tx),
			).To(MatchError(query.NotFound))
		})
		It("Should delete multiple policies", func() {
			Expect(w.Delete(ctx, policies[0].Key, policies[1].Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				WhereKeys(policies[0].Key, policies[1].Key).
				Entries(&[]policy.Policy{}).
				Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})
	Describe("SetOnRole", func() {
		var (
			r        *role.Role
			policies []policy.Policy
		)
		BeforeEach(func() {
			rw := roleSvc.NewWriter(tx)
			r = &role.Role{
				Name:        "test-role",
				Description: "Test role",
			}
			Expect(rw.Create(ctx, r)).To(Succeed())
			policies = []policy.Policy{
				{
					Name:   "policy-1",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
						Actions: []access.Action{access.ActionRetrieve},
					},
				},
				{
					Name:   "policy-2",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						IDs:     []ontology.ID{{Type: "workspace", Key: "ws1"}},
						Actions: []access.Action{access.ActionUpdate},
					},
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
				TraverseTo(ontology.Children).
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
				TraverseTo(ontology.Children).
				WhereTypes(policy.OntologyType).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID.Key).To(Equal(policies[0].Key.String()))
		})
	})
})
