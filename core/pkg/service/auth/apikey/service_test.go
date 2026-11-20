// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey_test

import (
	"context"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func create(
	ctx context.Context,
	svc *apikey.Service,
	sub resource.ID,
	name string,
) (apikey.APIKey, string) {
	GinkgoHelper()
	var (
		k     apikey.APIKey
		token string
	)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		k, token, err = svc.NewWriter(tx).Create(ctx, sub, name)
		return err
	})).To(Succeed())
	return k, token
}

func proof(key string) auth.Proof {
	return auth.Proof{Variant: auth.APIKeyProof{
		APIKeyCredentials: auth.APIKeyCredentials{Key: key},
	}}
}

var _ = Describe("ServiceConfig", func() {
	var cfg apikey.ServiceConfig
	BeforeEach(func() {
		cfg = apikey.ServiceConfig{
			DB:              db,
			Instrumentation: alamos.New("apikey-config-test"),
		}
	})
	Describe("Override", func() {
		It("Should take each field from the override when the base is zero", func() {
			result := apikey.ServiceConfig{}.Override(cfg)
			Expect(result.DB).To(Equal(db))
			Expect(result.Instrumentation).To(Equal(cfg.Instrumentation))
		})
		It("Should keep each field from the base when the override is zero", func() {
			result := cfg.Override(apikey.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
			Expect(result.Instrumentation).To(Equal(cfg.Instrumentation))
		})
	})
})

var _ = Describe("Service", func() {
	var svc *apikey.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
	})
	It("Should require a DB", func(ctx SpecContext) {
		Expect(apikey.OpenService(ctx)).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
	Describe("Create", func() {
		It("Should return a key that starts with syk_", func(ctx SpecContext) {
			sub := subject()
			k, token := create(ctx, svc, sub, "ci")
			Expect(token).To(HavePrefix("syk_"))
			Expect(k.Subject).To(Equal(sub))
			Expect(k.Name).To(Equal("ci"))
			Expect(k.Created).ToNot(BeZero())
			var stored apikey.APIKey
			Expect(svc.NewRetrieve().Where(apikey.MatchKeys(k.Key)).Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Hash).To(HaveLen(32))
		})
		It("Should reject an invalid subject", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, _, err := svc.NewWriter(tx).Create(ctx, resource.ID{}, "")
				return err
			})).
				To(MatchError(ContainSubstring("subject")))
		})
	})
	Describe("Authenticate", func() {
		It("Should prove the key's subject and credential", func(ctx SpecContext) {
			sub := subject()
			k, token := create(ctx, svc, sub, "")
			Expect(svc.Authenticate(ctx, proof(token))).To(Equal(auth.Identity{
				Subject:    sub,
				Credential: k.Key,
				Version:    k.Created,
			}))
		})
		It("Should reject a wrong secret", func(ctx SpecContext) {
			k, _ := create(ctx, svc, subject(), "")
			wrong, _ := MustSucceed2(auth.NewToken("syk_", k.Key))
			Expect(svc.Authenticate(ctx, proof(wrong))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an unknown key", func(ctx SpecContext) {
			token, _ := MustSucceed2(auth.NewToken("syk_", uuid.New()))
			Expect(svc.Authenticate(ctx, proof(token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			_, token := create(ctx, svc, subject(), "")
			session := strings.Replace(token, "syk_", "sys_", 1)
			Expect(svc.Authenticate(ctx, proof(session))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an empty key", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(""))).Error().
				To(MatchError(ContainSubstring("key: required")))
		})
		It("Should reject a proof for another method", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, auth.Proof{Variant: auth.PasswordProof{}})).
				Error().To(MatchError(validate.ErrValidation))
		})
	})
	Describe("Version", func() {
		It("Should return when the key was created", func(ctx SpecContext) {
			k, _ := create(ctx, svc, subject(), "")
			Expect(svc.Version(ctx, db, k.Key)).To(Equal(k.Created))
		})
		It("Should return a not found error for an unknown key", func(ctx SpecContext) {
			Expect(svc.Version(ctx, db, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
	Describe("Delete", func() {
		It("Should remove only the given keys", func(ctx SpecContext) {
			sub := subject()
			gone, goneToken := create(ctx, svc, sub, "gone")
			_, keptToken := create(ctx, svc, sub, "kept")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, gone.Key)
			})).To(Succeed())
			Expect(svc.NewRetrieve().Where(apikey.MatchKeys(gone.Key)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			Expect(svc.Authenticate(ctx, proof(goneToken))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			MustSucceed(svc.Authenticate(ctx, proof(keptToken)))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, uuid.New())
			})).To(Succeed())
		})
	})
	Describe("Subject", func() {
		It("Should delete every key of a subject when the subject is deleted", func(
			ctx SpecContext,
		) {
			gone, kept := subject(), subject()
			create(ctx, svc, gone, "a")
			create(ctx, svc, gone, "b")
			create(ctx, svc, kept, "c")
			deleteSubject(ctx, gone)
			var keys []apikey.APIKey
			Expect(svc.NewRetrieve().Where(apikey.MatchSubjects(gone, kept)).
				Entries(&keys).Exec(ctx, db)).To(Succeed())
			Expect(keys).To(HaveLen(1))
			Expect(keys[0].Subject).To(Equal(kept))
		})
	})
})
