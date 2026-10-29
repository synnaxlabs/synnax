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

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func requestCtx(ctx context.Context) freighter.Context {
	return freighter.Context{Context: ctx, Params: freighter.Params{}}
}

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apiauth.NewService(apiauth.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("auth: must be non-nil")))
	})
})

var _ = Describe("Methods", func() {
	It("Should list the methods the Core accepts", func(ctx SpecContext) {
		Expect(apiSvc.Methods(ctx, struct{}{})).To(Equal(apiauth.MethodsResponse{
			Methods: []auth.Method{
				auth.MethodAPIKey,
				auth.MethodOIDC,
				auth.MethodPassword,
			},
			Providers: []apiauth.OIDCLogin{{
				Provider:              "idp",
				AuthorizationEndpoint: provider.Issuer() + "/authorize",
				ClientID:              testutil.ClientID,
			}},
		}))
	})
})

var _ = Describe("Disabled methods", func() {
	disabled := func(ctx SpecContext, method auth.Method) *apiauth.Service {
		GinkgoHelper()
		authenticators := map[auth.Method]auth.Authenticator{
			auth.MethodPassword: passwordSvc,
			auth.MethodAPIKey:   apiKeySvc,
			auth.MethodOIDC:     oidcSvc,
		}
		delete(authenticators, method)
		svc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB:             node.DB,
			Authenticators: authenticators,
		}))
		return MustSucceed(apiauth.NewService(apiauth.ServiceConfig{
			Auth: svc,
			OIDC: oidcSvc,
		}))
	}
	It("Should leave a disabled method out of the list", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodPassword).Methods(ctx, struct{}{})).
			To(HaveField("Methods", []auth.Method{auth.MethodAPIKey, auth.MethodOIDC}))
	})
	It("Should list no providers while oidc is off", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodOIDC).Methods(ctx, struct{}{})).
			To(HaveField("Providers", BeEmpty()))
	})
})

var _ = Describe("Middleware", func() {
	var (
		mw      freighter.Middleware
		token   string
		session auth.Session
	)
	BeforeEach(func(ctx SpecContext) {
		mw = apiauth.Middleware(authSvc)
		session, token = MustSucceed2(authSvc.Login(ctx, auth.Proof{
			Variant: auth.PasswordProof{Credentials: creds},
		}))
	})
	withToken := func(ctx context.Context, variant freighter.Variant) freighter.Context {
		fctx := requestCtx(ctx)
		fctx.Variant = variant
		fctx.Set(fiber.HeaderAuthorization, "Bearer "+token)
		return fctx
	}
	exec := func(fctx freighter.Context, next freighter.Next) error {
		_, err := mw.Exec(fctx, next)
		return err
	}
	passthrough := freighter.FinalizerFunc(
		func(c freighter.Context) (freighter.Context, error) { return c, nil },
	)
	It("Should put the subject on a unary request", func(ctx SpecContext) {
		var seen freighter.Context
		MustSucceed(mw.Exec(
			withToken(ctx, freighter.VariantUnary),
			freighter.FinalizerFunc(
				func(c freighter.Context) (freighter.Context, error) {
					seen = c
					return c, nil
				},
			),
		))
		Expect(apiauth.GetSubject(seen)).To(Equal(root.OntologyID()))
		Expect(apiauth.GetSession(seen).Key).To(Equal(session.Key))
	})
	It("Should reject a request without a token", func(ctx SpecContext) {
		Expect(mw.Exec(requestCtx(ctx), passthrough)).Error().
			To(MatchError(auth.ErrAuth))
	})
	It("Should close a stream when its session ends", func(ctx SpecContext) {
		key := MustSucceed(authSvc.Authenticate(ctx, token)).Key
		errs := make(chan error, 1)
		var oCtx freighter.Context
		go func() {
			defer GinkgoRecover()
			var err error
			oCtx, err = mw.Exec(
				withToken(ctx, freighter.VariantStream),
				freighter.FinalizerFunc(
					func(c freighter.Context) (freighter.Context, error) {
						Eventually(c.Done()).Should(BeClosed())
						return c, c.Err()
					},
				),
			)
			errs <- err
		}()
		Consistently(errs).ShouldNot(Receive())
		Expect(authSvc.Logout(ctx, key)).To(Succeed())
		Eventually(errs).Should(Receive(MatchError(auth.ErrSessionEnded)))
		Expect(oCtx).To(BeZero())
	})
	It("Should pass a stream's own error through", func(ctx SpecContext) {
		errBoom := errors.New("boom")
		Expect(exec(
			withToken(ctx, freighter.VariantStream),
			freighter.FinalizerFunc(
				func(c freighter.Context) (freighter.Context, error) {
					return c, errBoom
				},
			),
		)).To(MatchError(errBoom))
	})
})
