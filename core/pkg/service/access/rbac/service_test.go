// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should validate properly shaped configs", func() {
				cfg := rbac.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}
				Expect(cfg.Validate()).To(Succeed())
			})
			It("Should validate configs with optional Signals field", func() {
				cfg := rbac.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc, Signals: nil}
				Expect(cfg.Validate()).To(Succeed())
			})
			It("Should fail to validate configs missing the DB field", func() {
				cfg := rbac.ServiceConfig{Ontology: otg, Group: groupSvc}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("Should fail to validate configs missing the Ontology field", func() {
				cfg := rbac.ServiceConfig{DB: db, Group: groupSvc}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
			It("Should fail to validate configs missing the Group field", func() {
				cfg := rbac.ServiceConfig{DB: db, Ontology: otg}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})
		Describe("Override", func() {
			It("Should override fields with non-nil values", func() {
				cfg := rbac.ServiceConfig{}
				other := rbac.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}
				result := cfg.Override(other)
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.Group).To(Equal(groupSvc))
			})
			It("Should not override fields with nil values", func() {
				cfg := rbac.ServiceConfig{DB: db, Ontology: otg, Group: groupSvc}
				other := rbac.ServiceConfig{}
				result := cfg.Override(other)
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.Group).To(Equal(groupSvc))
			})
		})
	})

	Describe("Open", func() {
		It("Should open a service with valid configuration", func() {
			s := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
			}))
			Expect(s.Close()).To(Succeed())
		})
		It("Should fail to open a service with an invalid configuration", func() {
			_, err := rbac.OpenService(ctx)
			Expect(err).To(HaveOccurred())
		})
		It("Should fail to open with missing DB", func() {
			Expect(rbac.OpenService(ctx, rbac.ServiceConfig{
				Ontology: otg,
				Group:    groupSvc,
			})).Error().To(MatchError(ContainSubstring("db")))
		})
		It("Should fail to open with missing Ontology", func() {
			Expect(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:    db,
				Group: groupSvc,
			})).Error().To(MatchError(ContainSubstring("ontology")))
		})
		It("Should fail to open with missing Group", func() {
			Expect(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
			})).Error().To(MatchError(ContainSubstring("group")))
		})
	})

	Describe("RetrievePoliciesForSubject", func() {
		var (
			tx           gorp.Tx
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
		)
		BeforeEach(func() {
			tx = db.OpenTx()
			policyWriter = svc.Policy.NewWriter(tx)
			roleWriter = svc.Role.NewWriter(tx)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})
		AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

		It("Should retrieve policies from assigned roles", func() {
			r := &role.Role{
				Name:        "admin",
				Description: "Administrator role",
			}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p1 := &policy.Policy{
				Name:   "policy-1",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindLogical,
					Operator: constraint.OpContainsAll,
					Constraints: []constraint.Constraint{
						{
							Kind:     constraint.KindMatch,
							Operator: constraint.OpContainsAny,
							IDs:      []ontology.ID{{Type: "channel", Key: "ch1"}},
						},
						{
							Kind:     constraint.KindAction,
							Operator: constraint.OpIsIn,
							Actions:  set.New(access.AllActions...),
						},
					},
				},
			}
			p2 := &policy.Policy{
				Name:   "policy-2",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindLogical,
					Operator: constraint.OpContainsAll,
					Constraints: []constraint.Constraint{
						{
							Kind:     constraint.KindMatch,
							Operator: constraint.OpContainsAny,
							IDs:      []ontology.ID{{Type: "workspace", Key: "ws1"}},
						},
						{
							Kind:     constraint.KindAction,
							Operator: constraint.OpIsIn,
							Actions:  set.New(access.ActionRetrieve),
						},
					},
				},
			}
			Expect(policyWriter.Create(ctx, p1)).To(Succeed())
			Expect(policyWriter.Create(ctx, p2)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, p1.Key, p2.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			policies, err := svc.RetrievePoliciesForSubject(ctx, subject, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(policies).To(HaveLen(2))
			policyKeys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(policyKeys).To(ContainElements(p1.Key, p2.Key))
		})

		It("Should return empty list when no roles assigned", func() {
			policies, err := svc.RetrievePoliciesForSubject(ctx, subject, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(policies).To(BeEmpty())
		})

		It("Should include system policies in retrieved policies", func() {
			systemPolicy := policy.Policy{
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
							IDs:      []ontology.ID{{Type: "metrics"}},
						},
						{
							Kind:     constraint.KindAction,
							Operator: constraint.OpIsIn,
							Actions:  set.New(access.ActionUpdate, access.ActionDelete),
						},
					},
				},
			}
			svc.Policy.AddSystemPolicies(systemPolicy)

			policies, err := svc.RetrievePoliciesForSubject(ctx, subject, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(policies).To(HaveLen(1))
			Expect(policies[0].Key).To(Equal(systemPolicy.Key))
		})

		It("Should combine system policies with role-based policies", func() {
			systemPolicy := policy.Policy{
				Key:    uuid.New(),
				Name:   "system-protection",
				Effect: policy.EffectDeny,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "metrics"}},
				},
			}
			svc.Policy.AddSystemPolicies(systemPolicy)

			r := &role.Role{Name: "test-role", Description: "Test role"}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			rolePolicy := &policy.Policy{
				Name:   "allow-channels",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindMatch,
					Operator: constraint.OpContainsAny,
					IDs:      []ontology.ID{{Type: "channel"}},
				},
			}
			Expect(policyWriter.Create(ctx, rolePolicy)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, rolePolicy.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			policies, err := svc.RetrievePoliciesForSubject(ctx, subject, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(policies).To(HaveLen(2))
			keys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(keys).To(ContainElements(systemPolicy.Key, rolePolicy.Key))
		})
	})

	Describe("Close", func() {
		It("Should close the service without error", func() {
			s := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
			}))
			Expect(s.Close()).To(Succeed())
		})

		It("Should be safe to close multiple times", func() {
			s := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
			}))
			Expect(s.Close()).To(Succeed())
			Expect(s.Close()).To(Succeed())
		})
	})
})
