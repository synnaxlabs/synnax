// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password_test

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apipassword "github.com/synnaxlabs/synnax/pkg/api/auth/password"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPassword(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Auth Password Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	node        testutil.Node
	authSvc     *auth.Service
	passwordSvc *password.Service
	apiKeySvc   *apikey.Service
	rbacSvc     *rbac.Service
	otg         *ontology.Ontology
	userSvc     *user.Service
	apiSvc      *apipassword.Service
	root        user.User
	creds       = auth.Credentials{Username: "api-password-suite-root", Password: "p"}
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db := node.DB
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc,
	}))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	root, _ = MustSucceed2(user.ProvisionRoot(ctx, userSvc, passwordSvc, creds))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		User:     userSvc,
	}))
	apiKeySvc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
	authSvc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
		DB: db,
		Authenticators: map[auth.Method]auth.Authenticator{
			auth.MethodPassword: passwordSvc,
		},
	}))
	apiSvc = newService(authSvc)
})

func newService(svc *auth.Service) *apipassword.Service {
	GinkgoHelper()
	return MustSucceed(apipassword.NewService(apipassword.ServiceConfig{
		Auth:     svc,
		Password: passwordSvc,
		Ontology: otg,
		RBAC:     rbacSvc,
	}))
}

func login(ctx context.Context, username, pwd string) string {
	GinkgoHelper()
	_, token := MustSucceed2(authSvc.Login(ctx, passwordProof(username, pwd)))
	return token
}

func requestCtx(ctx context.Context) freighter.Context {
	return freighter.Context{Context: ctx, Params: freighter.Params{}}
}

func authed(ctx context.Context, token string) freighter.Context {
	GinkgoHelper()
	fctx := requestCtx(ctx)
	fctx.Variant = freighter.VariantUnary
	fctx.Set(fiber.HeaderAuthorization, "Bearer "+token)
	var seen freighter.Context
	MustSucceed(apiauth.Middleware(authSvc).Exec(fctx, freighter.FinalizerFunc(
		func(c freighter.Context) (freighter.Context, error) {
			seen = c
			return c, nil
		},
	)))
	return seen
}

func createUser(ctx context.Context, username, pwd string) user.User {
	GinkgoHelper()
	u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Name: username}))
	Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), auth.Credentials{
		Username: username,
		Password: pwd,
	})).To(Succeed())
	return u
}

func passwordProof(username, pwd string) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{
		Username: username, Password: pwd,
	}}
}
