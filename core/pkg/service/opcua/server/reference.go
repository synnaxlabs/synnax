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
		Name: "connection.auth.password",
		Targets: func(s *Server) []secret.Key {
			auth, ok := s.Connection.Auth.Variant.(UsernameAuth)
			if !ok || auth.Password == nil {
				return nil
			}
			return []secret.Key{*auth.Password}
		},
		Rack: serverRack,
	})
}

// securedChannel returns the settings of a signed or encrypted connection, or false
// for a connection without security.
func securedChannel(c Connection) (SecuredChannel, bool) {
	switch v := c.Security.Variant.(type) {
	case SignSecurity:
		return v.SecuredChannel, true
	case SignAndEncryptSecurity:
		return v.SecuredChannel, true
	default:
		return SecuredChannel{}, false
	}
}

func certificateReference(certs *certificate.Service) gorp.Reference[Key, Server] {
	return certificate.NewReference(certs, certificate.ReferenceConfig[Key, Server]{
		Name: "connection.certificates",
		Targets: func(s *Server) []certificate.Key {
			var keys []certificate.Key
			if auth, ok := s.Connection.Auth.Variant.(CertificateAuth); ok {
				keys = append(keys, auth.Certificate)
			}
			if c, ok := securedChannel(s.Connection); ok {
				if c.ClientCertificate != nil {
					keys = append(keys, *c.ClientCertificate)
				}
				if pinned, ok := c.ServerTrust.Variant.(PinnedServerTrust); ok {
					keys = append(keys, pinned.Certificate)
				}
			}
			return keys
		},
		Rack: serverRack,
	})
}
