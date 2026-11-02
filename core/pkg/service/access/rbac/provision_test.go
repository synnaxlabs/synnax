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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Provision", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })

	Describe("Built-in roles", func() {
		It(
			"Should have created all built-in roles during OpenService",
			func(ctx SpecContext) {
				for _, name := range []string{"Owner", "Engineer", "Driver", "Operator", "Viewer"} {
					var r role.Role
					Expect(
						rbacSvc.Role.NewRetrieve().
							Where(role.MatchNames(name)).
							Entry(&r).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(r.Key).ToNot(Equal(uuid.Nil()))
					Expect(r.Internal).To(BeTrue())
				}
			},
		)
		It("Should have created policies for each role", func(ctx SpecContext) {
			for _, name := range []string{"Owner", "Engineer", "Driver", "Operator", "Viewer"} {
				var r role.Role
				Expect(
					rbacSvc.Role.NewRetrieve().
						Where(role.MatchNames(name)).
						Entry(&r).
						Exec(ctx, tx),
				).To(Succeed())
				var policies []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(r.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&policies).
					Exec(ctx, tx)).To(Succeed())
				Expect(policies).ToNot(BeEmpty())
			}
		})
	})

	Describe("Driver role", func() {
		var subject ontology.ID
		BeforeEach(func(ctx SpecContext) {
			var r role.Role
			Expect(
				rbacSvc.Role.NewRetrieve().
					Where(role.MatchNames("Driver")).
					Entry(&r).
					Exec(ctx, tx),
			).To(Succeed())
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResources(ctx, subject)).To(Succeed())
			Expect(
				rbacSvc.Role.NewWriter(tx, true).AssignRole(ctx, subject, r.Key),
			).To(Succeed())
		})

		DescribeTable("grants full access to driver resource types",
			func(ctx SpecContext, t ontology.ResourceType) {
				for _, action := range access.AllActions {
					Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
						Subject: subject,
						Action:  action,
						Objects: []ontology.ID{{Type: t, Key: "test-key"}},
					})).To(Succeed())
				}
			},
			Entry("range", ontology.ResourceTypeRange),
			Entry("device", ontology.ResourceTypeDevice),
			Entry("task", ontology.ResourceTypeTask),
			Entry("arc", ontology.ResourceTypeArc),
			Entry("status", ontology.ResourceTypeStatus),
		)

		DescribeTable("denies access to non-driver resource types",
			func(ctx SpecContext, t ontology.ResourceType) {
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Action:  access.ActionRetrieve,
					Objects: []ontology.ID{{Type: t, Key: "test-key"}},
				})).To(MatchError(access.ErrDenied))
			},
			Entry("range alias", ontology.ResourceTypeRangeAlias),
			Entry("label", ontology.ResourceTypeLabel),
			Entry("log", ontology.ResourceTypeLog),
			Entry("node", ontology.ResourceTypeNode),
			Entry("group", ontology.ResourceTypeGroup),
			Entry("project", ontology.ResourceTypeProject),
			Entry("schematic", ontology.ResourceTypeSchematic),
			Entry("lineplot", ontology.ResourceTypeLineplot),
			Entry("table", ontology.ResourceTypeTable),
			Entry("schematic symbol", ontology.ResourceTypeSchematicSymbol),
			Entry("view", ontology.ResourceTypeView),
			Entry("user", ontology.ResourceTypeUser),
			Entry("role", ontology.ResourceTypeRole),
			Entry("policy", ontology.ResourceTypePolicy),
			Entry("builtin", ontology.ResourceTypeBuiltin),
		)

		DescribeTable("grants retrieve and update on racks, but not create or delete",
			func(ctx SpecContext, action access.Action, matcher OmegaMatcher) {
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Action:  action,
					Objects: []ontology.ID{
						{Type: ontology.ResourceTypeRack, Key: "test-key"},
					},
				})).To(matcher)
			},
			Entry("retrieve", access.ActionRetrieve, Succeed()),
			Entry("update", access.ActionUpdate, Succeed()),
			Entry("create", access.ActionCreate, MatchError(access.ErrDenied)),
			Entry("delete", access.ActionDelete, MatchError(access.ErrDenied)),
		)

		DescribeTable("grants only retrieve on channels",
			func(ctx SpecContext, action access.Action, matcher OmegaMatcher) {
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Action:  action,
					Objects: []ontology.ID{
						{Type: ontology.ResourceTypeChannel, Key: "test-key"},
					},
				})).To(matcher)
			},
			Entry("retrieve", access.ActionRetrieve, Succeed()),
			Entry("create", access.ActionCreate, MatchError(access.ErrDenied)),
			Entry("update", access.ActionUpdate, MatchError(access.ErrDenied)),
			Entry("delete", access.ActionDelete, MatchError(access.ErrDenied)),
		)

		DescribeTable("grants writes and reads on framers, but not deletes",
			func(ctx SpecContext, action access.Action, matcher OmegaMatcher) {
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Action:  action,
					Objects: []ontology.ID{
						{Type: ontology.ResourceTypeFramer, Key: "test-key"},
					},
				})).To(matcher)
			},
			Entry("create", access.ActionCreate, Succeed()),
			Entry("retrieve", access.ActionRetrieve, Succeed()),
			Entry("update", access.ActionUpdate, MatchError(access.ErrDenied)),
			Entry("delete", access.ActionDelete, MatchError(access.ErrDenied)),
		)

		It(
			"Should attach exactly the driver resource types to the edit policy",
			func(ctx SpecContext) {
				var p policy.Policy
				Expect(rbacSvc.Policy.NewRetrieve().
					Where(policy.MatchNames("Driver Edit Access")).
					Entry(&p).
					Exec(ctx, tx)).To(Succeed())
				Expect(p.Actions).To(ConsistOf(access.AllActions))
				Expect(p.Objects).To(ConsistOf(
					ontology.ID{Type: ontology.ResourceTypeRange},
					ontology.ID{Type: ontology.ResourceTypeDevice},
					ontology.ID{Type: ontology.ResourceTypeTask},
					ontology.ID{Type: ontology.ResourceTypeArc},
					ontology.ID{Type: ontology.ResourceTypeStatus},
				))
			},
		)
	})

	Describe("Idempotency", func() {
		It(
			"Should produce the same role keys when opened again",
			func(ctx SpecContext) {
				var ownerBefore role.Role
				Expect(
					rbacSvc.Role.NewRetrieve().
						Where(role.MatchNames("Owner")).
						Entry(&ownerBefore).
						Exec(ctx, tx),
				).To(Succeed())

				svc2 := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
					DB:       db,
					Ontology: otg,
					Group:    groupSvc,
					User:     userSvc,
				}))

				var ownerAfter role.Role
				Expect(
					svc2.Role.NewRetrieve().
						Where(role.MatchNames("Owner")).
						Entry(&ownerAfter).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(ownerAfter.Key).To(Equal(ownerBefore.Key))
			},
		)
	})

	Describe("Renames", func() {
		It("Should rename a stored Host role to Driver", func(ctx SpecContext) {
			var driver role.Role
			Expect(rbacSvc.Role.NewRetrieve().
				Where(role.MatchNames("Driver")).
				Entry(&driver).
				Exec(ctx, nil)).To(Succeed())
			var edit policy.Policy
			Expect(rbacSvc.Policy.NewRetrieve().
				Where(policy.MatchNames("Driver Edit Access")).
				Entry(&edit).
				Exec(ctx, nil)).To(Succeed())
			staleTx := db.OpenTx()
			Expect(gorp.NewUpdate[role.Key, role.Role]().
				Where(gorp.MatchKeys[role.Key, role.Role](driver.Key)).
				Change(func(_ gorp.Context, r role.Role) role.Role {
					r.Name = "Host"
					return r
				}).Exec(ctx, staleTx)).To(Succeed())
			Expect(gorp.NewUpdate[policy.Key, policy.Policy]().
				Where(gorp.MatchKeys[policy.Key, policy.Policy](edit.Key)).
				Change(func(_ gorp.Context, p policy.Policy) policy.Policy {
					p.Name = "Host Edit Access"
					return p
				}).Exec(ctx, staleTx)).To(Succeed())
			Expect(staleTx.Commit(ctx)).To(Succeed())

			svc2 := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				User:     userSvc,
			}))
			Expect(svc2.Builtin.DriverKey).To(Equal(driver.Key))
			var roles []role.Role
			Expect(svc2.Role.NewRetrieve().
				Where(role.MatchNames("Driver", "Host")).
				Entries(&roles).
				Exec(ctx, nil)).To(Succeed())
			Expect(roles).To(HaveLen(1))
			Expect(roles[0].Name).To(Equal("Driver"))
			var policies []policy.Policy
			Expect(svc2.Policy.NewRetrieve().
				Where(policy.MatchNames("Driver Edit Access", "Host Edit Access")).
				Entries(&policies).
				Exec(ctx, nil)).To(Succeed())
			Expect(policies).To(HaveLen(1))
			Expect(policies[0].Key).To(Equal(edit.Key))
			Expect(policies[0].Name).To(Equal("Driver Edit Access"))
		})
	})

	Describe("Custom roles", func() {
		It("Should not take over a custom role named Driver", func(ctx SpecContext) {
			custom := role.Role{Name: "Driver", Description: "custom"}
			Expect(rbacSvc.Role.NewWriter(nil, false).Create(ctx, &custom)).
				To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(rbacSvc.Role.NewWriter(nil, false).Delete(ctx, custom.Key)).
					To(Succeed())
			})
			svc2 := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				User:     userSvc,
			}))
			Expect(svc2.Builtin.DriverKey).ToNot(Equal(custom.Key))
			var stored role.Role
			Expect(svc2.Role.NewRetrieve().
				Where(role.MatchKeys(custom.Key)).
				Entry(&stored).
				Exec(ctx, nil)).To(Succeed())
			Expect(stored.Internal).To(BeFalse())
			Expect(stored.Description).To(Equal("custom"))
		})
	})

	Describe("Policy updates", func() {
		It(
			"Should update existing policy objects on re-provision",
			func(ctx SpecContext) {
				var ownerPolicy policy.Policy
				Expect(rbacSvc.Policy.NewRetrieve().
					Where(policy.MatchNames("Owner")).
					Entry(&ownerPolicy).
					Exec(ctx, nil)).To(Succeed())
				originalObjects := ownerPolicy.Objects
				Expect(originalObjects).ToNot(BeEmpty())

				// Simulate stale DB by stripping objects in a committed transaction
				staleTx := db.OpenTx()
				Expect(gorp.NewUpdate[policy.Key, policy.Policy]().
					Where(gorp.MatchKeys[policy.Key, policy.Policy](ownerPolicy.Key)).
					Change(func(_ gorp.Context, p policy.Policy) policy.Policy {
						p.Objects = p.Objects[:1]
						return p
					}).Exec(ctx, staleTx)).To(Succeed())
				Expect(staleTx.Commit(ctx)).To(Succeed())

				// Re-open service, which re-provisions
				svc2 := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
					DB:       db,
					Ontology: otg,
					Group:    groupSvc,
					User:     userSvc,
				}))

				var updated policy.Policy
				Expect(svc2.Policy.NewRetrieve().
					Where(policy.MatchNames("Owner")).
					Entry(&updated).
					Exec(ctx, nil)).To(Succeed())
				Expect(updated.Objects).To(Equal(originalObjects))
			},
		)
	})
})
