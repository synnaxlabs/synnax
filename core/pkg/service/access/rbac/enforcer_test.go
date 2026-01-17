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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Enforcer", func() {
	Describe("NewEnforcer", func() {
		It("Should create an enforcer with nil transaction", func() {
			subject := ontology.ID{Type: "user", Key: uuid.New().String()}
			obj := ontology.ID{Type: "channel", Key: "ch1"}
			Expect(otg.NewWriter(nil).DefineResource(ctx, subject)).To(Succeed())

			enforcer := svc.NewEnforcer(nil)
			req := access.Request{
				Subject: subject,
				Objects: []ontology.ID{obj},
				Action:  access.ActionRetrieve,
			}
			Expect(enforcer.Enforce(ctx, req)).To(MatchError(access.ErrDenied))
		})

		It("Should use provided transaction for policy retrieval", func() {
			tx := db.OpenTx()

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
			Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
		})
	})

	Describe("Enforce", func() {
		var (
			tx           gorp.Tx
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
			obj1         ontology.ID
			obj2         ontology.ID
		)
		BeforeEach(func() {
			tx = db.OpenTx()
			policyWriter = svc.Policy.NewWriter(tx)
			roleWriter = svc.Role.NewWriter(tx)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			obj1 = ontology.ID{Type: "channel", Key: "channel-1"}
			obj2 = ontology.ID{Type: "channel", Key: "channel-2"}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})
		AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

		Context("with role-based policies", func() {
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
				err := svc.NewEnforcer(tx).Enforce(ctx, req)
				Expect(err).To(MatchError(access.ErrDenied))
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

			It("Should deny access when action doesn't match", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:   "allow-read-only",
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
					Action:  access.ActionDelete,
				}
				err := svc.NewEnforcer(tx).Enforce(ctx, req)
				Expect(err).To(MatchError(access.ErrDenied))
			})

			It("Should deny when multiple objects and only one is allowed", func() {
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
				err := svc.NewEnforcer(tx).Enforce(ctx, req)
				Expect(err).To(MatchError(access.ErrDenied))
			})
		})

		Context("with multiple roles", func() {
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
				err := svc.NewEnforcer(tx).Enforce(ctx, req)
				Expect(err).To(MatchError(access.ErrDenied))
			})

			It("Should allow access when any assigned role permits it", func() {
				r1 := &role.Role{Name: "role-1", Description: "Role 1"}
				r2 := &role.Role{Name: "role-2", Description: "Role 2"}
				Expect(roleWriter.Create(ctx, r1)).To(Succeed())
				Expect(roleWriter.Create(ctx, r2)).To(Succeed())

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
				Expect(policyWriter.SetOnRole(ctx, r2.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r1.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r2.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})
		})

		Context("with system policies", func() {
			It("Should deny access based on system deny policy", func() {
				metricsObj := ontology.ID{Type: "metrics", Key: "cpu-usage"}
				systemPolicy := policy.Policy{
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
				svc.Policy.AddSystemPolicies(systemPolicy)

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{metricsObj},
					Action:  access.ActionDelete,
				}
				err := svc.NewEnforcer(tx).Enforce(ctx, req)
				Expect(err).To(MatchError(access.ErrDenied))
			})

			It("Should allow access when system policy does not match", func() {
				channelObj := ontology.ID{Type: "channel", Key: "sensor-1"}

				systemDenyPolicy := policy.Policy{
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
				svc.Policy.AddSystemPolicies(systemDenyPolicy)

				systemAllowPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "allow-channels",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindMatch,
						Operator: constraint.OpContainsAny,
						IDs:      []ontology.ID{{Type: "channel"}},
					},
				}
				svc.Policy.AddSystemPolicies(systemAllowPolicy)

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{channelObj},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It("Should deny even with role allow when system deny matches", func() {
				metricsObj := ontology.ID{Type: "metrics", Key: "cpu-usage"}

				systemDenyPolicy := policy.Policy{
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
				svc.Policy.AddSystemPolicies(systemDenyPolicy)

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
				err := svc.NewEnforcer(tx).Enforce(ctx, req)
				Expect(err).To(MatchError(access.ErrDenied))
			})

			It("Should process deny policies before allow policies", func() {
				obj := ontology.ID{Type: "protected", Key: "resource-1"}

				allowPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "allow-all",
					Effect: policy.EffectAllow,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindMatch,
						Operator: constraint.OpContainsAny,
						IDs:      []ontology.ID{{Type: "protected"}},
					},
				}
				denyPolicy := policy.Policy{
					Key:    uuid.New(),
					Name:   "deny-delete",
					Effect: policy.EffectDeny,
					Constraint: constraint.Constraint{
						Kind:     constraint.KindLogical,
						Operator: constraint.OpContainsAll,
						Constraints: []constraint.Constraint{
							{
								Kind:     constraint.KindMatch,
								Operator: constraint.OpContainsAny,
								IDs:      []ontology.ID{{Type: "protected"}},
							},
							{
								Kind:     constraint.KindAction,
								Operator: constraint.OpIsIn,
								Actions:  set.New(access.ActionDelete),
							},
						},
					},
				}
				// Add allow first, then deny - deny should still take precedence
				svc.Policy.AddSystemPolicies(allowPolicy, denyPolicy)

				reqDelete := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj},
					Action:  access.ActionDelete,
				}
				err := svc.NewEnforcer(tx).Enforce(ctx, reqDelete)
				Expect(err).To(MatchError(access.ErrDenied))

				reqRetrieve := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, reqRetrieve)).To(Succeed())
			})
		})
	})
})
