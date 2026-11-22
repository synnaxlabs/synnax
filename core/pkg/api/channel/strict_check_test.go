// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Strict retrieve check", func() {
	refused := MatchError(ContainSubstring(
		"where: a strict retrieve takes only key, name, and path matches",
	))
	retrieve := func(ctx SpecContext, where *channel.Where) error {
		GinkgoHelper()
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
		)
		_, err := retrieveChannels(AuthedCtx(ctx, u), apichannel.RetrieveRequest{
			Where:  where,
			Strict: true,
		})
		return err
	}

	DescribeTable("Should refuse a strict retrieve with no where",
		func(ctx SpecContext, where *channel.Where) {
			Expect(retrieve(ctx, where)).To(refused)
		},
		Entry("no where", nil),
	)

	It("Should refuse an or inside a branch of the or", func(ctx SpecContext) {
		Expect(retrieve(ctx, &channel.Where{Or: []channel.Where{{
			Name: &filter.String{In: []string{"x"}},
			Or:   []channel.Where{{Name: &filter.String{In: []string{"y"}}}},
		}}})).To(refused)
	})
})
