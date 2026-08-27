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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// errSource is a cert.Source whose certificate load always fails.
type errSource struct{}

func (errSource) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return nil, errors.New("source failed")
}

// generateChain builds a three-tier chain: a self-signed root, an intermediate it
// signs, and a leaf for host the intermediate signs. It returns the root PEM, the
// leaf-plus-intermediate chain PEM, and the leaf key PEM.
func generateChain(host string) (rootPEM, chainPEM, keyPEM []byte) {
	GinkgoHelper()
	template := func(serial int64, cn string, ca bool) *x509.Certificate {
		return &x509.Certificate{
			SerialNumber:          big.NewInt(serial),
			Subject:               pkix.Name{CommonName: cn},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().Add(time.Hour),
			IsCA:                  ca,
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		}
	}
	rootKey := MustSucceed(rsa.GenerateKey(rand.Reader, mock.SmallKeySize))
	rootT := template(1, "Test Root", true)
	rootDER := MustSucceed(
		x509.CreateCertificate(rand.Reader, rootT, rootT, &rootKey.PublicKey, rootKey),
	)
	root := MustSucceed(x509.ParseCertificate(rootDER))
	interKey := MustSucceed(rsa.GenerateKey(rand.Reader, mock.SmallKeySize))
	interT := template(2, "Test Intermediate", true)
	interDER := MustSucceed(
		x509.CreateCertificate(rand.Reader, interT, root, &interKey.PublicKey, rootKey),
	)
	inter := MustSucceed(x509.ParseCertificate(interDER))
	leafKey := MustSucceed(rsa.GenerateKey(rand.Reader, mock.SmallKeySize))
	leafT := template(3, "Test Leaf", false)
	leafT.DNSNames = []string{host}
	leafDER := MustSucceed(
		x509.CreateCertificate(rand.Reader, leafT, inter, &leafKey.PublicKey, interKey),
	)
	encode := func(der []byte) []byte {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	}
	rootPEM = encode(rootDER)
	chainPEM = append(encode(leafDER), encode(interDER)...)
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(leafKey),
	})
	return rootPEM, chainPEM, keyPEM
}

// writeFile writes data to path on fs, creating the file.
func writeFile(fs xfs.FS, path string, data []byte) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(path, os.O_CREATE|os.O_WRONLY))
	MustSucceed(f.Write(data))
	Expect(f.Close()).To(Succeed())
}

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
				c = MustSucceed(
					config.GetClientCertificate(&tls.CertificateRequestInfo{}),
				)
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
		Describe("VerifyCertHost", func() {
			It("Should accept a certificate valid for the host", func() {
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
				Expect(prov.VerifyCertHost(src, "localhost")).To(Succeed())
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
				Expect(prov.VerifyCertHost(src, "other-host")).ToNot(Succeed())
			})
			It("Should accept a certificate an unrelated CA signed", func() {
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
				Expect(prov.VerifyCertHost(foreign, "localhost")).To(Succeed())
			})
			It("Should surface a certificate source failure", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				Expect(prov.VerifyCertHost(errSource{}, "localhost")).
					To(MatchError(ContainSubstring("source failed")))
			})
		})
		Describe("VerifyCertCoreCA", func() {
			It("Should accept a certificate the Core CA signed", func() {
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
				Expect(prov.VerifyCertCoreCA(src)).To(Succeed())
			})
			It("Should reject a certificate an unrelated CA signed", func() {
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
				Expect(prov.VerifyCertCoreCA(foreign)).ToNot(Succeed())
			})
			It("Should verify a chain through a presented intermediate", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				rootPEM, chainPEM, keyPEM := generateChain("localhost")
				writeFile(fs, "/usr/local/synnax/certs/root.crt", rootPEM)
				writeFile(fs, "/usr/local/synnax/certs/chain.crt", chainPEM)
				writeFile(fs, "/usr/local/synnax/certs/chain.key", keyPEM)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs, CACertPath: "root.crt"},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				src := MustSucceed(file.NewSource(fs,
					"/usr/local/synnax/certs/chain.crt",
					"/usr/local/synnax/certs/chain.key",
				))
				Expect(prov.VerifyCertCoreCA(src)).To(Succeed())
			})
			It("Should surface a certificate source failure", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				Expect(prov.VerifyCertCoreCA(errSource{})).
					To(MatchError(ContainSubstring("source failed")))
			})
		})
		Describe("VerifyCertTrustAnchors", func() {
			It("Should accept a certificate the Core CA signed", func() {
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
				Expect(prov.VerifyCertTrustAnchors(src)).To(Succeed())
			})
			It("Should accept the node certificate when its CA is absent", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs, CACertPath: "absent.crt"},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				src := MustSucceed(file.NewSource(fs,
					"/usr/local/synnax/certs/node.crt",
					"/usr/local/synnax/certs/node.key",
				))
				Expect(prov.VerifyCertTrustAnchors(src)).To(Succeed())
			})
			It("Should reject a certificate outside the trust anchors", func() {
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
				Expect(prov.VerifyCertTrustAnchors(foreign)).ToNot(Succeed())
			})
			It("Should verify a chain through a presented intermediate", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				rootPEM, chainPEM, keyPEM := generateChain("localhost")
				writeFile(fs, "/usr/local/synnax/certs/root.crt", rootPEM)
				writeFile(fs, "/usr/local/synnax/certs/chain.crt", chainPEM)
				writeFile(fs, "/usr/local/synnax/certs/chain.key", keyPEM)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs, CACertPath: "root.crt"},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				src := MustSucceed(file.NewSource(fs,
					"/usr/local/synnax/certs/chain.crt",
					"/usr/local/synnax/certs/chain.key",
				))
				Expect(prov.VerifyCertTrustAnchors(src)).To(Succeed())
			})
			It("Should surface a certificate source failure", func() {
				fs := xfs.NewMem()
				mock.GenerateCerts(fs)
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					LoaderConfig: cert.LoaderConfig{FS: fs},
					KeySize:      mock.SmallKeySize,
					Insecure:     new(false),
				}))
				Expect(prov.VerifyCertTrustAnchors(errSource{})).
					To(MatchError(ContainSubstring("source failed")))
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
		Describe("Certificate Verification", func() {
			It("Should be a no-op", func() {
				prov := MustSucceed(security.NewProvider(security.ProviderConfig{
					Insecure: new(true),
					KeySize:  mock.SmallKeySize,
				}))
				Expect(prov.VerifyCertHost(nil, "")).To(Succeed())
				Expect(prov.VerifyCertCoreCA(nil)).To(Succeed())
				Expect(prov.VerifyCertTrustAnchors(nil)).To(Succeed())
			})
		})
	})
})
