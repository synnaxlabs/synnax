// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

type Retrieve struct {
	baseTX   gorp.Tx
	gorp     gorp.Retrieve[Key, User]
	indexes  indexes
	password *password.Service
}

// MatchUsernames returns a filter that matches the users whose password has any of the
// usernames.
func MatchUsernames(usernames ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, User] {
		return gorp.MatchKeysFunc[Key, User](func(ctx gorp.Context) ([]Key, error) {
			var passwords []password.Password
			if err := r.password.NewRetrieve().
				Where(password.MatchUsernames(usernames...)).
				Entries(&passwords).
				Exec(ctx, ctx.Tx); err != nil {
				return nil, err
			}
			keys := make([]Key, 0, len(passwords))
			for _, p := range passwords {
				if p.Subject.Type != ontology.ResourceTypeUser {
					continue
				}
				key, err := KeyFromOntologyID(p.Subject)
				if err != nil {
					return nil, err
				}
				keys = append(keys, key)
			}
			return keys, nil
		})
	}
}
