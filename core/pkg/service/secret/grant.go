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
	"slices"
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
// rack to connect with. With a zero grant key, rack may read a secret when something on
// the rack uses it. With a grant key, rack may read only the secrets that the grant
// names, and Resolve spends the grant, so no later call can use it. Resolve returns
// access.ErrDenied when rack may not read a secret or the grant has ended,
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
	readable, err := s.readable(ctx, tx, rack, grantKey, keys)
	if err != nil {
		return nil, err
	}
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

// readable returns the secrets among keys that rack may read: the secrets that the
// grant names, or with a zero grant key, the secrets that something on the rack uses.
func (s *Service) readable(
	ctx context.Context,
	tx gorp.Tx,
	rack ontology.ID,
	grantKey grant.Key,
	keys []Key,
) (set.Set[Key], error) {
	if grantKey != uuid.Nil() {
		g, err := s.cfg.Grant.Redeem(ctx, grantKey, rack)
		if err != nil {
			return nil, err
		}
		return set.New(g.Secrets...), nil
	}
	uses, err := s.cfg.Uses(ctx, gorp.OverrideTx(s.cfg.DB, tx), keys)
	if err != nil {
		return nil, err
	}
	readable := make(set.Set[Key], len(keys))
	for _, use := range uses {
		if slices.Contains(use.Racks, rack) {
			readable.Add(use.Secret)
		}
	}
	return readable, nil
}
