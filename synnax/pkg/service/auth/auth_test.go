// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("KV", Ordered, Serial, func() {
	var (
		authenticator auth.Authenticator
		db            *gorp.DB
		tx            gorp.Tx
		creds         auth.InsecureCredentials
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		authenticator = auth.MultiAuthenticator{&auth.KV{DB: db}}
		creds = auth.InsecureCredentials{Username: "user", Password: "pass"}
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	BeforeEach(func() { tx = db.OpenTx() })
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
	Describe("Name", func() {
		It("Should register the credentials in the key-value store", func() {
			w := authenticator.NewWriter(tx)
			err := w.Register(ctx, creds)
			Expect(err).To(BeNil())
			var c auth.SecureCredentials
			Expect(gorp.NewRetrieve[string, auth.SecureCredentials]().
				WhereKeys(creds.Username).
				Entry(&c).
				Exec(ctx, tx)).To(Succeed())
		})
		It("Should return a UniqueViolation error when the username is already registered", func() {
			w := authenticator.NewWriter(tx)
			err := w.Register(ctx, creds)
			Expect(err).To(BeNil())
			err = w.Register(ctx, creds)
			Expect(errors.Is(err, errors.Wrap(auth.Error, "username already registered"))).To(BeTrue())
		})
	})
	Describe("Authenticate", func() {
		It("Should return a nil error for valid credentials", func() {
			err := authenticator.NewWriter(nil).Register(ctx, creds)
			Expect(err).To(BeNil())
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: creds.Username,
				Password: "invalid",
			}
			Expect(authenticator.Authenticate(ctx, invalidCreds)).
				To(MatchError(auth.InvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: "invalid",
				Password: "invalid",
			}
			Expect(authenticator.Authenticate(ctx, invalidCreds)).
				To(MatchError(auth.InvalidCredentials))

		})
	})
	Describe("UpdateUsername", func() {
		It("Should update the username", func() {
			w := authenticator.NewWriter(tx)
			Expect(w.UpdateUsername(ctx, creds, "new-user")).To(Succeed())
			var c auth.SecureCredentials
			Expect(gorp.NewRetrieve[string, auth.SecureCredentials]().
				WhereKeys("new-user").
				Entry(&c).
				Exec(ctx, tx)).To(Succeed())
			creds.Username = c.Username
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: creds.Username,
				Password: "invalid",
			}
			Expect(authenticator.NewWriter(tx).UpdateUsername(ctx, invalidCreds, "new-user")).
				To(MatchError(auth.InvalidCredentials))
		})
		It("Should return a UniqueViolation error when the username is already registered", func() {
			w := authenticator.NewWriter(tx)
			Expect(w.Register(ctx, auth.InsecureCredentials{
				Username: "old-user",
				Password: "pass",
			})).To(Succeed())
			Expect(w.Register(ctx, creds)).To(Succeed())
			err := w.UpdateUsername(ctx, creds, "old-user")
			Expect(err).ToNot(BeNil())
			Expect(errors.Is(errors.Wrap(auth.Error, "username already registered"), err)).To(BeTrue())
		})
	})
	Describe("UpdatePassword", func() {
		It("Should update the users password", func() {
			w := authenticator.NewWriter(tx)
			Expect(w.Register(ctx, creds)).To(Succeed())
			var newPass password.Raw = "new-pass"
			Expect(w.UpdatePassword(ctx, creds, newPass)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(authenticator.Authenticate(ctx, creds)).ToNot(Succeed())
			creds.Password = newPass
			Expect(authenticator.Authenticate(ctx, creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: creds.Username,
				Password: "invalid",
			}
			w := authenticator.NewWriter(tx)
			var newPass password.Raw = "new-pass"
			Expect(w.UpdatePassword(ctx, invalidCreds, newPass)).To(MatchError(auth.InvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: "invalid",
				Password: "invalid",
			}
			w := authenticator.NewWriter(tx)
			var newPass password.Raw = "new-pass"
			Expect(w.UpdatePassword(ctx, invalidCreds, newPass)).To(MatchError(auth.InvalidCredentials))
		})
	})
})
