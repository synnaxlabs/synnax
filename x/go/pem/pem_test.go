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
	stdpem "encoding/pem"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/pem"
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
				block := MustSucceed(pem.FromPrivateKey(key))
				Expect(block.Type).To(Equal(expectedBlockType))
				decoded := MustSucceed(pem.ToPrivateKey(block))
				Expect(decoded.(crypto.Signer).Public()).To(Equal(key.Public()))
			},
			Entry("RSA", func() crypto.Signer {
				return MustSucceed(rsa.GenerateKey(rand.Reader, rsaKeySize))
			}, pem.BlockTypeRSAPrivateKey),
			Entry("ECDSA", func() crypto.Signer {
				return MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
			}, pem.BlockTypeECPrivateKey),
			Entry("Ed25519", func() crypto.Signer {
				_, priv := MustSucceed2(ed25519.GenerateKey(rand.Reader))
				return priv
			}, pem.BlockTypePKCS8PrivateKey),
			Entry("ML-DSA-44", func() crypto.Signer {
				return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA44()))
			}, pem.BlockTypePKCS8PrivateKey),
			Entry("ML-DSA-65", func() crypto.Signer {
				return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA65()))
			}, pem.BlockTypePKCS8PrivateKey),
			Entry("ML-DSA-87", func() crypto.Signer {
				return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA87()))
			}, pem.BlockTypePKCS8PrivateKey),
		)

		It(
			"should return ErrUnsupportedKey when the algorithm has no encoding",
			func() {
				Expect(pem.FromPrivateKey("not-a-key")).Error().To(SatisfyAll(
					MatchError(pem.ErrUnsupportedKey),
					MatchError(validate.ErrValidation),
				))
			},
		)

		It("should return ErrUnsupportedKey when the block type is unknown", func() {
			block := &stdpem.Block{Type: "NONSENSE", Bytes: []byte("x")}
			Expect(pem.ToPrivateKey(block)).Error().To(SatisfyAll(
				MatchError(pem.ErrUnsupportedKey),
				MatchError(ContainSubstring("NONSENSE")),
			))
		})
	})

	Describe("Write and Read", func() {
		It("should read back the first of many written blocks", func() {
			key := MustSucceed(mldsa.GenerateKey(mldsa.MLDSA65()))
			block := MustSucceed(pem.FromPrivateKey(key))
			cert := pem.FromCertBytes([]byte("a-certificate"))
			buf := &bytes.Buffer{}
			Expect(pem.Write(buf, block, cert)).To(Succeed())
			Expect(MustSucceed(pem.Read(bytes.NewReader(buf.Bytes()))).Type).
				To(Equal(pem.BlockTypePKCS8PrivateKey))
			Expect(
				MustSucceed(pem.ReadMany(bytes.NewReader(buf.Bytes()))),
			).To(HaveLen(2))
		})
	})
})
