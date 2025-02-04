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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"time"
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
	)
	BeforeEach(func() {
		k, err := rsa.GenerateKey(rand.Reader, 1024)
		Expect(err).ToNot(HaveOccurred())
		svc = &token.Service{
			KeyProvider: &mockKeyService{key: k},
			Expiration:  5 * time.Second,
		}
	})
	It("Should generate a token for the given issuer", func() {
		issuer := uuid.New()
		token, err := svc.New(issuer)
		Expect(err).ToNot(HaveOccurred())
		Expect(token).ToNot(BeEmpty())
	})
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
