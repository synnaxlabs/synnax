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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/bcrypt"
)

func subject() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.New().String()}
}

func proof(username, pwd string) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{
		Credentials: auth.Credentials{Username: username, Password: pwd},
	}}
}

func storedHash(
	ctx context.Context,
	svc *password.Service,
	subject ontology.ID,
) []byte {
	GinkgoHelper()
	var p password.Password
	Expect(svc.NewRetrieve().Where(password.MatchSubjects(subject)).Entry(&p).
		Exec(ctx, nil)).To(Succeed())
	return p.Hash
}

// createHash bypasses the writer to plant a hash made by any algorithm.
func createHash(
	ctx context.Context,
	db *gorp.DB,
	subject ontology.ID,
	username string,
	hash []byte,
) {
	GinkgoHelper()
	Expect(
		gorp.WrapWriter[password.Key, password.Password](db).Set(ctx, password.Password{
			Key:      uuid.New(),
			Subject:  subject,
			Username: username,
			Hash:     hash,
		}),
	).To(Succeed())
}

var _ = Describe("ServiceConfig", func() {
	var cfg password.ServiceConfig
	BeforeEach(func() {
		cfg = password.ServiceConfig{DB: db, FIPS: new(true)}
	})
	Describe("Override", func() {
		It("Should take each field from the override when the base is zero", func() {
			result := password.ServiceConfig{}.Override(cfg)
			Expect(result.DB).To(Equal(db))
			Expect(*result.FIPS).To(BeTrue())
		})
		It("Should keep each field from the base when the override is zero", func() {
			result := cfg.Override(password.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
			Expect(*result.FIPS).To(BeTrue())
		})
	})
	Describe("Validate", func() {
		It("Should succeed when every required field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
		})
		It("Should return an error when DB is nil", func() {
			cfg.DB = nil
			Expect(
				cfg.Validate(),
			).To(MatchError(ContainSubstring("db: must be non-nil")))
		})
		It("Should return an error when FIPS is nil", func() {
			cfg.FIPS = nil
			Expect(
				cfg.Validate(),
			).To(MatchError(ContainSubstring("fips: must be non-nil")))
		})
	})
})

