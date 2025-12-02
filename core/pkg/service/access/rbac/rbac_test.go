// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Service", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("OpenService", func() {
		It("Should open a service with valid configuration", func() {
			s, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(s).ToNot(BeNil())
			Expect(s.Policy).ToNot(BeNil())
			Expect(s.Role).ToNot(BeNil())
			Expect(s.Close()).To(Succeed())
		})
		It("Should return error with missing DB", func() {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				Ontology: otg,
				Group:    g,
			})
			Expect(err).To(HaveOccurred())
		})
		It("Should return error with missing Ontology", func() {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:    db,
				Group: g,
			})
			Expect(err).To(HaveOccurred())
		})
		It("Should return error with missing Group", func() {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
			})
			Expect(err).To(HaveOccurred())
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
			policyWriter = svc.Policy.NewWriter(tx, true)
			roleWriter = svc.Role.NewWriter(tx, true)
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
					Name:    "allow-read",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
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

			It("Should deny access when policy denies action", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "deny-delete",
					Effect:  policy.EffectDeny,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionDelete},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionDelete,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.Denied))
			})

			It("Should deny access when no policy exists", func() {
				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.Denied))
			})

			It("Should allow access with ActionAll wildcard", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-all-actions",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionAll},
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
					Name:    "allow-all-channels",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{typeWildcard},
					Actions: []access.Action{access.ActionRetrieve},
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

			It("Should deny when multiple objects and only one is allowed", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-obj1",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1, obj2},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.Denied))
			})

			It("Should prioritize deny over allow", func() {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				allowPolicy := &policy.Policy{
					Name:    "allow-read",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				denyPolicy := &policy.Policy{
					Name:    "deny-read",
					Effect:  policy.EffectDeny,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, allowPolicy)).To(Succeed())
				Expect(policyWriter.Create(ctx, denyPolicy)).To(Succeed())
				Expect(policyWriter.SetOnRole(ctx, r.Key, allowPolicy.Key, denyPolicy.Key)).To(Succeed())
				Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.Denied))
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
					Name:    "allow-read",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
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
					Name:    "allow-read",
					Effect:  policy.EffectAllow,
					Objects: []ontology.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
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
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.Denied))
			})
		})
	})

	Describe("RetrievePoliciesForSubject", func() {
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
		)
		BeforeEach(func() {
			policyWriter = svc.Policy.NewWriter(tx, true)
			roleWriter = svc.Role.NewWriter(tx, true)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		It("Should retrieve policies from assigned roles", func() {
			r := &role.Role{
				Name:        "admin",
				Description: "Administrator role",
				Internal:    true,
			}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p1 := &policy.Policy{
				Name:     "policy-1",
				Effect:   policy.EffectAllow,
				Objects:  []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions:  []access.Action{access.ActionAll},
				Internal: true,
			}
			p2 := &policy.Policy{
				Name:    "policy-2",
				Effect:  policy.EffectAllow,
				Objects: []ontology.ID{{Type: "workspace", Key: "ws1"}},
				Actions: []access.Action{access.ActionRetrieve},
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
			Expect(enforcer.Enforce(ctx, req)).To(MatchError(access.Denied))
		})

		It("Should use provided transaction", func() {
			enforcer := svc.NewEnforcer(tx)
			Expect(enforcer).ToNot(BeNil())

			policyWriter := svc.Policy.NewWriter(tx, true)
			roleWriter := svc.Role.NewWriter(tx, true)
			subject := ontology.ID{Type: "user", Key: uuid.New().String()}
			obj := ontology.ID{Type: "channel", Key: "ch1"}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())

			r := &role.Role{Name: "test-role", Description: "Test role"}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p := &policy.Policy{
				Name:    "allow-read",
				Effect:  policy.EffectAllow,
				Objects: []ontology.ID{obj},
				Actions: []access.Action{access.ActionRetrieve},
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
})
