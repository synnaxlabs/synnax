// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package session_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	full := func() session.ServiceConfig {
		return session.ServiceConfig{
			DB:       &gorp.DB{},
			Auth:     &auth.Service{},
			Password: &password.Service{},
			User:     &user.Service{},
			Identity: node.Identity,
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(unset func(*session.ServiceConfig), msg string) {
				cfg := full()
				unset(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"db",
				func(c *session.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"auth",
				func(c *session.ServiceConfig) { c.Auth = nil },
				"auth: must be non-nil",
			),
			Entry(
				"password",
				func(c *session.ServiceConfig) { c.Password = nil },
				"password: must be non-nil",
			),
			Entry(
				"user",
				func(c *session.ServiceConfig) { c.User = nil },
				"user: must be non-nil",
			),
			Entry(
				"identity key",
				func(c *session.ServiceConfig) { c.Identity.Key = uuid.Nil() },
				"identity.key: must be set",
			),
			Entry(
				"identity host key",
				func(c *session.ServiceConfig) { c.Identity.HostKey = 0 },
				"identity.host_key: must be non-zero",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := session.ServiceConfig{}.Override(cfg)
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.Password).To(BeIdenticalTo(cfg.Password))
			Expect(merged.User).To(BeIdenticalTo(cfg.User))
			Expect(merged.Identity).To(Equal(cfg.Identity))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(session.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.Password).To(BeIdenticalTo(cfg.Password))
			Expect(merged.User).To(BeIdenticalTo(cfg.User))
			Expect(merged.Identity).To(Equal(cfg.Identity))
		})
	})
})

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(session.NewService(session.ServiceConfig{})).Error().
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
			expectLegacyUser(res.User, creds.Username)
			opened := MustSucceed(authSvc.Authenticate(ctx, res.Token))
			Expect(opened.Subject).To(Equal(root.ResourceID()))
			Expect(res.ClusterInfo.ClusterKey).To(Equal(node.Identity.Key.String()))
			Expect(res.ClusterInfo.NodeKey).To(Equal(node.Identity.HostKey))
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

var _ = Describe("Disabled password method", func() {
	It("Should refuse a password login and keep the password", func(ctx SpecContext) {
		own := auth.Credentials{Username: uuid.New().String(), Password: "old"}
		createUser(ctx, own.Username, own.Password)
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		disabled := newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: other,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodAPIKey: apiKeySvc,
			},
		})))
		proof := passwordProof(own.Username, own.Password)
		Expect(disabled.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: proof},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, proof))
	})
})

var _ = Describe("Create", func() {
	It("Should open a session for a password proof", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(
				creds.Username,
				creds.Password,
			)},
		))
		opened := MustSucceed(authSvc.Authenticate(ctx, res.Token))
		Expect(opened.Key).To(Equal(res.Session.Key))
		Expect(res.Session.Subject).To(Equal(root.ResourceID()))
		Expect(res.Session.Credential).To(Equal(opened.Credential))
		Expect(res.User.Key).To(Equal(root.Key))
	})
	It("Should send the user fields that 0.60 clients require", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(
				creds.Username,
				creds.Password,
			)},
		))
		expectLegacyUser(*res.User, creds.Username)
	})
	It("Should send an empty username for a user with no password", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		created := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: keyProof(createKey(ctx, u.ResourceID()))},
		))
		expectLegacyUser(*created.User, "")
		renewed := MustSucceed(apiSvc.Renew(authed(ctx, created.Token), struct{}{}))
		expectLegacyUser(*renewed.User, "")
	})
	It("Should leave the hash off the wire", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(
				creds.Username,
				creds.Password,
			)},
		))
		Expect(res.Session.Hash).To(HaveLen(32))
		var decoded map[string]any
		Expect(json.Codec.Decode(
			MustSucceed(json.Codec.Encode(res.Session)),
			&decoded,
		)).To(Succeed())
		Expect(decoded).To(HaveKey("key"))
		Expect(decoded).ToNot(HaveKey("hash"))
	})
	It("Should reject a wrong password", func(ctx SpecContext) {
		Expect(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(creds.Username, "wrong")},
		)).Error().To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should reject a proof with no method", func(ctx SpecContext) {
		Expect(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("Renew", func() {
	It("Should return the caller's session and user", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		res := MustSucceed(apiSvc.Renew(authed(ctx, created.Token), struct{}{}))
		Expect(res.Session.Key).To(Equal(created.Session.Key))
		Expect(res.Session.Subject).To(Equal(root.ResourceID()))
		Expect(res.User).ToNot(BeNil())
		Expect(res.User.Key).To(Equal(root.Key))
		expectLegacyUser(*res.User, creds.Username)
	})
})

