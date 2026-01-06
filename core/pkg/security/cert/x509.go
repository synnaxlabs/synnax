// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

const (
	validFrom    = -time.Hour * 24
	validFor     = time.Hour * 24 * 365
	caCommonName = "Synnax CA"
)

func newBasex509() (*x509.Certificate, error) {
	sn, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	cert := &x509.Certificate{
		SerialNumber: sn,
		Subject:      pkix.Name{CommonName: caCommonName},
		NotBefore:    time.Now().Add(validFrom),
		NotAfter:     time.Now().Add(validFor),
		KeyUsage:     x509.KeyUsageKeyAgreement | x509.KeyUsageDigitalSignature,
	}
	return cert, err
}
