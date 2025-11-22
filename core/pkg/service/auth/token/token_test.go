// Copyright 2025 Synnax Labs, Inc.
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
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	. "github.com/synnaxlabs/x/testutil"
)

type mockKeyService struct {
	key *rsa.PrivateKey
}

func (m *mockKeyService) NodePrivate() crypto.PrivateKey {
	return m.key
}

var _ = Describe("token", func() {
	var (
		svc *token.Service
		cfg token.ServiceConfig
	)
	JustBeforeEach(func() {
		k, err := rsa.GenerateKey(rand.Reader, 1024)
		Expect(err).ToNot(HaveOccurred())
		cfg.KeyProvider = &mockKeyService{key: k}
		svc = MustSucceed(token.OpenService(cfg))
	})
	Describe("Nominal", func() {
		BeforeEach(func() { cfg.Now = time.Now })
		Describe("Token Generation", func() {
			It("Should generate a token for the given issuer", func() {
				issuer := uuid.New()
				token, err := svc.New(issuer)
				Expect(err).ToNot(HaveOccurred())
				Expect(token).ToNot(BeEmpty())
			})
		})

		Describe("Token Validation", func() {
			It("Should validate a token", func() {
				issuer := uuid.New()
				token, err := svc.New(issuer)
				Expect(err).ToNot(HaveOccurred())
				Expect(token).ToNot(BeEmpty())
				issuer2, err := svc.Validate(token)
				Expect(err).ToNot(HaveOccurred())
				Expect(issuer).To(Equal(issuer2))
			})
		})
	})

	Context("Token Expiration Inside Refresh Interval", func() {
		var now time.Time
		BeforeEach(func() {
			cfg.Expiration = time.Second * 10
			cfg.RefreshThreshold = time.Second * 8
			cfg.Now = func() time.Time { return now }
		})

		It("Should refresh the token if the user submits a validation request within the refresh threshold", func() {
			now = time.Now()
			issuer := uuid.New()
			tk := MustSucceed(svc.New(issuer))

			id, newToken, err := svc.ValidateMaybeRefresh(tk)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(issuer))
			Expect(newToken).To(BeEmpty())

			now = now.Add(time.Second * 6)

			id, newToken, err = svc.ValidateMaybeRefresh(tk)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(issuer))
			Expect(newToken).ToNot(BeEmpty())
		})
	})
	Describe("Token Expiration Outside Refresh Interval", func() {
		// Unfortunately there's not much we can do get rid of the long sleep here,
		// as JWT has a method internally for checking expiration that we can't override
		// with our custom Now() function.
		BeforeEach(func() {
			cfg.Expiration = time.Second * 1
			cfg.RefreshThreshold = time.Second * 1
			cfg.Now = time.Now
		})
		It("Should not refresh the token if the user does not submit a validation request within the refresh threshold", func() {
			issuer := uuid.New()
			tk := MustSucceed(svc.New(issuer))
			id, newToken, err := svc.ValidateMaybeRefresh(tk)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(issuer))
			Expect(newToken).ToNot(BeEmpty())
			time.Sleep(time.Second * 2)
			Expect(svc.ValidateMaybeRefresh(tk)).Error().
				To(HaveOccurredAs(auth.ExpiredToken))
		})
	})
})
