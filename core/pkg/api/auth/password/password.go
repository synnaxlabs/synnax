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
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the password API service.
type ServiceConfig struct {
	// DB opens the snapshots that access checks and password lookups read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Auth reports whether the Core accepts the password method and ends the sessions
	// of replaced passwords.
	//
	// [REQUIRED]
	Auth *auth.Service
	// Password stores passwords and checks proofs of them.
	//
	// [REQUIRED]
	Password *password.Service
	// User checks that a user exists before it gets its first password.
	//
	// [REQUIRED]
	User *user.Service
	// RBAC checks access on the user whose password another caller creates or sets.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.Password = override.Nil(c.Password, other.Password)
	c.User = override.Nil(c.User, other.User)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.password")
	v.NotNil("db", c.DB)
	v.NotNil("auth", c.Auth)
	v.NotNil("password", c.Password)
	v.NotNil("user", c.User)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service creates, sets, changes, lists, and applies actions to passwords.
type Service struct {
	db       *gorp.DB
	auth     *auth.Service
	password *password.Service
	user     *user.Service
	access   *rbac.Service
}

// NewService opens a new password API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		auth:     cfg.Auth,
		password: cfg.Password,
		user:     cfg.User,
		access:   cfg.RBAC,
	}, nil
}

// ChangeRequest replaces the password that its credentials prove.
type ChangeRequest struct {
	auth.Credentials
	// NewPassword is the password that replaces the proven one.
	NewPassword string `json:"new_password" msgpack:"new_password"`
}

// PrepareChange checks the credentials of req and hashes its new password. It returns
// [auth.ErrInvalidCredentials] when the credentials match no password, and a validation
// error when the password method is off or the new password is empty.
func (s *Service) PrepareChange(
	ctx context.Context,
	req ChangeRequest,
) (password.Hash, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return password.Hash{}, err
	}
	return s.password.HashProven(
		ctx,
		auth.PasswordProof{Credentials: req.Credentials},
		req.NewPassword,
	)
}

// Change stores the password that PrepareChange hashed and ends every session the old
// password opened, the caller's included. It returns [auth.ErrInvalidCredentials] when
// the password changed after its proof.
func (s *Service) Change(
	ctx context.Context,
	tx gorp.Tx,
	_ ChangeRequest,
	hash password.Hash,
) (struct{}, error) {
	identity, _ := hash.Proven()
	if err := s.password.NewWriter(tx).
		SetHash(ctx, identity.Subject, hash); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, identity.Credential)
}

// CreateRequest gives a user its first password.
type CreateRequest struct {
	// Subject is the user to give the password. Empty means the caller.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Credentials is the username and password to create for the user.
	auth.Credentials
}

