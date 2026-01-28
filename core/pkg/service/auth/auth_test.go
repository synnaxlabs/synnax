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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("KV", Ordered, Serial, func() {
	var (
		authenticator    auth.Authenticator
		db               *gorp.DB
		creds            auth.InsecureCredentials
		invalidPassCreds auth.InsecureCredentials
		invalidUserCreds auth.InsecureCredentials
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		authenticator = &auth.KV{DB: db}
		creds = auth.InsecureCredentials{Username: "username", Password: "password"}
		invalidPassCreds = auth.InsecureCredentials{Username: creds.Username, Password: "invalid"}
		invalidUserCreds = auth.InsecureCredentials{Username: "invalid", Password: creds.Password}
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	BeforeEach(func() {
		Expect(authenticator.NewWriter(nil).Register(ctx, creds)).To(Succeed())
	})
	AfterEach(func() {
		Expect(authenticator.NewWriter(nil).InsecureDeactivate(ctx, creds.Username)).To(Succeed())
	})
	Describe("Registering", func() {
		It("Should register the credentials", func() {
			var secCreds auth.SecureCredentials
			tx := db.OpenTx()
			Expect(gorp.NewRetrieve[string, auth.SecureCredentials]().
				WhereKeys(creds.Username).
				Entry(&secCreds).
				Exec(ctx, tx)).To(Succeed())
			Expect(secCreds.Username).To(Equal(creds.Username))
			Expect(secCreds.Password.Validate(creds.Password)).To(Succeed())
		})
		It("Should return a RepeatedUsername error when the username is already registered", func() {
			Expect(errors.Is(authenticator.NewWriter(nil).Register(ctx, creds), auth.RepeatedUsername)).To(BeTrue())
		})
	})
	Describe("Authenticating", func() {
		It("Should return a nil error for valid credentials", func() {
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			Expect(authenticator.Authenticate(ctx, invalidPassCreds)).To(MatchError(auth.InvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func() {
			Expect(authenticator.Authenticate(ctx, invalidUserCreds)).To(MatchError(auth.InvalidCredentials))
		})
	})
	Describe("Changing the username", func() {
		var newCreds auth.InsecureCredentials
		BeforeAll(func() {
			newCreds = auth.InsecureCredentials{
				Username: "new-username",
				Password: creds.Password,
			}
		})
		AfterEach(func() {
			Expect(authenticator.NewWriter(nil).InsecureDeactivate(ctx, newCreds.Username)).To(Succeed())
		})
		When("using credentials", func() {
			It("Should update the username", func() {
				Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, creds, newCreds.Username)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.InvalidCredentials))
			})
			It("Should return an InvalidCredentials error when the password is wrong", func() {
				Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, invalidPassCreds, newCreds.Username)).To(MatchError(auth.InvalidCredentials))
				Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, newCreds)).To(MatchError(auth.InvalidCredentials))
			})
			It("Should return an InvalidCredentials error when the user can't be found", func() {
				Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, invalidUserCreds, newCreds.Username)).To(MatchError(auth.InvalidCredentials))
				Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, newCreds)).To(MatchError(auth.InvalidCredentials))
			})
			It("Should do nothing when the username is the same", func() {
				Expect(authenticator.NewWriter(nil).UpdateUsername(ctx, creds, creds.Username)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
			})
			It("Should raise a RepeatedUsername error when the username is already registered", func() {
				Expect(authenticator.NewWriter(nil).Register(ctx, newCreds)).To(Succeed())
				Expect(errors.Is(authenticator.NewWriter(nil).UpdateUsername(ctx, creds, newCreds.Username), auth.RepeatedUsername)).To(BeTrue())
				Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
			})
		})
		When("using usernames", func() {
			It("Should update the username", func() {
				Expect(authenticator.NewWriter(nil).InsecureUpdateUsername(ctx, creds.Username, newCreds.Username)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.InvalidCredentials))
			})
			It("Should do nothing when the username is the same", func() {
				Expect(authenticator.NewWriter(nil).InsecureUpdateUsername(ctx, creds.Username, creds.Username)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
			})
			It("Should return an RepeatedUsername error when the username is already registered", func() {
				Expect(authenticator.NewWriter(nil).Register(ctx, newCreds)).To(Succeed())
				Expect(errors.Is(authenticator.NewWriter(nil).InsecureUpdateUsername(ctx, creds.Username, newCreds.Username), auth.RepeatedUsername)).To(BeTrue())
				Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
				Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
			})
		})
	})
	Describe("Updating a password", func() {
		var newCreds auth.InsecureCredentials
		BeforeAll(func() {
			newCreds = auth.InsecureCredentials{
				Username: creds.Username,
				Password: "new-password",
			}
		})
		It("Should update the password", func() {
			Expect(authenticator.NewWriter(nil).UpdatePassword(ctx, creds, newCreds.Password)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.InvalidCredentials))
			Expect(authenticator.Authenticate(ctx, newCreds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			Expect(authenticator.NewWriter(nil).UpdatePassword(ctx, invalidPassCreds, newCreds.Password)).To(MatchError(auth.InvalidCredentials))
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, newCreds)).To(MatchError(auth.InvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func() {
			Expect(authenticator.NewWriter(nil).UpdatePassword(ctx, invalidUserCreds, newCreds.Password)).To(MatchError(auth.InvalidCredentials))
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, newCreds)).To(MatchError(auth.InvalidCredentials))
			Expect(authenticator.NewWriter(nil).InsecureDeactivate(ctx, creds.Username)).To(Succeed())
		})
	})
	Describe("Deactivating", func() {
		It("Should delete the credentials", func() {
			Expect(authenticator.NewWriter(nil).InsecureDeactivate(ctx, creds.Username)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.InvalidCredentials))
		})
		It("Should be idempotent", func() {
			for range 2 {
				Expect(authenticator.NewWriter(nil).InsecureDeactivate(ctx, creds.Username)).To(Succeed())
			}
		})
		It("Should delete multiple credentials", func() {
			creds2 := auth.InsecureCredentials{Username: "username2", Password: "password"}
			Expect(authenticator.NewWriter(nil).Register(ctx, creds2)).To(Succeed())
			Expect(authenticator.NewWriter(nil).InsecureDeactivate(ctx, creds.Username, creds2.Username)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).To(MatchError(auth.InvalidCredentials))
			Expect(authenticator.Authenticate(ctx, creds2)).To(MatchError(auth.InvalidCredentials))
		})
	})
	Describe("Error Encoding and Decoding", func() {
		DescribeTable(
			"Correctly encodes/decodes a network portable freighter error",
			func(err error) {
				pld := errors.Encode(ctx, err, false)
				oErr := errors.Decode(ctx, pld)
				Expect(oErr).To(MatchError(err))
			},
			Entry("InvalidCredentials", auth.InvalidCredentials),
			Entry("RepeatedUsername", auth.RepeatedUsername),
			Entry("InvalidToken", auth.InvalidToken),
			Entry("ExpiredToken", auth.ExpiredToken),
			Entry("Error", auth.Error),
		)

	})
})
