// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package builtin_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Provision", func() {
	var (
		db        *gorp.DB
		policySvc *policy.Service
		roleSvc   *role.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		roleSvc = MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		policySvc = MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
			DB:   db,
			Role: roleSvc,
		}))
	})
	provision := func(ctx context.Context) builtin.ProvisionResult {
		GinkgoHelper()
		return MustSucceed(builtin.Provision(ctx, db, policySvc, roleSvc))
	}
	retrieveRole := func(ctx context.Context, key role.Key) role.Role {
		GinkgoHelper()
		var r role.Role
		Expect(roleSvc.NewRetrieve().
			Where(role.MatchKeys(key)).
			Entry(&r).
			Exec(ctx, db)).To(Succeed())
		return r
	}

	It("Should provision the Driver role", func(ctx SpecContext) {
		r := retrieveRole(ctx, provision(ctx).DriverKey)
		Expect(r.Name).To(Equal("Driver"))
		Expect(r.Internal).To(BeTrue())
	})

	DescribeTable(
		"Should grant access to secrets and certificates",
		func(
			ctx SpecContext,
			name string,
			typ resource.Type,
			actions []access.Action,
		) {
			provision(ctx)
			var p policy.Policy
			Expect(policySvc.NewRetrieve().
				Where(policy.MatchNames(name)).
				Entry(&p).
				Exec(ctx, db)).To(Succeed())
			Expect(p.Objects).To(ContainElement(resource.ID{Type: typ}))
			Expect(p.Actions).To(Equal(actions))
		},
		Entry("Owner secrets", "Owner", resource.TypeSecret, access.AllActions),
		Entry(
			"Engineer certificates",
			"Engineer Edit Access",
			resource.TypeCertificate,
			access.AllActions,
		),
		Entry(
			"Operator secrets",
			"Operator View Access",
			resource.TypeSecret,
			[]access.Action{access.ActionRetrieve},
		),
		Entry(
			"Viewer certificates",
			"Viewer",
			resource.TypeCertificate,
			[]access.Action{access.ActionRetrieve},
		),
		Entry(
			"Driver secrets",
			"Driver Secret View Access",
			resource.TypeSecret,
			[]access.Action{access.ActionRetrieve},
		),
		Entry(
			"Driver certificates",
			"Driver Certificate Access",
			resource.TypeCertificate,
			[]access.Action{access.ActionCreate, access.ActionRetrieve},
		),
	)

	It("Should keep the same keys when it runs again", func(ctx SpecContext) {
		Expect(provision(ctx)).To(Equal(provision(ctx)))
	})

	It("Should rename a stored Host role and its policies", func(ctx SpecContext) {
		first := provision(ctx)
		var edit policy.Policy
		Expect(policySvc.NewRetrieve().
			Where(policy.MatchNames("Driver Edit Access")).
			Entry(&edit).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewUpdate[role.Key, role.Role]().
			Where(gorp.MatchKeys[role.Key, role.Role](first.DriverKey)).
			Change(func(_ gorp.Context, r role.Role) role.Role {
				r.Name = "Host"
				return r
			}).Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewUpdate[policy.Key, policy.Policy]().
			Where(gorp.MatchKeys[policy.Key, policy.Policy](edit.Key)).
			Change(func(_ gorp.Context, p policy.Policy) policy.Policy {
				p.Name = "Host Edit Access"
				return p
			}).Exec(ctx, db)).To(Succeed())
		second := provision(ctx)
		Expect(second.DriverKey).To(Equal(first.DriverKey))
		Expect(retrieveRole(ctx, second.DriverKey).Name).To(Equal("Driver"))
		var renamed policy.Policy
		Expect(policySvc.NewRetrieve().
			Where(policy.MatchKeys(edit.Key)).
			Entry(&renamed).
			Exec(ctx, db)).To(Succeed())
		Expect(renamed.Name).To(Equal("Driver Edit Access"))
	})

	It("Should restore the actions of a stored built-in policy", func(ctx SpecContext) {
		provision(ctx)
		var edit policy.Policy
		Expect(policySvc.NewRetrieve().
			Where(policy.MatchNames("Driver Edit Access")).
			Entry(&edit).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewUpdate[policy.Key, policy.Policy]().
			Where(gorp.MatchKeys[policy.Key, policy.Policy](edit.Key)).
			Change(func(_ gorp.Context, p policy.Policy) policy.Policy {
				p.Actions = []access.Action{access.ActionRetrieve}
				return p
			}).Exec(ctx, db)).To(Succeed())
		provision(ctx)
		var restored policy.Policy
		Expect(policySvc.NewRetrieve().
			Where(policy.MatchKeys(edit.Key)).
			Entry(&restored).
			Exec(ctx, db)).To(Succeed())
		Expect(restored).To(Equal(edit))
	})

	It("Should leave a custom role named Driver alone", func(ctx SpecContext) {
		custom := role.Role{Name: "Driver", Description: "custom"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return roleSvc.NewWriter(tx, false).Create(ctx, &custom)
		})).To(Succeed())
		result := provision(ctx)
		Expect(result.DriverKey).ToNot(Equal(custom.Key))
		Expect(retrieveRole(ctx, custom.Key).Description).To(Equal("custom"))
	})
})
