// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Auth Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db          *gorp.DB
	node        testutil.Node
	authSvc     *auth.Service
	passwordSvc *password.Service
	apiKeySvc   *apikey.Service
	rbacSvc     *rbac.Service
	userSvc     *user.Service
	rackSvc     *rack.Service
	apiSvc      *apiauth.Service
	// These handlers run their routes as the API layer binds them.
	changePassword freighter.UnaryHandler[apiauth.ChangePasswordRequest, struct{}]
	setPassword    freighter.UnaryHandler[
		apiauth.PasswordSetRequest,
		apiauth.PasswordSetResponse,
	]
	createKey freighter.UnaryHandler[
		apiauth.KeyCreateRequest,
		apiauth.KeyCreateResponse,
	]
	deleteKeys     freighter.UnaryHandler[apiauth.KeyDeleteRequest, struct{}]
	deleteSessions freighter.UnaryHandler[apiauth.SessionDeleteRequest, struct{}]
	retrieveKeys   freighter.UnaryHandler[
		apiauth.KeyRetrieveRequest,
		apiauth.KeyRetrieveResponse,
	]
	retrieveSessions freighter.UnaryHandler[
		apiauth.SessionRetrieveRequest,
		apiauth.SessionRetrieveResponse,
	]
	root  user.User
	creds = auth.Credentials{Username: "api-auth-suite-root", Password: "p"}
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db = node.DB
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Password: passwordSvc,
	}))
	apiKeySvc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	access := MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	}))
	root, _ = MustSucceed2(user.ProvisionRoot(
		ctx,
		userSvc,
		passwordSvc,
		creds,
		access.RBAC.Builtin.OwnerKey,
	))
	rbacSvc, rackSvc = access.RBAC, access.Rack
	authSvc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
		DB: db,
		Authenticators: map[auth.Method]auth.Authenticator{
			auth.MethodPassword: passwordSvc,
			auth.MethodAPIKey:   apiKeySvc,
		},
	}))
	apiSvc = MustSucceed(apiauth.NewService(apiauth.ServiceConfig{
		DB:       db,
		Auth:     authSvc,
		Password: passwordSvc,
		APIKey:   apiKeySvc,
		User:     userSvc,
		RBAC:     rbacSvc,
		Identity: node.Identity,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{DB: db, RBAC: rbacSvc}))
	changePassword = bind.WritePrepared(
		binder,
		apiSvc.PrepareChangePassword,
		apiSvc.ChangePassword,
	)
	setPassword = bind.WritePrepared(
		binder,
		apiSvc.PrepareSetPassword,
		apiSvc.SetPassword,
	)
	createKey = bind.Write(binder, apiSvc.CreateKey)
	retrieveKeys = bind.Read(binder, func(
		ctx context.Context,
		snap bind.Snapshot,
		req apiauth.KeyRetrieveRequest,
	) (apiauth.KeyRetrieveResponse, error) {
		return apiSvc.RetrieveKeys(ctx, snap, req)
	})
	retrieveSessions = bind.Read(binder, func(
		ctx context.Context,
		snap bind.Snapshot,
		req apiauth.SessionRetrieveRequest,
	) (apiauth.SessionRetrieveResponse, error) {
		return apiSvc.RetrieveSessions(ctx, snap, req)
	})
	deleteKeys = bind.Write(binder, apiSvc.DeleteKeys)
	deleteSessions = bind.Write(binder, apiSvc.DeleteSessions)
})