var _ = Describe("OpenService", func() {
	It("Should return an error when the config is invalid", func(ctx SpecContext) {
		Expect(password.OpenService(ctx, password.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
})

var _ = Describe("Service", func() {
	var (
		svc      *password.Service
		alice    ontology.ID
		username string
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		alice = subject()
		username = uuid.New().String()
		Expect(svc.NewWriter(nil).Set(ctx, alice, auth.Credentials{
			Username: username,
			Password: "secret",
		})).To(Succeed())
	})
	Describe("Authenticate", func() {
		It("Should return the identity the password proves", func(ctx SpecContext) {
			identity := MustSucceed(svc.Authenticate(ctx, proof(username, "secret")))
			Expect(identity.Subject).To(Equal(alice))
			var p password.Password
			Expect(svc.NewRetrieve().Where(password.MatchSubjects(alice)).Entry(&p).
				Exec(ctx, nil)).To(Succeed())
			Expect(identity.Credential).To(Equal(p.Key))
			Expect(identity.Version).To(Equal(p.Created))
		})
		It(
			"Should return the same error for a wrong password and an unknown username",
			func(ctx SpecContext) {
				Expect(svc.Authenticate(ctx, proof(username, "wrong"))).Error().
					To(MatchError(auth.ErrInvalidCredentials))
				Expect(svc.Authenticate(ctx, proof("nobody", "secret"))).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It("Should reject a proof of another method", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, auth.Proof{})).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject empty credentials", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(username, ""))).Error().
				To(MatchError(ContainSubstring("password: required")))
		})
		It(
			"Should check a bcrypt hash and store it again as Argon2id",
			func(ctx SpecContext) {
				bob := subject()
				hash := MustSucceed(
					bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost),
				)
				createHash(ctx, db, bob, uuid.New().String(), hash)
				var p password.Password
				Expect(svc.NewRetrieve().Where(password.MatchSubjects(bob)).Entry(&p).
					Exec(ctx, nil)).To(Succeed())
				Expect(
					MustSucceed(svc.Authenticate(ctx, proof(p.Username, "p"))).Subject,
				).
					To(Equal(bob))
				Expect(storedHash(ctx, svc, bob)).To(HavePrefix("$argon2id$"))
				MustSucceed(svc.Authenticate(ctx, proof(p.Username, "p")))
				Expect(svc.Authenticate(ctx, proof(p.Username, "wrong"))).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		DescribeTable(
			"Should fail on a malformed hash without blaming the password",
			func(ctx SpecContext, hash string) {
				bob := subject()
				name := uuid.New().String()
				createHash(ctx, db, bob, name, []byte(hash))
				Expect(svc.Authenticate(ctx, proof(name, "p"))).Error().To(SatisfyAll(
					MatchError(ContainSubstring("malformed")),
					Not(MatchError(auth.ErrInvalidCredentials)),
				))
			},
			Entry("argon2id with too few parts", "$argon2id$garbage"),
			Entry("argon2id with a bad salt", "$argon2id$v=19$m=1,t=1,p=1$!!$AAAA"),
			Entry("argon2id with a bad key", "$argon2id$v=19$m=1,t=1,p=1$AAAA$!!"),
			Entry("argon2id with bad parameters", "$argon2id$v=19$m=x$AAAA$AAAA"),
			Entry("pbkdf2 with bad parameters", "$pbkdf2-sha256$i=x$AAAA$AAAA"),
			Entry("bcrypt with a bad body", "$2a$garbage"),
		)
	})
	Describe("NewRetrieve", func() {
		var bob ontology.ID
		BeforeEach(func(ctx SpecContext) {
			bob = subject()
			Expect(svc.NewWriter(nil).Set(ctx, bob, auth.Credentials{
				Username: uuid.New().String(),
				Password: "secret",
			})).To(Succeed())
		})
		It("Should find a password by key", func(ctx SpecContext) {
			var p password.Password
			Expect(svc.NewRetrieve().Where(password.MatchSubjects(alice)).Entry(&p).
				Exec(ctx, nil)).To(Succeed())
			var byKey password.Password
			Expect(svc.NewRetrieve().Where(password.MatchKeys(p.Key)).Entry(&byKey).
				Exec(ctx, nil)).To(Succeed())
			Expect(byKey).To(Equal(p))
		})
		It("Should compose filters", func(ctx SpecContext) {
			var ps []password.Password
			Expect(svc.NewRetrieve().Where(password.Or(
				password.MatchSubjects(alice),
				password.MatchSubjects(bob),
			)).Entries(&ps).Exec(ctx, nil)).To(Succeed())
			Expect(ps).To(HaveLen(2))
			Expect(svc.NewRetrieve().Where(password.And(
				password.MatchSubjects(alice),
				password.Not(password.MatchSubjects(alice)),
			)).Count(ctx, nil)).To(Equal(0))
			Expect(svc.NewRetrieve().Where(password.Match(
				func(_ gorp.Context, _ password.Retrieve, p *password.Password) (bool, error) {
					return p.Subject == bob, nil
				},
			)).Count(ctx, nil)).To(Equal(1))
		})
		It("Should page through passwords", func(ctx SpecContext) {
			both := password.Or(
				password.MatchSubjects(alice),
				password.MatchSubjects(bob),
			)
			var first, second []password.Password
			Expect(svc.NewRetrieve().Where(both).Limit(1).Entries(&first).
				Exec(ctx, nil)).To(Succeed())
			Expect(svc.NewRetrieve().Where(both).Limit(1).Offset(1).Entries(&second).
				Exec(ctx, nil)).To(Succeed())
			Expect(first).To(HaveLen(1))
			Expect(second).To(HaveLen(1))
			Expect(first[0].Key).ToNot(Equal(second[0].Key))
		})
	})
	Describe("Version", func() {
		It("Should move each time the password is set", func(ctx SpecContext) {
			identity := MustSucceed(svc.Authenticate(ctx, proof(username, "secret")))
			w := svc.NewWriter(nil)
			Expect(w.ChangePassword(ctx, alice, "changed")).To(Succeed())
			changed := MustSucceed(svc.Version(ctx, nil, identity.Credential))
			Expect(changed).To(BeNumerically(">", identity.Version))
			Expect(w.Set(ctx, alice, auth.Credentials{
				Username: username,
				Password: "again",
			})).To(Succeed())
			Expect(svc.Version(ctx, nil, identity.Credential)).
				To(BeNumerically(">", changed))
		})
		It(
			"Should return a not found error for an unknown credential",
			func(ctx SpecContext) {
				Expect(svc.Version(ctx, nil, uuid.New())).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
	})
	Describe("DeleteFor", func() {
		It("Should remove the subject's password", func(ctx SpecContext) {
			Expect(svc.DeleteFor(ctx, nil, alice)).To(Succeed())
			Expect(svc.Authenticate(ctx, proof(username, "secret"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})

var _ = Describe("Service with FIPS", func() {
	var svc *password.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(password.OpenService(ctx, password.ServiceConfig{
			DB:   db,
			FIPS: new(true),
		}))
	})
	It("Should hash new passwords with PBKDF2", func(ctx SpecContext) {
		alice := subject()
		name := uuid.New().String()
		Expect(svc.NewWriter(nil).Set(ctx, alice, auth.Credentials{
			Username: name,
			Password: "secret",
		})).To(Succeed())
		Expect(storedHash(ctx, svc, alice)).To(HavePrefix("$pbkdf2-sha256$"))
		Expect(MustSucceed(svc.Authenticate(ctx, proof(name, "secret"))).Subject).
			To(Equal(alice))
		Expect(svc.Authenticate(ctx, proof(name, "wrong"))).Error().
			To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should refuse a bcrypt hash", func(ctx SpecContext) {
		name := uuid.New().String()
		hash := MustSucceed(bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost))
		createHash(ctx, db, subject(), name, hash)
		Expect(svc.Authenticate(ctx, proof(name, "p"))).Error().To(SatisfyAll(
			MatchError(auth.ErrInvalidCredentials),
			MatchError(ContainSubstring("does not accept")),
		))
	})
	It("Should refuse an Argon2id hash", func(ctx SpecContext) {
		alice := subject()
		name := uuid.New().String()
		plain := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		Expect(plain.NewWriter(nil).Set(ctx, alice, auth.Credentials{
			Username: name,
			Password: "p",
		})).To(Succeed())
		Expect(svc.Authenticate(ctx, proof(name, "p"))).Error().To(SatisfyAll(
			MatchError(auth.ErrInvalidCredentials),
			MatchError(ContainSubstring("does not accept")),
		))
	})
	It(
		"Should let a Core without FIPS check and replace a PBKDF2 hash",
		func(ctx SpecContext) {
			alice := subject()
			name := uuid.New().String()
			Expect(svc.NewWriter(nil).Set(ctx, alice, auth.Credentials{
				Username: name,
				Password: "p",
			})).To(Succeed())
			plain := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
			MustSucceed(plain.Authenticate(ctx, proof(name, "p")))
			Expect(storedHash(ctx, plain, alice)).To(HavePrefix("$argon2id$"))
		},
	)
})
