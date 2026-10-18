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
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// ProvisionRoot makes exactly one user the root user and sets its password to creds.
// The Core calls it at every start, so the start settings own the root password. A root
// user whose username already matches keeps its record. Else a user holding the
// username
// is promoted in place. Else a root user record is created. Every other root user is
// demoted and keeps its roles. The whole call runs in one transaction and returns the
// root user. It returns a validation error when creds is incomplete.
func ProvisionRoot(
	ctx context.Context,
	users *Service,
	passwords *password.Service,
	creds auth.Credentials,
) (User, error) {
	if err := creds.Validate(); err != nil {
		return User{}, err
	}
	var root User
	if err := users.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		roots, err := users.loadRootUsers(ctx, tx)
		if err != nil {
			return err
		}
		if root, err = users.resolveRoot(ctx, tx, roots, creds.Username); err != nil {
			return err
		}
		for _, r := range roots {
			if r.Key != root.Key {
				if err := users.demoteRoot(ctx, tx, r); err != nil {
					return err
				}
			}
		}
		return passwords.NewWriter(tx).Set(ctx, root.OntologyID(), creds)
	}); err != nil {
		return User{}, err
	}
	return root, nil
}

// loadRootUsers returns every user record with RootUser=true.
func (s *Service) loadRootUsers(ctx context.Context, tx gorp.Tx) ([]User, error) {
	var roots []User
	if err := s.NewRetrieve().
		Where(MatchRootUser(true)).
		Entries(&roots).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "load root users")
	}
	return roots, nil
}

// resolveRoot returns the user that holds username as the root user: an existing root
// with that username, else a user with that username promoted in place, else a new
// root user record.
func (s *Service) resolveRoot(
	ctx context.Context,
	tx gorp.Tx,
	roots []User,
	username string,
) (User, error) {
	for _, r := range roots {
		if r.Username == username {
			return r, nil
		}
	}
	var existing User
	err := s.NewRetrieve().
		Where(MatchUsernames(username)).
		Entry(&existing).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		created, err := s.NewWriter(tx).create(ctx, User{
			Username: username,
			RootUser: true,
		})
		if err != nil {
			return User{}, errors.Wrap(err, "create root user record")
		}
		s.cfg.L.Info("created root user record", zap.String("username", username))
		return created, nil
	}
	if err != nil {
		return User{}, errors.Wrap(err, "look up configured root username")
	}
	if err := s.NewWriter(tx).setRootUser(ctx, existing.Key, true); err != nil {
		return User{}, errors.Wrapf(err, "promote existing user %q to root", username)
	}
	existing.RootUser = true
	s.cfg.L.Info("promoted existing user to root", zap.String("username", username))
	return existing, nil
}

// demoteRoot clears the RootUser flag on u and keeps its roles. It always logs a
// warning so operators notice the change of ownership.
func (s *Service) demoteRoot(ctx context.Context, tx gorp.Tx, u User) error {
	if err := s.NewWriter(tx).setRootUser(ctx, u.Key, false); err != nil {
		return errors.Wrapf(err, "demote root user %q", u.Username)
	}
	s.cfg.L.Warn("demoted root user", zap.String("username", u.Username))
	return nil
}
