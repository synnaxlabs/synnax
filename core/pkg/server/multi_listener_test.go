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
	"crypto/x509"
	"io"
	stdnet "net"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/auto"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
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
		ca := MustSucceed(cert.NewFactory(cert.FactoryConfig{
			LoaderConfig: cert.LoaderConfig{FS: fs},
			KeySize:      mock.SmallKeySize,
		}))
		srcA := MustSucceed(auto.NewSource(ca, "hostA:1"))
		srcB := MustSucceed(auto.NewSource(ca, "hostB:1"))
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

	It("Should serve certificates a client verifies with the trust anchor", func() {
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
		l := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: fs}))
		autoAddr := address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
		fileAddr := address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
		autoSrc := MustSucceed(auto.NewSource(ca, autoAddr))
		fileSrc := MustSucceed(file.NewSource(
			fs,
			l.AbsoluteNodeCertPath(),
			l.AbsoluteNodeKeyPath(),
		))
		s := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{
				{Address: autoAddr, TLS: prov.TLSConfigFor(autoSrc)},
				{Address: fileAddr, TLS: prov.TLSConfigFor(fileSrc)},
			},
			Security: server.SecurityConfig{Insecure: new(false)},
			Branches: []server.Branch{
				&server.SecureHTTPBranch{MaxIdleWorkerDuration: 100 * time.Millisecond},
			},
		}))
		anchors := certPool(MustSucceed(l.TrustAnchorsPEM()))
		Expect(handshake(autoAddr, anchors)).To(Succeed())
		Expect(handshake(fileAddr, anchors)).To(Succeed())
		// The node certificate anchors only itself, so on its own it cannot verify the
		// separate certificate the auto source signs for its listener.
		nodeOnly := certPool(MustSucceed(readFile(fs, l.AbsoluteNodeCertPath())))
		Expect(handshake(autoAddr, nodeOnly)).
			To(MatchError(ContainSubstring("certificate signed by unknown authority")))
		Expect(s.Close()).To(Succeed())
	})

	It("Should serve a node certificate an unrelated CA did not sign", func() {
		fs := xfs.NewMem()
		mock.GenerateCerts(fs)
		l := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: fs}))
		// Stand in for a Core whose node certificate came from outside, leaving a CA on
		// disk that signed nothing the listeners serve.
		foreign := xfs.NewMem()
		mock.GenerateCerts(foreign)
		foreignL := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: foreign}))
		writeFile(fs, l.AbsoluteCACertPath(), MustSucceed(readFile(
			foreign,
			foreignL.AbsoluteCACertPath(),
		)))
		prov := MustSucceed(security.NewProvider(security.ProviderConfig{
			LoaderConfig: cert.LoaderConfig{FS: fs},
			KeySize:      mock.SmallKeySize,
			Insecure:     new(false),
		}))
		addr := address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
		src := MustSucceed(file.NewSource(
			fs,
			l.AbsoluteNodeCertPath(),
			l.AbsoluteNodeKeyPath(),
		))
		s := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{{Address: addr, TLS: prov.TLSConfigFor(src)}},
			Security:  server.SecurityConfig{Insecure: new(false)},
			Branches: []server.Branch{
				&server.SecureHTTPBranch{MaxIdleWorkerDuration: 100 * time.Millisecond},
			},
		}))
		anchors := certPool(MustSucceed(l.TrustAnchorsPEM()))
		Expect(handshake(addr, anchors)).To(Succeed())
		Expect(s.Close()).To(Succeed())
	})

	It("Should close earlier listeners when a later listener fails to bind", func() {
		portA := MustSucceed(net.FindOpenPort())
		portB := MustSucceed(net.FindOpenPort())
		addrA := address.Newf("localhost:%d", portA)
		addrB := address.Newf("localhost:%d", portB)
		occupied := MustSucceed(stdnet.Listen("tcp", addrB.PortString()))
		defer func() { Expect(occupied.Close()).To(Succeed()) }()
		Expect(server.Serve(server.Config{
			Debug:     new(false),
			Security:  server.SecurityConfig{Insecure: new(true)},
			Listeners: []server.Listener{{Address: addrA}, {Address: addrB}},
		})).Error().To(HaveOccurred())
		Eventually(func() error {
			conn, err := stdnet.DialTimeout("tcp", addrA.String(), 100*time.Millisecond)
			if err == nil {
				Expect(conn.Close()).To(Succeed())
			}
			return err
		}).Should(HaveOccurred())
	})
})

func readFile(fs xfs.FS, path string) ([]byte, error) {
	f, err := fs.Open(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return io.ReadAll(f)
}

func writeFile(fs xfs.FS, path string, b []byte) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	Expect(f.Write(b)).To(Equal(len(b)))
}

func certPool(pemBytes []byte) *x509.CertPool {
	GinkgoHelper()
	pool := x509.NewCertPool()
	Expect(pool.AppendCertsFromPEM(pemBytes)).To(BeTrue())
	return pool
}

func handshake(addr address.Address, pool *x509.CertPool) error {
	conn, err := tls.Dial("tcp", addr.String(), &tls.Config{
		RootCAs:    pool,
		ServerName: addr.Host(),
	})
	if err != nil {
		return err
	}
	return conn.Close()
}

func presentedSANs(addr address.Address) []string {
	conn := MustSucceed(tls.Dial(
		"tcp", addr.String(), &tls.Config{InsecureSkipVerify: true},
	))
	defer func() { Expect(conn.Close()).To(Succeed()) }()
	Expect(conn.Handshake()).To(Succeed())
	return conn.ConnectionState().PeerCertificates[0].DNSNames
}
