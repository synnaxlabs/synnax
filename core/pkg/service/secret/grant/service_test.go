// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grant_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	full := func() grant.ServiceConfig {
		return grant.ServiceConfig{DB: &gorp.DB{}, Auth: &auth.Service{}}
	}
	DescribeTable("Should fail when a dependency is missing",
		func(clear func(*grant.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(msg),
			))
		},
		Entry("db", func(c *grant.ServiceConfig) { c.DB = nil }, "db: must be non-nil"),
		Entry(
			"auth",
			func(c *grant.ServiceConfig) { c.Auth = nil },
			"auth: must be non-nil",
		),
	)
	It("Should take the other config's dependencies when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := grant.ServiceConfig{}.Override(cfg)
		Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
		Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
	})
})

var _ = Describe("Service", func() {
	var (
		db     *gorp.DB
		keys   *apikey.Service
		auths  *auth.Service
		svc    *grant.Service
		rk     ontology.ID
		secret uuid.UUID
	)
	login := func(ctx SpecContext) auth.SessionKey {
		GinkgoHelper()
		user := ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.New().String()}
		k, _ := MustSucceed2(keys.NewWriter(nil).Create(ctx, user, "Console"))
		session, _ := MustSucceed2(auths.NewWriter(nil).Open(ctx, user, k.Key))
		return session.Key
	}
	create := func(ctx SpecContext, session auth.SessionKey) grant.Grant {
		GinkgoHelper()
		g := grant.Grant{Rack: rk, Secrets: []uuid.UUID{secret}, Session: session}
		Expect(svc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		return g
	}
	retrieve := func(ctx SpecContext, filter grant.Filter) []grant.Grant {
		GinkgoHelper()
		var grants []grant.Grant
		Expect(svc.NewRetrieve().
			Where(filter).
			Entries(&grants).
			Exec(ctx, nil)).To(Succeed())
		return grants
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		keys = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
		auths = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB:             db,
			Authenticators: map[auth.Method]auth.Authenticator{auth.MethodAPIKey: keys},
		}))
		svc = MustOpen(grant.OpenService(ctx, grant.ServiceConfig{DB: db, Auth: auths}))
		rk = rack.Key(1).OntologyID()
		secret = uuid.New()
	})

	Describe("OpenService", func() {
		It("Should return an error when the config is invalid", func(ctx SpecContext) {
			Expect(grant.OpenService(ctx, grant.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Writer", func() {
		Describe("Create", func() {
			It("Should store the grant under a new key", func(ctx SpecContext) {
				session := login(ctx)
				g := create(ctx, session)
				Expect(g.Key).ToNot(Equal(uuid.Nil()))
				Expect(retrieve(ctx, grant.MatchKeys(g.Key))).To(Equal([]grant.Grant{{
					Key:     g.Key,
					Rack:    rk,
					Secrets: []uuid.UUID{secret},
					Session: session,
				}}))
			})
			DescribeTable("Should reject a grant that is not valid",
				func(ctx SpecContext, change func(*grant.Grant), msg string) {
					g := grant.Grant{
						Rack:    rk,
						Secrets: []uuid.UUID{secret},
						Session: uuid.New(),
					}
					change(&g)
					Expect(svc.NewWriter(nil).Create(ctx, &g)).To(SatisfyAll(
						BeAssignableToTypeOf(validate.PathError{}),
						MatchError(msg),
					))
					Expect(svc.NewRetrieve().Count(ctx, nil)).To(BeZero())
				},
				Entry(
					"a subject that is not a rack",
					func(g *grant.Grant) {
						g.Rack = ontology.ID{Type: ontology.ResourceTypeUser, Key: "u"}
					},
					"rack: user:u is not a rack",
				),
				Entry(
					"no secrets",
					func(g *grant.Grant) { g.Secrets = nil },
					"secrets: must be non-empty",
				),
				Entry(
					"no session",
					func(g *grant.Grant) { g.Session = uuid.Nil() },
					"session: required",
				),
			)
		})

		Describe("Delete", func() {
			It("Should delete the grants with the given keys", func(ctx SpecContext) {
				session := login(ctx)
				deleted := create(ctx, session)
				kept := create(ctx, session)
				Expect(svc.NewWriter(nil).Delete(ctx, deleted.Key)).To(Succeed())
				Expect(retrieve(ctx, grant.MatchSessions(session))).
					To(Equal([]grant.Grant{kept}))
			})
			It("Should ignore unknown keys", func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, uuid.New())).To(Succeed())
			})
		})
	})

	Describe("Get", func() {
		It("Should return a grant whose session is live", func(ctx SpecContext) {
			g := create(ctx, login(ctx))
			Expect(svc.Get(ctx, nil, g.Key)).To(Equal(g))
		})
		It("Should return not found for an unknown key", func(ctx SpecContext) {
			Expect(svc.Get(ctx, nil, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
		It(
			"Should return not found when the session has ended",
			func(ctx SpecContext) {
				g := create(ctx, uuid.New())
				Expect(svc.Get(ctx, nil, g.Key)).Error().To(SatisfyAll(
					MatchError(query.ErrNotFound),
					MatchError(ContainSubstring(
						"grant "+g.Key.String()+" ended with its session",
					)),
				))
			},
		)
	})

	Describe("Session end", func() {
		It("Should delete the grants of a session that ends", func(ctx SpecContext) {
			ended := login(ctx)
			live := login(ctx)
			create(ctx, ended)
			create(ctx, ended)
			kept := create(ctx, live)
			Expect(auths.Logout(ctx, ended)).To(Succeed())
			Eventually(func(g Gomega) {
				var grants []grant.Grant
				g.Expect(svc.NewRetrieve().
					Where(grant.MatchSessions(ended)).
					Entries(&grants).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(grants).To(BeEmpty())
			}).Should(Succeed())
			Expect(retrieve(ctx, grant.MatchSessions(live))).
				To(Equal([]grant.Grant{kept}))
		})
	})
})
