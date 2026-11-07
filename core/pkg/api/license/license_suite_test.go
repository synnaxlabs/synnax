// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apilicense "github.com/synnaxlabs/synnax/pkg/api/license"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPILicense(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API License Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	rbacSvc    *rbac.Service
	licenseSvc *license.Service
	apiSvc     *apilicense.Service
	userSvc    *user.Service
	signer     svctestutil.Signer
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	kvDB := memkv.New()
	db = DeferClose(gorp.Wrap(kvDB))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Password: passwordSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:       db,
		Ontology: otg,
		User:     userSvc,
		Status:   statusSvc,
	})).RBAC
	signer = svctestutil.NewSigner()
	licenseSvc = MustOpen(license.OpenService(ctx, license.ServiceConfig{
		DB: kvDB, Anchors: signer.Anchors,
	}))
	apiSvc = MustSucceed(apilicense.NewService(apilicense.ServiceConfig{
		RBAC:    rbacSvc,
		License: licenseSvc,
	}))
})

// freshUser creates a user with no role assignments.
func freshUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(
		userSvc.NewWriter(nil).
			Create(ctx, user.User{Name: "anon-" + uuid.New().String()}),
	)
}

// grantOn grants the actions on the objects to the subject through a new role.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(rbactestutil.Grant(ctx, rbacSvc, userSvc, key, actions, objects...))
}
