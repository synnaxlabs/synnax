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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// reconcileRootUser enforces, on every startup, the invariants the cluster's root user
// must satisfy.
//
// When [ServiceConfig.RootCredentials] is set:
//   - At most one user has RootUser=true, and that user's Username matches
//     cfg.RootCredentials.Username.
//   - The auth service has a credentials row for that username whose password
//     matches cfg.RootCredentials.Password. If the existing row has a different
//     password, it is rotated to match config; if no row exists, one is
//     registered. Config is the source of truth for the root password.
//   - Any user records previously flagged RootUser=true that do not match the
//     configured username are demoted (RootUser=false). Their existing role
//     assignments are left untouched.
//   - If a non-root user already owns the configured username, that user is
//     promoted in place: their RootUser flag is set to true and their stored
//     password is rotated to match cfg.RootCredentials.Password. This lets
//     operators turn an existing account into the root user simply by pointing
//     RootCredentials at it.
//
// When [ServiceConfig.RootCredentials] is unset, the reconciler does not touch the auth
// service. It collapses any state where multiple user records have RootUser=true by
// demoting all but the first user returned by the retrieve, emitting a warning.
// Clusters with zero or one root user are left as-is, so callers that do not need a
// root user (e.g. tests) can open the service without supplying credentials.
//
// All work runs inside a single transaction so partial failures roll back. The Owner
// role is assigned separately by the rbac service after this reconciliation completes;
// this service intentionally does not depend on rbac.
func (s *Service) reconcileRootUser(ctx context.Context) error {
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		roots, err := s.loadRootUsers(ctx, tx)
		if err != nil {
			return err
		}
		if s.cfg.RootCredentials.Username == "" {
			return s.collapseStaleRoots(ctx, tx, roots)
		}
		return s.reconcileWithCreds(ctx, tx, roots)
	})
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

// collapseStaleRoots is the no-credentials branch of the reconciler.
//   - Zero or one root: no-op. A cluster with no root is permitted so test setups
//     and other callers that do not need a root user can open the service without
//     supplying credentials.
//   - More than one root: keep the first user returned by the retrieve and
//     demote the rest, emitting a warning.
func (s *Service) collapseStaleRoots(
	ctx context.Context,
	tx gorp.Tx,
	roots []User,
) error {
	if len(roots) <= 1 {
		return nil
	}
	s.cfg.L.Warn(
		"multiple root users detected with no root credentials configured; demoting all but the first",
		zap.String("retained_username", roots[0].Username),
		zap.Int("demoted_count", len(roots)-1),
	)
	for _, r := range roots[1:] {
		if err := s.demoteRoot(ctx, tx, r); err != nil {
			return err
		}
	}
	return nil
}

// reconcileWithCreds is the credentialed branch of the reconciler. It enforces the full
// root-user invariant described on [Service.reconcileRootUser]. The flow is:
//  1. Identify the root user that matches the configured username, promoting an
//     existing non-root user with that username if one exists.
//  2. Sync the stored password to match config (rotating or registering as needed).
//  3. Create the root user record if no existing user matches.
//  4. Demote every other stale root.
func (s *Service) reconcileWithCreds(
	ctx context.Context,
	tx gorp.Tx,
	roots []User,
) error {
	matchingRoot, err := s.resolveMatchingRoot(ctx, tx, roots)
	if err != nil {
		return err
	}
	if err := s.ensureAuthSync(ctx, tx, s.cfg.RootCredentials); err != nil {
		return err
	}
	if matchingRoot.Key == (Key{}) {
		created, err := s.NewWriter(tx).create(ctx, User{
			Username: s.cfg.RootCredentials.Username,
			RootUser: true,
		})
		if err != nil {
			return errors.Wrap(err, "create root user record")
		}
		matchingRoot = created
		s.cfg.L.Info(
			"created root user record",
			zap.String("username", created.Username),
		)
	}
	for _, r := range roots {
		if r.Key == matchingRoot.Key {
			continue
		}
		if err := s.demoteRoot(ctx, tx, r); err != nil {
			return err
		}
	}
	return nil
}

// resolveMatchingRoot returns the user record that should hold RootUser=true after
// reconciliation, applying the rules described on [Service.reconcileRootUser]:
//   - If an existing root user already has the configured username, it is returned
//     unchanged.
//   - If a non-root user has the configured username, it is promoted in place to
//     RootUser=true and returned. The auth-password rotation happens later via
//     [Service.ensureAuthSync].
//   - If no user has the configured username, the zero User is returned and the
//     caller is responsible for creating the record.
func (s *Service) resolveMatchingRoot(
	ctx context.Context,
	tx gorp.Tx,
	roots []User,
) (User, error) {
	for _, r := range roots {
		if r.Username == s.cfg.RootCredentials.Username {
			return r, nil
		}
	}
	var existing User
	err := s.NewRetrieve().
		Where(MatchUsernames(s.cfg.RootCredentials.Username)).
		Entry(&existing).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return User{}, nil
	}
	if err != nil {
		return User{}, errors.Wrap(err, "look up configured root username")
	}
	if err := s.NewWriter(tx).setRootUser(ctx, existing.Key, true); err != nil {
		return User{}, errors.Wrapf(err,
			"promote existing user %q to root", existing.Username)
	}
	existing.RootUser = true
	s.cfg.L.Info(
		"promoted existing user to root",
		zap.String("username", existing.Username),
	)
	return existing, nil
}

// demoteRoot clears the RootUser flag on u. Existing role assignments are left
// untouched. Always logs a warning so operators notice ownership transitions.
func (s *Service) demoteRoot(ctx context.Context, tx gorp.Tx, u User) error {
	if err := s.NewWriter(tx).setRootUser(ctx, u.Key, false); err != nil {
		return errors.Wrapf(err, "demote root user %q", u.Username)
	}
	s.cfg.L.Warn("demoted root user", zap.String("username", u.Username))
	return nil
}

// ensureAuthSync makes the stored credentials for creds.Username match creds.Password.
// If no row exists, one is registered; if a row exists with a different password, the
// password is rotated. A row that already authenticates against creds is a no-op (no
// log). Config is the source of truth for the root password.
func (s *Service) ensureAuthSync(
	ctx context.Context,
	tx gorp.Tx,
	creds auth.Credentials,
) error {
	if err := s.cfg.Auth.Authenticate(ctx, tx, creds); err == nil {
		return nil
	} else if !errors.Is(err, auth.ErrInvalidCredentials) {
		return errors.Wrap(err, "check root credentials")
	}
	w := s.cfg.Auth.NewWriter(tx)
	if err := w.ChangePassword(ctx, creds); err == nil {
		s.cfg.L.Info(
			"rotated root user password to match config",
			zap.String("username", creds.Username),
		)
		return nil
	} else if !errors.Is(err, auth.ErrInvalidCredentials) {
		return errors.Wrap(err, "rotate root credentials")
	}
	if err := w.Register(ctx, creds); err != nil {
		return errors.Wrap(err, "register root credentials")
	}
	s.cfg.L.Info(
		"registered root user credentials",
		zap.String("username", creds.Username),
	)
	return nil
}
