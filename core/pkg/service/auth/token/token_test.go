// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package token_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	. "github.com/synnaxlabs/x/testutil"
)

type mockKeyService struct{ key crypto.PrivateKey }

func (m *mockKeyService) TokenPrivate() crypto.PrivateKey { return m.key }

var _ = Describe("token", func() {
	var (
		svc    *token.Service
		cfg    token.ServiceConfig
		issuer uuid.UUID
		tk     string
	)
	JustBeforeEach(func() {
		k := MustSucceed(rsa.GenerateKey(nil, 1024))
		cfg.KeyProvider = &mockKeyService{key: k}
		svc = MustSucceed(token.NewService(cfg))
		issuer = uuid.New()
		tk = MustSucceed(svc.New(issuer))
	})
	Describe("Signing keys", func() {
		BeforeEach(func() { cfg.Now = time.Now })
		DescribeTable("should sign and validate with every supported key type",
			func(generate func() crypto.PrivateKey) {
				svc := MustSucceed(token.NewService(token.ServiceConfig{
					KeyProvider: &mockKeyService{key: generate()},
					Now:         time.Now,
				}))
				issuer := uuid.New()
				Expect(svc.Validate(MustSucceed(svc.New(issuer)))).To(Equal(issuer))
			},
			Entry("RSA", func() crypto.PrivateKey {
				return MustSucceed(rsa.GenerateKey(rand.Reader, 1024))
			}),
			Entry("ECDSA P-256", func() crypto.PrivateKey {
				return MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
			}),
			Entry("Ed25519", func() crypto.PrivateKey {
				_, priv := MustSucceed2(ed25519.GenerateKey(rand.Reader))
				return priv
			}),
		)
	})

	Describe("Nominal", func() {
		BeforeEach(func() { cfg.Now = time.Now })
		It("Should generate a token for the given issuer", func() {
			Expect(tk).ToNot(BeEmpty())
		})
		It("Should validate a token", func() {
			Expect(svc.Validate(tk)).To(Equal(issuer))
		})
	})

	Describe("Token Expiration", func() {
		var now time.Time

		BeforeEach(func() {
			now = time.Time{}
			cfg.Expiration = time.Second * 10
			cfg.RefreshThreshold = time.Second * 8
			cfg.Now = func() time.Time { return now }
		})

		It(
			"Should refresh the token if the user submits a validation request within the refresh threshold",
			func() {
				id, newToken := MustSucceed2(svc.ValidateMaybeRefresh(tk))
				Expect(id).To(Equal(issuer))
				Expect(newToken).To(BeEmpty())
				now = now.Add(time.Second * 6)
				id, newToken = MustSucceed2(svc.ValidateMaybeRefresh(tk))
				Expect(id).To(Equal(issuer))
				Expect(newToken).ToNot(BeEmpty())
			},
		)

		It(
			"Should not refresh the token if the user does not submit a validation request within the refresh threshold",
			func() {
				id, newToken := MustSucceed2(svc.ValidateMaybeRefresh(tk))
				Expect(id).To(Equal(issuer))
				Expect(newToken).To(BeEmpty())
				now = now.Add(time.Second * 11)
				Expect(svc.ValidateMaybeRefresh(tk)).Error().
					To(MatchError(auth.ErrExpiredToken))
			},
		)
	})
})
