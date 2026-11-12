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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Grant creates a grant that lets subject read the secrets with the given keys once,
// until the grant expires. It returns query.ErrNotFound when a secret does not exist,
// and a validation error when subject is not a rack or keys is empty.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
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
// secret, whose value only the file holds. Resolve opens its own snapshot and
// transaction, so the caller must not hold either.
func (s *Service) Resolve(
	ctx context.Context,
	rack resource.ID,
	grantKey grant.Key,
	keys ...Key,
) (map[Key][]byte, error) {
	var secrets []Secret
	if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.NewRetrieve().
			Where(MatchKeys(keys...)).
			Entries(&secrets).
			Exec(ctx, snap)
	}); err != nil {
		return nil, err
	}
	g, err := s.cfg.Grant.Redeem(ctx, grantKey, rack)
	if err != nil {
		return nil, err
	}
	readable := set.New(g.Secrets...)
	for _, sec := range secrets {
		if base := sec.Base(); !readable.Contains(base.Key) {
			return nil, errors.Wrapf(
				access.ErrDenied,
				"%s cannot read secret %q",
				rack,
				base.Name,
			)
		}
	}
	return s.open(secrets)
}

// Reveal returns the plain text value of each stored secret with the given keys, with
// no access check. It is for Core code that connects with a secret itself, which must
// never send the values to a client. Reveal returns query.ErrNotFound when a secret
// does not exist, and a validation error for a file secret.
func (s *Service) Reveal(ctx context.Context, snap gorp.Snapshot, keys ...Key) (
	map[Key][]byte,
	error,
) {
	var secrets []Secret
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&secrets).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return s.open(secrets)
}

func (s *Service) open(secrets []Secret) (map[Key][]byte, error) {
	values := make(map[Key][]byte, len(secrets))
	for _, sec := range secrets {
		base := sec.Base()
		stored, ok := sec.Variant.(StoredSecret)
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"secret %q is a file, so only the file holds its value",
				base.Name,
			)
		}
		value, err := s.aead.Open(nil, nil, stored.Value, base.Key[:])
		if err != nil {
			return nil, errors.Wrapf(err, "decrypt secret %q", base.Name)
		}
		values[base.Key] = value
	}
	return values, nil
}
