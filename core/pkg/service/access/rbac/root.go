// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"go.uber.org/zap"
)

// reconcileRootUser enforces, on every startup, the invariants the cluster's root user
// must satisfy.
//
// When [ServiceConfig.RootCredentials] is set:
//   - At most one user has RootUser=true, and that user's Username matches
//     cfg.RootCredentials.Username.
//   - That user holds the Owner role.
//   - The auth service has a credentials row for that username whose password
//     matches cfg.RootCredentials.Password. If the existing row has a different
//     password, it is rotated to match config; if no row exists, one is
//     registered. Config is the source of truth for the root password.
//   - Any user records previously flagged RootUser=true that do not match the
//     configured username are demoted (RootUser=false). Their existing role
//     assignments are left untouched.
//   - If a non-root user already owns the configured username, startup fails
//     fast so the operator can resolve the conflict manually.
//
// When [ServiceConfig.RootCredentials] is unset, the reconciler does not touch the auth
// service. It collapses any state where multiple user records have RootUser=true by
// demoting all but the first user returned by the retrieve, emitting a warning. A
// cluster with zero root users returns an error, and a cluster with exactly one root
// user is left as-is.
//
// All work runs inside a single transaction so partial failures roll back.
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
func (s *Service) loadRootUsers(ctx context.Context, tx gorp.Tx) ([]user.User, error) {
	var roots []user.User
	if err := s.cfg.User.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entries(&roots).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "load root users")
	}
	return roots, nil
}

// collapseStaleRoots is the no-credentials branch of the reconciler.
//   - Zero roots: returns an error. The cluster would otherwise have no root user,
//     which is unrecoverable without re-bootstrapping. The operator must supply
//     RootCredentials to provision a root user.
//   - One root: no-op.
//   - More than one root: keep the first user returned by the retrieve and
//     demote the rest, emitting a warning.
func (s *Service) collapseStaleRoots(
	ctx context.Context,
	tx gorp.Tx,
	roots []user.User,
) error {
	if len(roots) == 0 {
		return errors.New(
			"rbac: no root user exists and no root credentials are configured; set --username / --password (or the corresponding config fields) to provision a root user",
		)
	}
	if len(roots) == 1 {
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
// root-user invariant described on [Service.reconcileRootUser].
func (s *Service) reconcileWithCreds(
	ctx context.Context,
	tx gorp.Tx,
	roots []user.User,
) error {
	matchingRootIdx := -1
	for i, r := range roots {
		if r.Username == s.cfg.RootCredentials.Username {
			matchingRootIdx = i
			break
		}
	}
	if matchingRootIdx < 0 {
		exists, err := s.cfg.User.NewRetrieve().
			Where(user.MatchUsernames(s.cfg.RootCredentials.Username)).
			Exists(ctx, tx)
		if err != nil {
			return errors.Wrap(err, "look up configured root username")
		}
		if exists {
			return errors.Newf(
				"cannot provision root user: a non-root user with username %q already exists",
				s.cfg.RootCredentials.Username,
			)
		}
	}
	if err := s.ensureAuthSync(ctx, tx, s.cfg.RootCredentials); err != nil {
		return err
	}
	var matchingRoot user.User
	if matchingRootIdx >= 0 {
		matchingRoot = roots[matchingRootIdx]
	} else {
		created, err := s.cfg.User.NewWriter(tx).Create(ctx, user.User{
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
	if err := s.Role.NewWriter(tx, false).
		AssignRole(ctx, user.OntologyID(matchingRoot.Key), s.builtinRoles.OwnerKey); err != nil {
		return errors.Wrapf(err,
			"assign Owner role to root user %q", matchingRoot.Username)
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

// demoteRoot clears the RootUser flag on u. Existing role assignments are left
// untouched. Always logs a warning so operators notice ownership transitions.
func (s *Service) demoteRoot(ctx context.Context, tx gorp.Tx, u user.User) error {
	if err := s.cfg.User.NewWriter(tx).SetRootUser(ctx, u.Key, false); err != nil {
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
	if err := s.cfg.Auth.Authenticate(ctx, creds); err == nil {
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
