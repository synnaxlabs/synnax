// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/crypto/cryptobyte"
	casn1 "golang.org/x/crypto/cryptobyte/asn1"
)

// plcPEM, badUTF8PEM, and badBMPPEM are the CERT_PLC, CERT_BAD_UTF8, and CERT_BAD_BMP
// vectors of x/cpp/x509, so the Core and the Driver read them the same way. Go refuses
// each one for its names. namesPEM and valuesPEM are CERT_NAMES and CERT_VALUES, whose
// names Go accepts.
const (
	namesPEM = `-----BEGIN CERTIFICATE-----
MIICGzCCAcGgAwIBAgIUB9VfRewbStY6eEYpBTBUmKVyCIwwCgYIKoZIzj0EAwIw
JzERMA8GA1UECgwIUGxhbnQgQ0ExEjAQBgNVBAMMCVJvb3Qgw6kgMTAeFw0yNjAx
MDEwMDAwMDBaFw0yNzAxMDEwMDAwMDBaMIHzMQswCQYDVQQGEwJVUzETMBEGA1UE
CAwKQ2FsaWZvcm5pYTETMBEGA1UEBxQKU8OjbyBQYXVsbzETMBEGA1UECgwKQWNt
ZSwgSW5jLjEMMAoGA1UECwwDRW5nMRcwFQYDVQQLHg4ATwBwAHMAIAA8ADEAPjEV
MBMGCSqGSIb3DQEJARYGYUBiLmNvMSIwDAYDVQQDDAVGaXJzdDASBgoJkiaJk/Is
ZAEBDAR1aWQxMR0wGwYDVQQDDBQgI0xlYWQrdHJhaWw7ICJxIiBcIDELMAkGA1UE
BRMCNDIxFzAVBgoJkiaJk/IsZAEZFgdleGFtcGxlMFkwEwYHKoZIzj0CAQYIKoZI
zj0DAQcDQgAEJJM9P9FoypCiKpA+sf0joTUjsn/8fXtQsMVmfCll58341BpIapc4
BD5Aj1vGkWPhV+AKJu5N5zvKwYyFtlcToDAKBggqhkjOPQQDAgNIADBFAiBA6VB9
rfX6G1JJjjTHxMPj/nkSh0AuTAWmd1ODKJ05yQIhAO05zs9wiDDkc+ob2L3wBKYW
zqJ/caxt+tVnaBYVCv4s
-----END CERTIFICATE-----
`
	valuesPEM = `-----BEGIN CERTIFICATE-----
MIIBfDCCASOgAwIBAgIBBzAKBggqhkjOPQQDAjBIMQ8wDQYDVQQDEwZ2YWx1ZXMx
CzAJBgMqAwQDAgeAMQ0wCwYDKgMFHAQAAABBMQswCQYDVQQKAwIHgDEMMAoGAyoD
BxYDaWE1MB4XDTI2MDEwMTAwMDAwMFoXDTI3MDEwMTAwMDAwMFowSDEPMA0GA1UE
AxMGdmFsdWVzMQswCQYDKgMEAwIHgDENMAsGAyoDBRwEAAAAQTELMAkGA1UECgMC
B4AxDDAKBgMqAwcWA2lhNTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABJxOCBU2
j2WhizBYenYrDLiPyDpk9CJn0pCu76QfzwOHqDof0EEDoEJ7HsjaFwWsvN43TA4V
HAdzwWtaDe9txt8wCgYIKoZIzj0EAwIDRwAwRAIgW6bitmeh7G2crRAhhNXCI0CJ
w+edEQUKCRIq0WNuqNQCIHCuCQTfnoU/SOPymY4QWEbkGNnAtbVukxUcHKEQI011
-----END CERTIFICATE-----
`
	plcPEM = `-----BEGIN CERTIFICATE-----
MIIBDzCBtaADAgECAgEHMAoGCCqGSM49BAMCMBExDzANBgNVBAMTBlBMQ18wMTAe
Fw0yNjAxMDEwMDAwMDBaFw0yNzAxMDEwMDAwMDBaMBExDzANBgNVBAMTBlBMQ18w
MTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABP6ShaQQAsVB4MXyrQhjY+i5TB2Q
mw0Kp2bTPGU1kbxeQ+YRsgu81HVE/oFLwg2RKKFplP5nX2hEqVdJ6eUzT/AwCgYI
KoZIzj0EAwIDSQAwRgIhAM6hUXFU3y33cK28Rggw0qUI/ESTeMYk3ruOD4Vi6f1m
AiEA61+Cv4ctjunZDcLqhyRWv1PVyPldZ5/gAOXI3QAozQs=
-----END CERTIFICATE-----
`
	badUTF8PEM = `-----BEGIN CERTIFICATE-----
MIIBKjCB0aADAgECAgEHMAoGCCqGSM49BAMCMB8xEDAOBgNVBAMTB2JhZHV0Zjgx
CzAJBgNVBAoMAmH/MB4XDTI2MDEwMTAwMDAwMFoXDTI3MDEwMTAwMDAwMFowHzEQ
MA4GA1UEAxMHYmFkdXRmODELMAkGA1UECgwCYf8wWTATBgcqhkjOPQIBBggqhkjO
PQMBBwNCAAQEJRpbpUnksYbGZeU8BVh1vrIRV4MfxBHU14HPNkLEjzUouSTtddfN
vKDj4STNfDjCXnplcH9mwaJkVA1+IdpoMAoGCCqGSM49BAMCA0gAMEUCIGPutBaD
lMq7daL4Yd2ZzjEg6sbaFUhwneVfE8lxRd+3AiEAt6TfnSIAPVDAtVGxHrAxSBja
gh3RZClLMTaDcCidbus=
-----END CERTIFICATE-----
`
	badBMPPEM = `-----BEGIN CERTIFICATE-----
MIIBKzCB06ADAgECAgEHMAoGCCqGSM49BAMCMCAxDzANBgNVBAMTBmJhZGJtcDEN
MAsGA1UECh4E2D3eADAeFw0yNjAxMDEwMDAwMDBaFw0yNzAxMDEwMDAwMDBaMCAx
DzANBgNVBAMTBmJhZGJtcDENMAsGA1UECh4E2D3eADBZMBMGByqGSM49AgEGCCqG
SM49AwEHA0IABO6bl2Q9qM6/tzxEbUSCkFaQL50cSbZEWKnzP+smOOfXHXPGL+yg
NSaRzEx68mr2P89mQpkCtzQqMO4eSX2H65swCgYIKoZIzj0EAwIDRwAwRAIgWejw
YodVMp8wh5nLPbwVeHHA5S1WL0mylnd/QF6IzvACIH1/vLFnpK7LRbV8ImCyT7iP
W9WtvFisAUytADsOcOuK
-----END CERTIFICATE-----
`
)

