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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var tx gorp.Tx
	BeforeEach(func(ctx SpecContext) { tx = db.OpenTx() })
	AfterEach(func(ctx SpecContext) { Expect(tx.Close()).To(Succeed()) })

	Describe("OpenService", func() {
		It("Should open a service with valid configuration", func(ctx SpecContext) {
			s := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			Expect(s).ToNot(BeNil())
			Expect(s.Policy).ToNot(BeNil())
			Expect(s.Role).ToNot(BeNil())
			Expect(s.Close()).To(Succeed())
		})
		It("Should return error with missing DB", func(ctx SpecContext) {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				Ontology: otg,
				Group:    g,
			})
			Expect(err).To(HaveOccurred())
		})
		It("Should return error with missing Ontology", func(ctx SpecContext) {
			_, err := rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:    db,
				Group: g,
			})
			Expect(err).To(HaveOccurred())
		})
		It("Should return error with missing Group", func(ctx SpecContext) {
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
		BeforeEach(func(ctx SpecContext) {
			policyWriter = svc.Policy.NewWriter(tx, true)
			roleWriter = svc.Role.NewWriter(tx, true)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			obj1 = ontology.ID{Type: "channel", Key: "channel-1"}
			obj2 = ontology.ID{Type: "channel", Key: "channel-2"}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		Describe("Enforce with role-based policies", func() {
			It("Should allow access when policy allows action", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-read",
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

			It("Should deny access when no policy exists", func(ctx SpecContext) {
				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})

			It("Should allow access with ActionAll wildcard", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-all-actions",
					Objects: []ontology.ID{obj1},
					Actions: access.AllActions,
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

			It("Should allow access with type-based matching", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				typeWildcard := ontology.ID{Type: "channel"}
				p := &policy.Policy{
					Name:    "allow-all-channels",
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

			It("Should deny when multiple objects and only one is allowed", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-obj1",
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
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})
		})

		Describe("Enforce with multiple roles", func() {
			It("Should allow access via role assignment", func(ctx SpecContext) {
				r := &role.Role{
					Name:        "reader",
					Description: "Read-only access",
				}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-read",
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

			It("Should deny access after role unassignment", func(ctx SpecContext) {
				r := &role.Role{
					Name:        "reader",
					Description: "Read-only access",
				}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Name:    "allow-read",
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
				Expect(svc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})
		})
	})

	Describe("RetrievePoliciesForSubject", func() {
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
		)
		BeforeEach(func(ctx SpecContext) {
			policyWriter = svc.Policy.NewWriter(tx, true)
			roleWriter = svc.Role.NewWriter(tx, true)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		It("Should retrieve policies from assigned roles", func(ctx SpecContext) {
			r := &role.Role{
				Name:        "admin",
				Description: "Administrator role",
				Internal:    true,
			}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p1 := &policy.Policy{
				Name:     "policy-1",
				Objects:  []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions:  access.AllActions,
				Internal: true,
			}
			p2 := &policy.Policy{
				Name:    "policy-2",
				Objects: []ontology.ID{{Type: "workspace", Key: "ws1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(policyWriter.Create(ctx, p1)).To(Succeed())
			Expect(policyWriter.Create(ctx, p2)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, p1.Key, p2.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			policies := MustSucceed(svc.RetrievePoliciesForSubject(ctx, subject, tx))
			Expect(policies).To(HaveLen(2))
			policyKeys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(policyKeys).To(ContainElements(p1.Key, p2.Key))
		})

		It("Should return empty list when no roles assigned", func(ctx SpecContext) {
			policies := MustSucceed(svc.RetrievePoliciesForSubject(ctx, subject, tx))
			Expect(policies).To(BeEmpty())
		})
	})

	Describe("NewEnforcer", func() {
		It("Should create a functional enforcer", func(ctx SpecContext) {
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

		It("Should use provided transaction", func(ctx SpecContext) {
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
