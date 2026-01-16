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
	"context"
	"iter"
	"slices"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/change"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology Integration", func() {
	Describe("OntologyID", func() {
		It("Should return correct ontology ID", func() {
			key := uuid.New()
			Expect(policy.OntologyID(key)).
				To(Equal(ontology.ID{Type: policy.OntologyType, Key: key.String()}))
		})
	})
	Describe("OntologyIDs", func() {
		It("Should return correct ontology IDs", func() {
			keys := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
			Expect(policy.OntologyIDs(keys)).To(Equal([]ontology.ID{
				policy.OntologyID(keys[0]),
				policy.OntologyID(keys[1]),
				policy.OntologyID(keys[2]),
			}))
		})
	})
	Describe("OntologyIDsFromPolicies", func() {
		It("Should return correct ontology IDs", func() {
			policies := []policy.Policy{
				{Key: uuid.New(), Name: "policy1", Effect: policy.EffectAllow},
				{Key: uuid.New(), Name: "policy2", Effect: policy.EffectAllow},
			}
			Expect(policy.OntologyIDsFromPolicies(policies)).To(Equal([]ontology.ID{
				policies[0].OntologyID(),
				policies[1].OntologyID(),
			}))
		})
	})
	Describe("KeysFromOntologyIDs", func() {
		It("Should return correct keys", func() {
			keys := []uuid.UUID{uuid.New(), uuid.New()}
			ids := []ontology.ID{policy.OntologyID(keys[0]), policy.OntologyID(keys[1])}
			Expect(policy.KeysFromOntologyIDs(ids)).
				To(Equal([]uuid.UUID{keys[0], keys[1]}))
		})
		It("Should return error for invalid UUID", func() {
			ids := []ontology.ID{{Type: policy.OntologyType, Key: "invalid-uuid"}}
			Expect(policy.KeysFromOntologyIDs(ids)).Error().To(HaveOccurred())
		})
	})
	Describe("Type", func() {
		It("Should return correct ontology type", func() {
			Expect(svc.Type()).To(Equal(policy.OntologyType))
		})
	})
	Describe("Schema", func() {
		// TODO: test this more
		It("Should return a valid schema", func() {
			schema := svc.Schema()
			Expect(schema).ToNot(BeNil())
		})
	})

	Describe("RetrieveResource", func() {
		It("Should retrieve a policy as an ontology resource", func() {
			w := svc.NewWriter(tx)
			p := &policy.Policy{
				Name:   "resource-test",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, p.Key.String(), tx))
			Expect(res.ID.Key).To(Equal(p.Key.String()))
			Expect(res.Name).To(Equal(p.Name))
		})
		It("Should return error for invalid UUID", func() {
			Expect(svc.RetrieveResource(ctx, "invalid-uuid", tx)).Error().
				To(HaveOccurred())
		})
		It("Should return error for non-existent policy", func() {
			Expect(svc.RetrieveResource(ctx, uuid.New().String(), tx)).
				Error().To(HaveOccurred())
		})
	})

	Describe("OnChange", func() {
		It("Should listen for changes to policies", func() {
			changesChan := make(chan ontology.Change, 1)
			disconnect := svc.OnChange(func(_ context.Context, changes iter.Seq[ontology.Change]) {
				for change := range changes {
					changesChan <- change
				}
			})
			defer disconnect()
			w := svc.NewWriter(tx)
			p := &policy.Policy{
				Name:   "onchange-test",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			ch := <-changesChan
			Expect(ch.Variant).To(Equal(change.Set))
			Expect(ch.Key).To(Equal(p.OntologyID()))
			Expect(ch.Value.Name).To(Equal(p.Name))
			tx := db.OpenTx()
			Expect(svc.NewWriter(tx).Delete(ctx, p.Key)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())

		})
	})
	Describe("OpenNexter", func() {
		It("Should iterate over all policies", func() {
			count := 3
			keys := make([]uuid.UUID, count)
			w := svc.NewWriter(tx)
			for i := range count {
				key := uuid.New()
				p := &policy.Policy{
					Key:    key,
					Name:   "nexter-test",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
						Actions: []access.Action{access.ActionRetrieve},
					},
				}
				keys[i] = key
				Expect(w.Create(ctx, p)).To(Succeed())
			}
			Expect(tx.Commit(ctx)).To(Succeed())
			nexter, closer := MustSucceed2(svc.OpenNexter(ctx))
			Expect(slices.Collect(nexter)).To(HaveLen(3))
			Expect(closer.Close()).To(Succeed())
			tx := db.OpenTx()
			Expect(svc.NewWriter(tx).Delete(ctx, keys...)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
		})
	})
})
