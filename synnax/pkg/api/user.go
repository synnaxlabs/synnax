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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"go/types"

	"github.com/synnaxlabs/x/gorp"
)

// UserService is the core authentication service for the delta API.
type UserService struct {
	dbProvider
	authProvider
	userProvider
	accessProvider
}

func NewUserService(p Provider) *UserService {
	return &UserService{
		dbProvider:     p.db,
		authProvider:   p.auth,
		userProvider:   p.user,
		accessProvider: p.access,
	}
}

// RegistrationRequest is an API request to register a new user.
type RegistrationRequest struct {
	auth.InsecureCredentials
}

// Register registers a new user with the provided credentials. If successful, returns a
// response containing a valid JWT along with the user's details.
func (s *UserService) Register(ctx context.Context, req RegistrationRequest) (tr TokenResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  "register",
		Objects: []ontology.ID{},
	}); err != nil {
		return TokenResponse{}, err
	}
	return tr, s.WithTx(ctx, func(txn gorp.Tx) error {
		if err := s.authenticator.NewWriter(txn).Register(ctx, req.InsecureCredentials); err != nil {
			return err
		}
		u := &user.User{Username: req.Username}
		if err := s.user.NewWriter(txn).Create(ctx, u); err != nil {
			return err
		}
		tr, err = s.tokenResponse(*u)
		return err
	})
}

// ChangePasswordRequest is an API request to change the password for a user.
type ChangePasswordRequest struct {
	auth.InsecureCredentials
	NewPassword password.Raw `json:"new_password" msgpack:"new_password" validate:"required"`
}

// ChangePassword changes the password for the user with the provided credentials.
func (s *UserService) ChangePassword(ctx context.Context, cpr ChangePasswordRequest) (types.Nil, error) {
	u, err := s.user.RetrieveByUsername(ctx, cpr.Username)
	if err != nil {
		return types.Nil{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  "change_password",
		Objects: []ontology.ID{user.OntologyID(u.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(txn gorp.Tx) error {
		return s.authenticator.NewWriter(txn).
			UpdatePassword(ctx, cpr.InsecureCredentials, cpr.NewPassword)
	})
}

// ChangeUsernameRequest is an API request to change the username for a user.
type ChangeUsernameRequest struct {
	auth.InsecureCredentials `json:"" msgpack:""`
	NewUsername              string `json:"new_username" msgpack:"new_username" validate:"required"`
}

// ChangeUsername changes the username for the user with the provided credentials.
func (s *UserService) ChangeUsername(ctx context.Context, cur ChangeUsernameRequest) (types.Nil, error) {
	u, err := s.user.RetrieveByUsername(ctx, cur.Username)
	if err != nil {
		return types.Nil{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  "change_username",
		Objects: []ontology.ID{user.OntologyID(u.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(txn gorp.Tx) error {
		u, err := s.user.RetrieveByUsername(ctx, cur.InsecureCredentials.Username)
		if err != nil {
			return err
		}
		if err := s.authenticator.NewWriter(txn).UpdateUsername(
			ctx,
			cur.InsecureCredentials,
			cur.NewUsername,
		); err != nil {
			return err
		}
		u.Username = cur.NewUsername
		return s.user.NewWriter(txn).Update(ctx, u)
	})
}

func (s *UserService) tokenResponse(u user.User) (TokenResponse, error) {
	tk, err := s.token.New(u.Key)
	return TokenResponse{User: u, Token: tk}, err
}
