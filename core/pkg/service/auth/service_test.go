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
	const password = "password"
	var (
		svc *auth.Service
		key auth.Key
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		key = uuid.New()
		Expect(svc.NewWriter(nil).Register(ctx, key, password)).To(Succeed())
	})

	Describe("Authenticate", func() {
		It("Should return a nil error for a valid password", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, nil, key, password)).To(Succeed())
		})
		It(
			"Should return an InvalidCredentials error when the password is wrong",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, key, "invalid")).
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should return an InvalidCredentials error when the key has no password",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, uuid.New(), password)).
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should return a validation error when the password is empty",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, nil, key, "")).
					To(MatchError(ContainSubstring("password: required")))
			},
		)
		It(
			"Should read from the supplied tx so an in-flight password rotation is observed",
			func(ctx SpecContext) {
				newPass := "rotated-" + uuid.New().String()
				tx := DeferClose(db.OpenTx())
				Expect(svc.NewWriter(tx).ChangePassword(ctx, key, newPass)).
					To(Succeed())
				Expect(svc.Authenticate(ctx, tx, key, newPass)).To(Succeed())
				Expect(svc.Authenticate(ctx, tx, key, password)).
					To(MatchError(auth.ErrInvalidCredentials))
				Expect(svc.Authenticate(ctx, nil, key, password)).To(Succeed())
			},
		)
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
