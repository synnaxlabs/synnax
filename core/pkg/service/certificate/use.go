// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/set"
)

var _ secret.Finder = FindSecretUses

// FindSecretUses is a secret.Finder that returns each stored certificate whose private
// key is one of the given secrets.
func FindSecretUses(
	ctx context.Context,
	tx gorp.Tx,
	keys []secret.Key,
) ([]secret.Use, error) {
	wanted := set.New(keys...)
	var certs []Certificate
	if err := gorp.NewRetrieve[Key, Certificate]().
		Where(gorp.Match(func(_ gorp.Context, c *Certificate) (bool, error) {
			key := privateKey(*c)
			return key != nil && wanted.Contains(*key), nil
		})).
		Entries(&certs).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	uses := make([]secret.Use, len(certs))
	for i, c := range certs {
		uses[i] = secret.Use{
			Secret:   *privateKey(c),
			Resource: c.OntologyID(),
			Name:     c.Base().Name,
		}
	}
	return uses, nil
}

func privateKey(c Certificate) *secret.Key {
	if stored, ok := c.Variant.(StoredCertificate); ok {
		return stored.PrivateKey
	}
	return nil
}
