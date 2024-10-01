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
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// AuthService is the core authentication service for the delta API.
type AuthService struct {
	authProvider
	userProvider
}

func NewAuthService(p Provider) *AuthService {
	return &AuthService{
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
