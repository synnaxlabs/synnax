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
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

func leafOf(c *tls.Certificate) *x509.Certificate {
	return MustSucceed(x509.ParseCertificate(c.Certificate[0]))
}

func copyFile(fs xfs.FS, from, to string) {
	src := MustSucceed(fs.Open(from, os.O_RDONLY))
	data := MustSucceed(io.ReadAll(src))
	Expect(src.Close()).To(Succeed())
	dst := MustSucceed(fs.Open(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
	MustSucceed(dst.Write(data))
	Expect(dst.Close()).To(Succeed())
}

func createNodePairIn(fs xfs.FS, dir string, host address.Address) {
	f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
		LoaderConfig: cert.LoaderConfig{FS: fs, CertsDir: dir},
		KeySize:      mock.SmallKeySize,
		Hosts:        []address.Address{host},
	}))
	Expect(f.CreateCAPair()).To(Succeed())
	Expect(f.CreateNodePair()).To(Succeed())
}

var _ = Describe("Source", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })

	Describe("Factory", func() {
		It("Should reject an unknown source type", func() {
			_, err := cert.NewSource(cert.SourceConfig{Type: "nope"})
			Expect(err).To(MatchError(ContainSubstring("unknown certificate source")))
		})
	})

	Describe("File", func() {
		It("Should serve the certificate from disk", func() {
			mock.GenerateCerts(fs)
			src := MustSucceed(cert.NewSource(cert.SourceConfig{
				Type: cert.SourceTypeFile,
				FS:   fs,
				Cert: "/usr/local/synnax/certs/node.crt",
				Key:  "/usr/local/synnax/certs/node.key",
			}))
			c := MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{}))
			Expect(c.Certificate).To(HaveLen(1))
		})
		It("Should require both a cert and a key", func() {
			_, err := cert.NewSource(cert.SourceConfig{
				Type: cert.SourceTypeFile,
				FS:   fs,
				Cert: "node.crt",
			})
			Expect(err).To(MatchError(ContainSubstring("requires both a cert and a key")))
		})
		It("Should reload the certificate when the files change", func() {
			createNodePairIn(fs, "a", "hostA:9090")
			createNodePairIn(fs, "b", "hostB:9090")
			copyFile(fs, "a/node.crt", "node.crt")
			copyFile(fs, "a/node.key", "node.key")
			src := MustSucceed(cert.NewSource(cert.SourceConfig{
				Type: cert.SourceTypeFile,
				FS:   fs,
				Cert: "node.crt",
				Key:  "node.key",
			}))
			first := leafOf(MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{})))
			Expect(first.DNSNames).To(ContainElement("hostA"))

			copyFile(fs, "b/node.crt", "node.crt")
			copyFile(fs, "b/node.key", "node.key")
			second := leafOf(MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{})))
			Expect(second.DNSNames).To(ContainElement("hostB"))
		})
	})

	Describe("Auto", func() {
		It("Should self-sign a certificate with SANs from the listener address", func() {
			src := MustSucceed(cert.NewSource(cert.SourceConfig{
				Type:    cert.SourceTypeAuto,
				FS:      fs,
				Address: "console.example.com:9091",
				KeySize: mock.SmallKeySize,
			}))
			c := MustSucceed(src.GetCertificate(&tls.ClientHelloInfo{}))
			Expect(leafOf(c).DNSNames).To(ContainElement("console.example.com"))
		})
		It("Should require a listener address", func() {
			_, err := cert.NewSource(cert.SourceConfig{
				Type:    cert.SourceTypeAuto,
				FS:      fs,
				KeySize: mock.SmallKeySize,
			})
			Expect(err).To(MatchError(ContainSubstring("requires a listener address")))
		})
	})
})
