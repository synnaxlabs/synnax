// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Grant", func() {
	It("Should let the user take the actions on the objects", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		users := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groups,
		}))
		rbacSvc := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
			DB:       db,
			Ontology: otg,
			User:     users,
		}))
		u := MustSucceed(users.NewWriter(nil).Create(ctx, user.User{Name: "u"}))
		req := access.Request{
			Subject: u.ResourceID(),
			Action:  access.ActionUpdate,
			Objects: []resource.ID{{Type: resource.TypeChannel, Key: "1"}},
		}
		Expect(rbacSvc.NewEnforcer(nil).Enforce(ctx, req)).
			To(MatchError(access.ErrDenied))
		r := MustSucceed(rbactestutil.Grant(
			ctx,
			rbacSvc,
			users,
			u.Key,
			[]access.Action{access.ActionUpdate},
			resource.ID{Type: resource.TypeChannel},
		))
		Expect(rbacSvc.NewEnforcer(nil).Enforce(ctx, req)).To(Succeed())
		var stored user.User
		Expect(users.NewRetrieve().
			Where(user.MatchKeys(u.Key)).
			Entry(&stored).
			Exec(ctx, nil)).To(Succeed())
		Expect(stored.Roles).To(Equal([]role.Key{r}))
	})
})
