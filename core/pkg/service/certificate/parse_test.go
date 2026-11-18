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
	"encoding/pem"
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

	It("Should read PEM only when it starts with a certificate", func() {
		data := append(encodeKey(newKey()), leaf...)
		Expect(certificate.ParseDetails(data)).Error().
			To(MatchError("x509: malformed certificate"))
	})

	It("Should fail for a certificate block that does not decode", func() {
		data := []byte("-----BEGIN CERTIFICATE-----\nnot base64")
		Expect(certificate.ParseDetails(data)).Error().
			To(MatchError("holds no certificate"))
	})

	It("Should fail for data that holds no certificate", func() {
		Expect(certificate.ParseDetails([]byte("no certificate"))).Error().
			To(MatchError("x509: malformed certificate"))
	})
})
