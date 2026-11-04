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
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Grant creates a grant that lets rack read the secrets with the given keys, whether or
// not anything on the rack uses them, until session ends or the grant is deleted. It
// returns query.ErrNotFound when a secret does not exist, and a validation error when
// rack is not a rack, keys is empty, or session is zero.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	rack ontology.ID,
	session auth.SessionKey,
	keys ...Key,
) (grant.Key, error) {
	if err := s.NewRetrieve().Where(MatchKeys(keys...)).Exec(ctx, tx); err != nil {
		return uuid.Nil(), err
	}
	g := grant.Grant{Rack: rack, Secrets: keys, Session: session}
	if err := s.cfg.Grant.NewWriter(tx).Create(ctx, &g); err != nil {
		return uuid.Nil(), err
	}
	return g.Key, nil
}

// granted returns the secrets that the grant with key lets rack read. A zero key lets
// rack read nothing.
func (s *Service) granted(
	ctx context.Context,
	tx gorp.Tx,
	rack ontology.ID,
	key grant.Key,
) (set.Set[Key], error) {
	if key == uuid.Nil() {
		return set.Set[Key]{}, nil
	}
	g, err := s.cfg.Grant.Get(ctx, tx, key)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil, errors.Wrapf(
				access.ErrDenied,
				"grant %s has ended or does not exist",
				key,
			)
		}
		return nil, err
	}
	if g.Rack != rack {
		return nil, errors.Wrapf(access.ErrDenied, "grant is not for %s", rack)
	}
	return set.New(g.Secrets...), nil
}

// Resolve returns the plain text value of each stored secret with the given keys, for
// rack to connect with. Rack may read a secret when something on the rack uses it, or
// when the grant with the given key names it. A zero grant key names no grant. Resolve
// returns access.ErrDenied when rack may not read a secret or the grant has ended,
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
	var granted set.Set[Key]
	values := make(map[Key][]byte, len(secrets))
	for _, sec := range secrets {
		base := sec.Base()
		if !readable.Contains(base.Key) {
			if granted == nil {
				if granted, err = s.granted(ctx, tx, rack, grantKey); err != nil {
					return nil, err
				}
			}
			if !granted.Contains(base.Key) {
				return nil, errors.Wrapf(
					access.ErrDenied,
					"%s cannot read secret %q",
					rack,
					base.Name,
				)
			}
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