var _ = Describe("Session cookie", func() {
	It(
		"Should hand the token to the transport instead of the body",
		func(ctx SpecContext) {
			fctx := requestCtx(ctx)
			res := MustSucceed(apiSvc.Create(fctx, session.CreateRequest{
				Proof:  passwordProof(creds.Username, creds.Password),
				Cookie: true,
			}))
			Expect(res.Token).To(BeEmpty())
			v := MustBeOk(fctx.Get(apiauth.SessionCookieParam))
			cookie, ok := v.(apiauth.SessionCookie)
			Expect(ok).To(BeTrue())
			Expect(cookie.Token).To(HavePrefix("sys_"))
			Expect(cookie.Lifetime).To(Equal(authSvc.IdleTimeout()))
			Expect(MustSucceed(apiSvc.Renew(
				authed(ctx, cookie.Token),
				struct{}{},
			)).Session.Key).To(Equal(res.Session.Key))
		},
	)
	It(
		"Should ask the transport to refresh the cookie on every unary request",
		func(ctx SpecContext) {
			created := MustSucceed(apiSvc.Create(
				requestCtx(ctx),
				session.CreateRequest{
					Proof: passwordProof(creds.Username, creds.Password),
				},
			))
			fctx := authed(ctx, created.Token)
			Expect(MustBeOk(fctx.Get(apiauth.SessionCookieParam))).
				To(Equal(apiauth.SessionCookie{Lifetime: authSvc.IdleTimeout()}))
		},
	)
	It("Should ask the transport to clear the cookie on logout", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		fctx := authed(ctx, created.Token)
		MustSucceed(deleteSessions(fctx, session.DeleteRequest{
			Keys: []auth.SessionKey{created.Session.Key},
		}))
		Expect(MustBeOk(fctx.Get(apiauth.SessionCookieParam))).
			To(Equal(apiauth.SessionCookie{}))
	})
	It("Should keep the cookie when ending other sessions", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		other := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		fctx := authed(ctx, created.Token)
		MustSucceed(deleteSessions(fctx, session.DeleteRequest{
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
	login := func(ctx context.Context, username, pwd string) session.CreateResponse {
		GinkgoHelper()
		return MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(username, pwd)},
		))
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, bob.Name, "p").Token
		rootToken = login(ctx, creds.Username, creds.Password).Token
	})
	Describe("Retrieve", func() {
		It("Should list the caller's own sessions by default", func(ctx SpecContext) {
			res := MustSucceed(retrieveSessions(
				authed(ctx, bobToken),
				session.RetrieveRequest{},
			))
			Expect(res.Sessions).To(HaveLen(1))
			Expect(res.Sessions[0].Subject).To(Equal(bob.ResourceID()))
		})
		It(
			"Should let the owner list another subject's sessions",
			func(ctx SpecContext) {
				res := MustSucceed(retrieveSessions(
					authed(ctx, rootToken),
					session.RetrieveRequest{
						Subjects: []resource.ID{bob.ResourceID()},
					},
				))
				Expect(res.Sessions).To(HaveLen(1))
				Expect(res.Sessions[0].Subject).To(Equal(bob.ResourceID()))
			},
		)
		It(
			"Should return an empty list for a subject with no sessions",
			func(ctx SpecContext) {
				idle := createUser(ctx, "idle-"+uuid.New().String(), "p")
				res := MustSucceed(retrieveSessions(
					authed(ctx, rootToken),
					session.RetrieveRequest{
						Subjects: []resource.ID{idle.ResourceID()},
					},
				))
				Expect(res.Sessions).ToNot(BeNil())
				Expect(res.Sessions).To(BeEmpty())
			},
		)
		It(
			"Should deny a user listing another subject's sessions",
			func(ctx SpecContext) {
				Expect(retrieveSessions(
					authed(ctx, bobToken),
					session.RetrieveRequest{
						Subjects: []resource.ID{root.ResourceID()},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
	Describe("Delete", func() {
		It("Should end the caller's own session", func(ctx SpecContext) {
			key := MustSucceed(authSvc.Authenticate(ctx, bobToken)).Key
			Expect(deleteSessions(
				authed(ctx, bobToken),
				session.DeleteRequest{Keys: []auth.SessionKey{key}},
			)).Error().ToNot(HaveOccurred())
			Expect(authSvc.Authenticate(ctx, bobToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It(
			"Should deny a user ending another subject's session",
			func(ctx SpecContext) {
				key := MustSucceed(authSvc.Authenticate(ctx, rootToken)).Key
				Expect(deleteSessions(
					authed(ctx, bobToken),
					session.DeleteRequest{Keys: []auth.SessionKey{key}},
				)).Error().To(MatchError(access.ErrDenied))
				MustSucceed(authSvc.Authenticate(ctx, rootToken))
			},
		)
		It(
			"Should let a caller that may delete on a subject end its session",
			func(ctx SpecContext) {
				eng := createUser(ctx, "eng-"+uuid.New().String(), "p")
				engToken := login(ctx, eng.Name, "p").Token
				key := MustSucceed(authSvc.Authenticate(ctx, engToken)).Key
				grant(ctx, bob.Key, access.ActionDelete, eng.ResourceID())
				Expect(deleteSessions(
					authed(ctx, bobToken),
					session.DeleteRequest{Keys: []auth.SessionKey{key}},
				)).Error().ToNot(HaveOccurred())
				Expect(authSvc.Authenticate(ctx, engToken)).Error().
					To(MatchError(auth.ErrInvalidToken))
			},
		)
		It("Should let the owner end another subject's session", func(ctx SpecContext) {
			key := MustSucceed(authSvc.Authenticate(ctx, bobToken)).Key
			Expect(deleteSessions(
				authed(ctx, rootToken),
				session.DeleteRequest{Keys: []auth.SessionKey{key}},
			)).Error().ToNot(HaveOccurred())
			Expect(authSvc.Authenticate(ctx, bobToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(deleteSessions(
				authed(ctx, bobToken),
				session.DeleteRequest{Keys: []auth.SessionKey{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})
})
