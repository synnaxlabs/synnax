// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Use is a resource that uses a secret.
type Use struct {
	// Resource identifies the resource that uses the secret.
	Resource ontology.ID
	// Name is the name of the resource.
	Name string
	// Secret is the key of the secret in use.
	Secret Key
}

// Finder returns the uses of the secrets with the given keys, read through tx.
type Finder func(ctx context.Context, tx gorp.Tx, keys []Key) ([]Use, error)

// Uses returns the uses of the secrets with the given keys. If tx is nil, Uses reads
// directly from the DB.
func (s *Service) Uses(ctx context.Context, tx gorp.Tx, keys ...Key) ([]Use, error) {
	return s.cfg.Uses(ctx, gorp.OverrideTx(s.cfg.DB, tx), keys)
}

// MultiFinder returns a Finder that returns the uses that each of finders returns, in
// the order of finders. It returns the first error that a finder returns.
func MultiFinder(finders ...Finder) Finder {
	return func(ctx context.Context, tx gorp.Tx, keys []Key) ([]Use, error) {
		var uses []Use
		for _, find := range finders {
			found, err := find(ctx, tx, keys)
			if err != nil {
				return nil, err
			}
			uses = append(uses, found...)
		}
		return uses, nil
	}
}
