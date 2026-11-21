// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
)

func serverRack(s *Server) rack.Key { return s.Rack }

func passwordReference(secrets *secret.Service) gorp.Reference[Key, Server] {
	return secret.NewReference(secrets, secret.ReferenceConfig[Key, Server]{
		Name: "connection.password",
		Targets: func(s *Server) []secret.Key {
			if s.Connection.Password == nil {
				return nil
			}
			return []secret.Key{*s.Connection.Password}
		},
		Rack: serverRack,
	})
}

func certificateReference(certs *certificate.Service) gorp.Reference[Key, Server] {
	return certificate.NewReference(certs, certificate.ReferenceConfig[Key, Server]{
		Name: "connection.certificates",
		Targets: func(s *Server) []certificate.Key {
			var keys []certificate.Key
			for _, k := range []*certificate.Key{
				s.Connection.ClientCertificate,
				s.Connection.ServerCertificate,
			} {
				if k != nil {
					keys = append(keys, *k)
				}
			}
			return keys
		},
		Rack: serverRack,
	})
}
