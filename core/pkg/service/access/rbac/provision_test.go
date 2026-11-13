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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Provision", func() {
	var tx gorp.Tx

	Describe("Built-in roles", func() {
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It(
			"Should have created all built-in roles during OpenRoles",
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
				var policies []policy.Policy
				Expect(rbacSvc.Policy.NewRetrieve().
					Where(policy.MatchRoles(r.Key)).
					Entries(&policies).
					Exec(ctx, tx)).To(Succeed())
				Expect(policies).ToNot(BeEmpty())
			}
		})
	})

	Describe("Driver role", func() {
		var subject resource.ID
		BeforeEach(func() {
			tx = DeferClose(db.OpenTx())
			subject = rackSvc.EmbeddedKey.ResourceID()
		})

		DescribeTable("grants full access to driver resource types",
			func(ctx SpecContext, t resource.Type) {
				for _, action := range access.AllActions {
					Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
						Subject: subject,
						Action:  action,
						Objects: []resource.ID{{Type: t, Key: "test-key"}},
					})).To(Succeed())
				}
			},
			Entry("range", resource.TypeRange),
			Entry("device", resource.TypeDevice),
			Entry("NI chassis", resource.TypeNiChassis),
			Entry("NI device", resource.TypeNiDevice),
			Entry("LabJack device", resource.TypeLabjackDevice),
			Entry("EtherCAT master", resource.TypeEthercatMaster),
			Entry("EtherCAT slave", resource.TypeEthercatSlave),
			Entry("OPC UA server", resource.TypeOpcuaServer),
			Entry("Modbus server", resource.TypeModbusServer),
			Entry("HTTP server", resource.TypeHTTPServer),
			Entry("task", resource.TypeTask),
			Entry("arc", resource.TypeArc),
			Entry("status", resource.TypeStatus),
		)

		DescribeTable("denies access to non-driver resource types",
			func(ctx SpecContext, t resource.Type) {
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Action:  access.ActionRetrieve,
					Objects: []resource.ID{{Type: t, Key: "test-key"}},
				})).To(MatchError(access.ErrDenied))
			},
			Entry("label", resource.TypeLabel),
			Entry("log", resource.TypeLog),
			Entry("node", resource.TypeNode),
			Entry("group", resource.TypeGroup),
			Entry("project", resource.TypeProject),
			Entry("schematic", resource.TypeSchematic),
			Entry("lineplot", resource.TypeLineplot),
			Entry("table", resource.TypeTable),
			Entry("schematic symbol", resource.TypeSchematicSymbol),
			Entry("view", resource.TypeView),
			Entry("user", resource.TypeUser),
			Entry("role", resource.TypeRole),
			Entry("policy", resource.TypePolicy),
			Entry("builtin", resource.TypeBuiltin),
		)

		DescribeTable("grants retrieve and update on racks, but not create or delete",
			func(ctx SpecContext, action access.Action, matcher OmegaMatcher) {
				Expect(rbacSvc.NewEnforcer(tx).Enforce(ctx, access.Request{
					Subject: subject,
					Action:  action,
					Objects: []resource.ID{
						{Type: resource.TypeRack, Key: "test-key"},
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
					Objects: []resource.ID{
						{Type: resource.TypeChannel, Key: "test-key"},
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
					Objects: []resource.ID{
						{Type: resource.TypeFramer, Key: "test-key"},
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
					resource.ID{Type: resource.TypeRange},
					resource.ID{Type: resource.TypeDevice},
					resource.ID{Type: resource.TypeEthercatMaster},
					resource.ID{Type: resource.TypeEthercatSlave},
					resource.ID{Type: resource.TypeHTTPServer},
					resource.ID{Type: resource.TypeLabjackDevice},
					resource.ID{Type: resource.TypeModbusServer},
					resource.ID{Type: resource.TypeNiChassis},
					resource.ID{Type: resource.TypeNiDevice},
					resource.ID{Type: resource.TypeOpcuaServer},
					resource.ID{Type: resource.TypeTask},
					resource.ID{Type: resource.TypeArc},
					resource.ID{Type: resource.TypeStatus},
				))
			},
		)
	})

	Describe("Idempotency", func() {
		It(
			"Should produce the same role keys when opened again",
			func(ctx SpecContext) {
				cfg := newRolesConfig(ctx)
				svc1 := MustSucceed(rbac.OpenRoles(ctx, cfg))
				var ownerBefore role.Role
				Expect(
					svc1.Role.NewRetrieve().
						Where(role.MatchNames("Owner")).
						Entry(&ownerBefore).
						Exec(ctx, cfg.DB),
				).To(Succeed())
				Expect(svc1.Close()).To(Succeed())

				svc2 := MustOpen(rbac.OpenRoles(ctx, cfg))
				var ownerAfter role.Role
				Expect(
					svc2.Role.NewRetrieve().
						Where(role.MatchNames("Owner")).
						Entry(&ownerAfter).
						Exec(ctx, cfg.DB),
				).To(Succeed())
				Expect(ownerAfter.Key).To(Equal(ownerBefore.Key))
			},
		)
	})

	Describe("Renames", func() {
		It("Should rename a stored Host role to Driver", func(ctx SpecContext) {
			cfg := newRolesConfig(ctx)
			svc1 := MustSucceed(rbac.OpenRoles(ctx, cfg))
			var driver role.Role
			Expect(svc1.Role.NewRetrieve().
				Where(role.MatchNames("Driver")).
				Entry(&driver).
				Exec(ctx, cfg.DB)).To(Succeed())
			var edit policy.Policy
			Expect(svc1.Policy.NewRetrieve().
				Where(policy.MatchNames("Driver Edit Access")).
				Entry(&edit).
				Exec(ctx, cfg.DB)).To(Succeed())
			Expect(svc1.Close()).To(Succeed())
			staleTx := cfg.DB.OpenTx()
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

			svc2 := MustOpen(rbac.OpenRoles(ctx, cfg))
			Expect(svc2.Builtin.DriverKey).To(Equal(driver.Key))
			var roles []role.Role
			Expect(svc2.Role.NewRetrieve().
				Where(role.MatchNames("Driver", "Host")).
				Entries(&roles).
				Exec(ctx, cfg.DB)).To(Succeed())
			Expect(roles).To(HaveLen(1))
			Expect(roles[0].Name).To(Equal("Driver"))
			var policies []policy.Policy
			Expect(svc2.Policy.NewRetrieve().
				Where(policy.MatchNames("Driver Edit Access", "Host Edit Access")).
				Entries(&policies).
				Exec(ctx, cfg.DB)).To(Succeed())
			Expect(policies).To(HaveLen(1))
			Expect(policies[0].Key).To(Equal(edit.Key))
			Expect(policies[0].Name).To(Equal("Driver Edit Access"))
		})
	})

	Describe("Custom roles", func() {
		It("Should not take over a custom role named Driver", func(ctx SpecContext) {
			cfg := newRolesConfig(ctx)
			svc1 := MustSucceed(rbac.OpenRoles(ctx, cfg))
			custom := role.Role{Name: "Driver", Description: "custom"}
			Expect(cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc1.Role.NewWriter(tx, false).Create(ctx, &custom)
			})).
				To(Succeed())
			Expect(svc1.Close()).To(Succeed())
			svc2 := MustOpen(rbac.OpenRoles(ctx, cfg))
			Expect(svc2.Builtin.DriverKey).ToNot(Equal(custom.Key))
			var stored role.Role
			Expect(svc2.Role.NewRetrieve().
				Where(role.MatchKeys(custom.Key)).
				Entry(&stored).
				Exec(ctx, cfg.DB)).To(Succeed())
			Expect(stored.Internal).To(BeFalse())
			Expect(stored.Description).To(Equal("custom"))
		})
	})

	Describe("Policy updates", func() {
		It(
			"Should update existing policy objects on re-provision",
			func(ctx SpecContext) {
				cfg := newRolesConfig(ctx)
				svc1 := MustSucceed(rbac.OpenRoles(ctx, cfg))
				var ownerPolicy policy.Policy
				Expect(svc1.Policy.NewRetrieve().
					Where(policy.MatchNames("Owner")).
					Entry(&ownerPolicy).
					Exec(ctx, cfg.DB)).To(Succeed())
				Expect(svc1.Close()).To(Succeed())
				originalObjects := ownerPolicy.Objects
				Expect(originalObjects).ToNot(BeEmpty())

				// Simulate stale DB by stripping objects in a committed transaction
				staleTx := cfg.DB.OpenTx()
				Expect(gorp.NewUpdate[policy.Key, policy.Policy]().
					Where(gorp.MatchKeys[policy.Key, policy.Policy](ownerPolicy.Key)).
					Change(func(_ gorp.Context, p policy.Policy) policy.Policy {
						p.Objects = p.Objects[:1]
						return p
					}).Exec(ctx, staleTx)).To(Succeed())
				Expect(staleTx.Commit(ctx)).To(Succeed())

				// Re-open service, which re-provisions
				svc2 := MustOpen(rbac.OpenRoles(ctx, cfg))

				var updated policy.Policy
				Expect(svc2.Policy.NewRetrieve().
					Where(policy.MatchNames("Owner")).
					Entry(&updated).
					Exec(ctx, cfg.DB)).To(Succeed())
				Expect(updated.Objects).To(Equal(originalObjects))
			},
		)
	})
})
