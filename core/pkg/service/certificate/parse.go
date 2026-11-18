// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"bytes"
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/cryptobyte"
	casn1 "golang.org/x/crypto/cryptobyte/asn1"
)

// parseChain parses the leaf-first chain of CERTIFICATE blocks in text. It returns the
// chain encoded again as PEM, so no text outside the blocks is kept, and the leaf. It
// returns a validation error when text holds no certificate, holds a block of another
// type, or holds a certificate that does not parse.
func parseChain(text string) (string, *x509.Certificate, error) {
	var (
		out  bytes.Buffer
		leaf *x509.Certificate
		rest = []byte(text)
	)
	for {
		var block *pem.Block
		if block, rest = pem.Decode(rest); block == nil {
			break
		}
		if strings.HasSuffix(block.Type, "PRIVATE KEY") {
			return "", nil, pemError(
				"holds a %s block. Send a private key as private_key_pem",
				block.Type,
			)
		}
		if block.Type != "CERTIFICATE" {
			return "", nil, pemError("holds a %s block, not a certificate", block.Type)
		}
		cert, err := parseCertificate(block.Bytes)
		if err != nil {
			return "", nil, pemError("%s", err.Error())
		}
		if leaf == nil {
			leaf = cert
		}
		if err = pem.Encode(&out, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: block.Bytes,
		}); err != nil {
			return "", nil, err
		}
	}
	if leaf == nil {
		return "", nil, pemError("holds no certificate")
	}
	return out.String(), leaf, nil
}

// ParseDetails returns the details of the leaf of the certificate chain in data. data
// is DER when it starts as a DER certificate does, with a SEQUENCE and a long-form
// length. Other data is PEM when it holds a CERTIFICATE block, else DER. The leaf is
// the first CERTIFICATE block, and blocks of other types are skipped. It returns an
// error when a certificate does not parse.
func ParseDetails(data []byte) (Details, error) {
	if isDER(data) {
		leaf, err := parseCertificate(data)
		if err != nil {
			return Details{}, err
		}
		return newDetails(leaf), nil
	}
	// A UTF-8 byte order mark hides a block on the first line from pem.Decode.
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))
	var leaf *x509.Certificate
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := parseCertificate(block.Bytes)
		if err != nil {
			return Details{}, err
		}
		if leaf == nil {
			leaf = cert
		}
	}
	if leaf == nil {
		var err error
		if leaf, err = parseCertificate(data); err != nil {
			return Details{}, err
		}
	}
	return newDetails(leaf), nil
}

// parseCertificate parses the DER certificate der. When Go refuses it, such as for a
// "_" in a PrintableString name, it reads only the names, the validity, and the key. A
// name value that Go refuses then keeps its bytes when they are UTF-8, and is left out
// when they are not. The result then holds only Raw, Issuer, Subject, NotBefore,
// NotAfter, and PublicKey. It returns the error of x509.ParseCertificate when der does
// not hold those parts.
func parseCertificate(der []byte) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(der)
	if err == nil {
		return cert, nil
	}
	if cert, ok := readParts(der); ok {
		return cert, nil
	}
	return nil, err
}

func readParts(der []byte) (*x509.Certificate, bool) {
	var (
		input                                 = cryptobyte.String(der)
		outer, tbs, validity, issuer, subject cryptobyte.String
		key                                   cryptobyte.String
	)
	if !input.ReadASN1(&outer, casn1.SEQUENCE) || !input.Empty() ||
		!outer.ReadASN1(&tbs, casn1.SEQUENCE) ||
		!tbs.SkipOptionalASN1(casn1.Tag(0).Constructed().ContextSpecific()) ||
		!tbs.SkipASN1(casn1.INTEGER) ||
		!tbs.SkipASN1(casn1.SEQUENCE) ||
		!tbs.ReadASN1Element(&issuer, casn1.SEQUENCE) ||
		!tbs.ReadASN1(&validity, casn1.SEQUENCE) ||
		!tbs.ReadASN1Element(&subject, casn1.SEQUENCE) ||
		!tbs.ReadASN1Element(&key, casn1.SEQUENCE) {
		return nil, false
	}
	cert := &x509.Certificate{Raw: der}
	var ok bool
	if cert.Issuer, ok = readName(issuer); !ok {
		return nil, false
	}
	if cert.Subject, ok = readName(subject); !ok {
		return nil, false
	}
	if cert.NotBefore, ok = readTime(&validity); !ok {
		return nil, false
	}
	if cert.NotAfter, ok = readTime(&validity); !ok {
		return nil, false
	}
	// x509.ParseCertificate also leaves the key nil for a type that it cannot read.
	if pub, err := x509.ParsePKIXPublicKey(key); err == nil {
		cert.PublicKey = pub
	}
	return cert, true
}

func readTime(der *cryptobyte.String) (time.Time, bool) {
	var (
		t  time.Time
		ok bool
	)
	switch {
	case der.PeekASN1Tag(casn1.UTCTime):
		ok = der.ReadASN1UTCTime(&t)
	case der.PeekASN1Tag(casn1.GeneralizedTime):
		ok = der.ReadASN1GeneralizedTime(&t)
	}
	return t, ok
}

func readName(der cryptobyte.String) (pkix.Name, bool) {
	var rdns pkix.RDNSequence
	if !der.ReadASN1(&der, casn1.SEQUENCE) {
		return pkix.Name{}, false
	}
	for !der.Empty() {
		var (
			set cryptobyte.String
			rdn pkix.RelativeDistinguishedNameSET
		)
		if !der.ReadASN1(&set, casn1.SET) {
			return pkix.Name{}, false
		}
		for !set.Empty() {
			var (
				atv, value cryptobyte.String
				tag        casn1.Tag
				attr       pkix.AttributeTypeAndValue
			)
			if !set.ReadASN1(&atv, casn1.SEQUENCE) ||
				!atv.ReadASN1ObjectIdentifier(&attr.Type) ||
				!atv.ReadAnyASN1Element(&value, &tag) {
				return pkix.Name{}, false
			}
			var ok bool
			if attr.Value, ok = nameValue(value, tag); ok {
				rdn = append(rdn, attr)
			}
		}
		rdns = append(rdns, rdn)
	}
	var name pkix.Name
	name.FillFromRDNSequence(&rdns)
	return name, true
}

