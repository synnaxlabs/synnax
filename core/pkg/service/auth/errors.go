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
	"strings"

	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrAuth is the base error for all authentication related errors.
	ErrAuth = errors.New("auth error")
	// ErrInvalidCredentials is returned when the supplied credentials do not
	// match a registered entity.
	ErrInvalidCredentials = errors.Wrap(ErrAuth, "invalid credentials")
	// ErrMissingCredentials is returned when a request carries no token or cookie.
	ErrMissingCredentials = errors.Wrap(ErrAuth, "missing credentials")
	// ErrRepeatedUsername is returned when a username is already taken.
	ErrRepeatedUsername = errors.Wrap(ErrAuth, "username already exists")
	// ErrInvalidToken is returned when a bearer token fails validation.
	ErrInvalidToken = errors.Wrap(ErrAuth, "invalid token")
	// ErrExpiredToken is returned when a bearer token has expired.
	ErrExpiredToken = errors.Wrap(ErrAuth, "expired token")
	// ErrSessionEnded closes a stream whose session was revoked or timed out while the
	// stream was open.
	ErrSessionEnded = errors.Wrap(ErrInvalidToken, "session ended")
	// ErrAccessDenied is returned when a subject does not have permission to
	// perform an action on a resource.
	ErrAccessDenied = errors.Wrap(ErrAuth, "access denied")
)

const (
	errorType              = "sy.auth"
	invalidCredentialsType = errorType + ".invalid-credentials"
	missingCredentialsType = errorType + ".missing_credentials"
	invalidTokenType       = errorType + ".invalid_token"
	sessionEndedType       = errorType + ".session_ended"
	expiredTokenType       = errorType + ".expired_token"
	repeatedUsernameType   = errorType + ".repeated-username"
	accessDeniedType       = errorType + ".access_denied"
)

// data returns the message of err without ref. A bare sentinel keeps its own message
// so that clients in other languages receive text.
func data(err, ref error) string {
	if msg := errors.MessageWithout(err, ref); msg != "" {
		return msg
	}
	return err.Error()
}

// wrap decodes data into ref. Data equal to the message of ref is the bare sentinel.
func wrap(ref error, data string) error {
	if data == ref.Error() {
		return ref
	}
	return errors.Wrap(ref, data)
}

func encode(_ *errors.Codec, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, ErrSessionEnded) {
		return errors.Payload{
			Type: sessionEndedType,
			Data: data(err, ErrSessionEnded),
		}, true
	}
	if errors.CheapIs(err, ErrInvalidToken) {
		return errors.Payload{
			Type: invalidTokenType,
			Data: data(err, ErrInvalidToken),
		}, true
	}
	if errors.CheapIs(err, ErrInvalidCredentials) {
		return errors.Payload{
			Type: invalidCredentialsType,
			Data: data(err, ErrInvalidCredentials),
		}, true
	}
	if errors.CheapIs(err, ErrMissingCredentials) {
		return errors.Payload{
			Type: missingCredentialsType,
			Data: data(err, ErrMissingCredentials),
		}, true
	}
	if errors.CheapIs(err, ErrExpiredToken) {
		return errors.Payload{
			Type: expiredTokenType,
			Data: data(err, ErrExpiredToken),
		}, true
	}
	if errors.CheapIs(err, ErrRepeatedUsername) {
		return errors.Payload{
			Type: repeatedUsernameType,
			Data: data(err, ErrRepeatedUsername),
		}, true
	}
	if errors.CheapIs(err, ErrAccessDenied) {
		return errors.Payload{
			Type: accessDeniedType,
			Data: data(err, ErrAccessDenied),
		}, true
	}
	if errors.CheapIs(err, ErrAuth) {
		return errors.Payload{Type: errorType, Data: data(err, ErrAuth)}, true
	}
	return errors.Payload{}, false
}

func decode(_ *errors.Codec, p errors.Payload) (error, bool) {
	switch p.Type {
	case invalidCredentialsType:
		return wrap(ErrInvalidCredentials, p.Data), true
	case missingCredentialsType:
		return wrap(ErrMissingCredentials, p.Data), true
	case invalidTokenType:
		return wrap(ErrInvalidToken, p.Data), true
	case sessionEndedType:
		return wrap(ErrSessionEnded, p.Data), true
	case repeatedUsernameType:
		return wrap(ErrRepeatedUsername, p.Data), true
	case expiredTokenType:
		return wrap(ErrExpiredToken, p.Data), true
	case accessDeniedType:
		return wrap(ErrAccessDenied, p.Data), true
	}
	if strings.HasPrefix(p.Type, errorType) {
		return wrap(ErrAuth, p.Data), true
	}
	return nil, false
}

// ErrorProvider encodes and decodes auth errors for an errors.Codec.
func ErrorProvider() errors.Provider {
	return errors.Provider{Encode: encode, Decode: decode}
}
