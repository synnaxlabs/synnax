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
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should validate properly shaped configs", func() {
				cfg := policy.ServiceConfig{DB: db, Ontology: otg}
				Expect(cfg.Validate()).To(Succeed())
			})
			It("Should validate configs with optional Signals field", func() {
				cfg := policy.ServiceConfig{DB: db, Ontology: otg, Signals: nil}
				Expect(cfg.Validate()).To(Succeed())
			})
			It("Should fail to validate configs missing the DB field", func() {
				cfg := policy.ServiceConfig{Ontology: otg}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("Should fail to validate configs missing the Ontology field", func() {
				cfg := policy.ServiceConfig{DB: db}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})
		Describe("Override", func() {
			It("Should override fields with non-nil values", func() {
				cfg := policy.ServiceConfig{}
				other := policy.ServiceConfig{DB: db, Ontology: otg}
				Expect(cfg.Override(other)).To(Equal(policy.ServiceConfig{
					DB:       db,
					Ontology: otg,
				}))
			})
			It("Should not override fields with nil values", func() {
				cfg := policy.ServiceConfig{DB: db, Ontology: otg}
				other := policy.ServiceConfig{}
				result := cfg.Override(other)
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
			})
		})
	})

	Describe("Open", func() {
		It("Should open a service with valid configuration", func() {
			s := MustSucceed(policy.OpenService(ctx, policy.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			Expect(s.Close()).To(Succeed())
		})
		It("Should fail to open a service with an invalid configuration", func() {
			Expect(policy.OpenService(ctx)).Error().To(HaveOccurred())
		})
		It("Should fail to open with missing DB", func() {
			Expect(policy.OpenService(ctx, policy.ServiceConfig{Ontology: otg})).
				Error().To(HaveOccurred())
		})
		It("Should fail to open with missing Ontology", func() {
			Expect(policy.OpenService(ctx, policy.ServiceConfig{DB: db})).
				Error().To(HaveOccurred())
		})
	})

	Describe("SystemPolicies", func() {
		It("Should return empty slice when no system policies are added", func() {
			Expect(svc.SystemPolicies()).To(BeEmpty())
		})

		It("Should add and retrieve a single system policy", func() {
			p := policy.Policy{
				Key:    uuid.New(),
				Name:   "system-protection",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindLogical,
					Operator: constraint.OpContainsAll,
					Constraints: []constraint.Constraint{
						{
							Kind:     constraint.KindMatch,
							Operator: constraint.OpContainsAny,
							IDs:      []ontology.ID{{Type: "metrics", Key: "cpu"}},
						},
						{
							Kind:     constraint.KindAction,
							Operator: constraint.OpIsIn,
							Actions:  set.New(access.ActionUpdate, access.ActionDelete),
						},
					},
				},
			}
			svc.AddSystemPolicies(p)
			policies := svc.SystemPolicies()
			Expect(policies).To(ConsistOf(p))
		})

		It("Should add multiple system policies via variadic call", func() {
			p1 := policy.Policy{
				Key:    uuid.New(),
				Name:   "protect-metrics",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "metrics"}},
				},
			}
			p2 := policy.Policy{
				Key:    uuid.New(),
				Name:   "protect-system-channels",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "system_channel"}},
				},
			}
			svc.AddSystemPolicies(p1, p2)
			policies := svc.SystemPolicies()
			Expect(policies).To(HaveLen(2))
			keys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(keys).To(ContainElements(p1.Key, p2.Key))
		})

		It("Should accumulate system policies across multiple AddSystem calls", func() {
			p1 := policy.Policy{
				Key:    uuid.New(),
				Name:   "policy-1",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "type1"}},
				},
			}
			p2 := policy.Policy{
				Key:    uuid.New(),
				Name:   "policy-2",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "type2"}},
				},
			}
			svc.AddSystemPolicies(p1)
			svc.AddSystemPolicies(p2)
			policies := svc.SystemPolicies()
			Expect(policies).To(HaveLen(2))
			keys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(keys).To(ContainElements(p1.Key, p2.Key))
		})

		It("Should generate UUID for policy with nil key", func() {
			p := policy.Policy{
				Name:   "auto-keyed-policy",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "protected"}},
				},
			}
			Expect(p.Key).To(Equal(uuid.Nil))
			svc.AddSystemPolicies(p)
			policies := svc.SystemPolicies()
			Expect(policies).To(HaveLen(1))
			Expect(policies[0].Key).ToNot(Equal(uuid.Nil))
			Expect(policies[0].Name).To(Equal("auto-keyed-policy"))
		})

		It("Should generate unique UUIDs for multiple policies with nil keys", func() {
			p1 := policy.Policy{
				Name:   "policy-1",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "type1"}},
				},
			}
			p2 := policy.Policy{
				Name:   "policy-2",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "type2"}},
				},
			}
			svc.AddSystemPolicies(p1, p2)
			policies := svc.SystemPolicies()
			Expect(policies).To(HaveLen(2))
			Expect(policies[0].Key).ToNot(Equal(uuid.Nil))
			Expect(policies[1].Key).ToNot(Equal(uuid.Nil))
			Expect(policies[0].Key).ToNot(Equal(policies[1].Key))
		})

		It("Should preserve existing key when provided", func() {
			existingKey := uuid.New()
			p := policy.Policy{
				Key:    existingKey,
				Name:   "pre-keyed-policy",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "resource"}},
				},
			}
			svc.AddSystemPolicies(p)
			policies := svc.SystemPolicies()
			Expect(policies).To(HaveLen(1))
			Expect(policies[0].Key).To(Equal(existingKey))
		})
	})
})