// plcName is the DER name CN=PLC_01, as a PrintableString that Go refuses.
func plcName() []byte {
	GinkgoHelper()
	var b cryptobyte.Builder
	b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
		b.AddASN1(casn1.SET, func(b *cryptobyte.Builder) {
			b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
				b.AddASN1ObjectIdentifier(asn1.ObjectIdentifier{2, 5, 4, 3})
				b.AddASN1(casn1.PrintableString, func(b *cryptobyte.Builder) {
					b.AddBytes([]byte("PLC_01"))
				})
			})
		})
	})
	return MustSucceed(b.Bytes())
}

// newRefusedCertificate returns a DER certificate issued by CN=PLC_01 to the DER name
// subject, for the DER public key info spki, that is valid from an hour before the
// start of each spec to an hour after, as a GeneralizedTime. Go refuses it for its
// issuer, and its signature is not valid.
func newRefusedCertificate(subject, spki []byte) []byte {
	GinkgoHelper()
	ecdsaWithSHA256 := asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	var b cryptobyte.Builder
	b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
		b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
			b.AddASN1Int64(1)
			b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
				b.AddASN1ObjectIdentifier(ecdsaWithSHA256)
			})
			b.AddBytes(plcName())
			b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
				b.AddASN1UTCTime(start.Add(-time.Hour))
				b.AddASN1GeneralizedTime(start.Add(time.Hour))
			})
			b.AddBytes(subject)
			b.AddBytes(spki)
		})
		b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
			b.AddASN1ObjectIdentifier(ecdsaWithSHA256)
		})
		b.AddASN1BitString([]byte{0})
	})
	return MustSucceed(b.Bytes())
}

// unreadableKey is the DER public key info of a key type that Go cannot read.
func unreadableKey() []byte {
	GinkgoHelper()
	var b cryptobyte.Builder
	b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
		b.AddASN1(casn1.SEQUENCE, func(b *cryptobyte.Builder) {
			b.AddASN1ObjectIdentifier(asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 2})
		})
		b.AddASN1BitString([]byte{1, 2, 3})
	})
	return MustSucceed(b.Bytes())
}

