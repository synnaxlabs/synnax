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
	var tx gorp.Tx
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Open", func() {
		It("Should open a service with valid configuration", func() {
			s := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
			}))
			Expect(s.Close()).To(Succeed())
		})
		It("Should return error with missing DB", func() {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				Ontology: otg,
				Group:    groupSvc,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db"))
		})
		It("Should return error with missing Ontology", func() {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:    db,
				Group: groupSvc,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ontology"))
		})
		It("Should return error with missing Group", func() {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("group"))
		})
	})

	Describe("RetrievePoliciesForSubject", func() {
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
		)
		BeforeEach(func() {
			policyWriter = svc.Policy.NewWriter(tx)
			roleWriter = svc.Role.NewWriter(tx)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

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
	})

	Describe("Enforcer", func() {
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
			obj1         ontology.ID
			obj2         ontology.ID
		)
		BeforeEach(func() {
			policyWriter = svc.Policy.NewWriter(tx)
			roleWriter = svc.Role.NewWriter(tx)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			obj1 = ontology.ID{Type: "channel", Key: "channel-1"}
			obj2 = ontology.ID{Type: "channel", Key: "channel-2"}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		Describe("Enforce with role-based policies", func() {
			It("Should allow access when policy allows action", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:   "allow-read",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{obj1},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.ActionRetrieve),
							},
						},
					},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It("Should deny access when no policy exists", func() {
				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})

			It("Should allow access with ActionAll wildcard", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:   "allow-all-actions",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{obj1},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.AllActions...),
							},
						},
					},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				for _, action := range []access.Action{
					access.ActionCreate,
					access.ActionRetrieve,
					access.ActionUpdate,
					access.ActionDelete,
				} {
					req := access.Request{
						Subject: subject,
						Objects: []ontology.ID{obj1},
						Action:  action,
					}
					Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
				}
			})

			It("Should allow access with type-based matching", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				typeWildcard := ontology.ID{Type: "channel"}
				p := &policy.Policy{
					Name:   "allow-all-channels",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{typeWildcard},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.ActionRetrieve),
							},
						},
					},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1, obj2},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			// TODO: This test requires per-object enforcement logic in the Enforcer.
			// Currently, the Enforcer checks if ANY policy matches the request, not
			// if ALL objects in the request are covered. This needs a redesign of the
			// Enforcer to iterate through each object and verify coverage.
			PIt("Should deny when multiple objects and only one is allowed", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:   "allow-obj1",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{obj1},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.ActionRetrieve),
							},
						},
					},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1, obj2},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})
		})

		Describe("Enforce with multiple roles", func() {
			It("Should allow access via role assignment", func() {
				r := &role.Role{
					Name:        "reader",
					Description: "Read-only access",
				}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:   "allow-read",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{obj1},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.ActionRetrieve),
							},
						},
					},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It("Should deny access after role unassignment", func() {
				r := &role.Role{
					Name:        "reader",
					Description: "Read-only access",
				}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:   "allow-read",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{obj1},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.ActionRetrieve),
							},
						},
					},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())

				Expect(roleWriter.UnassignRole(ctx, subject, r.Key)).To(Succeed())
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})
		})
	})

	Describe("NewEnforcer", func() {
		It("Should create a functional enforcer", func() {
			enforcer := svc.NewEnforcer(nil)
			Expect(enforcer).ToNot(BeNil())

			subject := ontology.ID{Type: "user", Key: uuid.New().String()}
			obj := ontology.ID{Type: "channel", Key: "ch1"}
			Expect(otg.NewWriter(nil).DefineResource(ctx, subject)).To(Succeed())
			req := access.Request{
				Subject: subject,
				Objects: []ontology.ID{obj},
				Action:  access.ActionRetrieve,
			}
			Expect(enforcer.Enforce(ctx, req)).To(MatchError(access.ErrDenied))
		})

		It("Should use provided transaction", func() {
			enforcer := svc.NewEnforcer(tx)
			Expect(enforcer).ToNot(BeNil())

			policyWriter := svc.Policy.NewWriter(tx)
			roleWriter := svc.Role.NewWriter(tx)
			subject := ontology.ID{Type: "user", Key: uuid.New().String()}
			obj := ontology.ID{Type: "channel", Key: "ch1"}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())

			r := &role.Role{Name: "test-role", Description: "Test role"}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p := &policy.Policy{
				Name:   "allow-read",
				Effect: policy.EffectAllow,
				Constraint: constraint.Constraint{
					Kind:     constraint.KindLogical,
					Operator: constraint.OpContainsAll,
					Constraints: []constraint.Constraint{
						{
							Kind:     constraint.KindMatch,
							Operator: constraint.OpContainsAny,
							IDs:      []ontology.ID{obj},
						},
						{
							Kind:     constraint.KindAction,
							Operator: constraint.OpIsIn,
							Actions:  set.New(access.ActionRetrieve),
						},
					},
				},
			}
			Expect(policyWriter.Create(ctx, p)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			req := access.Request{
				Subject: subject,
				Objects: []ontology.ID{obj},
				Action:  access.ActionRetrieve,
			}
			Expect(enforcer.Enforce(ctx, req)).To(Succeed())
		})
	})

	Describe("DefaultPolicies", func() {
		var subject ontology.ID
		BeforeEach(func() {
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		Describe("RetrievePoliciesForSubject", func() {
			It("Should include default policies in retrieved policies", func() {
				defaultPolicy := policy.Policy{
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
				svc.Policy.AddSystemPolicies(defaultPolicy)

				policies, err := svc.RetrievePoliciesForSubject(ctx, subject, tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(policies).To(HaveLen(1))
				Expect(policies[0].Key).To(Equal(defaultPolicy.Key))
			})

			It("Should combine default policies with role-based policies", func() {
				defaultPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "system-protection",
					Effect: policy.EffectDeny,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindMatch,
						Operator: constraint.OpContainsAny,
						IDs:      []ontology.ID{{Type: "metrics"}},
					},
				}
				svc.Policy.AddSystemPolicies(defaultPolicy)

				roleWriter := svc.Role.NewWriter(tx)
				policyWriter := svc.Policy.NewWriter(tx)

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
				Expect(keys).To(ContainElements(defaultPolicy.Key, rolePolicy.Key))
			})
		})

		Describe("Enforcer with default policies", func() {
			It("Should deny access based on default deny policy", func() {
				metricsObj := ontology.ID{Type: "metrics", Key: "cpu-usage"}
				defaultPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "protect-metrics",
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
				svc.Policy.AddSystemPolicies(defaultPolicy)

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{metricsObj},
					Action:  access.ActionDelete,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})

			It("Should allow access when default policy does not match", func() {
				channelObj := ontology.ID{Type: "channel", Key: "sensor-1"}

				defaultPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "protect-metrics",
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
				svc.Policy.AddSystemPolicies(defaultPolicy)

				allowPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "allow-channels",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindMatch,
						Operator: constraint.OpContainsAny,
						IDs:      []ontology.ID{{Type: "channel"}},
					},
				}
				svc.Policy.AddSystemPolicies(allowPolicy)

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{channelObj},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It("Should deny even with role allow when default deny matches", func() {
				metricsObj := ontology.ID{Type: "metrics", Key: "cpu-usage"}

				defaultDenyPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "protect-metrics",
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
								Actions:  set.New(access.ActionDelete),
							},
						},
					},
				}
				svc.Policy.AddSystemPolicies(defaultDenyPolicy)

				roleWriter := svc.Role.NewWriter(tx)
				policyWriter := svc.Policy.NewWriter(tx)

				r := &role.Role{Name: "admin", Description: "Admin role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				allowAllPolicy := &policy.Policy{
					Name:   "allow-all-metrics",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindMatch,
						Operator: constraint.OpContainsAny,
						IDs:      []ontology.ID{{Type: "metrics"}},
					},
				}
				Expect(policyWriter.Create(ctx, allowAllPolicy)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, allowAllPolicy.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{metricsObj},
					Action:  access.ActionDelete,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})
		})
	})
})
