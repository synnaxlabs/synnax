// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tailscale

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

type stubDaemon struct{ serverName string }

func (d *stubDaemon) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	d.serverName = hi.ServerName
	return &tls.Certificate{}, nil
}

var _ = Describe("source", func() {
	It("Should fall back to the configured host when the client omits SNI", func() {
		d := &stubDaemon{}
		s := &source{client: d, host: "node.tailnet.ts.net"}
		MustSucceed(s.GetCertificate(&tls.ClientHelloInfo{}))
		Expect(d.serverName).To(Equal("node.tailnet.ts.net"))
	})

	It("Should preserve the client's SNI when present", func() {
		d := &stubDaemon{}
		s := &source{client: d, host: "node.tailnet.ts.net"}
		MustSucceed(s.GetCertificate(&tls.ClientHelloInfo{ServerName: "explicit.example.com"}))
		Expect(d.serverName).To(Equal("explicit.example.com"))
	})
})
