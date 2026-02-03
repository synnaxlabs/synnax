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

	"github.com/synnaxlabs/synnax/pkg/service/auth/base"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"

	"github.com/synnaxlabs/x/errors"
)

var (
	// InvalidCredentials is returned when the credentials for a particular entity
	// are invalid.
	InvalidCredentials = password.ErrInvalid

	RepeatedUsername = errors.Wrap(base.ErrAuth, "username already exists")

	// Error is the base error for all authentication related errors.
	Error        = base.ErrAuth
	InvalidToken = errors.Wrap(base.ErrAuth, "invalid token")
	ExpiredToken = errors.Wrap(base.ErrAuth, "expired token")
)

const (
	errorType              = "sy.auth"
	invalidCredentialsType = errorType + ".invalid-credentials"
	invalidTokenType       = errorType + ".invalid_token"
	expiredTokenType       = errorType + ".expired_token"
	repeatedUsernameType   = errorType + ".repeated-username"
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, InvalidToken) {
		return errors.Payload{Type: invalidTokenType, Data: err.Error()}, true
	}
	if errors.Is(err, InvalidCredentials) {
		return errors.Payload{Type: invalidCredentialsType, Data: err.Error()}, true
	}
	if errors.Is(err, ExpiredToken) {
		return errors.Payload{Type: expiredTokenType, Data: err.Error()}, true
	}
	if errors.Is(err, RepeatedUsername) {
		return errors.Payload{Type: repeatedUsernameType, Data: err.Error()}, true
	}
	if errors.Is(err, Error) {
		return errors.Payload{Type: errorType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	switch p.Type {
	case invalidCredentialsType:
		return errors.Wrap(InvalidCredentials, p.Data), true
	case invalidTokenType:
		return errors.Wrap(InvalidToken, p.Data), true
	case repeatedUsernameType:
		return errors.Wrap(RepeatedUsername, p.Data), true
	case expiredTokenType:
		return errors.Wrap(ExpiredToken, p.Data), true
	}
	if strings.HasPrefix(p.Type, errorType) {
		return errors.Wrap(base.ErrAuth, p.Data), true
	}
	return nil, false
}

func init() {
	errors.Register(encode, decode)
}
