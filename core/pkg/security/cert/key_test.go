// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/mldsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	fs2 "io/fs"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("KeyAlgorithm", func() {
	Describe("GenerateKey", func() {
		DescribeTable("should generate a key of the algorithm's type",
			func(algo cert.KeyAlgorithm, expected any) {
				Expect(MustSucceed(algo.GenerateKey(mock.SmallKeySize))).
					To(BeAssignableToTypeOf(expected))
			},
			Entry("RSA", cert.KeyAlgorithmRSA, &rsa.PrivateKey{}),
			Entry("ML-DSA-44", cert.KeyAlgorithmMLDSA44, &mldsa.PrivateKey{}),
			Entry("ML-DSA-65", cert.KeyAlgorithmMLDSA65, &mldsa.PrivateKey{}),
			Entry("ML-DSA-87", cert.KeyAlgorithmMLDSA87, &mldsa.PrivateKey{}),
		)

		DescribeTable("should generate the ML-DSA parameter set the name selects",
			func(algo cert.KeyAlgorithm, expected mldsa.Parameters) {
				key := MustSucceed(algo.GenerateKey(mock.SmallKeySize))
				Expect(key.(*mldsa.PrivateKey).PublicKey().Parameters()).
					To(Equal(expected))
			},
			Entry("ML-DSA-44", cert.KeyAlgorithmMLDSA44, mldsa.MLDSA44()),
			Entry("ML-DSA-65", cert.KeyAlgorithmMLDSA65, mldsa.MLDSA65()),
			Entry("ML-DSA-87", cert.KeyAlgorithmMLDSA87, mldsa.MLDSA87()),
		)

		It("should return ErrUnsupportedKeyAlgorithm for an unknown algorithm", func() {
			Expect(cert.KeyAlgorithm("dilithium").GenerateKey(2048)).Error().
				To(SatisfyAll(
					MatchError(cert.ErrUnsupportedKeyAlgorithm),
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("dilithium")),
				))
		})
	})

	Describe("PostQuantum", func() {
		DescribeTable("should identify post-quantum algorithms",
			func(algo cert.KeyAlgorithm, expected bool) {
				Expect(algo.PostQuantum()).To(Equal(expected))
			},
			Entry("RSA", cert.KeyAlgorithmRSA, false),
			Entry("unknown", cert.KeyAlgorithm("dilithium"), false),
			Entry("ML-DSA-44", cert.KeyAlgorithmMLDSA44, true),
			Entry("ML-DSA-65", cert.KeyAlgorithmMLDSA65, true),
			Entry("ML-DSA-87", cert.KeyAlgorithmMLDSA87, true),
		)
	})
})

var _ = Describe("SignsJWT", func() {
	DescribeTable("should report whether the key has a JWT signing method",
		func(generate func() any, expected bool) {
			Expect(cert.SignsJWT(generate())).To(Equal(expected))
		},
		Entry("RSA", func() any {
			return MustSucceed(rsa.GenerateKey(rand.Reader, mock.SmallKeySize))
		}, true),
		Entry("ECDSA", func() any {
			return MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
		}, true),
		Entry("Ed25519", func() any {
			_, priv := MustSucceed2(ed25519.GenerateKey(rand.Reader))
			return priv
		}, true),
		Entry("ML-DSA-65", func() any {
			return MustSucceed(mldsa.GenerateKey(mldsa.MLDSA65()))
		}, false),
	)
})

