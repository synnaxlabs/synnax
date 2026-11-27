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
	var c certificate.Certificate
	if err := certs.NewRetrieve().
		Where(certificate.MatchKeys(*t.ClientCertificate)).
		Entry(&c).
		Exec(ctx, tx); err != nil {
		return err
	}
	v := validate.New("TLS")
	v.Ternaryf(
		"client_certificate",
		!c.HasPrivateKey(),
		"certificate %q has no private key",
		c.Base().Name,
	)
	return v.Error()
}

// Certificates returns the keys of the certificates that t names. An owner returns them
// from the Targets of a certificate.NewReference, so a certificate in use cannot be
// deleted.
func (t TLS) Certificates() []certificate.Key {
	var keys []certificate.Key
	if ca, ok := t.ServerTrust.Variant.(CertificateAuthorityServerTrust); ok {
		keys = append(keys, ca.Certificate)
	}
	if t.ClientCertificate != nil {
		keys = append(keys, *t.ClientCertificate)
	}
	return keys
}
