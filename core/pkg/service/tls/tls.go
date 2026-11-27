// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tls

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/x/validate"
)

// Validate returns a validation error when the client certificate of t has no private
// key, because the client signs with it. It reads the certificate in tx.
func Validate(
	ctx context.Context,
	tx gorp.Tx,
	certs *certificate.Service,
	t TLS,
) error {
	if t.ClientCertificate == nil {
		return nil
	}
	hasKey, err := certs.HasPrivateKey(ctx, tx, *t.ClientCertificate)
	if err != nil {
		return err
	}
	v := validate.New("TLS")
	v.Ternary("client_certificate", !hasKey, "certificate has no private key")
	return v.Error()
}

// Certificates returns the keys of the certificates that t names. An owner returns them
// from the Targets of a certificate.NewReference, so a certificate in use cannot be
// deleted.
func Certificates(t TLS) []certificate.Key {
	var keys []certificate.Key
	if ca, ok := t.ServerTrust.Variant.(CertificateAuthorityServerTrust); ok {
		keys = append(keys, ca.Certificate)
	}
	if t.ClientCertificate != nil {
		keys = append(keys, *t.ClientCertificate)
	}
	return keys
}
