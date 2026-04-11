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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Provision", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("Built-in roles", func() {
		It("Should have created all built-in roles during OpenService", func(ctx SpecContext) {
			for _, name := range []string{"Owner", "Engineer", "Operator", "Viewer"} {
				var r role.Role
				Expect(svc.Role.NewRetrieve().Where(role.MatchNames(name)).Entry(&r).Exec(ctx, tx)).To(Succeed())
				Expect(r.Key).ToNot(Equal(uuid.Nil))
				Expect(r.Internal).To(BeTrue())
			}
		})
		It("Should have created policies for each role", func(ctx SpecContext) {
			for _, name := range []string{"Owner", "Engineer", "Operator", "Viewer"} {
				var r role.Role
				Expect(svc.Role.NewRetrieve().Where(role.MatchNames(name)).Entry(&r).Exec(ctx, tx)).To(Succeed())
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

	Describe("Idempotency", func() {
		It("Should produce the same role keys when opened again", func(ctx SpecContext) {
			var ownerBefore role.Role
			Expect(svc.Role.NewRetrieve().Where(role.MatchNames("Owner")).Entry(&ownerBefore).Exec(ctx, tx)).To(Succeed())

			svc2 := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
				User:     userSvc,
			}))
			defer func() { Expect(svc2.Close()).To(Succeed()) }()

			var ownerAfter role.Role
			Expect(svc2.Role.NewRetrieve().Where(role.MatchNames("Owner")).Entry(&ownerAfter).Exec(ctx, tx)).To(Succeed())
			Expect(ownerAfter.Key).To(Equal(ownerBefore.Key))
		})
	})

	Describe("Policy updates", func() {
		It("Should update existing policy objects on re-provision", func(ctx SpecContext) {
			var ownerPolicy policy.Policy
			Expect(svc.Policy.NewRetrieve().
				Where(policy.MatchNames("Owner")).
				Entry(&ownerPolicy).
				Exec(ctx, nil)).To(Succeed())
			originalObjects := ownerPolicy.Objects
			Expect(originalObjects).ToNot(BeEmpty())

			// Simulate stale DB by stripping objects in a committed transaction
			staleTx := db.OpenTx()
			Expect(gorp.NewUpdate[uuid.UUID, policy.Policy]().
				WhereKeys(ownerPolicy.Key).
				Change(func(_ gorp.Context, p policy.Policy) policy.Policy {
					p.Objects = p.Objects[:1]
					return p
				}).Exec(ctx, staleTx)).To(Succeed())
			Expect(staleTx.Commit(ctx)).To(Succeed())

			// Re-open service, which re-provisions
			svc2 := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
				User:     userSvc,
			}))
			defer func() { Expect(svc2.Close()).To(Succeed()) }()

			var updated policy.Policy
			Expect(svc2.Policy.NewRetrieve().
				Where(policy.MatchNames("Owner")).
				Entry(&updated).
				Exec(ctx, nil)).To(Succeed())
			Expect(updated.Objects).To(Equal(originalObjects))
		})
	})
})
