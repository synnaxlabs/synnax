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
	"github.com/synnaxlabs/synnax/pkg/security/cert/tailscale"
	. "github.com/synnaxlabs/x/testutil"
)

type stubClient struct{ serverName string }

func (c *stubClient) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.serverName = hi.ServerName
	return &tls.Certificate{}, nil
}

var _ = Describe("Tailscale", func() {
	It("Should build a source with a host", func() {
		Expect(MustSucceed(tailscale.NewSource(&stubClient{}, "node01.tailnet.ts.net"))).ToNot(BeNil())
	})

	It("Should reject a listener with no host", func() {
		Expect(tailscale.NewSource(&stubClient{}, "")).
			Error().To(MatchError(ContainSubstring("requires a listener host")))
	})

	It("Should fall back to the configured host when the client omits SNI", func() {
		c := &stubClient{}
		src := MustSucceed(tailscale.NewSource(c, "node.tailnet.ts.net"))
		MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{}))
		Expect(c.serverName).To(Equal("node.tailnet.ts.net"))
	})

	It("Should preserve the client's SNI when present", func() {
		c := &stubClient{}
		src := MustSucceed(tailscale.NewSource(c, "node.tailnet.ts.net"))
		MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{ServerName: "explicit.example.com"}))
		Expect(c.serverName).To(Equal("explicit.example.com"))
	})
})
