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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func request(username, pwd string) apiauth.LoginRequest {
	creds := auth.Credentials{Username: username, Password: pwd}
	return apiauth.LoginRequest{Credentials: creds}
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
			Expect(res.User.Key).To(Equal(root.Key))
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
	var own auth.Credentials
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
			Auth:     svc,
			Password: passwordSvc,
			APIKey:   apiKeySvc,
			OIDC:     oidcSvc,
			Ontology: otg,
			User:     userSvc,
			RBAC:     rbacSvc,
			Cluster:  node.Cluster,
		}))
	}
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: uuid.New().String(), Password: "old"}
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: own.Username,
		}))
		Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), own)).To(Succeed())
	})
	It("Should leave a disabled method out of the list", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodPassword).Methods(ctx, struct{}{})).
			To(HaveField("Methods", []auth.Method{auth.MethodAPIKey, auth.MethodOIDC}))
	})
	It("Should list no providers while oidc is off", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodOIDC).Methods(ctx, struct{}{})).
			To(HaveField("Providers", BeEmpty()))
	})
	It("Should refuse to link a provider account", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodOIDC).LinkOIDC(
			requestCtx(ctx),
			nil,
			apiauth.OIDCLinkRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse a password login and keep the password", func(ctx SpecContext) {
		proof := passwordProof(own.Username, own.Password)
		Expect(disabled(ctx, auth.MethodPassword).CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: proof},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, proof))
	})
	It("Should refuse a password change and keep the password", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodPassword).ChangePassword(
			requestCtx(ctx),
			nil,
			apiauth.ChangePasswordRequest{Credentials: own, NewPassword: "new"},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, passwordProof(own.Username, "old")))
	})
	It("Should refuse to set a password and keep the old one", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodPassword).SetPassword(
			requestCtx(ctx),
			nil,
			apiauth.PasswordSetRequest{Password: "new", CurrentPassword: own.Password},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, passwordProof(own.Username, "old")))
	})
	It("Should refuse to create a password", func(ctx SpecContext) {
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		Expect(disabled(ctx, auth.MethodPassword).CreatePassword(
			requestCtx(ctx),
			nil,
			apiauth.PasswordCreateRequest{
				Subject:  u.OntologyID(),
				Username: u.Name, Password: "p",
			},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse a new API key", func(ctx SpecContext) {
		Expect(disabled(ctx, auth.MethodAPIKey).CreateKey(
			requestCtx(ctx),
			nil,
			apiauth.KeyCreateRequest{Name: "ci"},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("ChangePassword", func() {
	var own auth.Credentials
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: uuid.New().String(), Password: "old"}
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: own.Username,
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
			fctx.Set("Session", MustSucceed(authSvc.Authenticate(ctx, kept.Token)))
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

var _ = Describe("SetPassword", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	login := func(ctx context.Context, username, pwd string) string {
		GinkgoHelper()
		return MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: passwordProof(username, pwd)},
		)).Token
	}
	set := func(
		ctx context.Context,
		token string,
		req apiauth.PasswordSetRequest,
	) (struct{}, error) {
		return apiSvc.SetPassword(authed(ctx, token), nil, req)
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "old")
		bobToken = login(ctx, bob.Name, "old")
		rootToken = login(ctx, creds.Username, creds.Password)
	})
	It("Should set the caller's own password given the current one", func(
		ctx SpecContext,
	) {
		other := login(ctx, bob.Name, "old")
		Expect(set(ctx, bobToken, apiauth.PasswordSetRequest{
			Password:        "new",
			CurrentPassword: "old",
		})).Error().ToNot(HaveOccurred())
		login(ctx, bob.Name, "new")
		MustSucceed(authSvc.Authenticate(ctx, bobToken))
		Expect(authSvc.Authenticate(ctx, other)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	It("Should refuse the caller's own password with a wrong current one", func(
		ctx SpecContext,
	) {
		Expect(set(ctx, bobToken, apiauth.PasswordSetRequest{
			Password:        "new",
			CurrentPassword: "wrong",
		})).Error().To(MatchError(auth.ErrInvalidCredentials))
		login(ctx, bob.Name, "old")
	})
	It("Should refuse the caller's own password without the current one", func(
		ctx SpecContext,
	) {
		Expect(set(ctx, bobToken, apiauth.PasswordSetRequest{Password: "new"})).
			Error().To(MatchError(ContainSubstring("password: required")))
		login(ctx, bob.Name, "old")
	})
	It("Should let the owner set another user's password", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apiauth.PasswordSetRequest{
			Subject:  bob.OntologyID(),
			Password: "new",
		})).Error().ToNot(HaveOccurred())
		login(ctx, bob.Name, "new")
		Expect(authSvc.Authenticate(ctx, bobToken)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	It("Should deny a user setting another user's password", func(ctx SpecContext) {
		Expect(set(ctx, bobToken, apiauth.PasswordSetRequest{
			Subject:  root.OntologyID(),
			Password: "new",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject a subject that is not a user", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apiauth.PasswordSetRequest{
			Subject:  ontology.ID{Type: ontology.ResourceTypeRack, Key: "1"},
			Password: "new",
		})).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should report a user with no password as not found", func(ctx SpecContext) {
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		Expect(set(ctx, rootToken, apiauth.PasswordSetRequest{
			Subject:  u.OntologyID(),
			Password: "new",
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject an empty password", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apiauth.PasswordSetRequest{
			Subject: bob.OntologyID(),
		})).Error().To(MatchError(ContainSubstring("password: required")))
	})
})

var _ = Describe("CreatePassword", func() {
	var (
		bobToken  string
		rootToken string
		nopass    user.User
	)
	create := func(
		ctx context.Context,
		token string,
		req apiauth.PasswordCreateRequest,
	) (struct{}, error) {
		return apiSvc.CreatePassword(authed(ctx, token), nil, req)
	}
	login := func(ctx context.Context, username, pwd string) string {
		GinkgoHelper()
		return MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: passwordProof(username, pwd)},
		)).Token
	}
	BeforeEach(func(ctx SpecContext) {
		bob := createUser(ctx, "bob-"+uuid.New().String(), "old")
		bobToken = login(ctx, bob.Name, "old")
		rootToken = login(ctx, creds.Username, creds.Password)
		nopass = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
	})
	It("Should let the owner give a user its first password", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apiauth.PasswordCreateRequest{
			Subject:  nopass.OntologyID(),
			Username: nopass.Name, Password: "p",
		})).Error().ToNot(HaveOccurred())
		login(ctx, nopass.Name, "p")
	})
	It("Should deny a user giving another user a password", func(ctx SpecContext) {
		Expect(create(ctx, bobToken, apiauth.PasswordCreateRequest{
			Subject:  nopass.OntologyID(),
			Username: nopass.Name, Password: "p",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse a user that already has a password", func(ctx SpecContext) {
		Expect(create(ctx, bobToken, apiauth.PasswordCreateRequest{
			Username: "other", Password: "new",
		})).Error().To(MatchError(ContainSubstring("already has a password")))
	})
	It("Should refuse a username another user holds", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apiauth.PasswordCreateRequest{
			Subject:  nopass.OntologyID(),
			Username: creds.Username, Password: "p",
		})).Error().To(MatchError(auth.ErrRepeatedUsername))
	})
	It("Should reject a subject that is not a user", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apiauth.PasswordCreateRequest{
			Subject:  ontology.ID{Type: ontology.ResourceTypeRack, Key: "1"},
			Username: "rack", Password: "p",
		})).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should report a user that does not exist as not found", func(
		ctx SpecContext,
	) {
		Expect(create(ctx, rootToken, apiauth.PasswordCreateRequest{
			Subject:  user.OntologyID(uuid.New()),
			Username: "ghost", Password: "p",
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject an empty password", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apiauth.PasswordCreateRequest{
			Subject:  nopass.OntologyID(),
			Username: nopass.Name,
		})).Error().To(MatchError(ContainSubstring("password: required")))
	})
	It("Should reject an empty username", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apiauth.PasswordCreateRequest{
			Subject:  nopass.OntologyID(),
			Password: "p",
		})).Error().To(MatchError(ContainSubstring("username: required")))
	})
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