// nameValue returns the value of a name attribute from its DER element. A string reads
// as x509.ParseCertificate reads it, and another type keeps its DER, which
// pkix.Name.String prints as hex as Go does. A string that Go refuses keeps its bytes
// when they are UTF-8, and nameValue returns false when they are not.
func nameValue(element cryptobyte.String, tag casn1.Tag) (any, bool) {
	content := element
	if !content.ReadASN1(&content, tag) {
		return nil, false
	}
	switch tag {
	case casn1.T61String:
		// Go reads T.61 as Latin-1.
		out := make([]byte, 0, len(content))
		for _, b := range content {
			out = utf8.AppendRune(out, rune(b))
		}
		return string(out), true
	case casn1.PrintableString:
		if !slices.ContainsFunc(content, func(b byte) bool { return !isPrintable(b) }) {
			return string(content), true
		}
	case casn1.UTF8String:
		return string(content), utf8.Valid(content)
	case casn1.Tag(asn1.TagBMPString):
		if s, ok := readBMP(content); ok {
			return s, true
		}
	case casn1.IA5String:
		if !slices.ContainsFunc(
			content,
			func(b byte) bool { return b > unicode.MaxASCII },
		) {
			return string(content), true
		}
	case casn1.Tag(asn1.TagNumericString):
		if !slices.ContainsFunc(content, func(b byte) bool {
			return (b < '0' || b > '9') && b != ' '
		}) {
			return string(content), true
		}
	default:
		return asn1.RawValue{FullBytes: element}, true
	}
	return string(content), utf8.Valid(content)
}

// isPrintable reports whether x509.ParseCertificate accepts b in a PrintableString.
func isPrintable(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9' ||
		'\'' <= b && b <= ')' || '+' <= b && b <= '/' ||
		b == ' ' || b == ':' || b == '=' || b == '?' || b == '*' || b == '&'
}

// readBMP reads a BMPString as x509.ParseCertificate does. It returns false for a
// string that Go refuses.
func readBMP(b []byte) (string, bool) {
	if len(b)%2 != 0 {
		return "", false
	}
	b = bytes.TrimSuffix(b, []byte{0, 0})
	points := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		p := uint16(b[i])<<8 | uint16(b[i+1])
		if p == 0xfffe || p == 0xffff || (p >= 0xfdd0 && p <= 0xfdef) ||
			(p >= 0xd800 && p <= 0xdfff) {
			return "", false
		}
		points = append(points, p)
	}
	return string(utf16.Decode(points)), true
}

// isDER reports whether data starts with a SEQUENCE tag and a long-form length of one
// to four bytes, as every DER certificate does. No text starts that way, so a DER
// certificate that holds PEM text in an extension is never read as PEM.
func isDER(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x30 && data[1] >= 0x81 && data[1] <= 0x84
}

func pemError(format string, args ...any) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		"pem",
	)
}

// checkPrivateKey returns a validation error unless key holds a PEM private key that
// matches leaf. It skips the blocks before the key, such as EC PARAMETERS.
func checkPrivateKey(leaf *x509.Certificate, key []byte) error {
	block, rest := pem.Decode(key)
	for block != nil && !strings.HasSuffix(block.Type, "PRIVATE KEY") {
		block, rest = pem.Decode(rest)
	}
	if block == nil {
		return keyError("is not a PEM private key")
	}
	priv, ok := parsePrivateKey(block)
	if !ok {
		return keyError("is a key type Synnax cannot read")
	}
	if !samePublicKey(priv.Public(), leaf.PublicKey) {
		return keyError("does not match the certificate")
	}
	return nil
}

func parsePrivateKey(block *pem.Block) (crypto.Signer, bool) {
	var (
		key any
		err error
	)
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	signer, ok := key.(crypto.Signer)
	return signer, ok
}

func keyError(msg string) error {
	return validate.PathedError(
		errors.Wrap(validate.ErrValidation, msg),
		"private_key_pem",
	)
}

// samePublicKey reports whether a and b are the same public key.
func samePublicKey(a, b crypto.PublicKey) bool {
	pub, ok := a.(interface {
		Equal(other crypto.PublicKey) bool
	})
	return ok && pub.Equal(b)
}

// timeStamp converts t to a TimeStamp, saturating at its bounds: a certificate may use
// a date such as 9999-12-31 that a TimeStamp cannot hold.
func timeStamp(t time.Time) telem.TimeStamp {
	if t.After(telem.TimeStampMax.Time()) {
		return telem.TimeStampMax
	}
	if t.Before(telem.TimeStampMin.Time()) {
		return telem.TimeStampMin
	}
	return telem.NewTimeStamp(t)
}

func newDetails(leaf *x509.Certificate) Details {
	sum1 := sha1.Sum(leaf.Raw)
	sum256 := sha256.Sum256(leaf.Raw)
	return Details{
		Subject:   leaf.Subject.String(),
		Issuer:    leaf.Issuer.String(),
		NotBefore: timeStamp(leaf.NotBefore),
		NotAfter:  timeStamp(leaf.NotAfter),
		Sha1:      hex.EncodeToString(sum1[:]),
		Sha256:    hex.EncodeToString(sum256[:]),
	}
}
