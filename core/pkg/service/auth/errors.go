// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrAuth is the base error for all authentication related errors.
	ErrAuth = errors.New("auth error")
	// ErrInvalidCredentials is returned when the supplied credentials do not
	// match a registered entity.
	ErrInvalidCredentials = errors.Wrap(ErrAuth, "invalid credentials")
	// ErrRepeatedUsername is returned when a username is already taken.
	ErrRepeatedUsername = errors.Wrap(ErrAuth, "username already exists")
	// ErrInvalidToken is returned when a bearer token fails validation.
	ErrInvalidToken = errors.Wrap(ErrAuth, "invalid token")
	// ErrExpiredToken is returned when a bearer token has expired.
	ErrExpiredToken = errors.Wrap(ErrAuth, "expired token")
	// ErrAccessDenied is returned when a subject does not have permission to
	// perform an action on a resource.
	ErrAccessDenied = errors.Wrap(ErrAuth, "access denied")
)

const (
	errorType              = "sy.auth"
	invalidCredentialsType = errorType + ".invalid-credentials"
	invalidTokenType       = errorType + ".invalid_token"
	expiredTokenType       = errorType + ".expired_token"
	repeatedUsernameType   = errorType + ".repeated-username"
	accessDeniedType       = errorType + ".access_denied"
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, ErrInvalidToken) {
		return errors.Payload{Type: invalidTokenType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrInvalidCredentials) {
		return errors.Payload{Type: invalidCredentialsType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrExpiredToken) {
		return errors.Payload{Type: expiredTokenType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrRepeatedUsername) {
		return errors.Payload{Type: repeatedUsernameType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrAccessDenied) {
		return errors.Payload{Type: accessDeniedType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrAuth) {
		return errors.Payload{Type: errorType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	switch p.Type {
	case invalidCredentialsType:
		return errors.Wrap(ErrInvalidCredentials, p.Data), true
	case invalidTokenType:
		return errors.Wrap(ErrInvalidToken, p.Data), true
	case repeatedUsernameType:
		return errors.Wrap(ErrRepeatedUsername, p.Data), true
	case expiredTokenType:
		return errors.Wrap(ErrExpiredToken, p.Data), true
	case accessDeniedType:
		return errors.Wrap(ErrAccessDenied, p.Data), true
	}
	if strings.HasPrefix(p.Type, errorType) {
		return errors.Wrap(ErrAuth, p.Data), true
	}
	return nil, false
}

func init() { errors.Register(encode, decode) }
