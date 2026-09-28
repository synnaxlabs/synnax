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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	authv1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	userv1 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v1"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("ServiceConfig", func() {
	// fullCfg returns a ServiceConfig with every required field populated. Override and
	// Validate specs branch off this to test individual fields.
	var cfg user.ServiceConfig
	BeforeEach(func() {
		cfg = user.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
			Auth:     authSvc,
		}
	})
	Describe("Override", func() {
		It(
			"Should take each required field from the override when the base is zero",
			func() {
				result := user.ServiceConfig{}.Override(cfg)
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.Group).To(Equal(groupSvc))
				Expect(result.Search).To(Equal(searchIdx))
				Expect(result.Auth).To(Equal(authSvc))
			},
		)
		It(
			"Should keep each required field from the base when the override is zero",
			func() {
				result := cfg.Override(user.ServiceConfig{})
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.Group).To(Equal(groupSvc))
				Expect(result.Search).To(Equal(searchIdx))
				Expect(result.Auth).To(Equal(authSvc))
			},
		)
		It("Should take RootCredentials from the override when set", func() {
			creds := auth.Credentials{Username: "u", Password: "p"}
			result := cfg.Override(user.ServiceConfig{RootCredentials: creds})
			Expect(result.RootCredentials).To(Equal(creds))
		})
		It(
			"Should keep RootCredentials from the base when the override is zero",
			func() {
				creds := auth.Credentials{Username: "u", Password: "p"}
				cfg.RootCredentials = creds
				result := cfg.Override(user.ServiceConfig{})
				Expect(result.RootCredentials).To(Equal(creds))
			},
		)
	})
	Describe("Validate", func() {
		It("Should succeed when every required field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
		})
		It("Should require Auth when root credentials are configured", func() {
			cfg.Auth = nil
			cfg.RootCredentials = auth.Credentials{Username: "u", Password: "p"}
			Expect(
				cfg.Validate(),
			).To(MatchError(ContainSubstring("auth: must be non-nil")))
		})
		It("Should reject root credentials with an empty password", func() {
			cfg.RootCredentials = auth.Credentials{Username: "u", Password: ""}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("password")))
		})
		DescribeTable(
			"Should return an error when a required field is missing",
			func(mutate func(*user.ServiceConfig), errorMsg string) {
				mutate(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(errorMsg)))
			},
			Entry(
				"db",
				func(c *user.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"ontology",
				func(c *user.ServiceConfig) { c.Ontology = nil },
				"ontology: must be non-nil",
			),
			Entry(
				"group",
				func(c *user.ServiceConfig) { c.Group = nil },
				"group: must be non-nil",
			),
			Entry(
				"search",
				func(c *user.ServiceConfig) { c.Search = nil },
				"search: must be non-nil",
			),
			Entry(
				"auth",
				func(c *user.ServiceConfig) { c.Auth = nil },
				"auth: must be non-nil",
			),
		)
	})
})

var _ = Describe("Service", func() {
	Describe("ResolveUsernames", func() {
		It(
			"Should fill each user's username from its credentials",
			func(ctx SpecContext) {
				created := createWithCredentials(ctx, user.User{
					Username: uuid.New().String(),
				})
				users := []user.User{{Key: created.Key}, {Key: uuid.New()}}
				Expect(svc.ResolveUsernames(ctx, nil, users)).To(Succeed())
				Expect(users[0].Username).To(Equal(created.Username))
				Expect(users[1].Username).To(BeEmpty())
			},
		)
	})
	Describe("Open", func() {
		It("Should return an error when the config is invalid", func(ctx SpecContext) {
			Expect(user.OpenService(ctx, user.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})
	Describe("Retrieve", func() {
		var (
			tx gorp.Tx
			w  user.Writer
		)
		BeforeEach(func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			w = svc.NewWriter(tx)
		})
		It("Should retrieve a user by its key", func(ctx SpecContext) {
			created := MustSucceed(w.Create(ctx, user.User{FirstName: "Ada"}))
			var u user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(created.Key)).Entry(&u).
				Exec(ctx, tx)).To(Succeed())
			Expect(u).To(Equal(created))
		})
		It("Should retrieve multiple users by keys", func(ctx SpecContext) {
			a := MustSucceed(w.Create(ctx, user.User{FirstName: "Ada"}))
			b := MustSucceed(w.Create(ctx, user.User{FirstName: "Grace"}))
			var ret []user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(a.Key, b.Key)).Entries(&ret).
				Exec(ctx, tx)).To(Succeed())
			Expect(ret).To(ConsistOf(a, b))
		})
		It("Should not store the username on the user", func(ctx SpecContext) {
			created := createWithCredentials(ctx, user.User{
				Username: uuid.New().String(),
			})
			var u user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(created.Key)).Entry(&u).
				Exec(ctx, nil)).To(Succeed())
			Expect(u.Username).To(BeEmpty())
		})
	})
})

var _ = Describe("Credential re-key", func() {
	It(
		"Should move usernames onto credentials keyed by user when the service opens",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.OpenIndex())
			groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
			alice := userv0.User{Key: uuid.New(), Username: "alice"}
			hash := MustSucceed(
				bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost),
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := gorp.WrapWriter[userv0.Key, userv0.User](tx).
					Set(ctx, alice); err != nil {
					return err
				}
				return gorp.WrapWriter[string, authv1.SecureCredentials](tx).
					Set(ctx, authv1.SecureCredentials{Username: "alice", Password: hash})
			})).To(Succeed())
			MustOpen(user.OpenService(ctx, user.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				Search:   searchIdx,
				Auth:     authSvc,
			}))
			Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
				Username: "alice",
				Password: "p",
			})).To(Equal(alice.Key))
			var stored userv1.User
			Expect(gorp.NewRetrieve[userv1.Key, userv1.User]().
				Where(gorp.MatchKeys[userv1.Key, userv1.User](alice.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Username).To(BeEmpty())
		},
	)
})
