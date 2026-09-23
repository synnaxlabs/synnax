// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pem_test

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/mldsa"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xpem "github.com/synnaxlabs/x/encoding/pem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// rsaKeySize keeps generation fast; these specs exercise encoding, not key strength.
const rsaKeySize = 1024

var _ = Describe("PEM", func() {
	Describe("Private keys", func() {
		DescribeTable("should round trip through PEM",
			func(generate func() crypto.Signer, expectedBlockType string) {
				key := generate()
				block := MustSucceed(xpem.FromPrivateKey(key))
				Expect(block.Type).To(Equal(expectedBlockType))
				decoded := MustSucceed(xpem.ToPrivateKey(block))
				Expect(decoded.(crypto.Signer).Public()).To(Equal(key.Public()))
			},
			Entry("RSA", func() crypto.Signer {
				return MustSucceed(rsa.GenerateKey(rand.Reader, rsaKeySize))
			}, "RSA PRIVATE KEY"),
			Entry("ECDSA", func() crypto.Signer {
				return MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
			}, "EC PRIVATE KEY"),
			Entry("Ed25519", func() crypto.Signer {
				_, priv := MustSucceed2(ed25519.GenerateKey(rand.Reader))
				return priv
			}, "PRIVATE KEY"),
			Entry("ML-DSA-44", func() crypto.Signer {
				return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA44()))
			}, "PRIVATE KEY"),
			Entry("ML-DSA-65", func() crypto.Signer {
				return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA65()))
			}, "PRIVATE KEY"),
			Entry("ML-DSA-87", func() crypto.Signer {
				return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA87()))
			}, "PRIVATE KEY"),
		)

		It(
			"should return a validation error when the algorithm has no encoding",
			func() {
				Expect(xpem.FromPrivateKey("not-a-key")).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("unsupported key type")),
				))
			},
		)

		It("should return a validation error when the block type is unknown", func() {
			block := &pem.Block{Type: "NONSENSE", Bytes: []byte("x")}
			Expect(xpem.ToPrivateKey(block)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("NONSENSE")),
			))
		})
	})

	Describe("Write and Read", func() {
		It("should read back the first of many written blocks", func() {
			key := MustSucceed(mldsa.GenerateKey(mldsa.MLDSA65()))
			block := MustSucceed(xpem.FromPrivateKey(key))
			cert := xpem.FromCertBytes([]byte("a-certificate"))
			buf := &bytes.Buffer{}
			Expect(xpem.Write(buf, block, cert)).To(Succeed())
			Expect(MustSucceed(xpem.Read(bytes.NewReader(buf.Bytes()))).Type).
				To(Equal("PRIVATE KEY"))
			Expect(
				MustSucceed(xpem.ReadMany(bytes.NewReader(buf.Bytes()))),
			).To(HaveLen(2))
		})
	})
})
