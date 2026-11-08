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
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	db          *gorp.DB
	groupSvc    *group.Service
	rbacSvc     *rbac.Service
	roles       *rbac.Roles
	userSvc     *user.Service
	passwordSvc *password.Service
	rackSvc     *rack.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
		),
	)
	roles = MustOpen(rbac.OpenRoles(ctx, rbac.RolesConfig{DB: db}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           db,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       statusSvc,
		DriverRole:   roles.Builtin.DriverKey,
	}))
	rbacSvc = MustSucceed(rbac.NewService(ctx, rbac.ServiceConfig{
		DB:    db,
		Roles: roles,
		User:  userSvc,
		Rack:  rackSvc,
	}))
})

// newRolesConfig opens a database apart from the suite's database.
func newRolesConfig(ctx context.Context) rbac.RolesConfig {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	return rbac.RolesConfig{DB: db}
}

func TestRBAC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Access RBAC Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
