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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// createUser creates a user to act as the subject of access requests.
func createUser(ctx context.Context, tx gorp.Tx) resource.ID {
	GinkgoHelper()
	u := MustSucceed(userSvc.NewWriter(tx).Create(ctx, user.User{
		Name: "subject-" + uuid.New().String(),
	}))
	return u.ResourceID()
}

func userKey(subject resource.ID) user.Key {
	GinkgoHelper()
	return MustSucceed(user.KeyFromResourceID(subject))
}

func assignRole(ctx context.Context, tx gorp.Tx, subject resource.ID, r role.Key) {
	GinkgoHelper()
	Expect(userSvc.NewWriter(tx).AssignRole(ctx, r, userKey(subject))).To(Succeed())
}

var _ = Describe("ServiceConfig", func() {
	// fullCfg returns a ServiceConfig with every required field populated. Override and
	// Validate specs branch off this to exercise individual fields.
	var cfg rbac.ServiceConfig
	BeforeEach(func() {
		cfg = rbac.ServiceConfig{
			DB:       db,
			Ontology: otg,
			User:     userSvc,
		}
	})
	Describe("Override", func() {
		It(
			"Should take each required field from the override when the base is zero",
			func() {
				result := rbac.ServiceConfig{}.Override(cfg)
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.User).To(Equal(userSvc))
			},
		)
		It(
			"Should keep each required field from the base when the override is zero",
			func() {
				result := cfg.Override(rbac.ServiceConfig{})
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.User).To(Equal(userSvc))
			},
		)
	})
	Describe("Validate", func() {
		It("Should succeed when every required field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
		})
		DescribeTable(
			"Should return an error when a required field is missing",
			func(mutate func(*rbac.ServiceConfig), errorMsg string) {
				mutate(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(errorMsg)))
			},
			Entry(
				"db",
				func(c *rbac.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"ontology",
				func(c *rbac.ServiceConfig) { c.Ontology = nil },
				"ontology: must be non-nil",
			),
			Entry(
				"user",
				func(c *rbac.ServiceConfig) { c.User = nil },
				"user: must be non-nil",
			),
		)
	})
})

