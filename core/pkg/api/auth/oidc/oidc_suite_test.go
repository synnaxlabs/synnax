// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oidc_test

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apioidc "github.com/synnaxlabs/synnax/pkg/api/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestOIDC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Auth OIDC Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	node        mock.Node
	otg         *ontology.Ontology
	authSvc     *auth.Service
	apikeySvc   *apikey.Service
	passwordSvc *password.Service
	rbacSvc     *rbac.Service
	oidcSvc     *oidc.Service
	provider    *testutil.Provider
	userSvc     *user.Service
	apiSvc      *apioidc.Service
	root        user.User
	creds       = auth.Credentials{Username: "api-oidc-suite-root", Password: "p"}
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = mock.NewNode(ctx)
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
	apikeySvc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
	provider = testutil.NewProvider()
	DeferCleanup(provider.Close)
	oidcSvc = MustOpen(oidc.OpenService(ctx, oidc.ServiceConfig{
		DB:          db,
		User:        userSvc,
		Role:        rbacSvc.Role,
		NewUserRole: rbacSvc.Builtin.ViewerKey,
		Providers: []oidc.Provider{{
			Name:     "idp",
			Issuer:   provider.Issuer(),
			ClientID: testutil.ClientID,
		}},
	}))
	authSvc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
		DB: db,
		Authenticators: map[auth.Method]auth.Authenticator{
			auth.MethodPassword: passwordSvc,
			auth.MethodOIDC:     oidcSvc,
			auth.MethodAPIKey:   apikeySvc,
		},
	}))
	apiSvc = newService(authSvc)
})

func newService(svc *auth.Service) *apioidc.Service {
	GinkgoHelper()
	return MustSucceed(apioidc.NewService(apioidc.ServiceConfig{
		Auth: svc,
		OIDC: oidcSvc,
		RBAC: rbacSvc,
	}))
}

func login(ctx context.Context, proof auth.Proof) string {
	GinkgoHelper()
	_, token := MustSucceed2(authSvc.Login(ctx, proof))
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
