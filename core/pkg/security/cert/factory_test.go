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
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/testutil"
	"github.com/synnaxlabs/x/address"
	xpem "github.com/synnaxlabs/x/encoding/pem"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

func writePEM(fs xfs.FS, p string, b *pem.Block) {
	GinkgoHelper()
	file := MustSucceed(fs.Open(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
	Expect(xpem.Write(file, b)).To(Succeed())
	Expect(file.Close()).To(Succeed())
}

var _ = Describe("Factory", func() {
	var fs xfs.FS
	BeforeEach(func() {
		fs = xfs.NewMem()
	})
	Describe("CA Generation", func() {
		It("Should generate a CA and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			c, k := MustSucceed2(f.Loader.LoadCAPair())
			Expect(c).ToNot(BeNil())
			Expect(k).ToNot(BeNil())
			cas := MustSucceed(f.Loader.LoadCAs())
			Expect(cas).To(HaveLen(1))
		})
		It("Should not allow key reuse by default", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(
				f.CreateCAPair(),
			).To(MatchError(ContainSubstring(
				"already exists, but reuse is not allowed",
			)))
		})
		It("Should reuse the CA and key if they already exist", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:            fs,
				AllowKeyReuse: new(true),
				KeySize:       testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateCAPair()).To(Succeed())
		})
	})
	Describe("Node Generation", func() {
		It("Should generate a node certificate and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				Hosts:   []address.Address{"synnaxlabs.com"},
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).To(Succeed())
			c, k := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c).ToNot(BeNil())
			Expect(k).ToNot(BeNil())
			tlsC := MustSucceed(f.Loader.LoadNodeTLS())
			Expect(tlsC).ToNot(BeNil())
		})
		It("Should fail to generate a node cert and key if no CA is present", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateNodePair()).
				Error().To(MatchError(ContainSubstring("CA certificate not found")))
		})
		It("Should fail is no hosts are provided", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).
				Error().To(MatchError(ContainSubstring("no hosts provided")))
		})
	})
	Describe("CreateNodePairIfStale", func() {
		newFactory := func(hosts ...address.Address) *cert.Factory {
			GinkgoHelper()
			return MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				Hosts:   hosts,
				KeySize: testutil.SmallKeySize,
			}))
		}
		serialOf := func(f *cert.Factory) string {
			GinkgoHelper()
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			return c.SerialNumber.String()
		}
		It("Should create the pair when it does not exist", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c.DNSNames).To(ConsistOf("synnaxlabs.com"))
		})
		It("Should fail on a node certificate it cannot parse", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			for _, p := range []string{
				f.Config().AbsoluteNodeCertPath(),
				f.Config().AbsoluteNodeKeyPath(),
			} {
				file := MustSucceed(fs.Open(p, os.O_CREATE|os.O_WRONLY))
				MustSucceed(file.Write([]byte("not a certificate")))
				Expect(file.Close()).To(Succeed())
			}
			Expect(f.CreateNodePairIfStale()).
				To(MatchError(ContainSubstring("PEM")))
		})

		It("Should leave a certificate covering every host untouched", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			before := serialOf(f)
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			Expect(serialOf(f)).To(Equal(before))
		})
		It("Should replace a certificate missing a configured host", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			before := serialOf(f)
			grown := newFactory("synnaxlabs.com", "docs.synnaxlabs.com")
			Expect(grown.CreateNodePairIfStale()).To(Succeed())
			Expect(serialOf(grown)).ToNot(Equal(before))
			c, _ := MustSucceed2(grown.Loader.LoadNodePair())
			Expect(c.DNSNames).To(ConsistOf("synnaxlabs.com", "docs.synnaxlabs.com"))
		})
		It(
			"Should replace a certificate covering none of the configured hosts",
			func() {
				f := newFactory("old.synnaxlabs.com")
				Expect(f.CreateCAPair()).To(Succeed())
				Expect(f.CreateNodePairIfStale()).To(Succeed())
				moved := newFactory("new.synnaxlabs.com")
				Expect(moved.CreateNodePairIfStale()).To(Succeed())
				c, _ := MustSucceed2(moved.Loader.LoadNodePair())
				Expect(c.DNSNames).To(ConsistOf("new.synnaxlabs.com"))
			},
		)
		It("Should keep the existing pair when signing fails", func() {
			f := newFactory("old.synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			before := serialOf(f)
			Expect(fs.Remove("/usr/local/synnax/certs/ca.key")).To(Succeed())
			moved := newFactory("new.synnaxlabs.com")
			Expect(
				moved.CreateNodePairIfStale(),
			).To(MatchError(ContainSubstring("CA certificate not found")))
			Expect(serialOf(f)).To(Equal(before))
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c.DNSNames).To(ConsistOf("old.synnaxlabs.com"))
		})
		It("Should replace the pair despite a leftover from a failed attempt", func() {
			f := newFactory("old.synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			Expect(fs.Remove("/usr/local/synnax/certs/ca.key")).To(Succeed())
			moved := newFactory("new.synnaxlabs.com")
			Expect(
				moved.CreateNodePairIfStale(),
			).To(MatchError(ContainSubstring("CA certificate not found")))
			Expect(fs.Remove("/usr/local/synnax/certs/ca.crt")).To(Succeed())
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(moved.CreateNodePairIfStale()).To(Succeed())
			c, _ := MustSucceed2(moved.Loader.LoadNodePair())
			Expect(c.DNSNames).To(ConsistOf("new.synnaxlabs.com"))
		})
		It("Should keep the certificate chaining to the unchanged CA", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			caBefore, _ := MustSucceed2(f.Loader.LoadCAPair())
			grown := newFactory("synnaxlabs.com", "docs.synnaxlabs.com")
			Expect(grown.CreateNodePairIfStale()).To(Succeed())
			caAfter, _ := MustSucceed2(grown.Loader.LoadCAPair())
			Expect(caAfter.SerialNumber).To(Equal(caBefore.SerialNumber))
			c, _ := MustSucceed2(grown.Loader.LoadNodePair())
			Expect(c.AuthorityKeyId).To(Equal(caAfter.SubjectKeyId))
		})
		It("Should replace a certificate whose subject matches the CA's", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			ca, caKey := MustSucceed2(f.Loader.LoadCAPair())
			nodeKey := MustSucceed(
				f.Config().KeyAlgorithm.GenerateKey(testutil.SmallKeySize),
			)
			b := MustSucceed(x509.CreateCertificate(rand.Reader, &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject:      pkix.Name{CommonName: "Synnax CA"},
				NotBefore:    time.Now().Add(-time.Hour),
				NotAfter:     time.Now().Add(time.Hour),
				KeyUsage:     x509.KeyUsageDigitalSignature,
				ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				DNSNames:     []string{"synnaxlabs.com"},
			}, ca, nodeKey.Public(), caKey))
			writePEM(fs, f.Config().AbsoluteNodeCertPath(), xpem.FromCertBytes(b))
			writePEM(
				fs,
				f.Config().AbsoluteNodeKeyPath(),
				MustSucceed(xpem.FromPrivateKey(nodeKey)),
			)
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c.Subject.String()).ToNot(Equal(ca.Subject.String()))
			Expect(c.CheckSignatureFrom(ca)).To(Succeed())
		})
		It(
			"Should fail when it cannot replace a certificate another CA signed",
			func() {
				f := newFactory("synnaxlabs.com")
				Expect(f.CreateCAPair()).To(Succeed())
				Expect(f.CreateNodePairIfStale()).To(Succeed())
				before := serialOf(f)
				Expect(fs.Remove(f.Config().AbsoluteCAKeyPath())).To(Succeed())
				Expect(fs.Remove("/usr/local/synnax/certs/ca.crt")).To(Succeed())
				Expect(f.CreateCAPair()).To(Succeed())
				Expect(fs.Remove(f.Config().AbsoluteCAKeyPath())).To(Succeed())
				Expect(f.CreateNodePairIfStale()).To(MatchError(And(
					ContainSubstring("does not chain to the CA"),
					ContainSubstring("Restore the CA key"),
				)))
				Expect(serialOf(f)).To(Equal(before))
			},
		)
		It("Should replace a certificate another CA signed", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			Expect(fs.Remove(f.Config().AbsoluteCAKeyPath())).To(Succeed())
			Expect(fs.Remove("/usr/local/synnax/certs/ca.crt")).To(Succeed())
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			ca, _ := MustSucceed2(f.Loader.LoadCAPair())
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c.CheckSignatureFrom(ca)).To(Succeed())
		})
		It("Should keep a chaining certificate when the CA key is absent", func() {
			f := newFactory("synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			before := serialOf(f)
			Expect(fs.Remove(f.Config().AbsoluteCAKeyPath())).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			Expect(serialOf(f)).To(Equal(before))
		})
	})

	Describe("Chain Verification", func() {
		// A leaf whose subject matches its issuer's carries no authority key
		// identifier, and every verifier outside Go reads it as self-signed and stops
		// at depth zero.
		expectChainsToCA := func(f *cert.Factory, leaf *x509.Certificate) {
			GinkgoHelper()
			ca, _ := MustSucceed2(f.Loader.LoadCAPair())
			Expect(leaf.Subject.String()).ToNot(Equal(ca.Subject.String()))
			Expect(leaf.AuthorityKeyId).To(Equal(ca.SubjectKeyId))
			Expect(leaf.AuthorityKeyId).ToNot(BeEmpty())
			roots := x509.NewCertPool()
			roots.AddCert(ca)
			Expect(leaf.Verify(x509.VerifyOptions{
				Roots:     roots,
				DNSName:   "synnaxlabs.com",
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})).Error().ToNot(HaveOccurred())
		}
		It("Should chain a node certificate on disk to the CA", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				Hosts:   []address.Address{"synnaxlabs.com"},
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).To(Succeed())
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			expectChainsToCA(f, c)
		})
		It("Should chain an in-memory node certificate to the CA", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				FS:      fs,
				KeySize: testutil.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			tlsC := MustSucceed(f.SignNodeCert([]address.Address{"synnaxlabs.com"}))
			expectChainsToCA(f, MustSucceed(x509.ParseCertificate(tlsC.Certificate[0])))
		})
	})
})
