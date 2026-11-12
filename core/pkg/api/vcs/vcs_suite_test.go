// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apivcs "github.com/synnaxlabs/synnax/pkg/api/vcs"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIVCS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API VCS Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// env is a service layer and the version control API over it.
type env struct {
	db    *gorp.DB
	layer *service.Layer
	api   *apivcs.Service
}

// shared holds version control that never starts, so each route that needs it started
// returns vcs.ErrNotStarted once access allows it.
var shared env

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	shared = openEnv(ctx)
})

// openEnv opens a node, a service layer on it, and the API. They close when the
// current node of the spec tree ends.
func openEnv(ctx context.Context) env {
	GinkgoHelper()
	node := testutil.NewNode(ctx)
	layer := MustOpen(svctestutil.OpenLayer(ctx, node))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   node.DB,
		RBAC: layer.RBAC,
		Held: apiauth.HeldTables(),
	}))
	api := MustSucceed(apivcs.NewService(apivcs.ServiceConfig{
		DB:       node.DB,
		RBAC:     layer.RBAC,
		Binder:   binder,
		VCS:      layer.VCS,
		User:     layer.User,
		Password: layer.Password,
	}))
	return env{db: node.DB, layer: layer, api: api}
}

// createUser creates a user that can do nothing.
func (e env) createUser(ctx context.Context) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(
		ctx,
		e.db,
		e.layer.RBAC,
		e.layer.User,
		user.User{Name: "user-" + uuid.New().String()},
	))
}

// createUserWithRole creates a user that holds the role with the given key.
func (e env) createUserWithRole(ctx context.Context, key role.Key) user.User {
	GinkgoHelper()
	u := user.User{Key: uuid.New(), Name: "user-" + uuid.New().String(), Role: key}
	Expect(e.db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		u, err = e.layer.User.NewWriter(tx).Create(ctx, u)
		return err
	})).To(Succeed())
	return u
}

// grant grants the action on the objects to u.
func (e env) grant(
	ctx context.Context,
	u user.User,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	MustSucceed(rbactestutil.Grant(
		ctx,
		e.db,
		e.layer.RBAC,
		e.layer.User,
		u.Key,
		[]access.Action{action},
		objects...,
	))
}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}
