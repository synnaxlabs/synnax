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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// PasswordCreateRequest gives a user its first password.
type PasswordCreateRequest struct {
	// Subject is the user to give the password. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Credentials is the username and password to create for the user.
	Credentials
}

// CreatePassword gives the caller, or another user when the caller may create that
// user, its first password. It returns [auth.ErrRepeatedUsername] when another user
// holds the username, a validation error when the password method is off, the subject
// is not a user, or the user already has a password, and an error wrapping
// query.ErrNotFound when the user does not exist.
func (s *Service) CreatePassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordCreateRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	caller := GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionCreate,
		[]ontology.ID{subject},
	); err != nil {
		return struct{}{}, err
	}
	exists, err := s.ontology.NewRetrieve().WhereIDs(subject).Exists(ctx, tx)
	if err != nil {
		return struct{}{}, err
	}
	if !exists {
		return struct{}{}, errors.Wrapf(query.ErrNotFound, "%s", subject)
	}
	if exists, err = s.password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Exists(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if exists {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"%s already has a password",
			subject,
		)
	}
	return struct{}{}, s.password.NewWriter(tx).Set(ctx, subject, req.Credentials)
}

// PasswordSetRequest sets a user's password.
type PasswordSetRequest struct {
	// Subject is the user whose password to set. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
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
	// Token is the new session's token. It is empty when Session is nil or the token
	// went to a cookie.
	Token string `json:"token" msgpack:"token"`
	// Session is the new session, or nil when the caller's session did not end.
	Session *auth.Session `json:"session,omitempty" msgpack:"session,omitempty"`
}

// SetPassword sets the password of the caller, or of another user when the caller may
// update that user, and ends every session the old password opened. A caller setting
// its own password must send the current one. When the change ends the caller's
// session, SetPassword opens a new one on the new password and returns it. It returns
// [auth.ErrInvalidCredentials] for a wrong current password, a validation error when
// the password method is off or the subject is not a user, and an error wrapping
// query.ErrNotFound when the user has no password.
func (s *Service) SetPassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordSetRequest,
) (PasswordSetResponse, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return PasswordSetResponse{}, err
	}
	caller := GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser {
		return PasswordSetResponse{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionUpdate,
		[]ontology.ID{subject},
	); err != nil {
		return PasswordSetResponse{}, err
	}
	var current password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Entry(&current).
		Exec(ctx, tx); err != nil {
		return PasswordSetResponse{}, err
	}
	if subject == caller {
		if _, err := s.password.Authenticate(ctx, proofOf(Credentials{
			Username: current.Username,
			Password: req.CurrentPassword,
		})); err != nil {
			return PasswordSetResponse{}, err
		}
	}
	if err := s.password.NewWriter(tx).
		ChangePassword(ctx, subject, req.Password); err != nil {
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
