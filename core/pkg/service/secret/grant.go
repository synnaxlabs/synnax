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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Grant creates a grant that lets subject read the secrets with the given keys once,
// whether or not anything on subject uses them, until the grant expires. It returns
// query.ErrNotFound when a secret does not exist, and a validation error when subject
// is not a rack or keys is empty.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	subject ontology.ID,
	keys ...Key,
) (grant.Key, error) {
	if err := s.NewRetrieve().Where(MatchKeys(keys...)).Exec(ctx, tx); err != nil {
		return uuid.Nil(), err
	}
	g := grant.Grant{Subject: subject, Secrets: keys}
	if err := s.cfg.Grant.NewWriter(tx).Create(ctx, &g); err != nil {
		return uuid.Nil(), err
	}
	return g.Key, nil
}

// Resolve returns the plain text value of each stored secret with the given keys, for
// rack to connect with. Rack may read only the secrets that the grant names, and
// Resolve spends the grant, so no later call can use it. Resolve returns
// access.ErrDenied when the grant does not name a secret or has ended,
// query.ErrNotFound when a secret does not exist, and a validation error for a file
// secret, whose value only the file holds.
func (s *Service) Resolve(
	ctx context.Context,
	tx gorp.Tx,
	rack ontology.ID,
	grantKey grant.Key,
	keys ...Key,
) (map[Key][]byte, error) {
	var secrets []Secret
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&secrets).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	g, err := s.cfg.Grant.Redeem(ctx, grantKey, rack)
	if err != nil {
		return nil, err
	}
	readable := set.New(g.Secrets...)
	values := make(map[Key][]byte, len(secrets))
	for _, sec := range secrets {
		base := sec.Base()
		if !readable.Contains(base.Key) {
			return nil, errors.Wrapf(
				access.ErrDenied,
				"%s cannot read secret %q",
				rack,
				base.Name,
			)
		}
		stored, ok := sec.Variant.(StoredSecret)
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"secret %q is a file, so only the file holds its value",
				base.Name,
			)
		}
		if values[base.Key], err = s.aead.Open(
			nil,
			nil,
			stored.Value,
			base.Key[:],
		); err != nil {
			return nil, errors.Wrapf(err, "decrypt secret %q", base.Name)
		}
	}
	return values, nil
}
