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
	stdnet "net"
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
		ca := MustSucceed(cert.NewFactory(cert.FactoryConfig{
			LoaderConfig: cert.LoaderConfig{FS: fs},
			KeySize:      mock.SmallKeySize,
		}))
		srcA := MustSucceed(auto.NewSource(ca, "hostA:1"))
		srcB := MustSucceed(auto.NewSource(ca, "hostB:1"))
		s := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{
				{Address: "localhost:0", TLS: prov.TLSConfigFor(srcA)},
				{Address: "localhost:0", TLS: prov.TLSConfigFor(srcB)},
			},
			Security: server.SecurityConfig{Insecure: new(false)},
			Branches: []server.Branch{
				&server.SecureHTTPBranch{MaxIdleWorkerDuration: 100 * time.Millisecond},
			},
		}))
		Expect(presentedSANs(s.Addresses()[0])).To(ContainElement("hostA"))
		Expect(presentedSANs(s.Addresses()[1])).To(ContainElement("hostB"))
		Expect(s.Close()).To(Succeed())
	})

	It("Should close earlier listeners when a later listener fails to bind", func() {
		// The server binds every interface, so the port must be occupied the same way
		// for the second listener to collide with it.
		occupied := MustSucceed(stdnet.Listen("tcp", ":0"))
		defer func() { Expect(occupied.Close()).To(Succeed()) }()
		occupiedAddr := address.Newf(
			"localhost:%d", occupied.Addr().(*stdnet.TCPAddr).Port,
		)
		Expect(server.Serve(server.Config{
			Debug:    new(false),
			Security: server.SecurityConfig{Insecure: new(true)},
			Listeners: []server.Listener{
				{Address: "localhost:0"},
				{Address: occupiedAddr},
			},
		})).Error().To(MatchError(ContainSubstring("bind")))
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
