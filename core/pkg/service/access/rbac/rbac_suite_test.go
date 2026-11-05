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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	db          *gorp.DB
	otg         *ontology.Ontology
	groupSvc    *group.Service
	rbacSvc     *rbac.Service
	userSvc     *user.Service
	passwordSvc *password.Service
	rackSvc     *rack.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "suite-root", Password: "p"},
		),
	)
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		User:     userSvc,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       statusSvc,
		DriverRole:   rbacSvc.Builtin.DriverKey,
	}))
})

// newConfig opens a database and the services an RBAC service on it needs, apart from
// the suite's database.
func newConfig(ctx context.Context) rbac.ServiceConfig {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
	users := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	return rbac.ServiceConfig{DB: db, Ontology: otg, User: users}
}

func TestRBAC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Access RBAC Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
