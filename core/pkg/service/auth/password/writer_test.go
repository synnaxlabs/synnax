// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		svc   *password.Service
		w     password.Writer
		alice resource.ID
		name  string
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		w = svc.NewWriter(nil)
		alice = subject()
		name = uuid.New().String()
		Expect(w.Set(ctx, alice, name, hash(svc, "p"))).
			To(Succeed())
	})
	retrieve := func(ctx SpecContext, subject resource.ID) password.Password {
		GinkgoHelper()
		var p password.Password
		Expect(svc.NewRetrieve().Where(password.MatchSubjects(subject)).Entry(&p).
			Exec(ctx, nil)).To(Succeed())
		return p
	}
	Describe("Set", func() {
		It("Should store one password per subject", func(ctx SpecContext) {
			p := retrieve(ctx, alice)
			Expect(p.Username).To(Equal(name))
			Expect(p.Created).ToNot(BeZero())
			Expect(p.Hash).To(HavePrefix("$argon2id$"))
		})
		It(
			"Should replace the subject's password and keep its key",
			func(ctx SpecContext) {
				before := retrieve(ctx, alice)
				renamed := uuid.New().String()
				Expect(
					w.Set(ctx, alice, renamed, hash(svc, "q")),
				).
					To(Succeed())
				after := retrieve(ctx, alice)
				Expect(after.Key).To(Equal(before.Key))
				Expect(after.Username).To(Equal(renamed))
				Expect(after.Hash).ToNot(Equal(before.Hash))
				Expect(svc.NewRetrieve().Where(password.MatchSubjects(alice)).
					Count(ctx, nil)).To(Equal(1))
				MustSucceed(svc.Authenticate(ctx, proof(renamed, "q")))
				Expect(svc.Authenticate(ctx, proof(name, "p"))).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It("Should reject a username another subject holds", func(ctx SpecContext) {
			Expect(
				w.Set(ctx, subject(), name, hash(svc, "p")),
			).
				To(MatchError(auth.ErrRepeatedUsername))
		})
		It("Should reject an empty username", func(ctx SpecContext) {
			Expect(w.Set(ctx, alice, "", hash(svc, "p"))).
				To(MatchError(ContainSubstring("username: required")))
		})
		It("Should refuse a hash proven against a password that changed", func(
			ctx SpecContext,
		) {
			stale := MustSucceed(svc.HashProven(ctx, auth.PasswordProof{
				Credentials: auth.Credentials{Username: name, Password: "p"},
			}, "stale"))
			Expect(w.ChangePassword(ctx, alice, hash(svc, "q"))).To(Succeed())
			Expect(w.Set(ctx, alice, name, stale)).
				To(MatchError(auth.ErrInvalidCredentials))
			MustSucceed(svc.Authenticate(ctx, proof(name, "q")))
		})
	})
	Describe("ChangePassword", func() {
		It("Should replace the hash and keep the username", func(ctx SpecContext) {
			before := retrieve(ctx, alice)
			Expect(w.ChangePassword(ctx, alice, hash(svc, "q"))).To(Succeed())
			after := retrieve(ctx, alice)
			Expect(after.Key).To(Equal(before.Key))
			Expect(after.Username).To(Equal(name))
			Expect(after.Hash).ToNot(Equal(before.Hash))
			MustSucceed(svc.Authenticate(ctx, proof(name, "q")))
			Expect(svc.Authenticate(ctx, proof(name, "p"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should refuse a hash proven against a password that changed", func(
			ctx SpecContext,
		) {
			stale := MustSucceed(svc.HashProven(ctx, auth.PasswordProof{
				Credentials: auth.Credentials{Username: name, Password: "p"},
			}, "stale"))
			Expect(w.ChangePassword(ctx, alice, hash(svc, "q"))).To(Succeed())
			Expect(w.ChangePassword(ctx, alice, stale)).
				To(MatchError(auth.ErrInvalidCredentials))
			MustSucceed(svc.Authenticate(ctx, proof(name, "q")))
		})
		It(
			"Should return not found for a subject without a password",
			func(ctx SpecContext) {
				Expect(
					w.ChangePassword(ctx, subject(), hash(svc, "q")),
				).To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("Rename", func() {
		It("Should change the username", func(ctx SpecContext) {
			renamed := uuid.New().String()
			Expect(w.Rename(ctx, alice, renamed)).To(Succeed())
			MustSucceed(svc.Authenticate(ctx, proof(renamed, "p")))
			Expect(svc.Authenticate(ctx, proof(name, "p"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a username another subject holds", func(ctx SpecContext) {
			bob := subject()
			taken := uuid.New().String()
			Expect(w.Set(ctx, bob, taken, hash(svc, "p"))).
				To(Succeed())
			Expect(w.Rename(ctx, alice, taken)).To(MatchError(auth.ErrRepeatedUsername))
		})
		It(
			"Should return not found for a subject without a password",
			func(ctx SpecContext) {
				Expect(w.Rename(ctx, subject(), uuid.New().String())).
					To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("Set", func() {
		It("Should refuse a subject that does not exist", func(ctx SpecContext) {
			missing := resource.ID{Type: resource.TypeUser, Key: uuid.New().String()}
			Expect(w.Set(ctx, missing, uuid.New().String(), hash(svc, "p"))).To(
				MatchError(
					ContainSubstring(
						"subject: user " + missing.Key + " does not exist",
					),
				),
			)
		})
	})
})
