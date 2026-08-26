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
	"crypto/x509"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	var fs xfs.FS
	BeforeEach(func() {
		fs = xfs.NewMem()
	})
	Describe("CA Generation", func() {
		It("Should generate a CA and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
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
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateCAPair()).ToNot(Succeed())
		})
		It("Should reuse the CA and key if they already exist", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig:  cert.LoaderConfig{FS: fs},
				AllowKeyReuse: new(true),
				KeySize:       mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateCAPair()).To(Succeed())
		})
	})
	Describe("Node Generation", func() {
		It("Should generate a node certificate and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				Hosts:        []address.Address{"synnaxlabs.com"},
				KeySize:      mock.SmallKeySize,
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
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateNodePair()).
				Error().To(MatchError(ContainSubstring("CA certificate not found")))
		})
		It("Should fail is no hosts are provided", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
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
				LoaderConfig: cert.LoaderConfig{FS: fs},
				Hosts:        hosts,
				KeySize:      mock.SmallKeySize,
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
		It("Should replace a certificate covering none of the configured hosts", func() {
			f := newFactory("old.synnaxlabs.com")
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePairIfStale()).To(Succeed())
			moved := newFactory("new.synnaxlabs.com")
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
				LoaderConfig: cert.LoaderConfig{FS: fs},
				Hosts:        []address.Address{"synnaxlabs.com"},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).To(Succeed())
			c, _ := MustSucceed2(f.Loader.LoadNodePair())
			expectChainsToCA(f, c)
		})
		It("Should chain an in-memory node certificate to the CA", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			tlsC := MustSucceed(f.SignNodeCert([]address.Address{"synnaxlabs.com"}))
			expectChainsToCA(f, MustSucceed(x509.ParseCertificate(tlsC.Certificate[0])))
		})
	})
})
