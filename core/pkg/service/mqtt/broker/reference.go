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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
)

func brokerRack(b *Broker) rack.Key { return b.Rack }

func passwordReference(secrets *secret.Service) gorp.Reference[Key, Broker] {
	return secret.NewReference(secrets, secret.ReferenceConfig[Key, Broker]{
		Name: "auth.password",
		Targets: func(b *Broker) []secret.Key {
			auth, ok := b.Auth.Variant.(UsernameAuth)
			if !ok || auth.Password == nil {
				return nil
			}
			return []secret.Key{*auth.Password}
		},
		Rack: brokerRack,
	})
}

func certificateReference(certs *certificate.Service) gorp.Reference[Key, Broker] {
	return certificate.NewReference(certs, certificate.ReferenceConfig[Key, Broker]{
		Name: "transport.certificates",
		Targets: func(b *Broker) []certificate.Key {
			t, ok := b.Transport.Variant.(TlsTransport)
			if !ok {
				return nil
			}
			var keys []certificate.Key
			if t.CaCertificate != nil {
				keys = append(keys, *t.CaCertificate)
			}
			if t.ClientCertificate != nil {
				keys = append(keys, *t.ClientCertificate)
			}
			return keys
		},
		Rack: brokerRack,
	})
}
