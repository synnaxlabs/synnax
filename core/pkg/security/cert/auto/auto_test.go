// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auto_test

import (
	"crypto/tls"
	"crypto/x509"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/auto"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Auto", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })

	It("Should self-sign a certificate with SANs from the listener address", func() {
		src := MustSucceed(auto.Factory{}.NewSource(cert.SourceConfig{
			FS:      fs,
			Address: "console.example.com:9091",
			KeySize: mock.SmallKeySize,
		}))
		c := MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{}))
		leaf := MustSucceed(x509.ParseCertificate(c.Certificate[0]))
		Expect(leaf.DNSNames).To(ContainElement("console.example.com"))
	})

	It("Should require a listener address", func() {
		Expect(auto.Factory{}.NewSource(cert.SourceConfig{
			FS:      fs,
			KeySize: mock.SmallKeySize,
		})).Error().To(MatchError(ContainSubstring("requires a listener address")))
	})

	It("Should reject a cert or key", func() {
		Expect(auto.Factory{}.NewSource(cert.SourceConfig{
			FS:      fs,
			Address: "a:9090",
			Cert:    "node.crt",
		})).Error().To(MatchError(ContainSubstring("must not set cert or key")))
	})
})
