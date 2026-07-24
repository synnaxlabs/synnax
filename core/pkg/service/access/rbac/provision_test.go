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

var _ = Describe("Provision", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })

	Describe("Built-in roles", func() {
		It("Should have created all built-in roles during OpenService", func(ctx SpecContext) {
			for _, name := range []string{"Owner", "Engineer", "Host", "Operator", "Viewer"} {
				var r role.Role
				Expect(rbacSvc.Role.NewRetrieve().Where(role.MatchNames(name)).Entry(&r).Exec(ctx, tx)).To(Succeed())
				Expect(r.Key).ToNot(Equal(uuid.Nil))
				Expect(r.Internal).To(BeTrue())
			}
		})
		It("Should have created policies for each role", func(ctx SpecContext) {
			for _, name := range []string{"Owner", "Engineer", "Host", "Operator", "Viewer"} {
				var r role.Role
				Expect(rbacSvc.Role.NewRetrieve().Where(role.MatchNames(name)).Entry(&r).Exec(ctx, tx)).To(Succeed())
				var policies []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(role.OntologyID(r.Key)).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&policies).
					Exec(ctx, tx)).To(Succeed())
				Expect(policies).ToNot(BeEmpty())
			}
		})
	})

	Describe("Host role", func() {
		var subject ontology.ID
		BeforeEach(func(ctx SpecContext) {
			var r role.Role
			Expect(rbacSvc.Role.NewRetrieve().Where(role.MatchNames("Host")).Entry(&r).Exec(ctx, tx)).To(Succeed())
			subject = ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(otg.NewWriter(tx).DefineResource(ctx, subject)).To(Succeed())
			Expect(rbacSvc.Role.NewWriter(tx, true).AssignRole(ctx, subject, r.Key)).To(Succeed())
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
			Entry("channel", ontology.ResourceTypeChannel),
			Entry("range", ontology.ResourceTypeRange),
			Entry("framer", ontology.ResourceTypeFramer),
			Entry("range alias", ontology.ResourceTypeRangeAlias),
			Entry("rack", ontology.ResourceTypeRack),
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
			Entry("label", ontology.ResourceTypeLabel),
			Entry("log", ontology.ResourceTypeLog),
			Entry("node", ontology.ResourceTypeNode),
			Entry("group", ontology.ResourceTypeGroup),
			Entry("workspace", ontology.ResourceTypeWorkspace),
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

		It("Should attach exactly the driver resource types to the edit policy", func(ctx SpecContext) {
			var p policy.Policy
			Expect(rbacSvc.Policy.NewRetrieve().
				Where(policy.MatchNames("Host Edit Access")).
				Entry(&p).
				Exec(ctx, tx)).To(Succeed())
			Expect(p.Actions).To(ConsistOf(access.AllActions))
			Expect(p.Objects).To(ConsistOf(
				ontology.ID{Type: ontology.ResourceTypeChannel},
				ontology.ID{Type: ontology.ResourceTypeRange},
				ontology.ID{Type: ontology.ResourceTypeFramer},
				ontology.ID{Type: ontology.ResourceTypeRangeAlias},
				ontology.ID{Type: ontology.ResourceTypeRack},
				ontology.ID{Type: ontology.ResourceTypeDevice},
				ontology.ID{Type: ontology.ResourceTypeTask},
				ontology.ID{Type: ontology.ResourceTypeArc},
				ontology.ID{Type: ontology.ResourceTypeStatus},
			))
		})
	})

	Describe("Idempotency", func() {
		It("Should produce the same role keys when opened again", func(ctx SpecContext) {
			var ownerBefore role.Role
			Expect(rbacSvc.Role.NewRetrieve().Where(role.MatchNames("Owner")).Entry(&ownerBefore).Exec(ctx, tx)).To(Succeed())

			svc2 := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				Search:   searchIdx,
				User:     userSvc,
			}))

			var ownerAfter role.Role
			Expect(svc2.Role.NewRetrieve().Where(role.MatchNames("Owner")).Entry(&ownerAfter).Exec(ctx, tx)).To(Succeed())
			Expect(ownerAfter.Key).To(Equal(ownerBefore.Key))
		})
	})

	Describe("Policy updates", func() {
		It("Should update existing policy objects on re-provision", func(ctx SpecContext) {
			var ownerPolicy policy.Policy
			Expect(rbacSvc.Policy.NewRetrieve().
				Where(policy.MatchNames("Owner")).
				Entry(&ownerPolicy).
				Exec(ctx, nil)).To(Succeed())
			originalObjects := ownerPolicy.Objects
			Expect(originalObjects).ToNot(BeEmpty())

			// Simulate stale DB by stripping objects in a committed transaction
			staleTx := db.OpenTx()
			Expect(gorp.NewUpdate[uuid.UUID, policy.Policy]().
				Where(gorp.MatchKeys[uuid.UUID, policy.Policy](ownerPolicy.Key)).
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
				Search:   searchIdx,
				User:     userSvc,
			}))

			var updated policy.Policy
			Expect(svc2.Policy.NewRetrieve().
				Where(policy.MatchNames("Owner")).
				Entry(&updated).
				Exec(ctx, nil)).To(Succeed())
			Expect(updated.Objects).To(Equal(originalObjects))
		})
	})
})
