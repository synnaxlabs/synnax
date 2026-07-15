// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tailscale_test

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/tailscale"
	. "github.com/synnaxlabs/x/testutil"
)

type stubClient struct{ serverName string }

func (c *stubClient) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.serverName = hi.ServerName
	return &tls.Certificate{}, nil
}

var _ = Describe("Tailscale", func() {
	It("Should build a source when no cert or key is set", func() {
		Expect(MustSucceed(tailscale.Factory{}.NewSource(cert.SourceConfig{
			Address: "node01.tailnet.ts.net:9090",
		}))).ToNot(BeNil())
	})

	It("Should reject a cert or key", func() {
		Expect(tailscale.Factory{}.NewSource(cert.SourceConfig{Cert: "node.crt"})).
			Error().To(MatchError(ContainSubstring("must not set cert or key")))
	})

	It("Should reject a listener with no host", func() {
		Expect(tailscale.Factory{}.NewSource(cert.SourceConfig{Address: ":9090"})).
			Error().To(MatchError(ContainSubstring("requires a listener host")))
	})

	It("Should fall back to the configured host when the client omits SNI", func() {
		c := &stubClient{}
		f := tailscale.Factory{NewClient: func() cert.Source { return c }}
		src := MustSucceed(f.NewSource(cert.SourceConfig{Address: "node.tailnet.ts.net:9090"}))
		MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{}))
		Expect(c.serverName).To(Equal("node.tailnet.ts.net"))
	})

	It("Should preserve the client's SNI when present", func() {
		c := &stubClient{}
		f := tailscale.Factory{NewClient: func() cert.Source { return c }}
		src := MustSucceed(f.NewSource(cert.SourceConfig{Address: "node.tailnet.ts.net:9090"}))
		MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{ServerName: "explicit.example.com"}))
		Expect(c.serverName).To(Equal("explicit.example.com"))
	})
})
