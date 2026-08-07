// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert

import "crypto/tls"

// Source resolves the certificate a listener presents during a TLS handshake. It
// matches the crypto/tls GetCertificate callback, so a Source plugs directly into a
// listener's tls.Config. Each source strategy lives in its own package and is
// constructed with only the inputs that strategy needs; there is no shared source
// configuration.
type Source interface {
	// GetCertificate returns the certificate for the given handshake.
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}
