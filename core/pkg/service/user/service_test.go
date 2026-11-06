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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
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
			Password: passwordSvc,
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
				Expect(result.Password).To(Equal(passwordSvc))
			},
		)
		It(
			"Should keep each required field from the base when the override is zero",
			func() {
				result := cfg.Override(user.ServiceConfig{})
				Expect(result.DB).To(Equal(db))
				Expect(result.Ontology).To(Equal(otg))
				Expect(result.Group).To(Equal(groupSvc))
				Expect(result.Password).To(Equal(passwordSvc))
			},
		)
	})
	Describe("Validate", func() {
		It("Should succeed when every required field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
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
				"password",
				func(c *user.ServiceConfig) { c.Password = nil },
				"password: must be non-nil",
			),
		)
	})
})

var _ = Describe("Service", func() {
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
			created := MustSucceed(
				w.Create(ctx, user.User{Name: uuid.New().String()}),
			)
			var u user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(created.Key)).Entry(&u).
				Exec(ctx, tx)).To(Succeed())
			Expect(u).To(Equal(created))
		})
		It("Should retrieve multiple users by keys", func(ctx SpecContext) {
			a := MustSucceed(w.Create(ctx, user.User{Name: uuid.New().String()}))
			b := MustSucceed(w.Create(ctx, user.User{Name: uuid.New().String()}))
			var ret []user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(a.Key, b.Key)).Entries(&ret).
				Exec(ctx, tx)).To(Succeed())
			Expect(ret).To(ConsistOf(a, b))
		})
		Describe("MatchUsernames", func() {
			var (
				username string
				u        user.User
			)
			BeforeEach(func(ctx SpecContext) {
				username = uuid.New().String()
				u = MustSucceed(w.Create(ctx, user.User{Name: uuid.New().String()}))
				Expect(
					passwordSvc.NewWriter(tx).Set(ctx, u.OntologyID(), auth.Credentials{
						Username: username,
						Password: "p",
					}),
				).To(Succeed())
			})
			It(
				"Should match the user whose password has the username",
				func(ctx SpecContext) {
					var ret []user.User
					Expect(
						svc.NewRetrieve().
							Where(user.MatchUsernames(username)).
							Entries(&ret).
							Exec(ctx, tx),
					).To(Succeed())
					Expect(ret).To(ConsistOf(u))
				},
			)
			It("Should match nothing for an unknown username", func(ctx SpecContext) {
				var ret []user.User
				Expect(svc.NewRetrieve().
					Where(user.MatchUsernames(uuid.New().String())).
					Entries(&ret).
					Exec(ctx, tx)).To(Succeed())
				Expect(ret).To(BeEmpty())
			})
			It("Should match nothing for a user's name", func(ctx SpecContext) {
				var ret []user.User
				Expect(
					svc.NewRetrieve().Where(user.MatchUsernames(u.Name)).Entries(&ret).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(ret).To(BeEmpty())
			})
			It("Should exclude the matched users under Not", func(ctx SpecContext) {
				other := MustSucceed(
					w.Create(ctx, user.User{Name: uuid.New().String()}),
				)
				var ret []user.User
				Expect(svc.NewRetrieve().
					Where(user.MatchKeys(u.Key, other.Key)).
					Where(user.Not(user.MatchUsernames(username))).
					Entries(&ret).
					Exec(ctx, tx)).To(Succeed())
				Expect(ret).To(ConsistOf(other))
			})
			It("Should skip a password held by a subject that is not a user", func(
				ctx SpecContext,
			) {
				rackName := uuid.New().String()
				Expect(passwordSvc.NewWriter(tx).Set(
					ctx,
					ontology.ID{Type: ontology.ResourceTypeRack, Key: rackName},
					auth.Credentials{Username: rackName, Password: "p"},
				)).To(Succeed())
				var ret []user.User
				Expect(
					svc.NewRetrieve().
						Where(user.MatchUsernames(rackName)).
						Entries(&ret).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(ret).To(BeEmpty())
			})
		})
	})
})