// subjectOf returns the user that req names, or the caller, in canonical form.
func subjectOf(ctx context.Context, req CreateRequest) (resource.ID, error) {
	subject := lo.Ternary(req.Subject.IsZero(), apiauth.GetSubject(ctx), req.Subject)
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

// PrepareCreate hashes the password that Create stores. It returns a validation error
// when the password method is off or the subject is not a user, and an error wrapping
// access.ErrDenied when the caller cannot create and update another user.
func (s *Service) PrepareCreate(
	ctx context.Context,
	req CreateRequest,
) (password.Hash, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return password.Hash{}, err
	}
	subject, err := subjectOf(ctx, req)
	if err != nil {
		return password.Hash{}, err
	}
	if caller := apiauth.GetSubject(ctx); subject != caller {
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

// Create gives the caller, or another user, its first password. It returns
// [auth.ErrRepeatedUsername] when another user holds the username, a validation error
// when the user already has a password, and an error wrapping query.ErrNotFound when
// the user does not exist.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
	hash password.Hash,
) (struct{}, error) {
	subject, err := subjectOf(ctx, req)
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

// SetRequest sets a user's password.
type SetRequest struct {
	// Subject is the user whose password to set. Empty means the caller.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Password is the new password.
	Password string `json:"password" msgpack:"password"`
	// CurrentPassword is required when the caller changes its own password.
	CurrentPassword string `json:"current_password" msgpack:"current_password"`
	// Cookie asks for the new token in a cookie, as a session create request does.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// SetResponse is the session that replaces the caller's, when the password change
// ended it.
type SetResponse struct {
	// Token is empty when Session is nil or the token went to a cookie.
	Token string `json:"token" msgpack:"token"`
	// Session is the new session, or nil when the caller's session did not end.
	Session *auth.Session `json:"session,omitempty" msgpack:"session,omitempty"`
}

// PrepareSet hashes the new password of req. It first checks the current password that
// req carries when the caller sets its own, and update on the user when the caller sets
// another's. It returns [auth.ErrInvalidCredentials] for a wrong current password,
// access.ErrDenied when the caller cannot update the user, a validation error when the
// password method is off or the subject is not a user, and an error wrapping
// query.ErrNotFound when the caller has no password.
func (s *Service) PrepareSet(
	ctx context.Context,
	req SetRequest,
) (password.Hash, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return password.Hash{}, err
	}
	caller := apiauth.GetSubject(ctx)
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
		// binder checks delete on the user when Set ends its sessions.
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

// Set stores the password that PrepareSet hashed for the caller or another user, and
// ends every session the old password opened. When the change ends the caller's
// session, Set opens a new one on the new password and returns it. Setting another
// user's password needs update on that user, and delete on it when the change ends its
// sessions. It returns [auth.ErrInvalidCredentials] when the caller's password changed
// after its proof, and an error wrapping query.ErrNotFound when the user has no
// password.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
	hash password.Hash,
) (SetResponse, error) {
	subject := lo.Ternary(req.Subject.IsZero(), apiauth.GetSubject(ctx), req.Subject)
	var current password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Entry(&current).
		Exec(ctx, tx); err != nil {
		return SetResponse{}, err
	}
	if err := s.password.NewWriter(tx).
		SetHash(ctx, subject, hash); err != nil {
		return SetResponse{}, err
	}
	w := s.auth.NewWriter(tx)
	if err := w.RevokeCredential(ctx, current.Key); err != nil {
		return SetResponse{}, err
	}
	if apiauth.GetSession(ctx).Credential != current.Key {
		return SetResponse{}, nil
	}
	session, token, err := w.Open(ctx, subject, current.Key)
	if err != nil {
		return SetResponse{}, err
	}
	return SetResponse{
		Token:   apiauth.Deliver(ctx, s.auth, token, req.Cookie),
		Session: &session,
	}, nil
}

// Password is a password as the API returns it: its key, subject, and username,
// without its hash.
type Password struct {
	// Key is the unique identifier of the password.
	Key password.Key `json:"key" msgpack:"key"`
	// Subject is the user the password logs in as.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Username is the name the password logs in with.
	Username string `json:"username" msgpack:"username"`
}

// RetrieveRequest selects passwords to list. Empty fields mean the caller's.
type RetrieveRequest struct {
	// Subjects limits the result to these subjects' passwords.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
	// Usernames limits the result to the passwords with these usernames.
	Usernames []string `json:"usernames" msgpack:"usernames"`
}

// Holders implements bind.Held.
func (r RetrieveRequest) Holders() []resource.ID { return r.Subjects }

// RetrieveResponse lists passwords without their hashes.
type RetrieveResponse struct {
	// Passwords are the passwords that matched.
	Passwords []Password `json:"passwords" msgpack:"passwords"`
}

// Retrieve lists the caller's password in snap, or the passwords of the given subjects
// or usernames. It leaves out the passwords of other users for which allowed returns
// false, and ignores unknown subjects and usernames.
func (s *Service) Retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	allowed func(resource.ID) bool,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	caller := apiauth.GetSubject(ctx)
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
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Passwords: lo.FilterMap(
		passwords,
		func(p password.Password, _ int) (Password, bool) {
			return Password{Key: p.Key, Subject: p.Subject, Username: p.Username},
				p.Subject == caller || allowed(p.Subject)
		},
	)}, nil
}

// DispatchRequest carries actions to apply to one password, keyed by the password's
// key.
type DispatchRequest = actions.DispatchRequest[password.Key, password.Action]

// Dispatch applies the actions of req to a password in order. Sessions stay open. It
// returns access.ErrDenied when the caller cannot update the password's user,
// [auth.ErrRepeatedUsername] when another user holds the username that the actions
// set, a validation error when the password method is off or an action sets an empty
// username, and an error wrapping query.ErrNotFound when no password has the key.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return struct{}{}, err
	}
	var p password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchKeys(req.Key)).
		Entry(&p).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	// The binder lets a user change the passwords it holds, but the Core makes root
	// whichever user holds the root username, so renaming even one's own needs update.
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: apiauth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{p.Subject},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.password.NewWriter(tx).Apply(ctx, req.Key, req.Actions...)
}