var _ = Describe("ParseDetails", func() {
	var leaf, ca string
	BeforeEach(func() {
		leaf = newCertificate(newKey(), "Leaf", time.Hour)
		ca = newCertificate(newKey(), "CA", time.Hour)
	})

	It("Should give the details of the leaf of a PEM chain", func() {
		details := MustSucceed(certificate.ParseDetails([]byte(leaf + ca)))
		Expect(details.Subject).To(Equal("CN=Leaf"))
		Expect(details.NotAfter).To(Equal(MustSucceed(
			certificate.ParseDetails([]byte(leaf)),
		).NotAfter))
	})

	It("Should give the details of a DER certificate", func() {
		block, _ := pem.Decode([]byte(leaf))
		Expect(MustSucceed(certificate.ParseDetails(block.Bytes))).
			To(Equal(MustSucceed(certificate.ParseDetails([]byte(leaf)))))
	})

	It("Should skip a block of another type after the certificate", func() {
		data := append([]byte(leaf), encodeKey(newKey())...)
		Expect(MustSucceed(certificate.ParseDetails(data)).Subject).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate after a block of another type", func() {
		data := append(encodeKey(newKey()), leaf...)
		Expect(MustSucceed(certificate.ParseDetails(data)).Subject).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate after text", func() {
		data := "Bag Attributes\n    localKeyID: 01\nsubject=CN=Leaf\n" + leaf
		Expect(
			MustSucceed(certificate.ParseDetails([]byte(data))).Subject,
		).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate after a byte order mark", func() {
		data := "\uFEFF" + leaf
		Expect(
			MustSucceed(certificate.ParseDetails([]byte(data))).Subject,
		).To(Equal("CN=Leaf"))
	})

	// derWithPEM returns a DER certificate for CN=Outer that holds the PEM of leaf in
	// an extension.
	derWithPEM := func() []byte {
		GinkgoHelper()
		key := newKey()
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "Outer"},
			NotBefore:    start,
			NotAfter:     start.Add(time.Hour),
			ExtraExtensions: []pkix.Extension{{
				Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1},
				Value: []byte("\n" + leaf),
			}},
		}
		return MustSucceed(
			x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
		)
	}

	It("Should read a DER certificate that holds PEM text as DER", func() {
		Expect(
			MustSucceed(certificate.ParseDetails(derWithPEM())).Subject,
		).To(Equal("CN=Outer"))
	})

	It("Should fail for a cut DER certificate that holds PEM text", func() {
		der := derWithPEM()
		Expect(certificate.ParseDetails(der[:len(der)-16])).Error().
			To(MatchError("x509: malformed certificate"))
	})

	It("Should read PEM whose first line starts with 0", func() {
		data := "0 is the first byte of this file\n" + leaf
		Expect(
			MustSucceed(certificate.ParseDetails([]byte(data))).Subject,
		).To(Equal("CN=Leaf"))
	})

	It("Should read a certificate whose names Go refuses as the Driver does", func() {
		block, _ := pem.Decode([]byte(plcPEM))
		Expect(MustSucceed(certificate.ParseDetails([]byte(plcPEM)))).To(Equal(
			certificate.Details{
				Subject: "CN=PLC_01",
				Issuer:  "CN=PLC_01",
				NotBefore: telem.NewTimeStamp(
					time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				),
				NotAfter: telem.NewTimeStamp(
					time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
				),
				Sha1:   "695568fb733a0c23fbf960ef3586ee0b22c6ca9e",
				Sha256: "0cc6fe78113bed6954c52429c00afa0af0b34705c9b699dc29742d04e0b2d6d3",
			},
		))
		Expect(
			MustSucceed(certificate.ParseDetails(block.Bytes)).Subject,
		).To(Equal("CN=PLC_01"))
	})

	DescribeTable("Should leave out a name value that Go refuses and that is not UTF-8",
		func(pemText, subject string) {
			Expect(
				MustSucceed(certificate.ParseDetails([]byte(pemText))).Subject,
			).To(Equal(subject))
		},
		Entry("UTF8String that is not UTF-8", badUTF8PEM, "CN=badutf8"),
		Entry("BMPString with a surrogate pair", badBMPPEM, "CN=badbmp"),
	)

	DescribeTable("Should read each name value that Go accepts as Go does",
		func(pemText string) {
			block, _ := pem.Decode([]byte(pemText))
			accepted := MustSucceed(x509.ParseCertificate(block.Bytes))
			refused := newRefusedCertificate(
				accepted.RawSubject,
				accepted.RawSubjectPublicKeyInfo,
			)
			Expect(
				MustSucceed(certificate.ParseDetails(refused)).Subject,
			).To(Equal(accepted.Subject.String()))
		},
		Entry("each string type", namesPEM),
		Entry("values that are not text", valuesPEM),
	)

	It("Should give the details of a certificate whose key Go cannot read", func() {
		details := MustSucceed(
			certificate.ParseDetails(newRefusedCertificate(plcName(), unreadableKey())),
		)
		Expect(details.Subject).To(Equal("CN=PLC_01"))
		Expect(details.NotAfter).To(Equal(telem.NewTimeStamp(start.Add(time.Hour))))
	})

	It("Should fail for a cut certificate whose names Go refuses", func() {
		block, _ := pem.Decode([]byte(plcPEM))
		Expect(certificate.ParseDetails(block.Bytes[:len(block.Bytes)-16])).Error().
			To(MatchError("x509: malformed certificate"))
	})

	It("Should fail for a certificate block that does not parse", func() {
		data := append(
			[]byte(leaf),
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("x")})...,
		)
		Expect(certificate.ParseDetails(data)).Error().
			To(MatchError("x509: malformed certificate"))
	})

	It("Should fail for data that holds no certificate", func() {
		Expect(certificate.ParseDetails([]byte("no certificate"))).Error().
			To(MatchError("x509: malformed certificate"))
	})
})
