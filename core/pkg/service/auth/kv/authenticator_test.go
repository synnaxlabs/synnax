// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/kv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AuthenticatorConfig", func() {
	Describe("Override", func() {
		It("Should take DB from the override when the base is nil", func() {
			result := kv.AuthenticatorConfig{}.Override(kv.AuthenticatorConfig{DB: db})
			Expect(result.DB).To(Equal(db))
		})
		It("Should keep DB from the base when the override is nil", func() {
			result := kv.AuthenticatorConfig{DB: db}.Override(kv.AuthenticatorConfig{})
			Expect(result.DB).To(Equal(db))
		})
	})
	Describe("Validate", func() {
		It("Should return an error when DB is nil", func() {
			Expect(kv.AuthenticatorConfig{}.Validate()).To(
				MatchError(ContainSubstring("db: must be non-nil")),
			)
		})
		It("Should succeed when DB is set", func() {
			Expect(kv.AuthenticatorConfig{DB: db}.Validate()).To(Succeed())
		})
	})
})

var _ = Describe("OpenAuthenticator", func() {
	It("Should return an error when the config is invalid", func(ctx SpecContext) {
		Expect(kv.OpenAuthenticator(ctx, kv.AuthenticatorConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
})

var _ = Describe("Authenticator", func() {
	var (
		authenticator    *kv.Authenticator
		creds            auth.Credentials
		invalidPassCreds auth.Credentials
		invalidUserCreds auth.Credentials
	)
	BeforeEach(func(ctx SpecContext) {
		authenticator = MustOpen(kv.OpenAuthenticator(ctx, kv.AuthenticatorConfig{DB: db}))
		creds = auth.Credentials{Username: uuid.NewString(), Password: "password"}
		invalidPassCreds = auth.Credentials{Username: creds.Username, Password: "invalid"}
		invalidUserCreds = auth.Credentials{Username: uuid.NewString(), Password: creds.Password}
		Expect(authenticator.NewWriter(nil).Register(ctx, creds)).To(Succeed())
	})

	Describe("Authenticate", func() {
		It("Should return a nil error for valid credentials", func(ctx SpecContext) {
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func(ctx SpecContext) {
			Expect(authenticator.Authenticate(ctx, invalidPassCreds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func(ctx SpecContext) {
			Expect(authenticator.Authenticate(ctx, invalidUserCreds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should return a validation error when the username is empty", func(ctx SpecContext) {
			Expect(authenticator.Authenticate(ctx, auth.Credentials{Password: "password"})).To(
				MatchError(ContainSubstring("username")),
			)
		})
		It("Should return a validation error when the password is empty", func(ctx SpecContext) {
			Expect(authenticator.Authenticate(ctx, auth.Credentials{Username: uuid.NewString()})).To(
				MatchError(ContainSubstring("password")),
			)
		})
	})
})
