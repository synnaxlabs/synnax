// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/gorp"
)

// UserService is the core authentication service for the delta API.
type UserService struct {
	dbProvider
	authProvider
	userProvider
	accessProvider
	internal *user.Service
}

func NewUserService(p Provider) *UserService {
	return &UserService{
		dbProvider:     p.db,
		authProvider:   p.auth,
		userProvider:   p.user,
		accessProvider: p.access,
		internal:       p.Config.User,
	}
}

// UserRegisterRequest is an API request to register a new user.
type UserRegisterRequest struct {
	auth.InsecureCredentials
	FirstName string `json:"first_name" msgpack:"first_name"`
	LastName  string `json:"last_name" msgpack:"last_name"`
}

// Register registers a new user with the provided credentials. If successful, returns a
// response containing a valid JWT along with the user's details.
func (s *UserService) Register(ctx context.Context, req UserRegisterRequest) (tr TokenResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Action(user.Register),
		Objects: []ontology.ID{},
	}); err != nil {
		return tr, err
	}
	return tr, s.WithTx(ctx, func(txn gorp.Tx) error {
		if err := s.authenticator.NewWriter(txn).Register(ctx, req.InsecureCredentials); err != nil {
			return err
		}
		u := &user.User{Username: req.Username, FirstName: req.FirstName, LastName: req.LastName}
		if err := s.user.NewWriter(txn).Create(ctx, u); err != nil {
			return err
		}
		tr, err = s.tokenResponse(*u)
		return err
	})
}

// UserChangePasswordRequest is an API request to change the password for a user.
type UserChangePasswordRequest struct {
	auth.InsecureCredentials
	NewPassword password.Raw `json:"new_password" msgpack:"new_password" validate:"required"`
}

// ChangePassword changes the password for the user with the provided credentials.
func (s *UserService) ChangePassword(ctx context.Context, req UserChangePasswordRequest) (res types.Nil, err error) {
	u, err := s.user.RetrieveByUsername(ctx, req.Username)
	if err != nil {
		return res, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Action(user.ChangePassword),
		Objects: []ontology.ID{user.OntologyID(u.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(txn gorp.Tx) error {
		return s.authenticator.NewWriter(txn).
			UpdatePassword(ctx, req.InsecureCredentials, req.NewPassword)
	})
}

// UserChangeUserNameRequest is an API request to change the username for a user.
type UserChangeUserNameRequest struct {
	auth.InsecureCredentials `json:"" msgpack:""`
	NewUsername              string `json:"new_username" msgpack:"new_username" validate:"required"`
}

// ChangeUsername changes the username for the user with the provided credentials.
func (s *UserService) ChangeUsername(ctx context.Context, req UserChangeUserNameRequest) (types.Nil, error) {
	u, err := s.user.RetrieveByUsername(ctx, req.Username)
	if err != nil {
		return types.Nil{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Action(user.ChangeUsername),
		Objects: []ontology.ID{user.OntologyID(u.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(txn gorp.Tx) error {
		u, err := s.user.RetrieveByUsername(ctx, req.InsecureCredentials.Username)
		if err != nil {
			return err
		}
		if err := s.authenticator.NewWriter(txn).UpdateUsername(
			ctx,
			req.InsecureCredentials,
			req.NewUsername,
		); err != nil {
			return err
		}
		u.Username = req.NewUsername
		return s.user.NewWriter(txn).Update(ctx, u)
	})
}

type UserChangeNameRequest struct {
	Key       uuid.UUID `json:"key" msgpack:"key"`
	FirstName string    `json:"first_name" msgpack:"first_name"`
	LastName  string    `json:"last_name" msgpack:"last_name"`
}

// ChangeName changes the name for the user with the provided key. If either the first
// or last name is empty, the corresponding field will not be updated.
func (s *UserService) ChangeName(ctx context.Context, req UserChangeNameRequest) (res types.Nil, err error) {
	u, err := s.user.Retrieve(ctx, req.Key)
	if err != nil {
		return
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Update,
		Objects: []ontology.ID{user.OntologyID(req.Key)},
	}); err != nil {
		return
	}
	if req.FirstName != "" {
		u.FirstName = req.FirstName
	}
	if req.LastName != "" {
		u.LastName = req.LastName
	}

	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Update(ctx, u)
	})
}

type UserUnregisterRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

// Unregister removes the user with the provided key from the Synnax cluster.
func (s *UserService) Unregister(ctx context.Context, req UserUnregisterRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Action(user.Unregister),
		Objects: []ontology.ID{},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(txn gorp.Tx) error {
		return s.internal.NewWriter(txn).Delete(ctx, req.Keys...)
	})
}

func (s *UserService) tokenResponse(u user.User) (TokenResponse, error) {
	tk, err := s.token.New(u.Key)
	return TokenResponse{User: u, Token: tk}, err
}
