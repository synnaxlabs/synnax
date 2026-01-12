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
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/errors"
)

// secureProvider implements the Provider interface for use in a secure cluster.
type secureProvider struct {
	ProviderConfig
	loader   *cert.Loader
	tls      *tls.Certificate
	certPool *x509.CertPool
}

func newSecureProvider(cfg ProviderConfig) (Provider, error) {
	l, err := cert.NewLoader(cfg.LoaderConfig)
	if err != nil {
		return nil, err
	}
	p := &secureProvider{ProviderConfig: cfg, loader: l, certPool: x509.NewCertPool()}
	cas, err := l.LoadCAs()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, ca := range cas {
		p.certPool.AddCert(ca)
	}
	p.tls, err = l.LoadNodeTLS()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// defaultCipherSuites is the default list of cipher suites used by the node. Order of
// the list is irrelevant, as prioritization is hard-coded in the go standard library.
// This is a subset of the cipher suites supported by the go standard library[1], which
// are also marked as recommended by IETF[2]. Mozilla recommends the same suites because
// athey provide forward secrecy and authentication[3].
//
// Thanks to the CockroachDB team for this list of cipher suites[4].
//
// [1]: https://github.com/golang/go/blob/4aa1efed4853ea067d665a952eee77c52faac774/src/crypto/tls/cipher_suites.go#L215-L270
// [2]: https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#tls-parameters-4
// [3]: https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28recommended.29
// [4]: https://github.com/cockroachdb/cockroach/blob/a9fcbd15b24bcf09e139785921be8d9ff8cad729/pkg/security/tls.go#L106
var defaultCipherSuites = []uint16{
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

// TLS implements TLSProvider.
func (p *secureProvider) TLS() *tls.Config {
	return &tls.Config{
		GetCertificate:       p.getCert,
		RootCAs:              p.certPool,
		ClientAuth:           tls.NoClientCert,
		ClientCAs:            p.certPool,
		GetClientCertificate: p.getClientCert,
		CipherSuites:         defaultCipherSuites,
		MinVersion:           tls.VersionTLS10,
		NextProtos:           []string{"http/1.1", "h2"},
	}
}

// NodePrivate implements KeyProvider.
func (p *secureProvider) NodePrivate() crypto.PrivateKey { return p.tls.PrivateKey }

func (p *secureProvider) getClientCert(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return p.tls, nil
}

func (p *secureProvider) getCert(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return p.tls, nil
}
