// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package security_test

import (
	"crypto/tls"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OtelProvider", func() {
	Describe("Secure", func() {
		Describe("TLS Properties", func() {
			It("Should load and return the correct TLS configuration", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				src := MustSucceed(file.NewSource(fs,
					"/usr/local/synnax/certs/node.crt",
					"/usr/local/synnax/certs/node.key",
				))
				config := prov.TLSConfigFor(src)
				Expect(config).ToNot(BeNil())
				Expect(config.GetCertificate).ToNot(BeNil())
				c := MustSucceed(config.GetCertificate(&tls.ClientHelloInfo{}))
				Expect(c.Certificate).To(HaveLen(1))
				Expect(config.GetClientCertificate).ToNot(BeNil())
				c = MustSucceed(config.GetClientCertificate(&tls.CertificateRequestInfo{}))
				Expect(c.Certificate).To(HaveLen(1))
				Expect(config.RootCAs).ToNot(BeNil())
				Expect(config.ClientAuth).To(Equal(tls.NoClientCert))
				Expect(config.MinVersion).To(Equal(uint16(tls.VersionTLS13)))
				Expect(config.ClientCAs).ToNot(BeNil())
			})
			It("Should return an error if the node certificate is not found", func() {
				fs := xfs.NewMem()
				_, err := security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				})
				Expect(err).To(MatchError(os.ErrNotExist))
			})
		})
		Describe("Node Private", func() {
			It("Should return the node private key", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				Expect(prov.NodePrivate()).ToNot(BeNil())
			})

		})
		Describe("VerifyCoreCert", func() {
			It("Should accept a certificate signed by the Core CA", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				src := MustSucceed(file.NewSource(fs,
					"/usr/local/synnax/certs/node.crt",
					"/usr/local/synnax/certs/node.key",
				))
				Expect(prov.VerifyCoreCert(src, "localhost")).To(Succeed())
			})
			It("Should reject a certificate that is not valid for the host", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				src := MustSucceed(file.NewSource(fs,
					"/usr/local/synnax/certs/node.crt",
					"/usr/local/synnax/certs/node.key",
				))
				Expect(prov.VerifyCoreCert(src, "other-host")).ToNot(Succeed())
			})
			It("Should reject a certificate signed by a foreign CA", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				foreignFS := xfs.NewMem()
				mock.GenerateCerts(foreignFS)
				foreign := MustSucceed(file.NewSource(foreignFS,
					"/usr/local/synnax/certs/node.crt",
					"/usr/local/synnax/certs/node.key",
				))
				Expect(prov.VerifyCoreCert(foreign, "localhost")).ToNot(Succeed())
			})
		})
	})
	Describe("Insecure", func() {
		Describe("TLS Properties", func() {
			It("Should return an empty TLS configuration", func() {
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					Insecure: new(true),
					KeySize:  mock.SmallKeySize,
				}))
				Expect(prov.TLSConfigFor(nil)).To(BeNil())
			})
		})
		Describe("Node Private", func() {
			It("Should return the randomly generated private key", func() {
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					Insecure: new(true),
					KeySize:  mock.SmallKeySize,
				}))
				Expect(prov.NodePrivate()).ToNot(BeNil())
			})
		})
		Describe("VerifyCoreCert", func() {
			It("Should be a no-op", func() {
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					Insecure: new(true),
					KeySize:  mock.SmallKeySize,
				}))
				Expect(prov.VerifyCoreCert(nil, "")).To(Succeed())
			})
		})
	})
})
