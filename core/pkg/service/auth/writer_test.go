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
	const password = "password"
	var (
		svc    *auth.Service
		writer auth.Writer
		key    auth.Key
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		writer = svc.NewWriter(nil)
		key = uuid.New()
		Expect(writer.Register(ctx, key, password)).To(Succeed())
	})
	Describe("Register", func() {
		It(
			"Should make the registered password valid for Authenticate",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, key, password)).To(Succeed())
			},
		)
		It(
			"Should replace the password when the key is already registered",
			func(ctx SpecContext) {
				Expect(writer.Register(ctx, key, "replaced")).To(Succeed())
				Expect(svc.Authenticate(ctx, nil, key, "replaced")).To(Succeed())
				Expect(svc.Authenticate(ctx, nil, key, password)).
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should propagate the bcrypt error verbatim when the password is too long to hash",
			func(ctx SpecContext) {
				err := writer.Register(ctx, uuid.New(), strings.Repeat("a", 73))
				Expect(err).To(MatchError(bcrypt.ErrPasswordTooLong))
				Expect(errors.Is(err, auth.ErrInvalidCredentials)).To(BeFalse())
			},
		)
		It(
			"Should reject an empty password before touching the store",
			func(ctx SpecContext) {
				Expect(writer.Register(ctx, key, "")).
					To(MatchError(ContainSubstring("password: required")))
				Expect(svc.Authenticate(ctx, nil, key, password)).To(Succeed())
			},
		)
	})
	Describe("ChangePassword", func() {
		const newPassword = "new-password"
		It("Should set a new password for the given key", func(ctx SpecContext) {
			Expect(writer.ChangePassword(ctx, key, newPassword)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, key, password)).
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Authenticate(ctx, nil, key, newPassword)).To(Succeed())
		})
		It(
			"Should return query.ErrNotFound when the key has no stored password",
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
		It(
			"Should reject an empty password before touching the store",
			func(ctx SpecContext) {
				Expect(writer.ChangePassword(ctx, key, "")).
					To(MatchError(ContainSubstring("password: required")))
				Expect(svc.Authenticate(ctx, nil, key, password)).To(Succeed())
			},
		)
	})
	Describe("Deactivate", func() {
		It("Should delete the stored password", func(ctx SpecContext) {
			Expect(writer.Deactivate(ctx, key)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, key, password)).
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should be idempotent", func(ctx SpecContext) {
			for range 2 {
				Expect(writer.Deactivate(ctx, key)).To(Succeed())
			}
		})
		It("Should delete multiple stored passwords", func(ctx SpecContext) {
			key2 := uuid.New()
			Expect(writer.Register(ctx, key2, password)).To(Succeed())
			Expect(writer.Deactivate(ctx, key, key2)).To(Succeed())
			Expect(svc.Authenticate(ctx, nil, key, password)).
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Authenticate(ctx, nil, key2, password)).
				To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})
