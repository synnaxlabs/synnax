// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package security

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
)

// insecureProvider is an implementation of Provider for use in insecure clusters.
type insecureProvider struct {
	nodeSecret *rsa.PrivateKey
}

func newInsecureProvider(cfg ProviderConfig) (Provider, error) {
	key, err := rsa.GenerateKey(rand.Reader, cfg.KeySize)
	return &insecureProvider{nodeSecret: key}, err
}

// TLS implements TLSProvider.
func (p *insecureProvider) TLS() *tls.Config { return nil }

// NodePrivate implements KeyProvider.
func (p *insecureProvider) NodePrivate() crypto.PrivateKey { return p.nodeSecret }
