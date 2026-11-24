// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
)

func clusterRack(c *Cluster) rack.Key { return c.Rack }

func credentials(a Auth) (Credentials, bool) {
	switch v := a.Variant.(type) {
	case PlainAuth:
		return v.Credentials, true
	case ScramSha256Auth:
		return v.Credentials, true
	case ScramSha512Auth:
		return v.Credentials, true
	default:
		return Credentials{}, false
	}
}

func passwordReference(secrets *secret.Service) gorp.Reference[Key, Cluster] {
	return secret.NewReference(secrets, secret.ReferenceConfig[Key, Cluster]{
		Name: "auth.password",
		Targets: func(c *Cluster) []secret.Key {
			creds, ok := credentials(c.Auth)
			if !ok {
				return nil
			}
			return []secret.Key{creds.Password}
		},
		Rack: clusterRack,
	})
}

func certificateReference(certs *certificate.Service) gorp.Reference[Key, Cluster] {
	return certificate.NewReference(certs, certificate.ReferenceConfig[Key, Cluster]{
		Name: "transport.certificates",
		Targets: func(c *Cluster) []certificate.Key {
			t, ok := c.Transport.Variant.(TlsTransport)
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
		Rack: clusterRack,
	})
}
