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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
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
	// DB opens the snapshots that access checks read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// RBAC checks the user writes that it must check before they run.
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
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the user API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.User = override.Nil(c.User, other.User)
	c.Password = override.Nil(c.Password, other.Password)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.user")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("user", c.User)
	v.NotNil("password", c.Password)
	return v.Error()
}

// Service is the core authentication service for the Synnax API.
type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *user.Service
	password *password.Service
}

// NewService creates a new Service that allows for registering, updating, and removing
// users.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		access:   cfg.RBAC,
		internal: cfg.User,
		password: cfg.Password,
	}, nil
}

// NewUser is the create-request payload for a single user. It bundles the user-record
// fields and the credentials the auth service will store. An empty Name defaults to the
// username.
type NewUser struct {
	svcauth.Credentials
	Name string   `json:"name" msgpack:"name"`
	Key  user.Key `json:"key"  msgpack:"key"`
	Role role.Key `json:"role" msgpack:"role"`
}

type (
	CreateRequest struct {
		Users []NewUser `json:"users" msgpack:"users"`
	}
	CreateResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// PrepareCreate hashes the password of each new user in req. It returns
// access.ErrDenied when the subject cannot create users, and a validation error when a
// password is empty.
func (s *Service) PrepareCreate(
	ctx context.Context,
	req CreateRequest,
) ([]password.Hash, error) {
	// Each hash is slow, so a subject that cannot create users costs none.
	if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.access.NewEnforcer(snap).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionCreate,
			Objects: []resource.ID{{Type: resource.TypeUser}},
		})
	}); err != nil {
		return nil, err
	}
	hashes := make([]password.Hash, len(req.Users))
	for i, nu := range req.Users {
		var err error
		if hashes[i], err = s.password.Hash(nu.Password); err != nil {
			return nil, err
		}
	}
	return hashes, nil
}

// Create creates the new users with the roles in req and the passwords that
// PrepareCreate hashed, and returns them.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
	hashes []password.Hash,
) (CreateResponse, error) {
	passwordW := s.password.NewWriter(tx)
	userW := s.internal.NewWriter(tx)
	newUsers := make([]user.User, len(req.Users))
	for i, nu := range req.Users {
		name := nu.Name
		if name == "" {
			name = nu.Username
		}
		u, err := userW.Create(ctx, user.User{Name: name, Key: nu.Key, Role: nu.Role})
		if err != nil {
			return CreateResponse{}, err
		}
		if err = passwordW.Set(
			ctx,
			u.ResourceID(),
			nu.Username,
			hashes[i],
		); err != nil {
			return CreateResponse{}, err
		}
		newUsers[i] = u
	}
	return CreateResponse{Users: newUsers}, nil
}

// checkOtherUser returns an error when the user with key is the subject, as a subject
// cannot change its own username.
func checkOtherUser(ctx context.Context, key user.Key) error {
	if auth.GetSubject(ctx) != user.ResourceID(key) {
		return nil
	}
	return errors.New("you cannot change your own username through the user service")
}

// DispatchRequest carries an action sequence to apply to a single user. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[user.Key, user.Action]

// Dispatch applies the action sequence to the target user atomically. It returns
// access.ErrDenied when the subject cannot update the user, checked before the write
// reads the user, an error when a set username action targets the subject, and the
// errors of user.Writer.Apply.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.enforceOnUsers(
		ctx,
		access.ActionUpdate,
		[]user.Key{req.Key},
	); err != nil {
		return struct{}{}, err
	}
	if slices.ContainsFunc(req.Actions, func(a user.Action) bool {
		return a.Type == user.ActionTypeSetUsername
	}) {
		if err := checkOtherUser(ctx, req.Key); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the users that the top level of Where selects
// by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, user.ResourceID)
}

// Retrieve returns the users that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Users))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, user.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []user.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the users with the provided keys, with their passwords, API keys, and
// sessions. It returns access.ErrDenied when the subject cannot delete the users, and
// an error when a user is a root user.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.enforceOnUsers(ctx, access.ActionDelete, req.Keys); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// enforceOnUsers checks the action on the users with the provided keys before a write
// reads them, so a subject without access learns nothing from the root-user guard. It
// reads committed policies, so a policy staged in the same transaction grants nothing.
func (s *Service) enforceOnUsers(
	ctx context.Context,
	action access.Action,
	keys []user.Key,
) error {
	return s.access.NewEnforcer(s.db).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  action,
		Objects: user.ResourceIDsFromKeys(keys),
	})
}
