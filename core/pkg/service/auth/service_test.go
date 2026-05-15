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
	"github.com/google/uuid"
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
		svc              *auth.Service
		creds            auth.Credentials
		invalidPassCreds auth.Credentials
		invalidUserCreds auth.Credentials
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		creds = auth.Credentials{Username: uuid.NewString(), Password: "password"}
		invalidPassCreds = auth.Credentials{Username: creds.Username, Password: "invalid"}
		invalidUserCreds = auth.Credentials{Username: uuid.NewString(), Password: creds.Password}
		Expect(svc.NewWriter(nil).Register(ctx, creds)).To(Succeed())
	})

	Describe("Authenticate", func() {
		It("Should return a nil error for valid credentials", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, invalidPassCreds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, invalidUserCreds)).To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should return a validation error when the username is empty", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, auth.Credentials{Password: "password"})).To(
				MatchError(ContainSubstring("username")),
			)
		})
		It("Should return a validation error when the password is empty", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, auth.Credentials{Username: uuid.NewString()})).To(
				MatchError(ContainSubstring("password")),
			)
		})
	})
})
