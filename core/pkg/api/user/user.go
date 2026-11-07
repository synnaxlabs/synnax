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
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
// fields and the credentials the auth service will store. An empty Name defaults to the
// username.
type NewUser struct {
	svcauth.Credentials
	Name  string     `json:"name"  msgpack:"name"`
	Key   user.Key   `json:"key"   msgpack:"key"`
	Roles []role.Key `json:"roles" msgpack:"roles"`
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
// returns a slice of the new users. Creating a user with roles also requires update on
// users and on each of the roles.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	enforcer := s.access.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeUser}},
	}); err != nil {
		return CreateResponse{}, err
	}
	var roles []role.Key
	for _, u := range req.Users {
		roles = append(roles, u.Roles...)
	}
	if len(roles) > 0 {
		if err := enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  access.ActionUpdate,
			Objects: append(
				[]resource.ID{{Type: resource.TypeUser}},
				role.ResourceIDs(roles)...,
			),
		}); err != nil {
			return CreateResponse{}, err
		}
	}
	passwordW := s.password.NewWriter(tx)
	userW := s.internal.NewWriter(tx)
	newUsers := make([]user.User, len(req.Users))
	for i, nu := range req.Users {
		name := nu.Name
		if name == "" {
			name = nu.Username
		}
		u, err := userW.Create(ctx, user.User{Name: name, Key: nu.Key, Roles: nu.Roles})
		if err != nil {
			return CreateResponse{}, err
		}
		if err := passwordW.Set(ctx, u.ResourceID(), nu.Credentials); err != nil {
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

// ChangeUsername changes the username of the password held by the user with the given
// key.
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
	var p password.Password
	if err := s.password.NewRetrieve().
		Where(password.MatchSubjects(user.ResourceID(req.Key))).
		Entry(&p).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if p.Username == req.Username {
		return struct{}{}, nil
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: []resource.ID{user.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.password.NewWriter(tx).
		Rename(ctx, user.ResourceID(req.Key), req.Username)
}

type RenameRequest struct {
	Name string   `json:"name" msgpack:"name"`
	Key  user.Key `json:"key"  msgpack:"key"`
}

// Rename changes the name of the user with the provided key.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{user.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		ChangeName(ctx, req.Key, req.Name)
}

// Retrieve returns the users that the request selects.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Users))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: user.ResourceIDsFromUsers(res.Users),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
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
		Objects: user.ResourceIDsFromKeys(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	if err := s.internal.NewWriter(tx).Delete(ctx, req.Keys...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).
		DeleteFor(ctx, user.ResourceIDsFromKeys(req.Keys)...)
}

// SetRolesRequest removes the roles in Remove from the users with Keys, then adds the
// roles in Add.
type SetRolesRequest struct {
	Keys   []user.Key `json:"keys"   msgpack:"keys"`
	Add    []role.Key `json:"add"    msgpack:"add"`
	Remove []role.Key `json:"remove" msgpack:"remove"`
}

// SetRoles removes and adds roles on the users with the provided keys in one
// transaction. It returns access.ErrDenied unless the subject can update the users and
// every added and removed role, and a validation error when a user is a root user.
func (s *Service) SetRoles(
	ctx context.Context,
	tx gorp.Tx,
	req SetRolesRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: slices.Concat(
			user.ResourceIDsFromKeys(req.Keys),
			role.ResourceIDs(req.Add),
			role.ResourceIDs(req.Remove),
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		SetRoles(ctx, req.Add, req.Remove, req.Keys...)
}
