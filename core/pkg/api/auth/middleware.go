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
	"uuid"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// Middleware authenticates the session token on every request and puts the session's
// subject, credential, and key on the request for handlers to read.
func Middleware(svc *auth.Service) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		tk, err := tryParseToken(ctx.Params)
		if err != nil {
			return ctx, err
		}
		session, err := svc.Authenticate(ctx, tk)
		if err != nil {
			return ctx, err
		}
		ctx.Set(subjectKey, session.Subject)
		ctx.Set(credentialKey, session.Credential)
		ctx.Set(sessionKey, session.Key)
		return next(ctx)
	})
}

const tokenParamPrefix = "Bearer "

var (
	errInvalidAuthenticationParam = errors.Wrapf(
		auth.ErrAuth,
		`invalid authorization token. Format should be
		'Authorization: %s <token>'`,
		tokenParamPrefix,
	)
	errNoAuthenticationParam = errors.Wrapf(
		auth.ErrAuth,
		"no authentication token provided",
	)
)

func tryParseToken(params freighter.Params) (string, error) {
	tkParam, ok := params.Get(fiber.HeaderAuthorization)
	if !ok {
		// GRPC sends a lowercase header
		tkParam, ok = params.Get(strings.ToLower(fiber.HeaderAuthorization))
		if !ok {
			return "", errNoAuthenticationParam
		}
	}
	tkStr, ok := tkParam.(string)
	if !ok {
		return "", errNoAuthenticationParam
	}
	if !strings.HasPrefix(tkStr, tokenParamPrefix) {
		return "", errInvalidAuthenticationParam
	}
	return strings.TrimPrefix(tkStr, tokenParamPrefix), nil
}

const (
	subjectKey    = "Subject"
	credentialKey = "Credential"
	sessionKey    = "Session"
)

// GetSubject returns the subject of the request's session.
func GetSubject(ctx context.Context) ontology.ID {
	s, ok := freighter.MDFromContext(ctx).Get(subjectKey)
	if !ok {
		zap.S().DPanic("[api] - no subject found in context")
		return user.OntologyID(uuid.Nil())
	}
	return s.(ontology.ID)
}

// GetCredential returns the key of the credential the request's session was opened
// with.
func GetCredential(ctx context.Context) uuid.UUID {
	c, ok := freighter.MDFromContext(ctx).Get(credentialKey)
	if !ok {
		zap.S().DPanic("[api] - no credential found in context")
		return uuid.Nil()
	}
	return c.(uuid.UUID)
}

// GetSession returns the key of the request's session.
func GetSession(ctx context.Context) auth.SessionKey {
	s, ok := freighter.MDFromContext(ctx).Get(sessionKey)
	if !ok {
		zap.S().DPanic("[api] - no session found in context")
		return uuid.Nil()
	}
	return s.(auth.SessionKey)
}
