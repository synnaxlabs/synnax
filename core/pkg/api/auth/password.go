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
}

// SetPassword sets the password of the caller, or of another user when the caller may
// update that user, and ends every other session the old password opened. A caller
// setting its own password must send the current one. It returns
// [auth.ErrInvalidCredentials] for a wrong current password, a validation error when
// the password method is off or the subject is not a user, and an error wrapping
// query.ErrNotFound when the user has no password.
func (s *Service) SetPassword(
	ctx context.Context,
	tx gorp.Tx,
	req PasswordSetRequest,
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
		access.ActionUpdate,
		[]ontology.ID{subject},
	); err != nil {
		return struct{}{}, err
	}
	var current password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Entry(&current).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	var keep []auth.SessionKey
	if subject == caller {
		if _, err := s.password.Authenticate(ctx, proofOf(Credentials{
			Username: current.Username,
			Password: req.CurrentPassword,
		})); err != nil {
			return struct{}{}, err
		}
		keep = []auth.SessionKey{getSession(ctx).Key}
	}
	if err := s.password.NewWriter(tx).
		ChangePassword(ctx, subject, req.Password); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, keep, current.Key)
}

// Password is a password as the API returns it: its subject and username, without its
// hash.
type Password struct {
	// Subject is the user the password logs in as.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Username is the name the password logs in with.
	Username string `json:"username" msgpack:"username"`
}

// PasswordRetrieveRequest selects passwords to list. Empty fields mean the caller's.
type PasswordRetrieveRequest struct {
	// Subjects limits the result to these subjects' passwords.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	// Usernames limits the result to the passwords with these usernames.
	Usernames []string `json:"usernames" msgpack:"usernames"`
}

// PasswordRetrieveResponse lists passwords without their hashes.
type PasswordRetrieveResponse struct {
	// Passwords are the passwords that matched.
	Passwords []Password `json:"passwords" msgpack:"passwords"`
}

// RetrievePasswords lists the caller's password, or the passwords of the given subjects
// or usernames when the caller may retrieve their subjects. Unknown subjects and
// usernames are ignored.
func (s *Service) RetrievePasswords(
	ctx context.Context,
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
	if err := q.Entries(&passwords).Exec(ctx, nil); err != nil {
		return PasswordRetrieveResponse{}, err
	}
	res := lo.Map(passwords, func(p password.Password, _ int) Password {
		return Password{Subject: p.Subject, Username: p.Username}
	})
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionRetrieve,
		lo.Map(res, func(p Password, _ int) ontology.ID { return p.Subject }),
	); err != nil {
		return PasswordRetrieveResponse{}, err
	}
	return PasswordRetrieveResponse{Passwords: res}, nil
}

// PasswordRenameRequest changes the username of a password.
type PasswordRenameRequest struct {
	// Subject is the user whose password to rename. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
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
	caller := GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionUpdate,
		[]ontology.ID{subject},
	); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.password.NewWriter(tx).Rename(ctx, subject, req.Username)
}