var _ = Describe("CreateSession", func() {
	It("Should open a session for a password proof", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: passwordProof(
				creds.Username,
				creds.Password,
			)},
		))
		session := MustSucceed(authSvc.Authenticate(ctx, res.Token))
		Expect(session.Key).To(Equal(res.Session.Key))
		Expect(res.Session.Subject).To(Equal(root.OntologyID()))
		Expect(res.Session.Credential).To(Equal(session.Credential))
		Expect(res.User.Key).To(Equal(root.Key))
	})
	DescribeTable("Should leave the hash off the wire", func(
		ctx SpecContext,
		codec encoding.Codec,
	) {
		res := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: passwordProof(
				creds.Username,
				creds.Password,
			)},
		))
		Expect(res.Session.Hash).To(HaveLen(32))
		var decoded map[string]any
		Expect(codec.Decode(
			ctx,
			MustSucceed(codec.Encode(ctx, res.Session)),
			&decoded,
		)).To(Succeed())
		Expect(decoded).To(HaveKey("key"))
		Expect(decoded).ToNot(HaveKey("hash"))
	},
		Entry("JSON", json.Codec),
		Entry("msgpack", msgpack.Codec),
	)
	It("Should reject a wrong password", func(ctx SpecContext) {
		Expect(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: passwordProof(creds.Username, "wrong")},
		)).Error().To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should reject a proof with no method", func(ctx SpecContext) {
		Expect(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("RenewSession", func() {
	It("Should return the caller's session and user", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		res := MustSucceed(apiSvc.RenewSession(authed(ctx, created.Token), struct{}{}))
		Expect(res.Session.Key).To(Equal(created.Session.Key))
		Expect(res.Session.Subject).To(Equal(root.OntologyID()))
		Expect(res.User).ToNot(BeNil())
		Expect(res.User.Key).To(Equal(root.Key))
	})
})

