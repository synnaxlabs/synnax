// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/service/auth/base"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrInvalidCredentials is returned when the credentials for a particular entity
	// are invalid.
	ErrInvalidCredentials = password.ErrInvalid

	ErrRepeatedUsername = errors.Wrap(base.ErrAuth, "username already exists")

	// Err is the base error for all authentication related errors.
	Err             = base.ErrAuth
	ErrInvalidToken = errors.Wrap(base.ErrAuth, "invalid token")
	ErrExpiredToken = errors.Wrap(base.ErrAuth, "expired token")
)

const (
	errorType              = "sy.auth"
	invalidCredentialsType = errorType + ".invalid-credentials"
	invalidTokenType       = errorType + ".invalid_token"
	expiredTokenType       = errorType + ".expired_token"
	repeatedUsernameType   = errorType + ".repeated-username"
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, ErrInvalidToken) {
		return errors.Payload{Type: invalidTokenType, Data: err.Error()}, true
	}
	if errors.Is(err, ErrInvalidCredentials) {
		return errors.Payload{Type: invalidCredentialsType, Data: err.Error()}, true
	}
	if errors.Is(err, ErrExpiredToken) {
		return errors.Payload{Type: expiredTokenType, Data: err.Error()}, true
	}
	if errors.Is(err, ErrRepeatedUsername) {
		return errors.Payload{Type: repeatedUsernameType, Data: err.Error()}, true
	}
	if errors.Is(err, Err) {
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
	}
	if strings.HasPrefix(p.Type, errorType) {
		return errors.Wrap(base.ErrAuth, p.Data), true
	}
	return nil, false
}

func init() {
	errors.Register(encode, decode)
}
