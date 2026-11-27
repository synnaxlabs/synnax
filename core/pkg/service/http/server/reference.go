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
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
)

func authReference(secrets *secret.Service) gorp.Reference[Key, Server] {
	return secret.NewReference(secrets, secret.ReferenceConfig[Key, Server]{
		Name: "auth",
		Targets: func(s *Server) []secret.Key {
			switch a := s.Auth.Variant.(type) {
			case BearerAuth:
				return []secret.Key{a.Token}
			case BasicAuth:
				return []secret.Key{a.Password}
			case APIKeyAuth:
				return []secret.Key{a.Key}
			default:
				return nil
			}
		},
		Rack: func(s *Server) rack.Key { return s.Rack },
	})
}
