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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Grant", func() {
	var (
		db      *gorp.DB
		users   *user.Service
		rbacSvc *rbac.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		passwords := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		users = MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Password: passwords,
		}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
			DB:     db,
			User:   users,
			Status: statuses,
		})).RBAC
	})

	It("Should create a user that can do nothing", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			users,
			user.User{Name: "u"},
		))
		Expect(rbacSvc.NewEnforcer(db).Enforce(ctx, access.Request{
			Subject: u.ResourceID(),
			Action:  access.ActionRetrieve,
			Objects: []resource.ID{u.ResourceID()},
		})).To(MatchError(access.ErrDenied))
	})

	It("Should let the user take the actions on the objects", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			users,
			user.User{Name: "u"},
		))
		req := access.Request{
			Subject: u.ResourceID(),
			Action:  access.ActionUpdate,
			Objects: []resource.ID{{Type: resource.TypeChannel, Key: "1"}},
		}
		Expect(rbacSvc.NewEnforcer(db).Enforce(ctx, req)).
			To(MatchError(access.ErrDenied))
		p := MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			users,
			u.Key,
			[]access.Action{access.ActionUpdate},
			resource.ID{Type: resource.TypeChannel},
		))
		Expect(rbacSvc.NewEnforcer(db).Enforce(ctx, req)).To(Succeed())
		Expect(p.Role).To(Equal(u.Role))
	})

	It("Should add up the grants to one user", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			users,
			user.User{Name: "u"},
		))
		for _, a := range []access.Action{access.ActionUpdate, access.ActionDelete} {
			MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				rbacSvc,
				users,
				u.Key,
				[]access.Action{a},
				resource.ID{Type: resource.TypeChannel},
			))
		}
		for _, a := range []access.Action{access.ActionUpdate, access.ActionDelete} {
			Expect(rbacSvc.NewEnforcer(db).Enforce(ctx, access.Request{
				Subject: u.ResourceID(),
				Action:  a,
				Objects: []resource.ID{{Type: resource.TypeChannel, Key: "1"}},
			})).To(Succeed())
		}
	})

	It(
		"Should give a user that holds a shared role a role of its own",
		func(ctx SpecContext) {
			viewer := rbacSvc.Builtin.ViewerKey
			var u user.User
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				u, err = users.NewWriter(tx).
					Create(ctx, user.User{Name: "u", Role: viewer})
				return err
			})).To(Succeed())
			p := MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				rbacSvc,
				users,
				u.Key,
				[]access.Action{access.ActionUpdate},
				resource.ID{Type: resource.TypeChannel},
			))
			Expect(p.Role).ToNot(Equal(viewer))
			var stored user.User
			Expect(users.NewRetrieve().
				Where(user.MatchKeys(u.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Role).To(Equal(p.Role))
			var policies []policy.Policy
			Expect(rbacSvc.Policy.NewRetrieve().
				Where(policy.MatchRoles(viewer)).
				Entries(&policies).
				Exec(ctx, db)).To(Succeed())
			Expect(policies).To(HaveLen(1))
		},
	)

	It("Should return not found when the user does not exist", func(ctx SpecContext) {
		Expect(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			users,
			uuid.New(),
			[]access.Action{access.ActionUpdate},
			resource.ID{Type: resource.TypeChannel},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})
