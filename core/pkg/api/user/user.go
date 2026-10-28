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
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the user API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// User stores and retrieves users.
	//
	// [REQUIRED]
	User *user.Service
	// Password stores passwords and checks proofs of them.
	//
	// [REQUIRED]
	Password *password.Service
	// Auth logs subjects in and checks their session tokens.
	//
	// [REQUIRED]
	Auth *svcauth.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the user API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.User = override.Nil(c.User, other.User)
	c.Password = override.Nil(c.Password, other.Password)
	c.Auth = override.Nil(c.Auth, other.Auth)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.user")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("user", c.User)
	v.NotNil("password", c.Password)
	v.NotNil("auth", c.Auth)
	return v.Error()
}

// Service is the core authentication service for the Synnax API.
type Service struct {
	access   *rbac.Service
	internal *user.Service
	password *password.Service
	auth     *svcauth.Service
}

// NewService creates a new Service that allows for registering, updating, and removing
// users.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.User,
		password: cfg.Password,
		auth:     cfg.Auth,
	}, nil
}

// NewUser is the create-request payload for a single user. It bundles the user-record
// fields and the credentials the auth service will store.
type NewUser struct {
	svcauth.Credentials
	FirstName string   `json:"first_name" msgpack:"first_name"`
	LastName  string   `json:"last_name"  msgpack:"last_name"`
	Key       user.Key `json:"key"        msgpack:"key"`
}

type (
	CreateRequest struct {
		Users []NewUser `json:"users" msgpack:"users"`
	}
	CreateResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Create registers the new users with the provided credentials. If successful, Create
// returns a slice of the new users.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeUser}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.auth.Accepts(svcauth.MethodPassword); err != nil {
		return CreateResponse{}, err
	}
	passwordW := s.password.NewWriter(tx)
	userW := s.internal.NewWriter(tx)
	newUsers := make([]user.User, len(req.Users))
	for i, nu := range req.Users {
		u, err := userW.Create(ctx, user.User{
			Username:  nu.Username,
			FirstName: nu.FirstName,
			LastName:  nu.LastName,
			Key:       nu.Key,
		})
		if err != nil {
			return CreateResponse{}, err
		}
		if err := passwordW.Set(ctx, u.OntologyID(), nu.Credentials); err != nil {
			return CreateResponse{}, err
		}
		newUsers[i] = u
	}
	return CreateResponse{Users: newUsers}, nil
}

type ChangeUsernameRequest struct {
	Username string   `json:"username" msgpack:"username"`
	Key      user.Key `json:"key"      msgpack:"key"`
}

// ChangeUsername changes the username for the user with the given key.
func (s *Service) ChangeUsername(
	ctx context.Context,
	tx gorp.Tx,
	req ChangeUsernameRequest,
) (struct{}, error) {
	subject := auth.GetSubject(ctx)
	if subject.Key == req.Key.String() {
		return struct{}{}, errors.New(
			"you cannot change your own username through the user service",
		)
	}
	var u user.User
	if err := s.internal.NewRetrieve().
		Where(user.MatchKeys(req.Key)).Entry(&u).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if u.Username == req.Username {
		return struct{}{}, nil
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	if err := s.internal.NewWriter(tx).
		ChangeUsername(ctx, req.Key, req.Username); err != nil {
		return struct{}{}, err
	}
	// a user who logs in only through a provider has no password to rename
	hasPassword, err := s.password.NewRetrieve().
		Where(password.MatchSubjects(u.OntologyID())).
		Exists(ctx, tx)
	if err != nil || !hasPassword {
		return struct{}{}, err
	}
	return struct{}{}, s.password.NewWriter(tx).
		Rename(ctx, u.OntologyID(), req.Username)
}

type RenameRequest struct {
	FirstName string   `json:"first_name" msgpack:"first_name"`
	LastName  string   `json:"last_name"  msgpack:"last_name"`
	Key       user.Key `json:"key"        msgpack:"key"`
}

// Rename changes the name for the user with the provided key. If either the first or
// last name is empty, the corresponding field will not be updated.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		ChangeName(ctx, req.Key, req.FirstName, req.LastName)
}

type (
	RetrieveRequest struct {
		Keys      []user.Key `json:"keys"      msgpack:"keys"`
		Usernames []string   `json:"usernames" msgpack:"usernames"`
	}
	RetrieveResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Retrieve returns the users with the provided keys or usernames, each with the
// username of its password.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(user.MatchKeys(req.Keys...))
	}
	if len(req.Usernames) > 0 {
		keys, err := s.keysByUsername(ctx, req.Usernames)
		if err != nil {
			return RetrieveResponse{}, err
		}
		if len(keys) == 0 {
			return RetrieveResponse{Users: []user.User{}}, nil
		}
		q = q.Where(user.MatchKeys(keys...))
	}
	var users []user.User
	if err := q.Entries(&users).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: user.OntologyIDsFromUsers(users),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.fillUsernames(ctx, users); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Users: users}, nil
}

func (s *Service) keysByUsername(
	ctx context.Context,
	usernames []string,
) ([]user.Key, error) {
	var passwords []password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchUsernames(usernames...)).
		Entries(&passwords).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	keys := make([]user.Key, 0, len(passwords))
	for _, p := range passwords {
		if p.Subject.Type != ontology.ResourceTypeUser {
			continue
		}
		key, err := user.KeyFromOntologyID(p.Subject)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// fillUsernames sets each user's username from its password, leaving it empty for a
// user with no password.
func (s *Service) fillUsernames(ctx context.Context, users []user.User) error {
	if len(users) == 0 {
		return nil
	}
	var passwords []password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(user.OntologyIDsFromUsers(users)...)).
		Entries(&passwords).
		Exec(ctx, nil); err != nil {
		return err
	}
	usernames := make(map[ontology.ID]string, len(passwords))
	for _, p := range passwords {
		usernames[p.Subject] = p.Username
	}
	for i := range users {
		users[i].Username = usernames[users[i].OntologyID()]
	}
	return nil
}

type DeleteRequest struct {
	Keys []user.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the users with the provided keys from the Synnax cluster.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: user.OntologyIDsFromKeys(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	if err := s.internal.NewWriter(tx).Delete(ctx, req.Keys...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).
		DeleteFor(ctx, user.OntologyIDsFromKeys(req.Keys)...)
}
