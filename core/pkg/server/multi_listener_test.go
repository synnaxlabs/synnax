// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"crypto/tls"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/auto"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/net"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MultiListener", func() {
	It("Should present each listener's own certificate", func() {
		fs := xfs.NewMem()
		mock.GenerateCerts(fs)
		prov := MustSucceed(security.NewProvider(security.ProviderConfig{
			LoaderConfig: cert.LoaderConfig{FS: fs},
			KeySize:      mock.SmallKeySize,
			Insecure:     new(false),
		}))
		portA := MustSucceed(net.FindOpenPort())
		portB := MustSucceed(net.FindOpenPort())
		addrA := address.Newf("localhost:%d", portA)
		addrB := address.Newf("localhost:%d", portB)
		srcA := MustSucceed(auto.Factory{}.NewSource(cert.SourceConfig{
			FS: fs, Address: "hostA:1", KeySize: mock.SmallKeySize,
		}))
		srcB := MustSucceed(auto.Factory{}.NewSource(cert.SourceConfig{
			FS: fs, Address: "hostB:1", KeySize: mock.SmallKeySize,
		}))
		s := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{
				{Address: addrA, TLS: prov.TLSConfigFor(srcA)},
				{Address: addrB, TLS: prov.TLSConfigFor(srcB)},
			},
			Security: server.SecurityConfig{Insecure: new(false)},
			Branches: []server.Branch{
				&server.SecureHTTPBranch{MaxIdleWorkerDuration: 100 * time.Millisecond},
			},
		}))
		Expect(presentedSANs(addrA)).To(ContainElement("hostA"))
		Expect(presentedSANs(addrB)).To(ContainElement("hostB"))
		Expect(s.Close()).To(Succeed())
	})
})

func presentedSANs(addr address.Address) []string {
	conn := MustSucceed(tls.Dial(
		"tcp", addr.String(), &tls.Config{InsecureSkipVerify: true},
	))
	defer func() { Expect(conn.Close()).To(Succeed()) }()
	Expect(conn.Handshake()).To(Succeed())
	return conn.ConnectionState().PeerCertificates[0].DNSNames
}
