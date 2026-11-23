// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package broker

import (
	"context"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Integration is the name that a rack lists in its integrations when it runs MQTT.
const Integration = "mqtt"

// checkRack refuses a rack that does not run the MQTT integration.
func (w Writer) checkRack(ctx context.Context, b Broker) error {
	var r rack.Rack
	if err := w.racks.NewRetrieve().
		Where(rack.MatchKeys(b.Rack)).
		Entry(&r).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if !slices.Contains(r.Integrations, Integration) {
		return errors.Wrapf(
			validate.ErrValidation,
			"rack: %s does not run the %s integration",
			r.Name,
			Integration,
		)
	}
	return nil
}

// checkClientCertificate refuses a client certificate that holds no private key.
func (w Writer) checkClientCertificate(ctx context.Context, b Broker) error {
	t, ok := b.Transport.Variant.(TlsTransport)
	if !ok || t.ClientCertificate == nil {
		return nil
	}
	var c certificate.Certificate
	err := w.certs.NewRetrieve().
		Where(certificate.MatchKeys(*t.ClientCertificate)).
		Entry(&c).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return errors.Wrapf(
			validate.ErrValidation,
			"transport.client_certificate: certificate %s does not exist",
			*t.ClientCertificate,
		)
	}
	if err != nil {
		return err
	}
	var hasKey bool
	switch v := c.Variant.(type) {
	case certificate.StoredCertificate:
		hasKey = v.PrivateKey != nil
	case certificate.FileCertificate:
		hasKey = v.PrivateKeyFile != nil
	}
	if !hasKey {
		return errors.Wrapf(
			validate.ErrValidation,
			"transport.client_certificate: certificate %q holds no private key",
			c.Base().Name,
		)
	}
	return nil
}