var _ = Describe("Service", func() {
	var tx gorp.Tx

	Describe("OpenService", func() {
		It("Should open a service with a valid configuration", func(ctx SpecContext) {
			s := MustSucceed(rbac.OpenService(ctx, newConfig(ctx)))
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
			subject      resource.ID
			obj1         resource.ID
			obj2         resource.ID
		)
		BeforeEach(func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			policyWriter = rbacSvc.Policy.NewWriter(tx, true)
			roleWriter = rbacSvc.Role.NewWriter(tx, true)
			subject = createUser(ctx, tx)
			obj1 = resource.ID{Type: "channel", Key: "channel-1"}
			obj2 = resource.ID{Type: "channel", Key: "channel-2"}
		})

		Describe("Enforce with role-based policies", func() {
			It("Should allow access when policy allows action", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Role:    r.Key,
					Name:    "allow-read",
					Objects: []resource.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It("Should deny access when no policy exists", func(ctx SpecContext) {
				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(
					rbacSvc.NewEnforcer(tx).Enforce(ctx, req),
				).To(MatchError(access.ErrDenied))
			})

			It("Should name the subject, action, and denied object", func(
				ctx SpecContext,
			) {
				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(MatchError(
					ContainSubstring(
						"%s cannot retrieve %s",
						subject.String(),
						obj1.String(),
					),
				))
			})

			It("Should name the object that stopped a multi-object request", func(
				ctx SpecContext,
			) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())
				p := &policy.Policy{
					Role:    r.Key,
					Name:    "allow-read-obj1",
					Objects: []resource.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1, obj2},
					Action:  access.ActionRetrieve,
				}
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(obj2.String())),
					Not(MatchError(ContainSubstring(obj1.String()))),
				))
			})

			It("Should allow access with ActionAll wildcard", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Role:    r.Key,
					Name:    "allow-all-actions",
					Objects: []resource.ID{obj1},
					Actions: access.AllActions,
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				for _, action := range []access.Action{
					access.ActionCreate,
					access.ActionRetrieve,
					access.ActionUpdate,
					access.ActionDelete,
				} {
					req := access.Request{
						Subject: subject,
						Objects: []resource.ID{obj1},
						Action:  action,
					}
					Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
				}
			})

			It("Should allow access with type-based matching", func(ctx SpecContext) {
				r := &role.Role{Name: "test-role", Description: "Test role"}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				typeWildcard := resource.ID{Type: "channel"}
				p := &policy.Policy{
					Role:    r.Key,
					Name:    "allow-all-channels",
					Objects: []resource.ID{typeWildcard},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1, obj2},
					Action:  access.ActionRetrieve,
				}
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			})

			It(
				"Should deny when multiple objects and only one is allowed",
				func(ctx SpecContext) {
					r := &role.Role{Name: "test-role", Description: "Test role"}
					Expect(roleWriter.Create(ctx, r)).To(Succeed())

					p := &policy.Policy{
						Role:    r.Key,
						Name:    "allow-obj1",
						Objects: []resource.ID{obj1},
						Actions: []access.Action{access.ActionRetrieve},
					}
					Expect(policyWriter.Create(ctx, p)).To(Succeed())
					assignRole(ctx, tx, subject, r.Key)

					req := access.Request{
						Subject: subject,
						Objects: []resource.ID{obj1, obj2},
						Action:  access.ActionRetrieve,
					}
					Expect(
						rbacSvc.NewEnforcer(tx).Enforce(ctx, req),
					).To(MatchError(access.ErrDenied))
				},
			)
		})

		Describe("Enforce with multiple roles", func() {
			It("Should allow access via role assignment", func(ctx SpecContext) {
				r := &role.Role{
					Name:        "reader",
					Description: "Read-only access",
				}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())

				p := &policy.Policy{
					Role:    r.Key,
					Name:    "allow-read",
					Objects: []resource.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1},
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
					Role:    r.Key,
					Name:    "allow-read",
					Objects: []resource.ID{obj1},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				req := access.Request{
					Subject: subject,
					Objects: []resource.ID{obj1},
					Action:  access.ActionRetrieve,
				}
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())

				Expect(userSvc.NewWriter(tx).SetRoles(ctx, nil, userKey(subject))).
					To(Succeed())
				Expect(
					rbacSvc.NewEnforcer(tx).Enforce(ctx, req),
				).To(MatchError(access.ErrDenied))
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
			subject      resource.ID
			obj          resource.ID
		)
		BeforeEach(func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			policyWriter = rbacSvc.Policy.NewWriter(tx, true)
			roleWriter = rbacSvc.Role.NewWriter(tx, true)
			subject = createUser(ctx, tx)
			obj = resource.ID{Type: "channel", Key: uuid.New().String()}
		})

		It(
			"Should deny when the matching policy's actions do not contain the requested action",
			func(ctx SpecContext) {
				r := &role.Role{Name: "read-only-" + uuid.New().String()}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())
				p := &policy.Policy{
					Role:    r.Key,
					Name:    "read-only-policy-" + uuid.New().String(),
					Objects: []resource.ID{obj},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(policyWriter.Create(ctx, p)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Objects: []resource.ID{obj},
					Action:  access.ActionUpdate,
				})).To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should fall through a non-matching-action policy to a matching-action policy",
			func(ctx SpecContext) {
				r := &role.Role{Name: "split-actions-" + uuid.New().String()}
				Expect(roleWriter.Create(ctx, r)).To(Succeed())
				retrievePolicy := &policy.Policy{
					Role:    r.Key,
					Name:    "retrieve-only-" + uuid.New().String(),
					Objects: []resource.ID{obj},
					Actions: []access.Action{access.ActionRetrieve},
				}
				updatePolicy := &policy.Policy{
					Role:    r.Key,
					Name:    "update-only-" + uuid.New().String(),
					Objects: []resource.ID{obj},
					Actions: []access.Action{access.ActionUpdate},
				}
				Expect(policyWriter.Create(ctx, retrievePolicy)).To(Succeed())
				Expect(policyWriter.Create(ctx, updatePolicy)).To(Succeed())
				assignRole(ctx, tx, subject, r.Key)

				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Objects: []resource.ID{obj},
					Action:  access.ActionUpdate,
				})).To(Succeed())
			},
		)
	})

	Describe("Role policies", func() {
		var (
			policyWriter policy.Writer
			roleWriter   role.Writer
			subject      resource.ID
		)
		BeforeEach(func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			policyWriter = rbacSvc.Policy.NewWriter(tx, true)
			roleWriter = rbacSvc.Role.NewWriter(tx, true)
			subject = createUser(ctx, tx)
		})

		It("Should enforce the policies of an assigned role", func(ctx SpecContext) {
			r := &role.Role{
				Name:        "admin",
				Description: "Administrator role",
				Internal:    true,
			}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p1 := &policy.Policy{
				Role:     r.Key,
				Name:     "policy-1",
				Objects:  []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions:  access.AllActions,
				Internal: true,
			}
			p2 := &policy.Policy{
				Role:    r.Key,
				Name:    "policy-2",
				Objects: []resource.ID{{Type: "project", Key: "p1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(policyWriter.Create(ctx, p1)).To(Succeed())
			Expect(policyWriter.Create(ctx, p2)).To(Succeed())
			assignRole(ctx, tx, subject, r.Key)

			enforcer := rbacSvc.NewEnforcer(tx)
			Expect(enforcer.Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Action:  access.ActionDelete,
			})).To(Succeed())
			Expect(enforcer.Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []resource.ID{{Type: "project", Key: "p1"}},
				Action:  access.ActionRetrieve,
			})).To(Succeed())
			Expect(enforcer.Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []resource.ID{{Type: "project", Key: "p1"}},
				Action:  access.ActionUpdate,
			})).To(MatchError(access.ErrDenied))
		})

		It("Should deny a subject with no assigned roles", func(ctx SpecContext) {
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Action:  access.ActionRetrieve,
			})).To(MatchError(access.ErrDenied))
		})
	})

	Describe("Role deletion", func() {
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It("Should delete the role's policies and remove it from its users", func(
			ctx SpecContext,
		) {
			roleWriter := rbacSvc.Role.NewWriter(tx, true)
			r := &role.Role{Name: "doomed"}
			kept := &role.Role{Name: "kept"}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())
			Expect(roleWriter.Create(ctx, kept)).To(Succeed())
			p := &policy.Policy{
				Role:    r.Key,
				Name:    "doomed-policy",
				Objects: []resource.ID{{Type: resource.TypeChannel}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(rbacSvc.Policy.NewWriter(tx, true).Create(ctx, p)).To(Succeed())
			subject := createUser(ctx, tx)
			assignRole(ctx, tx, subject, r.Key)
			assignRole(ctx, tx, subject, kept.Key)
			Expect(roleWriter.Delete(ctx, r.Key)).To(Succeed())
			Expect(rbacSvc.Policy.NewRetrieve().
				Where(policy.MatchKeys(p.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(rbacSvc.RolesOf(ctx, tx, subject)).
				To(Equal([]role.Key{kept.Key}))
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: subject,
				Objects: []resource.ID{{Type: resource.TypeChannel, Key: "ch1"}},
				Action:  access.ActionRetrieve,
			})).To(MatchError(access.ErrDenied))
		})

		It("Should remove a deleted role from its racks", func(ctx SpecContext) {
			subject := rackSvc.EmbeddedKey.ResourceID()
			Expect(rbacSvc.RolesOf(ctx, tx, subject)).
				To(ContainElement(rbacSvc.Builtin.DriverKey))
			Expect(rbacSvc.Role.NewWriter(tx, true).
				Delete(ctx, rbacSvc.Builtin.DriverKey)).To(Succeed())
			Expect(rbacSvc.RolesOf(ctx, tx, subject)).
				ToNot(ContainElement(rbacSvc.Builtin.DriverKey))
			Expect(rbacSvc.Policy.NewRetrieve().
				Where(policy.MatchRoles(rbacSvc.Builtin.DriverKey)).
				Exists(ctx, tx)).To(BeFalse())
		})
	})

	Describe("RolesOf", func() {
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It("Should return the roles of several subjects once each", func(
			ctx SpecContext,
		) {
			r := &role.Role{Name: "shared"}
			Expect(rbacSvc.Role.NewWriter(tx, true).Create(ctx, r)).To(Succeed())
			first, second := createUser(ctx, tx), createUser(ctx, tx)
			assignRole(ctx, tx, first, r.Key)
			assignRole(ctx, tx, second, r.Key)
			Expect(rbacSvc.RolesOf(ctx, tx, first, second)).
				To(Equal([]role.Key{r.Key}))
		})

		It("Should return not found for a user that does not exist", func(
			ctx SpecContext,
		) {
			subject := createUser(ctx, tx)
			Expect(userSvc.NewWriter(tx).Delete(ctx, userKey(subject))).To(Succeed())
			Expect(rbacSvc.RolesOf(ctx, tx, subject)).Error().
				To(MatchError(query.ErrNotFound))
		})

		It(
			"Should return a validation error for a subject that cannot hold roles",
			func(ctx SpecContext) {
				Expect(rbacSvc.RolesOf(ctx, tx, resource.ID{
					Type: resource.TypeChannel,
					Key:  "ch1",
				})).Error().To(MatchError(ContainSubstring("cannot hold roles")))
			},
		)
	})

	Describe("Missing subjects", func() {
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It("Should deny a user that was deleted", func(ctx SpecContext) {
			r := &role.Role{Name: "reader"}
			Expect(rbacSvc.Role.NewWriter(tx, true).Create(ctx, r)).To(Succeed())
			Expect(rbacSvc.Policy.NewWriter(tx, true).Create(ctx, &policy.Policy{
				Role:    r.Key,
				Name:    "read-channels",
				Objects: []resource.ID{{Type: resource.TypeChannel}},
				Actions: []access.Action{access.ActionRetrieve},
			})).To(Succeed())
			subject := createUser(ctx, tx)
			assignRole(ctx, tx, subject, r.Key)
			req := access.Request{
				Subject: subject,
				Objects: []resource.ID{{Type: resource.TypeChannel, Key: "ch1"}},
				Action:  access.ActionRetrieve,
			}
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			Expect(userSvc.NewWriter(tx).Delete(ctx, userKey(subject))).To(Succeed())
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).
				To(MatchError(access.ErrDenied))
		})

		It("Should deny a rack that was deleted", func(ctx SpecContext) {
			r := &rack.Rack{Name: "doomed"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			req := access.Request{
				Subject: r.ResourceID(),
				Objects: []resource.ID{{Type: resource.TypeDevice, Key: "d1"}},
				Action:  access.ActionRetrieve,
			}
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).To(Succeed())
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, r.Key)).To(Succeed())
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, req)).
				To(MatchError(access.ErrDenied))
		})

		It("Should reject a subject that cannot hold roles", func(ctx SpecContext) {
			Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
				Subject: resource.ID{Type: resource.TypeChannel, Key: "ch1"},
				Objects: []resource.ID{{Type: resource.TypeChannel, Key: "ch1"}},
				Action:  access.ActionRetrieve,
			})).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("NewEnforcer", func() {
		It("Should create a functional enforcer", func(ctx SpecContext) {
			enforcer := rbacSvc.NewEnforcer(nil)
			Expect(enforcer).ToNot(BeNil())

			subject := createUser(ctx, nil)
			obj := resource.ID{Type: "channel", Key: "ch1"}
			req := access.Request{
				Subject: subject,
				Objects: []resource.ID{obj},
				Action:  access.ActionRetrieve,
			}
			Expect(enforcer.Enforce(ctx, req)).To(MatchError(access.ErrDenied))
		})

		It("Should use provided transaction", func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			enforcer := rbacSvc.NewEnforcer(tx)
			Expect(enforcer).ToNot(BeNil())

			policyWriter := rbacSvc.Policy.NewWriter(tx, true)
			roleWriter := rbacSvc.Role.NewWriter(tx, true)
			subject := createUser(ctx, tx)
			obj := resource.ID{Type: "channel", Key: "ch1"}

			r := &role.Role{Name: "test-role", Description: "Test role"}
			Expect(roleWriter.Create(ctx, r)).To(Succeed())

			p := &policy.Policy{
				Role:    r.Key,
				Name:    "allow-read",
				Objects: []resource.ID{obj},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(policyWriter.Create(ctx, p)).To(Succeed())
			assignRole(ctx, tx, subject, r.Key)

			req := access.Request{
				Subject: subject,
				Objects: []resource.ID{obj},
				Action:  access.ActionRetrieve,
			}
			Expect(enforcer.Enforce(ctx, req)).To(Succeed())
		})
	})
})
