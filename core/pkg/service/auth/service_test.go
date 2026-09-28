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
	"iter"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	Describe("Override", func() {
		It("Should take DB from the override when the base is nil", func() {
			result := auth.ServiceConfig{}.Override(auth.ServiceConfig{DB: db})
			Expect(result.DB).To(Equal(db))
		})
		It("Should keep DB from the base when the override is nil", func() {
			result := auth.ServiceConfig{DB: db}.Override(auth.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
		})
	})
	Describe("Validate", func() {
		It("Should return an error when DB is nil", func() {
			Expect(auth.ServiceConfig{}.Validate()).To(
				MatchError(ContainSubstring("db: must be non-nil")),
			)
		})
		It("Should succeed when DB is set", func() {
			Expect(auth.ServiceConfig{DB: db}.Validate()).To(Succeed())
		})
	})
})

var _ = Describe("OpenService", func() {
	It("Should return an error when the config is invalid", func(ctx SpecContext) {
		Expect(auth.OpenService(ctx, auth.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
})

var _ = Describe("Service", func() {
	var (
		svc   *auth.Service
		key   auth.Key
		creds auth.Credentials
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		key = uuid.New()
		creds = auth.Credentials{Username: uuid.New().String(), Password: "password"}
		Expect(svc.NewWriter(nil).Register(ctx, key, creds)).To(Succeed())
	})

	Describe("Authenticate", func() {
		It("Should return the key for valid credentials", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
		})
		It(
			"Should return an InvalidCredentials error when the password is wrong",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, auth.Credentials{
					Username: creds.Username,
					Password: "invalid",
				})).Error().To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should return an InvalidCredentials error when no user holds the username",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, auth.Credentials{
					Username: uuid.New().String(),
					Password: creds.Password,
				})).Error().To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should return a validation error when the username is empty",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, auth.Credentials{Password: "p"})).
					Error().To(MatchError(ContainSubstring("username: required")))
			},
		)
		It(
			"Should return a validation error when the password is empty",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, auth.Credentials{
					Username: creds.Username,
				})).Error().To(MatchError(ContainSubstring("password: required")))
			},
		)
		It(
			"Should read from the supplied tx so an in-flight password rotation is observed",
			func(ctx SpecContext) {
				rotated := auth.Credentials{
					Username: creds.Username,
					Password: "rotated-" + uuid.New().String(),
				}
				tx := DeferClose(db.OpenTx())
				Expect(svc.NewWriter(tx).ChangePassword(ctx, key, rotated.Password)).
					To(Succeed())
				Expect(svc.Authenticate(ctx, tx, rotated)).To(Equal(key))
				Expect(svc.Authenticate(ctx, tx, creds)).Error().
					To(MatchError(auth.ErrInvalidCredentials))
				Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
			},
		)
	})

	Describe("UsernamesByKey", func() {
		It("Should return the username of each key", func(ctx SpecContext) {
			Expect(svc.UsernamesByKey(ctx, nil, key)).
				To(Equal(map[auth.Key]string{key: creds.Username}))
		})
		It("Should skip keys without credentials", func(ctx SpecContext) {
			Expect(svc.UsernamesByKey(ctx, nil, key, uuid.New())).
				To(Equal(map[auth.Key]string{key: creds.Username}))
		})
		It("Should return an empty map for no keys", func(ctx SpecContext) {
			Expect(svc.UsernamesByKey(ctx, nil)).To(BeEmpty())
		})
	})

	Describe("Usernames", func() {
		It("Should include every user with credentials", func(ctx SpecContext) {
			Expect(svc.Usernames(ctx, nil)).To(HaveKeyWithValue(key, creds.Username))
		})
	})

	Describe("KeysByUsername", func() {
		It("Should return the key holding each username", func(ctx SpecContext) {
			Expect(svc.KeysByUsername(ctx, nil, creds.Username, uuid.New().String())).
				To(Equal([]auth.Key{key}))
		})
	})

	Describe("OnChange", func() {
		It(
			"Should report the keys whose credentials a commit sets",
			func(ctx SpecContext) {
				changed := make(chan auth.Key, 10)
				disconnect := svc.OnChange(
					func(_ context.Context, keys iter.Seq[auth.Key]) {
						for k := range keys {
							changed <- k
						}
					},
				)
				defer disconnect()
				Expect(
					svc.NewWriter(nil).ChangeUsername(ctx, key, uuid.New().String()),
				).
					To(Succeed())
				Eventually(changed).Should(Receive(Equal(key)))
			},
		)
		It("Should not report deleted credentials", func(ctx SpecContext) {
			changed := make(chan auth.Key, 10)
			disconnect := svc.OnChange(
				func(_ context.Context, keys iter.Seq[auth.Key]) {
					for k := range keys {
						changed <- k
					}
				},
			)
			defer disconnect()
			Expect(svc.NewWriter(nil).Deactivate(ctx, key)).To(Succeed())
			Consistently(changed).ShouldNot(Receive())
		})
	})
})

var _ = Describe("SecureCredentials", func() {
	Describe("GorpKey", func() {
		It("Should return the user key", func() {
			key := uuid.New()
			Expect(auth.SecureCredentials{Key: key}.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(auth.SecureCredentials{}.SetOptions()).To(BeNil())
		})
	})
})
