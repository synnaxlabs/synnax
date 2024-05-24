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

	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/gorp"
)

// AuthService is the core authentication service for the delta API.
type AuthService struct {
	dbProvider
	authProvider
	userProvider
}

func NewAuthServer(p Provider) *AuthService {
	return &AuthService{
		dbProvider:   p.db,
		authProvider: p.auth,
		userProvider: p.user,
	}
}

// Login attempts to authenticate a user with the provided credentials. If successful,
// returns a response containing a valid JWT along with the user's details.
func (s *AuthService) Login(ctx context.Context, cred auth.InsecureCredentials) (tr TokenResponse, err error) {
	if err = s.authenticator.Authenticate(ctx, cred); err != nil {
		return
	}
	u, err := s.user.RetrieveByUsername(ctx, cred.Username)
	if err != nil {
		return tr, err
	}
	return s.tokenResponse(u)
}

// RegistrationRequest is an API request to register a new user.
type RegistrationRequest struct {
	auth.InsecureCredentials
}

// Register registers new user with the provided credentials. If successful, returns a
// response containing a valid JWT along with the user's details.
func (s *AuthService) Register(ctx context.Context, req RegistrationRequest) (tr TokenResponse, err error) {
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
func (s *AuthService) ChangePassword(ctx context.Context, cpr ChangePasswordRequest) (types.Nil, error) {
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
func (s *AuthService) ChangeUsername(ctx context.Context, cur ChangeUsernameRequest) (types.Nil, error) {
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

// TokenResponse is a response containing a valid JWT along with details about the user
// the token is associated with.
type TokenResponse struct {
	// User is the user the token is associated with.
	User user.User `json:"user"`
	// Token is the JWT.
	Token string `json:"token"`
}

func (s *AuthService) tokenResponse(u user.User) (TokenResponse, error) {
	tk, err := s.token.New(u.Key)
	return TokenResponse{User: u, Token: tk}, err
}
