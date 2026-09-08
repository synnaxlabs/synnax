// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	v1 "github.com/synnaxlabs/freighter/grpc/v1"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// selfSignedCert returns a certificate valid only for dnsName, along with a pool that
// trusts it.
func selfSignedCert(dnsName string) (tls.Certificate, *x509.CertPool) {
	GinkgoHelper()
	key := MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: dnsName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{dnsName},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der := MustSucceed(
		x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
	)
	leaf := MustSucceed(x509.ParseCertificate(der))
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
		Leaf:        leaf,
	}, roots
}

var _ = Describe("Pool", func() {
	Describe("Acquire", func() {
		It("Should dial the address through the passthrough resolver", func() {
			pool := DeferClose(fgrpc.OpenPool(
				"",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
			conn := MustSucceed(pool.Acquire("localhost:12345"))
			Expect(conn.CanonicalTarget()).To(Equal("passthrough:///localhost:12345"))
		})

		It("Should lead the prefixed address with the scheme", func() {
			pool := DeferClose(fgrpc.OpenPool(
				"scoped",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			))
			conn := MustSucceed(pool.Acquire("localhost:12345"))
			Expect(conn.CanonicalTarget()).To(
				Equal("passthrough:///scoped/localhost:12345"),
			)
		})
	})

	Describe("TLS", func() {
		// The Core dials peers with a tls.Config that leaves ServerName empty, so gRPC
		// derives the name from the target. A target the dialer cannot reduce to the
		// bare host breaks certificate verification for every secure cluster.
		It("Should verify a peer certificate against the bare hostname", func(
			ctx SpecContext,
		) {
			serverCert, roots := selfSignedCert("localhost")
			lis := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
			srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{serverCert},
				MinVersion:   tls.VersionTLS13,
			})))
			pool := DeferClose(fgrpc.OpenPool("", grpc.WithTransportCredentials(
				credentials.NewTLS(&tls.Config{
					RootCAs:    roots,
					MinVersion: tls.VersionTLS13,
				}),
			)))
			go func() {
				defer GinkgoRecover()
				if err := srv.Serve(lis); !errors.Is(err, grpc.ErrServerStopped) {
					Expect(err).To(Succeed())
				}
			}()
			DeferCleanup(srv.GracefulStop)
			port := lis.Addr().(*net.TCPAddr).Port
			conn := MustSucceed(pool.Acquire(address.Newf("localhost:%d", port)))
			// An Unimplemented status can only come from the server, so reaching it
			// means the handshake verified the certificate.
			Expect(conn.Invoke(
				ctx,
				"/nonexistent.Service/Method",
				new(v1.Request),
				new(v1.Request),
			)).Error().To(MatchError(ContainSubstring("unknown service")))
		})
	})
})
