// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NewPair", func() {
	It("Should return a certificate for plant and its private key", func() {
		certPEM, keyPEM := certtestutil.NewPair()
		pair := MustSucceed(tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)))
		cert := MustSucceed(x509.ParseCertificate(pair.Certificate[0]))
		Expect(cert.Subject.CommonName).To(Equal("plant"))
		inAYear := time.Now().AddDate(1, 0, 0)
		Expect(cert.NotAfter).To(BeTemporally("~", inAYear, time.Hour))
		block, _ := pem.Decode([]byte(keyPEM))
		Expect(block.Type).To(Equal("PRIVATE KEY"))
	})

	It("Should return a new pair on each call", func() {
		first, _ := certtestutil.NewPair()
		second, _ := certtestutil.NewPair()
		Expect(first).ToNot(Equal(second))
	})
})
