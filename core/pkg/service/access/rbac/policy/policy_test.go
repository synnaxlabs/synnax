// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Writer", func() {
	var (
		tx gorp.Tx
		w  policy.Writer
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Create", func() {
		It("Should create a policy with auto-generated UUID", func() {
			p := &policy.Policy{
				Name:    "test-policy",
				Effect:  policy.EffectAllow,
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
				Effect:  policy.EffectAllow,
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).To(Equal(key))
		})

		It("Should create a policy with multiple objects", func() {
			p := &policy.Policy{
				Name:   "multi-object-policy",
				Effect: policy.EffectAllow,
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
				Effect:  policy.EffectAllow,
				Objects: []ontology.ID{{Type: "channel"}},
				Actions: []access.Action{access.ActionAll},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
		})

		It("Should create a policy with EffectDeny", func() {
			p := &policy.Policy{
				Name:    "deny-policy",
				Effect:  policy.EffectDeny,
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionDelete},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
		})

		It("Should define policy in ontology", func() {
			p := &policy.Policy{
				Name:    "ontology-test",
				Effect:  policy.EffectAllow,
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
	})

	Describe("Delete", func() {
		var policies []policy.Policy
		BeforeEach(func() {
			policies = []policy.Policy{
				{
					Name:    "policy-1",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
				{
					Name:    "policy-2",
					Effect:  policy.EffectAllow,
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
})

var _ = Describe("Retriever", func() {
	var (
		tx       gorp.Tx
		w        policy.Writer
		policies []policy.Policy
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
		policies = []policy.Policy{
			{
				Name:    "alpha-policy",
				Effect:  policy.EffectAllow,
				Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			},
			{
				Name:    "beta-policy",
				Effect:  policy.EffectDeny,
				Objects: []ontology.ID{{Type: "workspace", Key: "ws1"}},
				Actions: []access.Action{access.ActionDelete},
			},
			{
				Name:    "gamma-policy",
				Effect:  policy.EffectAllow,
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
			w := svc.NewWriter(tx)
			p := &policy.Policy{
				Name:    "resource-test",
				Effect:  policy.EffectAllow,
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
			w := svc.NewWriter(tx)
			for i := 0; i < 3; i++ {
				p := &policy.Policy{
					Name:    "nexter-test",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(w.Create(ctx, p)).To(Succeed())
			}
			Expect(tx.Commit(ctx)).To(Succeed())

			nexter, err := svc.OpenNexter()
			Expect(err).ToNot(HaveOccurred())
			defer func() { Expect(nexter.Close()).To(Succeed()) }()

			count := 0
			for {
				_, ok := nexter.Next(ctx)
				if !ok {
					break
				}
				count++
			}
			Expect(count).To(BeNumerically(">=", 3))
		})
	})
})
