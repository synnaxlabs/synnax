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
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/kv"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		authenticator *kv.Authenticator
		creds         auth.Credentials
	)
	BeforeEach(func(ctx SpecContext) {
		authenticator = MustOpen(kv.OpenAuthenticator(ctx, kv.AuthenticatorConfig{DB: db}))
		creds = auth.Credentials{Username: uuid.NewString(), Password: "password"}
		Expect(authenticator.NewWriter(nil).Register(ctx, creds)).To(Succeed())
	})
	Describe("Register", func() {
		It("Should make the registered credentials valid for Authenticate", func(ctx SpecContext) {
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return a RepeatedUsername error when the username is already registered", func(ctx SpecContext) {
			Expect(errors.Is(authenticator.NewWriter(nil).Register(ctx, creds), auth.ErrRepeatedUsername)).To(BeTrue())
		})
		It("Should return ErrInvalidCredentials when bcrypt cannot hash the password", func(ctx SpecContext) {
			err := authenticator.NewWriter(nil).Register(ctx, auth.Credentials{
				Username: uuid.NewString(),
				Password: strings.Repeat("a", 73),
			})
			Expect(err).To(MatchError(auth.ErrInvalidCredentials))
		})
	})
	Describe("UpdateUsername", func() {
		It("Should rename the credential entry", func(ctx SpecContext) {
			newUsername := uuid.NewString()
			newCreds := auth.Credentials{Username: newUsername, Password: creds.Password}
			Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, creds.Username, newUsername)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should do nothing when the new username equals the old", func(ctx SpecContext) {
			Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, creds.Username, creds.Username)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return RepeatedUsername when the target name is taken", func(ctx SpecContext) {
			newUsername := uuid.NewString()
			newCreds := auth.Credentials{Username: newUsername, Password: creds.Password}
			Expect(authenticator.NewWriter(nil).Register(ctx, newCreds)).To(Succeed())
			Expect(errors.Is(
				authenticator.NewWriter(nil).UpdateUsername(ctx, creds.Username, newUsername),
				auth.ErrRepeatedUsername,
			)).To(BeTrue())
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
		})
		It("Should return an error when the old username is not registered", func(ctx SpecContext) {
			Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, uuid.NewString(), uuid.NewString())).
				To(MatchError(query.ErrNotFound))
		})
	})
	Describe("ChangePassword", func() {
		const newPassword = "new-password"
		It("Should set a new password for the given username", func(ctx SpecContext) {
			newCreds := auth.Credentials{Username: creds.Username, Password: newPassword}
			Expect(authenticator.NewWriter(nil).ChangePassword(ctx, newCreds)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.ErrInvalidCredentials))
			Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
		})
		It("Should return InvalidCredentials when the username is not registered", func(ctx SpecContext) {
			Expect(authenticator.NewWriter(nil).ChangePassword(ctx, auth.Credentials{
				Username: uuid.NewString(),
				Password: newPassword,
			})).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should return InvalidCredentials when the new password is too long for bcrypt", func(ctx SpecContext) {
			Expect(authenticator.NewWriter(nil).ChangePassword(ctx, auth.Credentials{
				Username: creds.Username,
				Password: strings.Repeat("a", 73),
			})).To(MatchError(auth.ErrInvalidCredentials))
		})
	})
	Describe("Deactivate", func() {
		It("Should delete the credentials", func(ctx SpecContext) {
			Expect(authenticator.NewWriter(nil).Deactivate(ctx, creds.Username)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should be idempotent", func(ctx SpecContext) {
			for range 2 {
				Expect(authenticator.NewWriter(nil).Deactivate(ctx, creds.Username)).To(Succeed())
			}
		})
		It("Should delete multiple credentials", func(ctx SpecContext) {
			creds2 := auth.Credentials{Username: uuid.NewString(), Password: "password"}
			Expect(authenticator.NewWriter(nil).Register(ctx, creds2)).To(Succeed())
			Expect(authenticator.NewWriter(nil).Deactivate(ctx, creds.Username, creds2.Username)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.ErrInvalidCredentials))
			Expect(authenticator.Authenticate(ctx, creds2)).To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})
