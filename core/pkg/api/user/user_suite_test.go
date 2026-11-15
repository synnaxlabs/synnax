// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiuser "github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPIUser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API User Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db          *gorp.DB
	passwordSvc *password.Service
	userSvc     *user.Service
	apiSvc      *apiuser.Service
	rbacSvc     *rbac.Service
	root        user.User
	// The handlers below run their routes as the API layer binds them.
	create   freighter.UnaryHandler[apiuser.CreateRequest, apiuser.CreateResponse]
	del      freighter.UnaryHandler[apiuser.DeleteRequest, struct{}]
	dispatch freighter.UnaryHandler[apiuser.DispatchRequest, struct{}]
	retrieve freighter.UnaryHandler[
		apiuser.RetrieveRequest,
		apiuser.RetrieveResponse,
	]
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
		DB: db,
		Authenticators: map[auth.Method]auth.Authenticator{
			auth.MethodPassword: passwordSvc,
		},
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Password: passwordSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     db,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	MustSucceed2(
		user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "api-user-suite-root", Password: "p"},
			rbacSvc.Builtin.OwnerKey,
		),
	)
	apiSvc = MustSucceed(apiuser.NewService(apiuser.ServiceConfig{
		DB:       db,
		RBAC:     rbacSvc,
		User:     userSvc,
		Password: passwordSvc,
	}))
	root = findRoot(ctx, userSvc)
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: rbacSvc,
		Held: apiauth.HeldTables(),
	}))
	create = bind.WritePrepared(binder, apiSvc.PrepareCreate, apiSvc.Create)
	del = bind.Write(binder, apiSvc.Delete)
	dispatch = bind.Write(binder, apiSvc.Dispatch)
	retrieve = bind.Read(binder, apiSvc.Retrieve)
})

func login(ctx context.Context, username, pwd string) (auth.Identity, error) {
	return passwordSvc.Authenticate(ctx, auth.Proof{
		Variant: auth.PasswordProof{
			Credentials: auth.Credentials{Username: username, Password: pwd},
		},
	})
}

func findRoot(ctx SpecContext, svc *user.Service) user.User {
	GinkgoHelper()
	var u user.User
	Expect(
		svc.NewRetrieve().Where(user.MatchRootUser(true)).Entry(&u).Exec(ctx, db),
	).To(Succeed())
	return u
}

// rootCtx returns a freighter.Context with the suite-level root user installed as the
// request subject. The root user holds the Owner role, so every RBAC enforcement call
// passes.
func rootCtx(ctx SpecContext) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", root.ResourceID())
	return fctx
}

// hash hashes plaintext with svc.
func hash(svc *password.Service, plaintext string) password.Hash {
	GinkgoHelper()
	return MustSucceed(svc.Hash(plaintext))
}
