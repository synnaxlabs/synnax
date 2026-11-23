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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
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
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.User = override.Nil(c.User, other.User)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.user")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("user", c.User)
	return v.Error()
}

// Service is the core authentication service for the Synnax API.
type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *user.Service
}

// NewService creates a new Service that allows for registering, updating, and removing
// users.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		access:   cfg.RBAC,
		internal: cfg.User,
	}, nil
}

type (
	CreateRequest struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
	CreateResponse struct {
		Users []user.User `json:"users" msgpack:"users"`
	}
)

// Create creates users and returns them. A user with a zero key gets a new one. It
// returns a validation error for a root user, a missing role, or a key that already
// exists. A new user has no login method until it is given a password, API key, or
// provider link.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.internal.NewWriter(tx)
	created := make([]user.User, len(req.Users))
	for i, u := range req.Users {
		var err error
		if created[i], err = w.Create(ctx, u); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse{Users: created}, nil
}

// DispatchRequest carries an action sequence to apply to a single user. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[user.Key, user.Action]

// Dispatch applies the action sequence to the target user atomically. It returns
// access.ErrDenied when the subject cannot update the user, checked before the write
// reads the user, and the errors of user.Writer.Apply.
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
