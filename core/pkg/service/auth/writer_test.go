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
	"uuid"

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
		key    auth.Key
		creds  auth.Credentials
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		writer = svc.NewWriter(nil)
		key = uuid.New()
		creds = auth.Credentials{Username: uuid.New().String(), Password: "password"}
		Expect(writer.Register(ctx, key, creds)).To(Succeed())
	})
	Describe("Register", func() {
		It(
			"Should make the registered credentials valid for Authenticate",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
			},
		)
		It(
			"Should replace the credentials when the key is already registered",
			func(ctx SpecContext) {
				replaced := auth.Credentials{
					Username: uuid.New().String(),
					Password: "replaced",
				}
				Expect(writer.Register(ctx, key, replaced)).To(Succeed())
				Expect(svc.Authenticate(ctx, nil, replaced)).To(Equal(key))
				Expect(svc.Authenticate(ctx, nil, creds)).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should return ErrUniqueViolation when another user holds the username",
			func(ctx SpecContext) {
				Expect(writer.Register(ctx, uuid.New(), creds)).
					To(MatchError(query.ErrUniqueViolation))
			},
		)
		It(
			"Should propagate the bcrypt error verbatim when the password is too long to hash",
			func(ctx SpecContext) {
				err := writer.Register(ctx, uuid.New(), auth.Credentials{
					Username: uuid.New().String(),
					Password: strings.Repeat("a", 73),
				})
				Expect(err).To(MatchError(bcrypt.ErrPasswordTooLong))
				Expect(errors.Is(err, auth.ErrInvalidCredentials)).To(BeFalse())
			},
		)
		It("Should reject an empty username", func(ctx SpecContext) {
			Expect(writer.Register(ctx, uuid.New(), auth.Credentials{
				Password: "password",
			})).To(MatchError(ContainSubstring("username: required")))
		})
		It("Should reject an empty password", func(ctx SpecContext) {
			Expect(writer.Register(ctx, key, auth.Credentials{
				Username: creds.Username,
			})).To(MatchError(ContainSubstring("password: required")))
			Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
		})
	})
	Describe("ChangeUsername", func() {
		It(
			"Should rename the credentials and keep the password",
			func(ctx SpecContext) {
				renamed := auth.Credentials{
					Username: uuid.New().String(),
					Password: creds.Password,
				}
				Expect(writer.ChangeUsername(ctx, key, renamed.Username)).To(Succeed())
				Expect(svc.Authenticate(ctx, nil, renamed)).To(Equal(key))
				Expect(svc.Authenticate(ctx, nil, creds)).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It("Should succeed when the username is unchanged", func(ctx SpecContext) {
			Expect(writer.ChangeUsername(ctx, key, creds.Username)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
		})
		It(
			"Should return ErrUniqueViolation when another user holds the username",
			func(ctx SpecContext) {
				other := auth.Credentials{
					Username: uuid.New().String(),
					Password: "password",
				}
				Expect(writer.Register(ctx, uuid.New(), other)).To(Succeed())
				Expect(writer.ChangeUsername(ctx, key, other.Username)).
					To(MatchError(query.ErrUniqueViolation))
				Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
			},
		)
		It(
			"Should return query.ErrNotFound when the key has no credentials",
			func(ctx SpecContext) {
				Expect(writer.ChangeUsername(ctx, uuid.New(), uuid.New().String())).
					To(MatchError(query.ErrNotFound))
			},
		)
		It("Should reject an empty username", func(ctx SpecContext) {
			Expect(writer.ChangeUsername(ctx, key, "")).
				To(MatchError(ContainSubstring("username: required")))
		})
	})
	Describe("ChangePassword", func() {
		const newPassword = "new-password"
		It("Should set a new password for the given key", func(ctx SpecContext) {
			Expect(writer.ChangePassword(ctx, key, newPassword)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Authenticate(ctx, nil, auth.Credentials{
				Username: creds.Username,
				Password: newPassword,
			})).To(Equal(key))
		})
		It(
			"Should return query.ErrNotFound when the key has no credentials",
			func(ctx SpecContext) {
				Expect(writer.ChangePassword(ctx, uuid.New(), newPassword)).
					To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should propagate the bcrypt error verbatim when the new password is too long to hash",
			func(ctx SpecContext) {
				err := writer.ChangePassword(ctx, key, strings.Repeat("a", 73))
				Expect(err).To(MatchError(bcrypt.ErrPasswordTooLong))
				Expect(errors.Is(err, auth.ErrInvalidCredentials)).To(BeFalse())
			},
		)
		It("Should reject an empty password", func(ctx SpecContext) {
			Expect(writer.ChangePassword(ctx, key, "")).
				To(MatchError(ContainSubstring("password: required")))
			Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(key))
		})
	})
	Describe("Deactivate", func() {
		It("Should delete the credentials", func(ctx SpecContext) {
			Expect(writer.Deactivate(ctx, key)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should be idempotent", func(ctx SpecContext) {
			for range 2 {
				Expect(writer.Deactivate(ctx, key)).To(Succeed())
			}
		})
		It("Should free the username for another user", func(ctx SpecContext) {
			Expect(writer.Deactivate(ctx, key)).To(Succeed())
			other := uuid.New()
			Expect(writer.Register(ctx, other, creds)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, creds)).To(Equal(other))
		})
	})
})
