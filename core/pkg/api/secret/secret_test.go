// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	secretType = ontology.ID{Type: ontology.ResourceTypeSecret}
	deviceType = ontology.ID{Type: ontology.ResourceTypeDevice}
	all        = access.AllActions
)

func createSecret(ctx SpecContext, name, value string) secret.Secret {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: name}}}
	Expect(svc.Secret.NewWriter(nil).Create(ctx, &s, []byte(value))).To(Succeed())
	return s
}

func retrieveSecret(ctx SpecContext, key secret.Key) secret.Secret {
	GinkgoHelper()
	var s secret.Secret
	Expect(svc.Secret.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, nil)).To(Succeed())
	return s
}

// sessionCtx returns a request context for u that passed the auth middleware, and the
// key of the session u opened for it.
func sessionCtx(ctx context.Context, u user.User) (freighter.Context, auth.SessionKey) {
	GinkgoHelper()
	k, _ := MustSucceed2(svc.APIKey.NewWriter(nil).Create(ctx, u.OntologyID(), "c"))
	session, token := MustSucceed2(
		svc.Auth.NewWriter(nil).Open(ctx, u.OntologyID(), k.Key),
	)
	fctx := freighter.Context{
		Context: ctx,
		Params:  freighter.Params{},
		Variant: freighter.VariantUnary,
	}
	fctx.Set(fiber.HeaderAuthorization, "Bearer "+token)
	var authed freighter.Context
	MustSucceed(apiauth.Middleware(svc.Auth).Exec(fctx, freighter.FinalizerFunc(
		func(c freighter.Context) (freighter.Context, error) {
			authed = c
			return c, nil
		},
	)))
	return authed, session.Key
}

