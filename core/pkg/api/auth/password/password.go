// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the password API service.
type ServiceConfig struct {
	// Auth checks that the Core accepts passwords and ends the sessions they opened.
	//
	// [REQUIRED]
	Auth *svcauth.Service
	// Password stores passwords and checks proofs of them.
	//
	// [REQUIRED]
	Password *password.Service
	// Ontology resolves the users that passwords belong to.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the password API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.Password = override.Nil(c.Password, other.Password)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.password")
	v.NotNil("auth", c.Auth)
	v.NotNil("password", c.Password)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service creates, changes, renames, and lists passwords.
type Service struct {
	auth     *svcauth.Service
	password *password.Service
	ontology *ontology.Ontology
	rbac     *rbac.Service
}

// NewService opens a new password API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		auth:     cfg.Auth,
		password: cfg.Password,
		ontology: cfg.Ontology,
		rbac:     cfg.RBAC,
	}, nil
}

// ChangeRequest changes the password of the user with the given credentials.
type ChangeRequest struct {
	svcauth.Credentials
	// NewPassword is the password to change to.
	NewPassword string `json:"new_password" msgpack:"new_password"`
}

// Change changes the password for the user with the provided credentials and ends
// every other session that password opened.
func (s *Service) Change(
	ctx context.Context,
	tx gorp.Tx,
	req ChangeRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(svcauth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	identity, err := s.password.Authenticate(ctx, proofOf(req.Credentials))
	if err != nil {
		return struct{}{}, err
	}
	if err = s.password.NewWriter(tx).
		ChangePassword(ctx, identity.Subject, req.NewPassword); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(
		ctx,
		[]svcauth.SessionKey{auth.GetSession(ctx).Key},
		identity.Credential,
	)
}

func proofOf(creds svcauth.Credentials) svcauth.Proof {
	return svcauth.Proof{Variant: svcauth.PasswordProof{Credentials: creds}}
}

// CreateRequest gives a user its first password.
type CreateRequest struct {
	// Subject is the user to give the password. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	svcauth.Credentials
}

// Create gives the caller, or another user when the caller may create that user, its
// first password. It returns [svcauth.ErrRepeatedUsername] when another user holds the
// username, a validation error when the password method is off, the subject is not a
// user, or the user already has a password, and an error wrapping query.ErrNotFound
// when the user does not exist.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(svcauth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	caller := auth.GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
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

// SetRequest sets a user's password.
type SetRequest struct {
	// Subject is the user whose password to set. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Password is the new password.
	Password string `json:"password" msgpack:"password"`
	// CurrentPassword is required when the caller changes its own password.
	CurrentPassword string `json:"current_password" msgpack:"current_password"`
}

// Set sets the password of the caller, or of another user when the caller may update
// that user, and ends every other session the old password opened. A caller setting its
// own password must send the current one. It returns [svcauth.ErrInvalidCredentials]
// for a wrong current password, a validation error when the password method is off or
// the subject is not a user, and an error wrapping query.ErrNotFound when the user has
// no password.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(svcauth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	caller := auth.GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s has no password",
			subject.Type,
		)
	}
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
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
	var keep []svcauth.SessionKey
	if subject == caller {
		if _, err := s.password.Authenticate(ctx, proofOf(svcauth.Credentials{
			Username: current.Username,
			Password: req.CurrentPassword,
		})); err != nil {
			return struct{}{}, err
		}
		keep = []svcauth.SessionKey{auth.GetSession(ctx).Key}
	}
	if err := s.password.NewWriter(tx).
		ChangePassword(ctx, subject, req.Password); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, keep, current.Key)
}

// RetrieveRequest selects passwords to list. Empty fields mean the caller's.
type RetrieveRequest struct {
	// Subjects limits the result to these subjects' passwords.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	// Usernames limits the result to the passwords with these usernames.
	Usernames []string `json:"usernames" msgpack:"usernames"`
}

// RetrieveResponse lists passwords without their hashes.
type RetrieveResponse struct {
	// Passwords are the passwords that matched.
	Passwords []Password `json:"passwords" msgpack:"passwords"`
}

// Retrieve lists the caller's password, or the passwords of the given subjects or
// usernames when the caller may retrieve their subjects. Unknown subjects and usernames
// are ignored.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	caller := auth.GetSubject(ctx)
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
		return RetrieveResponse{}, err
	}
	res := lo.Map(passwords, func(p password.Password, _ int) Password {
		return Password{Subject: p.Subject, Username: p.Username}
	})
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		caller,
		access.ActionRetrieve,
		lo.Map(res, func(p Password, _ int) ontology.ID { return p.Subject }),
	); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Passwords: res}, nil
}

// RenameRequest changes the username of a password.
type RenameRequest struct {
	// Subject is the user whose password to rename. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Username is the new username.
	Username string `json:"username" msgpack:"username"`
}

// Rename changes the username of the caller's password, or of another user's when the
// caller may update that user. Sessions stay open. It returns
// [svcauth.ErrRepeatedUsername] when another user holds the username, a validation
// error when the password method is off or the username is empty, and an error wrapping
// query.ErrNotFound when the user has no password.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(svcauth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	if req.Username == "" {
		return struct{}{}, errors.Wrap(validate.ErrValidation, "username: required")
	}
	caller := auth.GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		caller,
		access.ActionUpdate,
		[]ontology.ID{subject},
	); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.password.NewWriter(tx).Rename(ctx, subject, req.Username)
}
