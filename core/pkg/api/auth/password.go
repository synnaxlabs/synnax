// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// PasswordSetRequest sets a user's password.
type PasswordSetRequest struct {
	// Subject is the user whose password to set. Empty means the caller.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Password is the new password.
	Password string `json:"password" msgpack:"password"`
	// CurrentPassword is required when the caller changes its own password.
	CurrentPassword string `json:"current_password" msgpack:"current_password"`
	// Cookie asks for the new token in a cookie, as [SessionCreateRequest] does.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// PasswordSetResponse is the session that replaces the caller's, when the password
// change ended it.
type PasswordSetResponse struct {
	// Token is empty when Session is nil or the token went to a cookie.
	Token string `json:"token" msgpack:"token"`
	// Session is the new session, or nil when the caller's session did not end.
	Session *auth.Session `json:"session,omitempty" msgpack:"session,omitempty"`
}

// PrepareSetPassword hashes the new password of req. It first checks the current
// password that req carries when the caller sets its own, and update on the user when
// the caller sets another's. It returns [auth.ErrInvalidCredentials] for a wrong
// current password, access.ErrDenied when the caller cannot update the user, a
// validation error when the subject is not a user, and an error wrapping
// query.ErrNotFound when the caller has no password.
func (s *Service) PrepareSetPassword(
	ctx context.Context,
	req PasswordSetRequest,
) (password.Hash, error) {
	caller := GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != resource.TypeUser {
		return password.Hash{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	if subject != caller {
		// Each hash is slow, so a caller that cannot update the user costs none.
		if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			return s.access.NewEnforcer(snap).Enforce(ctx, access.Request{
				Subject: caller,
				Action:  access.ActionUpdate,
				Objects: []resource.ID{subject},
			})
		}); err != nil {
			return password.Hash{}, err
		}
		return s.password.Hash(req.Password)
	}
	var current password.Password
	if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.password.NewRetrieve().
			Where(password.MatchSubjects(subject)).
			Entry(&current).
			Exec(ctx, snap)
	}); err != nil {
		return password.Hash{}, err
	}
	proof := auth.PasswordProof{
		Username: current.Username,
		Password: req.CurrentPassword,
	}
	return s.password.HashProven(ctx, proof, req.Password)
}

// SetPassword stores the password that PrepareSetPassword hashed for the caller or
// another user, and ends every session the old password opened. When the change ends
// the caller's session, SetPassword opens a new one on the new password and returns
// it. It returns [auth.ErrInvalidCredentials] when the caller's password changed after
// its proof, and an error wrapping query.ErrNotFound when the user has no password.
func (s *Service) SetPassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordSetRequest,
	hash password.Hash,
) (PasswordSetResponse, error) {
	subject := lo.Ternary(req.Subject.IsZero(), GetSubject(ctx), req.Subject)
	var current password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Entry(&current).
		Exec(ctx, tx); err != nil {
		return PasswordSetResponse{}, err
	}
	if err := s.password.NewWriter(tx).
		Apply(ctx, subject, password.NewSetHashAction(hash)); err != nil {
		return PasswordSetResponse{}, err
	}
	w := s.auth.NewWriter(tx)
	if err := w.RevokeCredential(ctx, current.Key); err != nil {
		return PasswordSetResponse{}, err
	}
	if getSession(ctx).Credential != current.Key {
		return PasswordSetResponse{}, nil
	}
	session, token, err := w.Open(ctx, subject, current.Key)
	if err != nil {
		return PasswordSetResponse{}, err
	}
	return PasswordSetResponse{
		Token:   s.deliver(ctx, token, req.Cookie),
		Session: &session,
	}, nil
}