var _ = Describe("Session cookie", func() {
	It(
		"Should hand the token to the transport instead of the body",
		func(ctx SpecContext) {
			fctx := requestCtx(ctx)
			res := MustSucceed(apiSvc.CreateSession(fctx, apiauth.SessionCreateRequest{
				Proof:  passwordProof(creds.Username, creds.Password),
				Cookie: true,
			}))
			Expect(res.Token).To(BeEmpty())
			cookie := MustBeOk(fctx.Get(apiauth.SessionCookieParam)).(apiauth.SessionCookie)
			Expect(cookie.Token).To(HavePrefix("sys_"))
			Expect(cookie.Lifetime).To(Equal(authSvc.IdleTimeout()))
			Expect(MustSucceed(apiSvc.RenewSession(
				authed(ctx, cookie.Token),
				struct{}{},
			)).Session.Key).To(Equal(res.Session.Key))
		},
	)
	It(
		"Should ask the transport to refresh the cookie on every unary request",
		func(ctx SpecContext) {
			created := MustSucceed(apiSvc.CreateSession(
				requestCtx(ctx),
				apiauth.SessionCreateRequest{
					Proof: passwordProof(creds.Username, creds.Password),
				},
			))
			fctx := authed(ctx, created.Token)
			Expect(MustBeOk(fctx.Get(apiauth.SessionCookieParam))).
				To(Equal(apiauth.SessionCookie{Lifetime: authSvc.IdleTimeout()}))
		},
	)
	It("Should ask the transport to clear the cookie on logout", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		fctx := authed(ctx, created.Token)
		MustSucceed(apiSvc.DeleteSessions(fctx, apiauth.SessionDeleteRequest{
			Keys: []auth.SessionKey{created.Session.Key},
		}))
		Expect(MustBeOk(fctx.Get(apiauth.SessionCookieParam))).
			To(Equal(apiauth.SessionCookie{}))
	})
	It("Should keep the cookie when ending other sessions", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		other := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		fctx := authed(ctx, created.Token)
		MustSucceed(apiSvc.DeleteSessions(fctx, apiauth.SessionDeleteRequest{
			Keys: []auth.SessionKey{other.Session.Key},
		}))
		Expect(MustBeOk(fctx.Get(apiauth.SessionCookieParam))).
			To(Equal(apiauth.SessionCookie{Lifetime: authSvc.IdleTimeout()}))
	})
})

