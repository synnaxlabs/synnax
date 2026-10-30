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
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func request(username, pwd string) session.LoginRequest {
	creds := auth.Credentials{Username: username, Password: pwd}
	return session.LoginRequest{Credentials: creds}
}

var _ = Describe("ServiceConfig", func() {
	full := func() session.ServiceConfig {
		return session.ServiceConfig{
			Auth:    &auth.Service{},
			User:    &user.Service{},
			RBAC:    &rbac.Service{},
			Cluster: node.Cluster,
		}
	}
	It("Should succeed when every service is set", func() {
		Expect(full().Validate()).To(Succeed())
	})
	DescribeTable(
		"Should fail when a service is missing",
		func(clear func(*session.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry(
			"auth",
			func(c *session.ServiceConfig) { c.Auth = nil },
			"auth: must be non-nil",
		),
		Entry(
			"user",
			func(c *session.ServiceConfig) { c.User = nil },
			"user: must be non-nil",
		),
		Entry(
			"rbac",
			func(c *session.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil",
		),
		Entry(
			"cluster",
			func(c *session.ServiceConfig) { c.Cluster = nil },
			"cluster: must be non-nil",
		),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		merged := session.ServiceConfig{}.Override(cfg)
		Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
		Expect(merged.User).To(BeIdenticalTo(cfg.User))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		Expect(merged.Cluster).To(BeIdenticalTo(cfg.Cluster))
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

var _ = Describe("Disabled password method", func() {
	It("Should refuse a password login and keep the password", func(ctx SpecContext) {
		own := auth.Credentials{Username: uuid.New().String(), Password: "old"}
		createUser(ctx, own.Username, own.Password)
		svc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: node.DB,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodAPIKey: apiKeySvc,
			},
		}))
		disabled := MustSucceed(session.NewService(session.ServiceConfig{
			Auth:    svc,
			User:    userSvc,
			RBAC:    rbacSvc,
			Cluster: node.Cluster,
		}))
		proof := passwordProof(own.Username, own.Password)
		Expect(disabled.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: proof},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, proof))
	})
})

var _ = Describe("CreateSession", func() {
	It("Should open a session for a password proof", func(ctx SpecContext) {
		res := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(
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
		res := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{Proof: passwordProof(
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

var _ = Describe("RenewSession", func() {
	It("Should return the caller's session and user", func(ctx SpecContext) {
		created := MustSucceed(apiSvc.Create(
			requestCtx(ctx),
			session.CreateRequest{
				Proof: passwordProof(creds.Username, creds.Password),
			},
		))
		res := MustSucceed(apiSvc.Renew(authed(ctx, created.Token), struct{}{}))
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
			res := MustSucceed(apiSvc.Create(fctx, session.CreateRequest{
				Proof:  passwordProof(creds.Username, creds.Password),
				Cookie: true,
			}))
			Expect(res.Token).To(BeEmpty())
			cookie := MustBeOk(fctx.Get(apiauth.SessionCookieParam)).(apiauth.SessionCookie)
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
		MustSucceed(apiSvc.Delete(fctx, session.DeleteRequest{
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
		MustSucceed(apiSvc.Delete(fctx, session.DeleteRequest{
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
	Describe("RetrieveSessions", func() {
		It("Should list the caller's own sessions by default", func(ctx SpecContext) {
			res := MustSucceed(apiSvc.Retrieve(
				authed(ctx, bobToken),
				session.RetrieveRequest{},
			))
			Expect(res.Sessions).To(HaveLen(1))
			Expect(res.Sessions[0].Subject).To(Equal(bob.OntologyID()))
		})
		It(
			"Should let the owner list another subject's sessions",
			func(ctx SpecContext) {
				res := MustSucceed(apiSvc.Retrieve(
					authed(ctx, rootToken),
					session.RetrieveRequest{
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
				res := MustSucceed(apiSvc.Retrieve(
					authed(ctx, rootToken),
					session.RetrieveRequest{
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
				Expect(apiSvc.Retrieve(
					authed(ctx, bobToken),
					session.RetrieveRequest{
						Subjects: []ontology.ID{root.OntologyID()},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
	Describe("DeleteSessions", func() {
		It("Should end the caller's own session", func(ctx SpecContext) {
			key := MustSucceed(authSvc.Authenticate(ctx, bobToken)).Key
			Expect(apiSvc.Delete(
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
				Expect(apiSvc.Delete(
					authed(ctx, bobToken),
					session.DeleteRequest{Keys: []auth.SessionKey{key}},
				)).Error().To(MatchError(access.ErrDenied))
				MustSucceed(authSvc.Authenticate(ctx, rootToken))
			},
		)
		It("Should let the owner end another subject's session", func(ctx SpecContext) {
			key := MustSucceed(authSvc.Authenticate(ctx, bobToken)).Key
			Expect(apiSvc.Delete(
				authed(ctx, rootToken),
				session.DeleteRequest{Keys: []auth.SessionKey{key}},
			)).Error().ToNot(HaveOccurred())
			Expect(authSvc.Authenticate(ctx, bobToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(apiSvc.Delete(
				authed(ctx, bobToken),
				session.DeleteRequest{Keys: []auth.SessionKey{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})
})
