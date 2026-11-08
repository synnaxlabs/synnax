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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var roleType = resource.ID{Type: resource.TypeRole}

var _ = Describe("Retrieve", func() {
	It("Should return the roles the request selects when retrieve is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		r := grantOn(ctx, u.ResourceID(), access.ActionRetrieve, roleType)
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