var _ = Describe("Factory key algorithms", func() {
	DescribeTable("should sign a node certificate the CA can verify",
		func(algo cert.KeyAlgorithm, expectedSigAlgo x509.SignatureAlgorithm) {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:           xfs.NewMem(),
				Hosts:        []address.Address{"synnaxlabs.com"},
				KeySize:      mock.SmallKeySize,
				KeyAlgorithm: algo,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).To(Succeed())
			ca, _ := MustSucceed2(f.Loader.LoadCAPair())
			node, _ := MustSucceed2(f.Loader.LoadNodePair())
			Expect(node.SignatureAlgorithm).To(Equal(expectedSigAlgo))
			Expect(node.CheckSignatureFrom(ca)).To(Succeed())
		},
		Entry("RSA", cert.KeyAlgorithmRSA, x509.SHA256WithRSA),
		Entry("ML-DSA-44", cert.KeyAlgorithmMLDSA44, x509.MLDSA44),
		Entry("ML-DSA-65", cert.KeyAlgorithmMLDSA65, x509.MLDSA65),
		Entry("ML-DSA-87", cert.KeyAlgorithmMLDSA87, x509.MLDSA87),
	)

	It("should reject an unknown algorithm at construction", func() {
		Expect(cert.NewFactory(cert.FactoryConfig{
			FS:           xfs.NewMem(),
			KeyAlgorithm: cert.KeyAlgorithm("dilithium"),
		})).Error().To(MatchError(cert.ErrUnsupportedKeyAlgorithm))
	})
})

var _ = Describe("Token key", func() {
	var (
		fs xfs.FS
		f  *cert.Factory
	)
	BeforeEach(func() {
		fs = xfs.NewMem()
		f = MustSucceed(cert.NewFactory(cert.FactoryConfig{
			FS:      fs,
			Hosts:   []address.Address{"synnaxlabs.com"},
			KeySize: mock.SmallKeySize,
		}))
	})

	It("should create an Ed25519 key that round trips through the loader", func() {
		Expect(f.CreateTokenKeyIfMissing()).To(Succeed())
		Expect(MustSucceed(f.Loader.LoadTokenKey())).
			To(BeAssignableToTypeOf(ed25519.PrivateKey{}))
	})

	It("should leave an existing key untouched", func() {
		Expect(f.CreateTokenKeyIfMissing()).To(Succeed())
		first := MustSucceed(f.Loader.LoadTokenKey())
		Expect(f.CreateTokenKeyIfMissing()).To(Succeed())
		Expect(MustSucceed(f.Loader.LoadTokenKey())).To(Equal(first))
	})

	It("should report a missing key", func() {
		Expect(f.Loader.LoadTokenKey()).Error().To(SatisfyAll(
			MatchError(fs2.ErrNotExist),
			MatchError(ContainSubstring("token signing key not found")),
		))
	})

	It("should resolve the key's absolute path", func() {
		Expect(f.AbsoluteTokenKeyPath()).To(HaveSuffix("/token.key"))
	})
})

var _ = Describe("ML-DSA handshake", func() {
	It("should negotiate an ML-DSA signature scheme in TLS 1.3", func(ctx SpecContext) {
		fs := xfs.NewMem()
		f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
			FS:           fs,
			Hosts:        []address.Address{"synnaxlabs.com"},
			KeyAlgorithm: cert.KeyAlgorithmMLDSA65,
		}))
		Expect(f.CreateCAPair()).To(Succeed())
		Expect(f.CreateNodePair()).To(Succeed())
		nodeTLS := MustSucceed(f.Loader.LoadNodeTLS())
		ca, _ := MustSucceed2(f.Loader.LoadCAPair())
		roots := x509.NewCertPool()
		roots.AddCert(ca)

		clientRaw, serverRaw := net.Pipe()
		server := tls.Server(serverRaw, &tls.Config{
			Certificates: []tls.Certificate{*nodeTLS},
			MinVersion:   tls.VersionTLS13,
		})
		client := tls.Client(clientRaw, &tls.Config{
			RootCAs:    roots,
			ServerName: "synnaxlabs.com",
			MinVersion: tls.VersionTLS13,
		})
		// Close the pipe ends, not the tls.Conns: a tls.Conn Close writes a
		// closeNotify alert, and an unbuffered net.Pipe blocks with no reader left.
		DeferClose(serverRaw)
		DeferClose(clientRaw)

		served := make(chan error, 1)
		go func() {
			defer GinkgoRecover()
			served <- server.HandshakeContext(ctx)
		}()
		Expect(client.HandshakeContext(ctx)).To(Succeed())
		Eventually(served).Should(Receive(BeNil()))

		state := client.ConnectionState()
		Expect(state.Version).To(Equal(uint16(tls.VersionTLS13)))
		Expect(state.PeerCertificates[0].SignatureAlgorithm).To(Equal(x509.MLDSA65))
	})
})
