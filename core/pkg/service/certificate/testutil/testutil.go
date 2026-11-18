// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/x/testutil"
)

// NewPair returns a self-signed PEM certificate for the common name "plant", valid
// for one year, and its PEM private key.
func NewPair() (certificate, privateKey string) {
	ginkgo.GinkgoHelper()
	key := testutil.MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "plant"},
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	der := testutil.MustSucceed(
		x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
	)
	keyDER := testutil.MustSucceed(x509.MarshalPKCS8PrivateKey(key))
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}
