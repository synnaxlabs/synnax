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
	"time"
	"uuid"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func requestCtx(ctx context.Context) freighter.Context {
	return freighter.Context{Context: ctx, Params: freighter.Params{}}
}

func passwordProof(username, pwd string) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{
		Username: username, Password: pwd,
	}}
}

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apiauth.NewService(apiauth.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("auth: must be non-nil")))
	})
})

var _ = Describe("HeldTables", func() {
	It("Should name the tables of the records that a user holds", func(
		ctx SpecContext,
	) {
		tx := DeferClose(db.OpenTx())
		k, _ := MustSucceed2(
			apiKeySvc.NewWriter(tx).Create(ctx, root.ResourceID(), "held"),
		)
		h := MustSucceed(passwordSvc.Hash(creds.Password))
		passwords := passwordSvc.NewWriter(tx)
		Expect(passwords.Set(ctx, root.ResourceID(), creds.Username, h)).To(Succeed())
		MustSucceed2(authSvc.NewWriter(tx).Open(ctx, root.ResourceID(), k.Key))
		MustSucceed(oidcSvc.NewWriter(tx).Create(
			ctx,
			root.Key,
			oidc.Account{
				Provider: "idp",
				Issuer:   provider.Issuer(),
				ID:       uuid.New().String(),
			},
		),
		)
		tables := lo.Map(gorp.ChangesOf(tx), func(c gorp.Change, _ int) string {
			return c.Record.Table
		})
		Expect(tables).To(ConsistOf(apiauth.HeldTables()))
	})
})

var _ = Describe("Methods", func() {
	It("Should list the methods the Core accepts and each provider's issuer", func(
		ctx SpecContext,
	) {
		Eventually(func() (apiauth.MethodsResponse, error) {
			return apiSvc.Methods(ctx, struct{}{})
		}).Should(Equal(apiauth.MethodsResponse{
			Methods: []auth.Method{
				auth.MethodAPIKey,
				auth.MethodOIDC,
				auth.MethodPassword,
			},
			Providers: []apiauth.OIDCLogin{{
				Provider:              "idp",
				Issuer:                provider.Issuer(),
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
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		svc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB:             other,
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
		mw    freighter.Middleware
		token string
	)
	BeforeEach(func(ctx SpecContext) {
		mw = apiauth.Middleware(authSvc)
		_, token = MustSucceed2(
			authSvc.Login(ctx, passwordProof(creds.Username, creds.Password)),
		)
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
		Expect(apiauth.GetSubject(seen)).To(Equal(root.ResourceID()))
	})
	It("Should put the session on a unary request", func(ctx SpecContext) {
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
		Expect(apiauth.GetSession(seen)).
			To(Equal(MustSucceed(authSvc.Authenticate(ctx, token))))
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

var _ = Describe("Deliver", func() {
	const token = "session-token"
	It("Should return the token unchanged", func(ctx SpecContext) {
		fctx := requestCtx(ctx)
		Expect(apiauth.Deliver(fctx, authSvc, token, false)).To(Equal(token))
		Expect(fctx.Params).ToNot(HaveKey(apiauth.SessionCookieParam))
	})
	It("Should put the token in the session cookie", func(ctx SpecContext) {
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		svc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: other,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodPassword: passwordSvc,
			},
			IdleTimeout: time.Hour,
		}))
		fctx := requestCtx(ctx)
		Expect(apiauth.Deliver(fctx, svc, token, true)).To(BeEmpty())
		Expect(MustBeOk(fctx.Get(apiauth.SessionCookieParam))).To(Equal(
			apiauth.SessionCookie{Token: token, Lifetime: time.Hour},
		))
	})
})
