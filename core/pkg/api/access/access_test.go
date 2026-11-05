// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiaccess "github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("RetrievePolicy", func() {
	It("Should retrieve the policies of the roles a subject holds", func(
		ctx SpecContext,
	) {
		res := MustSucceed(apiSvc.RetrievePolicy(
			rootCtx(ctx),
			apiaccess.RetrievePolicyRequest{Subjects: []resource.ID{root.ResourceID()}},
		))
		Expect(res.Policies).ToNot(BeEmpty())
		Expect(res.Policies).To(HaveEach(HaveField("Role", rbacSvc.Builtin.OwnerKey)))
	})

	It("Should retrieve the policies of a role", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.RetrievePolicy(
			rootCtx(ctx),
			apiaccess.RetrievePolicyRequest{
				Roles: []role.Key{rbacSvc.Builtin.ViewerKey},
			},
		))
		Expect(res.Policies).ToNot(BeEmpty())
		Expect(res.Policies).To(HaveEach(HaveField("Role", rbacSvc.Builtin.ViewerKey)))
	})

	It("Should retrieve no policies for a subject with no roles", func(
		ctx SpecContext,
	) {
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Name: "none"}))
		res := MustSucceed(apiSvc.RetrievePolicy(
			rootCtx(ctx),
			apiaccess.RetrievePolicyRequest{Subjects: []resource.ID{u.ResourceID()}},
		))
		Expect(res.Policies).To(BeEmpty())
	})
})
