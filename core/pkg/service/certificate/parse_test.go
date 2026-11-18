// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ParseDetails", func() {
	var leaf, ca string
	BeforeEach(func() {
		leaf = newCertificate(newKey(), "Leaf", time.Hour)
		ca = newCertificate(newKey(), "CA", time.Hour)
	})

	It("Should give the details of the leaf of a PEM chain", func() {
		details := MustSucceed(certificate.ParseDetails([]byte(leaf + ca)))
		Expect(details.Subject).To(Equal("CN=Leaf"))
		Expect(details.NotAfter).To(Equal(MustSucceed(
			certificate.ParseDetails([]byte(leaf)),
		).NotAfter))
	})

	It("Should give the details of a DER certificate", func() {
		block, _ := pem.Decode([]byte(leaf))
		Expect(MustSucceed(certificate.ParseDetails(block.Bytes))).
			To(Equal(MustSucceed(certificate.ParseDetails([]byte(leaf)))))
	})

	It("Should skip a block of another type after the certificate", func() {
		data := append([]byte(leaf), encodeKey(newKey())...)
		Expect(MustSucceed(certificate.ParseDetails(data)).Subject).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate after a block of another type", func() {
		data := append(encodeKey(newKey()), leaf...)
		Expect(MustSucceed(certificate.ParseDetails(data)).Subject).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate after text", func() {
		data := "Bag Attributes\n    localKeyID: 01\nsubject=CN=Leaf\n" + leaf
		Expect(
			MustSucceed(certificate.ParseDetails([]byte(data))).Subject,
		).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate after a byte order mark", func() {
		data := "\uFEFF" + leaf
		Expect(
			MustSucceed(certificate.ParseDetails([]byte(data))).Subject,
		).To(Equal("CN=Leaf"))
	})

	// derWithPEM returns a DER certificate for CN=Outer that holds the PEM of leaf in
	// an
	// extension.
	derWithPEM := func() []byte {
		GinkgoHelper()
		key := newKey()
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "Outer"},
			NotBefore:    start,
			NotAfter:     start.Add(time.Hour),
			ExtraExtensions: []pkix.Extension{{
				Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1},
				Value: []byte("\n" + leaf),
			}},
		}
		return MustSucceed(
			x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
		)
	}

	It("Should read a DER certificate that holds PEM text as DER", func() {
		Expect(
			MustSucceed(certificate.ParseDetails(derWithPEM())).Subject,
		).To(Equal("CN=Outer"))
	})

	It("Should fail for a cut DER certificate that holds PEM text", func() {
		der := derWithPEM()
		Expect(certificate.ParseDetails(der[:len(der)-16])).Error().
			To(MatchError("x509: malformed certificate"))
	})

	It("Should read PEM whose first line starts with 0", func() {
		data := "0 is the first byte of this file\n" + leaf
		Expect(
			MustSucceed(certificate.ParseDetails([]byte(data))).Subject,
		).To(Equal("CN=Leaf"))
	})

	It("Should fail for a certificate block that does not parse", func() {
		data := append(
			[]byte(leaf),
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("x")})...,
		)
		Expect(certificate.ParseDetails(data)).Error().
			To(MatchError("x509: malformed certificate"))
	})

	It("Should fail for data that holds no certificate", func() {
		Expect(certificate.ParseDetails([]byte("no certificate"))).Error().
			To(MatchError("x509: malformed certificate"))
	})
})
