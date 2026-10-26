// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apirole "github.com/synnaxlabs/synnax/pkg/api/access/role"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var roleType = ontology.ID{Type: ontology.ResourceTypeRole}

var _ = Describe("Retrieve", func() {
	It("Should return the roles the request selects when retrieve is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		r := grantOn(ctx, u.OntologyID(), access.ActionRetrieve, roleType)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apirole.RetrieveRequest{
			Where: &role.Where{Key: &filter.Set[role.Key]{Eq: &r.Key}},
		}))
		Expect(res.Roles).To(HaveExactElements(HaveField("Key", r.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := createRole(ctx)
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, createUser(ctx)), apirole.RetrieveRequest{
			Where: &role.Where{Key: &filter.Set[role.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Assign", func() {
	It("Should reject the request when update is not granted on the user", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		target := createUser(ctx)
		r := createRole(ctx)
		Expect(apiSvc.Assign(
			AuthedCtx(ctx, u), nil,
			apirole.AssignRequest{User: target.Key, Role: r.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Unassign", func() {
	It("Should reject the request when update is not granted on the user", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		target := createUser(ctx)
		r := createRole(ctx)
		Expect(svc.RBAC.Role.NewWriter(nil, true).AssignRole(
			ctx, target.OntologyID(), r.Key,
		)).To(Succeed())
		Expect(apiSvc.Unassign(
			AuthedCtx(ctx, u), nil,
			apirole.UnassignRequest{User: target.Key, Role: r.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
})
