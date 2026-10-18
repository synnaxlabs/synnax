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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		svc   *password.Service
		w     password.Writer
		alice ontology.ID
		name  string
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		w = svc.NewWriter(nil)
		alice = subject()
		name = uuid.New().String()
		Expect(w.Set(ctx, alice, auth.Credentials{Username: name, Password: "p"})).
			To(Succeed())
	})
	retrieve := func(ctx SpecContext, subject ontology.ID) password.Password {
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
					w.Set(
						ctx,
						alice,
						auth.Credentials{Username: renamed, Password: "q"},
					),
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
				w.Set(ctx, subject(), auth.Credentials{Username: name, Password: "p"}),
			).
				To(MatchError(auth.ErrRepeatedUsername))
		})
		It("Should reject empty credentials", func(ctx SpecContext) {
			Expect(w.Set(ctx, alice, auth.Credentials{Username: name})).
				To(MatchError(ContainSubstring("password: required")))
		})
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
			Expect(w.Set(ctx, bob, auth.Credentials{Username: taken, Password: "p"})).
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
	Describe("DeleteFor", func() {
		It("Should remove the passwords of the given subjects", func(ctx SpecContext) {
			bob := subject()
			Expect(w.Set(ctx, bob, auth.Credentials{
				Username: uuid.New().String(),
				Password: "p",
			})).To(Succeed())
			Expect(w.DeleteFor(ctx, alice, bob)).To(Succeed())
			Expect(svc.NewRetrieve().Where(password.MatchSubjects(alice, bob)).
				Exists(ctx, nil)).To(BeFalse())
		})
		It("Should succeed for a subject without a password", func(ctx SpecContext) {
			Expect(w.DeleteFor(ctx, subject())).To(Succeed())
		})
	})
})
