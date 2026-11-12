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
	"encoding/hex"
	"encoding/pem"
	"strings"
	"time"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
		cert, err := x509.ParseCertificate(block.Bytes)
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
	pub, ok := a.(interface{ Equal(crypto.PublicKey) bool })
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
