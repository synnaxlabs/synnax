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
	"uuid"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func request(username, pwd string) apiauth.LoginRequest {
	creds := auth.Credentials{Username: username, Password: pwd}
	return apiauth.LoginRequest{Credentials: creds}
}

func requestCtx(ctx context.Context) freighter.Context {
	return freighter.Context{Context: ctx, Params: freighter.Params{}}
}

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apiauth.NewService(apiauth.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("auth: must be non-nil")))
	})
})

var _ = Describe("Login", func() {
	It(
		"Should return a token for the user and the cluster info",
		func(ctx SpecContext) {
			res := MustSucceed(
				apiSvc.Login(requestCtx(ctx), request(creds.Username, creds.Password)),
			)
			Expect(res.User).To(Equal(root))
			session := MustSucceed(authSvc.Authenticate(ctx, res.Token))
			Expect(session.Subject).To(Equal(root.OntologyID()))
			Expect(res.ClusterInfo.ClusterKey).To(Equal(node.Cluster.Key().String()))
			Expect(res.ClusterInfo.NodeKey).To(Equal(node.Cluster.HostKey()))
			Expect(res.ClusterInfo.NodeTime).ToNot(BeZero())
		},
	)
	It("Should reject a wrong password", func(ctx SpecContext) {
		Expect(
			apiSvc.Login(requestCtx(ctx), request(creds.Username, "wrong")),
		).Error().
			To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should reject an unknown username", func(ctx SpecContext) {
		Expect(
			apiSvc.Login(requestCtx(ctx), request("nobody", creds.Password)),
		).Error().
			To(MatchError(auth.ErrInvalidCredentials))
	})
})

var _ = Describe("ChangePassword", func() {
	var own auth.Credentials
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: uuid.New().String(), Password: "old"}
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Username: own.Username,
		}))
		Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), own)).To(Succeed())
	})
	It(
		"Should replace the password when the current one is right",
		func(ctx SpecContext) {
			Expect(
				apiSvc.ChangePassword(
					requestCtx(ctx),
					nil,
					apiauth.ChangePasswordRequest{
						Credentials: own,
						NewPassword: "new",
					},
				),
			).Error().
				ToNot(HaveOccurred())
			MustSucceed(
				apiSvc.Login(requestCtx(ctx), request(own.Username, "new")),
			)
			Expect(
				apiSvc.Login(requestCtx(ctx), request(own.Username, own.Password)),
			).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		},
	)
	It("Should keep the password when the current one is wrong", func(ctx SpecContext) {
		creds := auth.Credentials{Username: own.Username, Password: "wrong"}
		Expect(
			apiSvc.ChangePassword(requestCtx(ctx), nil, apiauth.ChangePasswordRequest{
				Credentials: creds,
				NewPassword: "new",
			}),
		).Error().
			To(MatchError(auth.ErrInvalidCredentials))
		MustSucceed(apiSvc.Login(requestCtx(ctx), request(own.Username, own.Password)))
	})
	It(
		"Should end the password's other sessions and keep the caller's",
		func(ctx SpecContext) {
			kept := MustSucceed(
				apiSvc.Login(requestCtx(ctx), request(own.Username, own.Password)),
			)
			other := MustSucceed(
				apiSvc.Login(requestCtx(ctx), request(own.Username, own.Password)),
			)
			fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
			fctx.Set("Session", MustSucceed(authSvc.Authenticate(ctx, kept.Token)).Key)
			Expect(apiSvc.ChangePassword(fctx, nil, apiauth.ChangePasswordRequest{
				Credentials: own,
				NewPassword: "new",
			})).Error().ToNot(HaveOccurred())
			MustSucceed(authSvc.Authenticate(ctx, kept.Token))
			Expect(authSvc.Authenticate(ctx, other.Token)).Error().
				To(MatchError(auth.ErrInvalidToken))
		},
	)
})

var _ = Describe("Middleware", func() {
	var (
		mw    freighter.Middleware
		token string
	)
	BeforeEach(func(ctx SpecContext) {
		mw = apiauth.Middleware(authSvc)
		token = MustSucceed(apiSvc.Login(requestCtx(ctx), request(
			creds.Username,
			creds.Password,
		))).Token
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
		Expect(apiauth.GetSubject(seen)).To(Equal(root.OntologyID()))
		Expect(apiauth.GetCredential(seen)).ToNot(Equal(uuid.Nil()))
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
