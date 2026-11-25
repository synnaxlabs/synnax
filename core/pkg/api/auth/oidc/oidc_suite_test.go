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
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apioidc "github.com/synnaxlabs/synnax/pkg/api/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestOIDC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Auth OIDC Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db          *gorp.DB
	node        disttestutil.Node
	authSvc     *auth.Service
	passwordSvc *password.Service
	apiKeySvc   *apikey.Service
	rbacSvc     *rbac.Service
	userSvc     *user.Service
	rackSvc     *rack.Service
	oidcSvc     *oidc.Service
	provider    *testutil.Provider
	apiSvc      *apioidc.Service
	// These handlers run their routes as the API layer binds them.
	linkOIDC          freighter.UnaryHandler[apioidc.LinkRequest, apioidc.LinkResponse]
	retrieveOIDCLinks freighter.UnaryHandler[
		apioidc.RetrieveRequest,
		apioidc.RetrieveResponse,
	]
	unlinkOIDC freighter.UnaryHandler[apioidc.UnlinkRequest, struct{}]
	root       user.User
	creds      = auth.Credentials{Username: "api-oidc-suite-root", Password: "p"}
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = disttestutil.NewNode(ctx)
	db = node.DB
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db,
	}))
	apiKeySvc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	fixture := MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	}))
	root, _ = MustSucceed2(user.ProvisionRoot(
		ctx,
		userSvc,
		passwordSvc,
		creds,
		fixture.RBAC.Builtin.OwnerKey,
	))
	rbacSvc, rackSvc = fixture.RBAC, fixture.Rack
	provider = testutil.NewProvider()
	DeferCleanup(provider.Close)
	oidcSvc = MustOpen(oidc.OpenService(ctx, oidc.ServiceConfig{
		DB:          db,
		User:        userSvc,
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
			auth.MethodAPIKey:   apiKeySvc,
			auth.MethodOIDC:     oidcSvc,
		},
	}))
	apiSvc = newService(authSvc)
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		Holders: taskconfig.NewHolders(),
		DB:      db,
		RBAC:    rbacSvc,
		Held:    apiauth.HeldTables(),
	}))
	linkOIDC = bind.WritePrepared(binder, apiSvc.PrepareLink, apiSvc.Link)
	retrieveOIDCLinks = bind.Read(binder, func(
		ctx context.Context,
		snap bind.Snapshot,
		req apioidc.RetrieveRequest,
	) (apioidc.RetrieveResponse, error) {
		return apiSvc.Retrieve(ctx, snap, req)
	})
	unlinkOIDC = bind.Write(binder, apiSvc.Unlink)
})

func newService(svc *auth.Service) *apioidc.Service {
	GinkgoHelper()
	return MustSucceed(apioidc.NewService(apioidc.ServiceConfig{
		Auth: svc,
		OIDC: oidcSvc,
	}))
}

// login opens a session with proof and returns its token.
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
	u := MustSucceed(
		rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{Name: username}),
	)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return passwordSvc.NewWriter(tx).Set(
			ctx,
			u.ResourceID(),
			username,
			MustSucceed(passwordSvc.Hash(pwd)),
		)
	})).To(Succeed())
	return u
}

// createKey creates an API key for subject and returns its token.
func createKey(ctx context.Context, subject resource.ID) string {
	GinkgoHelper()
	var token string
	Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		_, token, err = apiKeySvc.NewWriter(tx).Create(ctx, subject, "spec")
		return err
	})).To(Succeed())
	return token
}

func passwordProof(username, pwd string) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{
		Username: username, Password: pwd,
	}}
}

func keyProof(key string) auth.Proof {
	return auth.Proof{Variant: auth.APIKeyProof{
		APIKeyCredentials: auth.APIKeyCredentials{Key: key},
	}}
}
