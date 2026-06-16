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

var _ = Describe("ServiceConfig", func() {
	// fullCfg returns a ServiceConfig with every required field populated. Override and
	// Validate specs branch off this to exercise individual fields.
	var cfg rbac.ServiceConfig
	BeforeEach(func() {
		cfg = rbac.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
			User:     userSvc,
		}
	})
	Describe("Override", func() {
		It("Should take each required field from the override when the base is zero", func() {
			result := rbac.ServiceConfig{}.Override(cfg)
			Expect(result.DB).To(Equal(db))
			Expect(result.Ontology).To(Equal(otg))
			Expect(result.Group).To(Equal(groupSvc))
			Expect(result.Search).To(Equal(searchIdx))
			Expect(result.User).To(Equal(userSvc))
		})
		It("Should keep each required field from the base when the override is zero", func() {
			result := cfg.Override(rbac.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
			Expect(result.Ontology).To(Equal(otg))
			Expect(result.Group).To(Equal(groupSvc))
			Expect(result.Search).To(Equal(searchIdx))
			Expect(result.User).To(Equal(userSvc))
		})
	})
	Describe("Validate", func() {
		It("Should succeed when every required field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
		})
		DescribeTable("Should return an error when a required field is missing",
			func(mutate func(*rbac.ServiceConfig), errorMsg string) {
				mutate(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(errorMsg)))
			},
			Entry("db", func(c *rbac.ServiceConfig) { c.DB = nil }, "db: must be non-nil"),
			Entry("ontology", func(c *rbac.ServiceConfig) { c.Ontology = nil }, "ontology: must be non-nil"),
			Entry("group", func(c *rbac.ServiceConfig) { c.Group = nil }, "group: must be non-nil"),
			Entry("search", func(c *rbac.ServiceConfig) { c.Search = nil }, "search: must be non-nil"),
			Entry("user", func(c *rbac.ServiceConfig) { c.User = nil }, "user: must be non-nil"),
		)
	})
})

var _ = Describe("Service", func() {
	var tx gorp.Tx
	BeforeEach(func(ctx SpecContext) { tx = db.OpenTx() })
	AfterEach(func(ctx SpecContext) { Expect(tx.Close()).To(Succeed()) })

	Describe("OpenService", func() {
		It("Should open a service with a valid configuration", func(ctx SpecContext) {
			s := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				Search:   searchIdx,
				User:     userSvc,
			}))
			Expect(s).ToNot(BeNil())
			Expect(s.Policy).ToNot(BeNil())
			Expect(s.Role).ToNot(BeNil())
			Expect(s.Close()).To(Succeed())
		})
		It("Should propagate ServiceConfig.Validate errors", func(ctx SpecContext) {
			Expect(rbac.OpenService(ctx, rbac.ServiceConfig{})).
				Error().To(MatchError(ContainSubstring("db: must be non-nil")))
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
			policyWriter = rbacSvc.Policy.NewWriter(tx, true)
			roleWriter = rbacSvc.Role.NewWriter(tx, true)
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
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It("Should deny access when no policy exists", func(ctx SpecContext) {
				req := access.Request{
					Subject: subject,
					Objects: []ontology.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
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
					Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
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
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
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
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
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
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
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
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())

				Expect(roleWriter.UnassignRole(ctx, subject, r.Key)).To(Succeed())
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(access.ErrDenied))
			})
		})
	})

	Describe("allowRequest action filtering", func() {
		// These specs exercise the !hasAction branch of allowRequest: a policy is
		// attached to the subject and its objects match the request, but its
		// Actions slice does not contain the requested action, so the policy must
		// be skipped during the search.
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
			obj          ontology.ID
		)
		BeforeEach(func(ctx SpecContext) {
			policyWriter = rbacSvc.Policy.NewWriter(tx, true)
			roleWriter = rbacSvc.Role.NewWriter(tx, true)
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			obj = ontology.ID{Type: "channel", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
		})

		It("Should deny when the matching policy's actions do not contain the requested action", func(ctx SpecContext) {
			r := &role.Role{Name: "read-only-" + uuid.NewString()}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())
			p := &policy.Policy{
				Name:    "read-only-policy-" + uuid.NewString(),
				Objects: []ontology.ID{obj},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(policyWriter.Create(ctx, p)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []ontology.ID{obj},
				Action:  access.ActionUpdate,
			})).To(MatchError(access.ErrDenied))
		})

		It("Should fall through a non-matching-action policy to a matching-action policy", func(ctx SpecContext) {
			r := &role.Role{Name: "split-actions-" + uuid.NewString()}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())
			retrievePolicy := &policy.Policy{
				Name:    "retrieve-only-" + uuid.NewString(),
				Objects: []ontology.ID{obj},
				Actions: []access.Action{access.ActionRetrieve},
			}
			updatePolicy := &policy.Policy{
				Name:    "update-only-" + uuid.NewString(),
				Objects: []ontology.ID{obj},
				Actions: []access.Action{access.ActionUpdate},
			}
			Expect(policyWriter.Create(ctx, retrievePolicy)).To(Succeed())
			Expect(policyWriter.Create(ctx, updatePolicy)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, retrievePolicy.Key, updatePolicy.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []ontology.ID{obj},
				Action:  access.ActionUpdate,
			})).To(Succeed())
		})
	})

	Describe("RetrievePoliciesForSubject", func() {
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      ontology.ID
		)
		BeforeEach(func(ctx SpecContext) {
			policyWriter = rbacSvc.Policy.NewWriter(tx, true)
			roleWriter = rbacSvc.Role.NewWriter(tx, true)
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
				Objects: []ontology.ID{{Type: "project", Key: "p1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(policyWriter.Create(ctx, p1)).To(Succeed())
			Expect(policyWriter.Create(ctx, p2)).To(Succeed())
			Expect(policyWriter.SetOnRole(ctx, r.Key, p1.Key, p2.Key)).To(Succeed())
			Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())

			policies := MustSucceed(rbacSvc.RetrievePoliciesForSubject(ctx, subject, tx))
			Expect(policies).To(HaveLen(2))
			policyKeys := []uuid.UUID{policies[0].Key, policies[1].Key}
			Expect(policyKeys).To(ContainElements(p1.Key, p2.Key))
		})

		It("Should return empty list when no roles assigned", func(ctx SpecContext) {
			policies := MustSucceed(rbacSvc.RetrievePoliciesForSubject(ctx, subject, tx))
			Expect(policies).To(BeEmpty())
		})
	})

	Describe("NewEnforcer", func() {
		It("Should create a functional enforcer", func(ctx SpecContext) {
			enforcer := rbacSvc.NewEnforcer(nil)
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
			enforcer := rbacSvc.NewEnforcer(tx)
			Expect(enforcer).ToNot(BeNil())

			policyWriter := rbacSvc.Policy.NewWriter(tx, true)
			roleWriter := rbacSvc.Role.NewWriter(tx, true)
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
