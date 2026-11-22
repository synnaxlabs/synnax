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
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// PasswordCreateRequest gives a user its first password.
type PasswordCreateRequest struct {
	// Subject is the user to give the password. Empty means the caller.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Credentials is the username and password to create for the user.
	Credentials
}

// passwordSubject returns the user that req names, or the caller, in canonical form.
func passwordSubject(
	ctx context.Context,
	req PasswordCreateRequest,
) (resource.ID, error) {
	subject := lo.Ternary(req.Subject.IsZero(), GetSubject(ctx), req.Subject)
	if subject.Type != resource.TypeUser {
		return resource.ID{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	key, err := user.KeyFromResourceID(subject)
	if err != nil {
		return resource.ID{}, err
	}
	return user.ResourceID(key), nil
}

// PrepareCreatePassword hashes the password that CreatePassword stores. It returns a
// validation error when the password method is off or the subject is not a user, and
// an error wrapping access.ErrDenied when the caller cannot create and update another
// user.
func (s *Service) PrepareCreatePassword(
	ctx context.Context,
	req PasswordCreateRequest,
) (password.Hash, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return password.Hash{}, err
	}
	subject, err := passwordSubject(ctx, req)
	if err != nil {
		return password.Hash{}, err
	}
	if caller := GetSubject(ctx); subject != caller {
		// A first password logs its holder in as the user, so it needs update, as
		// setting one does, and the binder checks create when the password is stored.
		// Each hash is slow, so a denied caller costs none.
		if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			enforcer := s.access.NewEnforcer(snap)
			for _, action := range []access.Action{
				access.ActionCreate,
				access.ActionUpdate,
			} {
				if err := enforcer.Enforce(ctx, access.Request{
					Subject: caller,
					Action:  action,
					Objects: []resource.ID{subject},
				}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return password.Hash{}, err
		}
	}
	return s.password.Hash(req.Password)
}

// CreatePassword gives the caller, or another user, its first password. It returns
// [auth.ErrRepeatedUsername] when another user holds the username, a validation error
// when the user already has a password, and an error wrapping query.ErrNotFound when
// the user does not exist.
func (s *Service) CreatePassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordCreateRequest,
	hash password.Hash,
) (struct{}, error) {
	subject, err := passwordSubject(ctx, req)
	if err != nil {
		return struct{}{}, err
	}
	key, err := user.KeyFromResourceID(subject)
	if err != nil {
		return struct{}{}, err
	}
	exists, err := s.user.NewRetrieve().Where(user.MatchKeys(key)).Exists(ctx, tx)
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
	return struct{}{}, s.password.NewWriter(tx).Set(ctx, subject, req.Username, hash)
}

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
// validation error when the password method is off or the subject is not a user, and
// an error wrapping query.ErrNotFound when the caller has no password.
func (s *Service) PrepareSetPassword(
	ctx context.Context,
	req PasswordSetRequest,
) (password.Hash, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return password.Hash{}, err
	}
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
		// Each hash is slow, so a caller that cannot update the user costs none. The
		// binder checks delete on the user when SetPassword ends its sessions.
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
// it. Setting another user's password needs update on that user, and delete on it when
// the change ends its sessions. It returns [auth.ErrInvalidCredentials] when the
// caller's password changed after its proof, and an error wrapping query.ErrNotFound
// when the user has no password.
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

// Password is a password as the API returns it: its subject and username, without its
// hash.
type Password struct {
	// Subject is the user the password logs in as.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Username is the name the password logs in with.
	Username string `json:"username" msgpack:"username"`
}

// PasswordRetrieveRequest selects passwords to list. Empty fields mean the caller's.
type PasswordRetrieveRequest struct {
	// Subjects limits the result to these subjects' passwords.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
	// Usernames limits the result to the passwords with these usernames.
	Usernames []string `json:"usernames" msgpack:"usernames"`
}

// Holders implements bind.Held.
func (r PasswordRetrieveRequest) Holders() []resource.ID { return r.Subjects }

// PasswordRetrieveResponse lists passwords without their hashes.
type PasswordRetrieveResponse struct {
	// Passwords are the passwords that matched.
	Passwords []Password `json:"passwords" msgpack:"passwords"`
}

// RetrievePasswords lists the caller's password in snap, or the passwords of the given
// subjects or usernames. It leaves out the passwords of other users for which allowed
// returns false, and ignores unknown subjects and usernames.
func (s *Service) RetrievePasswords(
	ctx context.Context,
	snap gorp.Snapshot,
	allowed func(resource.ID) bool,
	req PasswordRetrieveRequest,
) (PasswordRetrieveResponse, error) {
	caller := GetSubject(ctx)
	q := s.password.NewRetrieve()
	if len(req.Subjects) == 0 && len(req.Usernames) == 0 {
		q = q.Where(password.MatchSubjects(caller))
	}
	if len(req.Subjects) > 0 {
		q = q.Where(password.MatchSubjects(req.Subjects...))
	}
	if len(req.Usernames) > 0 {
		q = q.Where(password.MatchUsernames(req.Usernames...))
	}
	var passwords []password.Password
	if err := q.Entries(&passwords).Exec(ctx, snap); err != nil &&
		!errors.Is(err, query.ErrNotFound) {
		return PasswordRetrieveResponse{}, err
	}
	return PasswordRetrieveResponse{Passwords: lo.FilterMap(
		passwords,
		func(p password.Password, _ int) (Password, bool) {
			return Password{Subject: p.Subject, Username: p.Username},
				p.Subject == caller || allowed(p.Subject)
		},
	)}, nil
}

// PasswordRenameRequest changes the username of a password.
type PasswordRenameRequest struct {
	// Subject is the user whose password to rename. Empty means the caller.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Username is the new username.
	Username string `json:"username" msgpack:"username"`
}

// RenamePassword changes the username of the caller's password, or of another user's
// when the caller may update that user. Sessions stay open. It returns
// [auth.ErrRepeatedUsername] when another user holds the username, a validation error
// when the password method is off or the username is empty, and an error wrapping
// query.ErrNotFound when the user has no password.
func (s *Service) RenamePassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordRenameRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	if req.Username == "" {
		return struct{}{}, errors.Wrap(validate.ErrValidation, "username: required")
	}
	subject := lo.Ternary(req.Subject.IsZero(), GetSubject(ctx), req.Subject)
	return struct{}{}, s.password.NewWriter(tx).
		Apply(ctx, subject, password.NewSetUsernameAction(req.Username))
}