var _ = Describe("Sessions", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	login := func(ctx context.Context, username, pwd string) apiauth.SessionCreateResponse {
		GinkgoHelper()
		return MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: passwordProof(username, pwd)},
		))
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, bob.Name, "p").Token
		rootToken = login(ctx, creds.Username, creds.Password).Token
	})
	Describe("RetrieveSessions", func() {
		It("Should list the caller's own sessions by default", func(ctx SpecContext) {
			res := MustSucceed(apiSvc.RetrieveSessions(
				authed(ctx, bobToken),
				apiauth.SessionRetrieveRequest{},
			))
			Expect(res.Sessions).To(HaveLen(1))
			Expect(res.Sessions[0].Subject).To(Equal(bob.OntologyID()))
		})
		It(
			"Should let the owner list another subject's sessions",
			func(ctx SpecContext) {
				res := MustSucceed(apiSvc.RetrieveSessions(
					authed(ctx, rootToken),
					apiauth.SessionRetrieveRequest{
						Subjects: []ontology.ID{bob.OntologyID()},
					},
				))
				Expect(res.Sessions).To(HaveLen(1))
				Expect(res.Sessions[0].Subject).To(Equal(bob.OntologyID()))
			},
		)
		It(
			"Should return an empty list for a subject with no sessions",
			func(ctx SpecContext) {
				idle := createUser(ctx, "idle-"+uuid.New().String(), "p")
				res := MustSucceed(apiSvc.RetrieveSessions(
					authed(ctx, rootToken),
					apiauth.SessionRetrieveRequest{
						Subjects: []ontology.ID{idle.OntologyID()},
					},
				))
				Expect(res.Sessions).ToNot(BeNil())
				Expect(res.Sessions).To(BeEmpty())
			},
		)
		It(
			"Should deny a user listing another subject's sessions",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveSessions(
					authed(ctx, bobToken),
					apiauth.SessionRetrieveRequest{
						Subjects: []ontology.ID{root.OntologyID()},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
	Describe("DeleteSessions", func() {
		It("Should end the caller's own session", func(ctx SpecContext) {
			key := MustSucceed(authSvc.Authenticate(ctx, bobToken)).Key
			Expect(apiSvc.DeleteSessions(
				authed(ctx, bobToken),
				apiauth.SessionDeleteRequest{Keys: []auth.SessionKey{key}},
			)).Error().ToNot(HaveOccurred())
			Expect(authSvc.Authenticate(ctx, bobToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It(
			"Should deny a user ending another subject's session",
			func(ctx SpecContext) {
				key := MustSucceed(authSvc.Authenticate(ctx, rootToken)).Key
				Expect(apiSvc.DeleteSessions(
					authed(ctx, bobToken),
					apiauth.SessionDeleteRequest{Keys: []auth.SessionKey{key}},
				)).Error().To(MatchError(access.ErrDenied))
				MustSucceed(authSvc.Authenticate(ctx, rootToken))
			},
		)
		It("Should let the owner end another subject's session", func(ctx SpecContext) {
			key := MustSucceed(authSvc.Authenticate(ctx, bobToken)).Key
			Expect(apiSvc.DeleteSessions(
				authed(ctx, rootToken),
				apiauth.SessionDeleteRequest{Keys: []auth.SessionKey{key}},
			)).Error().ToNot(HaveOccurred())
			Expect(authSvc.Authenticate(ctx, bobToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(apiSvc.DeleteSessions(
				authed(ctx, bobToken),
				apiauth.SessionDeleteRequest{Keys: []auth.SessionKey{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})
})
