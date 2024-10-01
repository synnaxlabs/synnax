// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/auth/base"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/errors"
	"strings"
)

var (
	// InvalidCredentials is returned when the credentials for a particular entity
	// are invalid.
	InvalidCredentials = password.Invalid
	// Error is the base error for all authentication related errors.
	Error        = base.AuthError
	InvalidToken = errors.Wrap(base.AuthError, "invalid token")
)

const (
	errorType              = "sy.auth"
	invalidCredentialsType = errorType + ".invalid-credentials"
	invalidTokenType       = errorType + ".invalid-token"
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, InvalidToken) {
		return errors.Payload{Type: invalidTokenType, Data: err.Error()}, true
	}
	if errors.Is(err, InvalidCredentials) {
		return errors.Payload{Type: invalidCredentialsType, Data: err.Error()}, true
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
	}
	if strings.HasPrefix(p.Type, errorType) {
		return errors.Wrap(base.AuthError, p.Data), true
	}
	return nil, false
}

func init() {
	errors.Register(encode, decode)
}