var _ = Describe("ServiceConfig", func() {
	full := func() apisecret.ServiceConfig {
		return apisecret.ServiceConfig{
			Secret: &secret.Service{},
			Grant:  &grant.Service{},
			RBAC:   &rbac.Service{},
		}
	}
	DescribeTable("Should fail when a service is missing",
		func(clear func(*apisecret.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry("secret", func(c *apisecret.ServiceConfig) { c.Secret = nil },
			"secret: must be non-nil"),
		Entry("grant", func(c *apisecret.ServiceConfig) { c.Grant = nil },
			"grant: must be non-nil"),
		Entry("rbac", func(c *apisecret.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil"),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := apisecret.ServiceConfig{}.Override(cfg)
		Expect(merged.Secret).To(BeIdenticalTo(cfg.Secret))
		Expect(merged.Grant).To(BeIdenticalTo(cfg.Grant))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
	})
})

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It("Should create stored and file secrets and never return a value",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.OntologyID(), all, secretType)
				res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apisecret.CreateRequest{Secrets: []apisecret.Input{
						{
							Source: secret.SourceStored,
							Name:   "Password",
							Value:  "hunter2",
						},
						{Source: secret.SourceFile, Name: "Key", Path: "/k.pem"},
					}},
				))
				Expect(res.Secrets).To(HaveLen(2))
				Expect(res.Secrets[0].Base().Key).ToNot(Equal(uuid.Nil()))
				Expect(retrieveSecret(ctx, res.Secrets[1].Base().Key).Variant).
					To(BeAssignableToTypeOf(secret.FileSecret{}))
				encoded := string(MustSucceed(json.Marshal(res)))
				Expect(encoded).ToNot(ContainSubstring("value"))
				Expect(encoded).ToNot(ContainSubstring("hunter2"))
			},
		)
		It(
			"Should replace the secret whose key the request gives",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.OntologyID(), all, secretType)
				s := createSecret(ctx, "Before", "a")
				MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apisecret.CreateRequest{Secrets: []apisecret.Input{{
						Key:    s.Base().Key,
						Source: secret.SourceStored,
						Name:   "After",
						Value:  "b",
					}}},
				))
				Expect(retrieveSecret(ctx, s.Base().Key).Base().Name).To(Equal("After"))
			},
		)
		DescribeTable("Should reject an invalid input",
			func(ctx SpecContext, in apisecret.Input, msg string) {
				u := createUser(ctx)
				allow(ctx, u.OntologyID(), all, secretType)
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apisecret.CreateRequest{Secrets: []apisecret.Input{in}},
				)).Error().To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"an unknown source",
				apisecret.Input{Source: "cloud", Name: "A", Value: "v"},
				`source: must be stored or file, not "cloud"`,
			),
			Entry(
				"a stored secret with a path",
				apisecret.Input{
					Source: secret.SourceStored,
					Name:   "A",
					Value:  "v",
					Path:   "/a",
				},
				"path: must be empty for a stored secret",
			),
			Entry(
				"a file secret with a value",
				apisecret.Input{
					Source: secret.SourceFile,
					Name:   "A",
					Value:  "v",
					Path:   "/a",
				},
				"value: must be empty for a file secret",
			),
		)
		It("Should deny a user without create access", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
				apisecret.CreateRequest{Secrets: []apisecret.Input{
					{Source: secret.SourceStored, Name: "A", Value: "v"},
				}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve secrets by key", func(ctx SpecContext) {
			u := createUser(ctx)
			allow(
				ctx,
				u.OntologyID(),
				[]access.Action{access.ActionRetrieve},
				secretType,
			)
			s := createSecret(ctx, "Password", "hunter2")
			res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u),
				apisecret.RetrieveRequest{Keys: []secret.Key{s.Base().Key}},
			))
			Expect(res.Secrets).To(HaveLen(1))
			Expect(res.Secrets[0].Base().Name).To(Equal("Password"))
		})
		It("Should deny a user without retrieve access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "hunter2")
			Expect(apiSvc.Retrieve(AuthedCtx(ctx, u),
				apisecret.RetrieveRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Rename", func() {
		It("Should rename the secret when update is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Before", "a")
			allow(
				ctx,
				u.OntologyID(),
				[]access.Action{access.ActionUpdate},
				s.OntologyID(),
			)
			MustSucceed(apiSvc.Rename(AuthedCtx(ctx, u), nil,
				apisecret.RenameRequest{Key: s.Base().Key, Name: "After"},
			))
			Expect(retrieveSecret(ctx, s.Base().Key).Base().Name).To(Equal("After"))
		})
		It("Should deny a user without update access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Before", "a")
			Expect(apiSvc.Rename(AuthedCtx(ctx, u), nil,
				apisecret.RenameRequest{Key: s.Base().Key, Name: "After"},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should delete the secret when delete is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "a")
			allow(
				ctx,
				u.OntologyID(),
				[]access.Action{access.ActionDelete},
				s.OntologyID(),
			)
			MustSucceed(apiSvc.Delete(AuthedCtx(ctx, u), nil,
				apisecret.DeleteRequest{Keys: []secret.Key{s.Base().Key}},
			))
			Expect(svc.Secret.NewRetrieve().
				Where(secret.MatchKeys(s.Base().Key)).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should deny a user without delete access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "a")
			Expect(apiSvc.Delete(AuthedCtx(ctx, u), nil,
				apisecret.DeleteRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Grants and Resolve", func() {
		rk := rack.Key(7)
		allowDevices := func(ctx SpecContext, u user.User) {
			GinkgoHelper()
			allow(ctx, u.OntologyID(), []access.Action{access.ActionCreate}, deviceType)
		}
		createGrant := func(
			ctx freighter.Context,
			keys ...secret.Key,
		) apisecret.GrantCreateResponse {
			GinkgoHelper()
			return MustSucceed(
				apiSvc.CreateGrant(ctx, nil, apisecret.GrantCreateRequest{
					Rack:    rk,
					Secrets: keys,
				}),
			)
		}
		retrieveGrant := func(ctx SpecContext, key grant.Key) grant.Grant {
			GinkgoHelper()
			var g grant.Grant
			Expect(svc.Grant.NewRetrieve().
				Where(grant.MatchKeys(key)).
				Entry(&g).
				Exec(ctx, nil)).To(Succeed())
			return g
		}

		It("Should let the rack resolve the granted secrets in request order",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allowDevices(ctx, u)
				a := createSecret(ctx, "A", "first")
				b := createSecret(ctx, "B", "second")
				keys := []secret.Key{b.Base().Key, a.Base().Key}
				uCtx, _ := sessionCtx(ctx, u)
				res := createGrant(uCtx, keys...)
				values := MustSucceed(apiSvc.Resolve(SubjectCtx(ctx, rk.OntologyID()),
					apisecret.ResolveRequest{Keys: keys, Grant: res.Grant},
				))
				Expect(values.Values).To(Equal([]apisecret.Value{
					{Key: b.Base().Key, Value: "second"},
					{Key: a.Base().Key, Value: "first"},
				}))
			},
		)
		It("Should record the caller's session on the grant", func(ctx SpecContext) {
			u := createUser(ctx)
			allowDevices(ctx, u)
			s := createSecret(ctx, "A", "v")
			uCtx, session := sessionCtx(ctx, u)
			res := createGrant(uCtx, s.Base().Key)
			Expect(retrieveGrant(ctx, res.Grant)).To(Equal(grant.Grant{
				Key:     res.Grant,
				Rack:    rk.OntologyID(),
				Secrets: []secret.Key{s.Base().Key},
				Session: session,
			}))
		})
		It(
			"Should end the grant when the caller's session ends",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allowDevices(ctx, u)
				s := createSecret(ctx, "A", "v")
				uCtx, session := sessionCtx(ctx, u)
				res := createGrant(uCtx, s.Base().Key)
				Expect(svc.Auth.Logout(ctx, session)).To(Succeed())
				Expect(apiSvc.Resolve(SubjectCtx(ctx, rk.OntologyID()),
					apisecret.ResolveRequest{
						Keys:  []secret.Key{s.Base().Key},
						Grant: res.Grant,
					},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"grant "+res.Grant.String()+" has ended or does not exist",
					)),
				))
			},
		)
		It("Should deny a grant to a user who cannot create devices",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.OntologyID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				uCtx, _ := sessionCtx(ctx, u)
				Expect(apiSvc.CreateGrant(uCtx, nil, apisecret.GrantCreateRequest{
					Rack:    rk,
					Secrets: []secret.Key{s.Base().Key},
				})).Error().To(MatchError(access.ErrDenied))
				Expect(svc.Grant.NewRetrieve().
					Where(grant.MatchSessions(apiauth.GetSessionKey(uCtx))).
					Count(ctx, nil)).To(BeZero())
			},
		)
		It("Should delete grants", func(ctx SpecContext) {
			u := createUser(ctx)
			allowDevices(ctx, u)
			s := createSecret(ctx, "A", "v")
			uCtx, _ := sessionCtx(ctx, u)
			res := createGrant(uCtx, s.Base().Key)
			MustSucceed(apiSvc.DeleteGrants(uCtx, nil, apisecret.GrantDeleteRequest{
				Grants: []grant.Key{res.Grant, uuid.New()},
			}))
			Expect(svc.Grant.NewRetrieve().
				Where(grant.MatchKeys(res.Grant)).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should deny a delete to a user who cannot create devices",
			func(ctx SpecContext) {
				owner := createUser(ctx)
				allowDevices(ctx, owner)
				s := createSecret(ctx, "A", "v")
				ownerCtx, _ := sessionCtx(ctx, owner)
				res := createGrant(ownerCtx, s.Base().Key)
				u := createUser(ctx)
				uCtx, _ := sessionCtx(ctx, u)
				Expect(apiSvc.DeleteGrants(uCtx, nil, apisecret.GrantDeleteRequest{
					Grants: []grant.Key{res.Grant},
				})).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveGrant(ctx, res.Grant).Key).To(Equal(res.Grant))
			},
		)
		It(
			"Should deny resolve to a subject that is not a rack",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.OntologyID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				Expect(apiSvc.Resolve(AuthedCtx(ctx, u),
					apisecret.ResolveRequest{Keys: []secret.Key{s.Base().Key}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring("only a rack can resolve secrets")),
				))
			},
		)
		It("Should deny resolve without a grant", func(ctx SpecContext) {
			s := createSecret(ctx, "A", "v")
			Expect(apiSvc.Resolve(SubjectCtx(ctx, rk.OntologyID()),
				apisecret.ResolveRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(`rack:7 cannot read secret "A"`)),
			))
		})
	})
})
