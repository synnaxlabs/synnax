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
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Writer", func() {
	var (
		svc    *auth.Service
		writer auth.Writer
		creds  auth.Credentials
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		writer = svc.NewWriter(nil)
		creds = auth.Credentials{Username: uuid.NewString(), Password: "password"}
		Expect(writer.Register(ctx, creds)).To(Succeed())
	})
	Describe("Register", func() {
		It("Should make the registered credentials valid for Authenticate", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, nil, creds)).To(Succeed())
		})
		It("Should return a RepeatedUsername error when the username is already registered", func(ctx SpecContext) {
			Expect(errors.Is(writer.Register(ctx, creds), auth.ErrRepeatedUsername)).To(BeTrue())
		})
		It("Should propagate the bcrypt error verbatim when the password is too long to hash", func(ctx SpecContext) {
			err := writer.Register(ctx, auth.Credentials{
				Username: uuid.NewString(),
				Password: strings.Repeat("a", 73),
			})
			Expect(err).To(MatchError(bcrypt.ErrPasswordTooLong))
			Expect(errors.Is(err, auth.ErrInvalidCredentials)).To(BeFalse())
		})
		It("Should reject an empty username before touching the store", func(ctx SpecContext) {
			Expect(writer.Register(ctx, auth.Credentials{
				Username: "",
				Password: "password",
			})).To(MatchError(ContainSubstring("username")))
		})
		It("Should reject an empty password before touching the store", func(ctx SpecContext) {
			emptyPassUser := uuid.NewString()
			Expect(writer.Register(ctx, auth.Credentials{
				Username: emptyPassUser,
				Password: "",
			})).To(MatchError(ContainSubstring("password")))
			// Ensure no orphaned row was created: a subsequent Register with the same
			// username and a real password must succeed.
			Expect(writer.Register(ctx, auth.Credentials{
				Username: emptyPassUser,
				Password: "password",
			})).To(Succeed())
		})
	})
	Describe("UpdateUsername", func() {
		It("Should rename the credential entry", func(ctx SpecContext) {
			newUsername := uuid.NewString()
			newCreds := auth.Credentials{Username: newUsername, Password: creds.Password}
			Expect(writer.UpdateUsername(ctx, creds.Username, newUsername)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, newCreds)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should do nothing when the new username equals the old", func(ctx SpecContext) {
			Expect(writer.UpdateUsername(ctx, creds.Username, creds.Username)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(Succeed())
		})
		It("Should return RepeatedUsername when the target name is taken", func(ctx SpecContext) {
			newUsername := uuid.NewString()
			newCreds := auth.Credentials{Username: newUsername, Password: creds.Password}
			Expect(writer.Register(ctx, newCreds)).To(Succeed())
			Expect(errors.Is(
				writer.UpdateUsername(ctx, creds.Username, newUsername),
				auth.ErrRepeatedUsername,
			)).To(BeTrue())
			Expect(svc.Authenticate(ctx, nil, creds)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, newCreds)).To(Succeed())
		})
		It("Should return an error when the old username is not registered", func(ctx SpecContext) {
			Expect(writer.UpdateUsername(ctx, uuid.NewString(), uuid.NewString())).
				To(MatchError(query.ErrNotFound))
		})
	})
	Describe("ChangePassword", func() {
		const newPassword = "new-password"
		It("Should set a new password for the given username", func(ctx SpecContext) {
			newCreds := auth.Credentials{Username: creds.Username, Password: newPassword}
			Expect(writer.ChangePassword(ctx, newCreds)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Authenticate(ctx, nil, newCreds)).To(Succeed())
		})
		It("Should return InvalidCredentials when the username is not registered", func(ctx SpecContext) {
			Expect(writer.ChangePassword(ctx, auth.Credentials{
				Username: uuid.NewString(),
				Password: newPassword,
			})).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should propagate the bcrypt error verbatim when the new password is too long to hash", func(ctx SpecContext) {
			err := writer.ChangePassword(ctx, auth.Credentials{
				Username: creds.Username,
				Password: strings.Repeat("a", 73),
			})
			Expect(err).To(MatchError(bcrypt.ErrPasswordTooLong))
			Expect(errors.Is(err, auth.ErrInvalidCredentials)).To(BeFalse())
		})
		It("Should reject an empty username before touching the store", func(ctx SpecContext) {
			Expect(writer.ChangePassword(ctx, auth.Credentials{
				Username: "",
				Password: newPassword,
			})).To(MatchError(ContainSubstring("username")))
			// Original password must still work.
			Expect(svc.Authenticate(ctx, nil, creds)).To(Succeed())
		})
		It("Should reject an empty password before touching the store", func(ctx SpecContext) {
			Expect(writer.ChangePassword(ctx, auth.Credentials{
				Username: creds.Username,
				Password: "",
			})).To(MatchError(ContainSubstring("password")))
			// Original password must still work.
			Expect(svc.Authenticate(ctx, nil, creds)).To(Succeed())
		})
	})
	Describe("Deactivate", func() {
		It("Should delete the credentials", func(ctx SpecContext) {
			Expect(writer.Deactivate(ctx, creds.Username)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should be idempotent", func(ctx SpecContext) {
			for range 2 {
				Expect(writer.Deactivate(ctx, creds.Username)).To(Succeed())
			}
		})
		It("Should delete multiple credentials", func(ctx SpecContext) {
			creds2 := auth.Credentials{Username: uuid.NewString(), Password: "password"}
			Expect(writer.Register(ctx, creds2)).To(Succeed())
			Expect(writer.Deactivate(ctx, creds.Username, creds2.Username)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Authenticate(ctx, nil, creds2)).To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})
