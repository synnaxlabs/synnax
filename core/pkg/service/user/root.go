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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// ProvisionRoot makes the user whose password holds creds.Username the only root user,
// promoting or creating it, and demotes every other root user without touching its
// roles. It writes the password only when creds does not verify, and reports that
// write. It runs in one transaction and returns a validation error when creds is
// incomplete.
func ProvisionRoot(
	ctx context.Context,
	users *Service,
	passwords *password.Service,
	creds auth.Credentials,
) (User, bool, error) {
	if err := creds.Validate(); err != nil {
		return User{}, false, err
	}
	changed, err := passwordChanged(ctx, passwords, creds)
	if err != nil {
		return User{}, false, err
	}
	var root User
	if err := users.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		roots, err := users.loadRootUsers(ctx, tx)
		if err != nil {
			return err
		}
		if root, err = users.resolveRoot(
			ctx,
			tx,
			passwords,
			roots,
			creds.Username,
		); err != nil {
			return err
		}
		for _, r := range roots {
			if r.Key != root.Key {
				if err := users.demoteRoot(ctx, tx, r); err != nil {
					return err
				}
			}
		}
		if !changed {
			return nil
		}
		return passwords.NewWriter(tx).Set(ctx, root.OntologyID(), creds)
	}); err != nil {
		return User{}, false, err
	}
	return root, changed, nil
}

// passwordChanged counts a missing record as changed.
func passwordChanged(
	ctx context.Context,
	passwords *password.Service,
	creds auth.Credentials,
) (bool, error) {
	_, err := passwords.Authenticate(ctx, auth.Proof{
		Variant: auth.PasswordProof{Credentials: creds},
	})
	if err == nil {
		return false, nil
	}
	if errors.Is(err, auth.ErrInvalidCredentials) {
		return true, nil
	}
	return false, err
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

// resolveRoot returns the user whose password holds username, creating a root user
// when no password holds it.
func (s *Service) resolveRoot(
	ctx context.Context,
	tx gorp.Tx,
	passwords *password.Service,
	roots []User,
	username string,
) (User, error) {
	var p password.Password
	err := passwords.NewRetrieve().
		Where(password.MatchUsernames(username)).
		Entry(&p).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		created, err := s.NewWriter(tx).create(ctx, User{
			Name:     username,
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
	for _, r := range roots {
		if r.OntologyID() == p.Subject {
			return r, nil
		}
	}
	if p.Subject.Type != ontology.ResourceTypeUser {
		return User{}, errors.Newf("password %q is not held by a user", username)
	}
	key, err := KeyFromOntologyID(p.Subject)
	if err != nil {
		return User{}, err
	}
	var existing User
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&existing).
		Exec(ctx, tx); err != nil {
		return User{}, errors.Wrapf(err, "look up user holding %q", username)
	}
	if err := s.NewWriter(tx).setRootUser(ctx, existing.Key, true); err != nil {
		return User{}, errors.Wrapf(err, "promote existing user %q to root", username)
	}
	existing.RootUser = true
	s.cfg.L.Info("promoted existing user to root", zap.String("username", username))
	return existing, nil
}

func (s *Service) demoteRoot(ctx context.Context, tx gorp.Tx, u User) error {
	if err := s.NewWriter(tx).setRootUser(ctx, u.Key, false); err != nil {
		return errors.Wrapf(err, "demote root user %q", u.Name)
	}
	s.cfg.L.Warn("demoted root user", zap.String("name", u.Name))
	return nil
}
