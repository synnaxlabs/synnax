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
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"

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
				"holds a %s block. Send a private key as private_key",
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

// checkPrivateKey returns a validation error unless key is a PEM private key that
// matches the leaf of chain.
func checkPrivateKey(chain string, key []byte) error {
	if _, err := tls.X509KeyPair([]byte(chain), key); err != nil {
		return validate.PathedError(
			errors.Wrap(
				validate.ErrValidation,
				strings.TrimPrefix(err.Error(), "tls: "),
			),
			"private_key",
		)
	}
	return nil
}

// samePublicKey reports whether a and b hold the same public key.
func samePublicKey(a, b *x509.Certificate) bool {
	pub, ok := a.PublicKey.(interface{ Equal(crypto.PublicKey) bool })
	return ok && pub.Equal(b.PublicKey)
}

func newDetails(leaf *x509.Certificate) Details {
	sum1 := sha1.Sum(leaf.Raw)
	sum256 := sha256.Sum256(leaf.Raw)
	return Details{
		Subject:   leaf.Subject.String(),
		Issuer:    leaf.Issuer.String(),
		NotBefore: telem.NewTimeStamp(leaf.NotBefore),
		NotAfter:  telem.NewTimeStamp(leaf.NotAfter),
		Sha1:      hex.EncodeToString(sum1[:]),
		Sha256:    hex.EncodeToString(sum256[:]),
	}
}
