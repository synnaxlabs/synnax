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

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

// AuthService is the core authentication service for the Synnax API.
type AuthService struct {
	dbProvider
	authProvider
	userProvider
}

func NewAuthService(p Provider) *AuthService {
	return &AuthService{
		dbProvider:   p.db,
		authProvider: p.auth,
		userProvider: p.user,
	}
}

type AuthLoginResponse struct {
	// User is the user the token is associated with.
	User user.User `json:"user" msgpack:"user"`
	// Token is the JWT.
	Token string `json:"token" msgpack:"token"`
}

type AuthLoginRequest struct {
	auth.InsecureCredentials
}

// Login attempts to authenticate a user with the provided credentials. If successful,
// returns a response containing a valid JWT along with the user's details.
func (s *AuthService) Login(ctx context.Context, req AuthLoginRequest) (AuthLoginResponse, error) {
	if err := s.authenticator.Authenticate(ctx, req.InsecureCredentials); err != nil {
		return AuthLoginResponse{}, err
	}
	var u user.User
	if err := s.user.NewRetrieve().WhereUsernames(req.Username).Entry(&u).Exec(ctx, nil); err != nil {
		return AuthLoginResponse{}, err
	}
	tk, err := s.token.New(u.Key)
	return AuthLoginResponse{User: u, Token: tk}, err
}

type AuthChangePasswordRequest struct {
	auth.InsecureCredentials
	NewPassword password.Raw `json:"new_password" msgpack:"new_password" validate:"required"`
}

// ChangePassword changes the password for the user with the provided credentials.
func (s *AuthService) ChangePassword(ctx context.Context, req AuthChangePasswordRequest) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.authenticator.NewWriter(tx).
			UpdatePassword(ctx, req.InsecureCredentials, req.NewPassword)
	})
}
