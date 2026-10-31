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

// Middleware authenticates the session token on every request and puts the session and
// its subject on the request for handlers to read. A unary request gives a session
// cookie it carried a fresh lifetime. A stream holds its session open for as long as
// it runs, and its context is cancelled with [auth.ErrSessionEnded] when the session
// ends, which the middleware then returns.
func Middleware(svc *auth.Service) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		tk, err := tryParseToken(ctx.Params)
		if err != nil {
			return freighter.Context{}, err
		}
		session, err := svc.Authenticate(ctx, tk)
		if err != nil {
			return freighter.Context{}, err
		}
		ctx.Set(subjectKey, session.Subject)
		ctx.Set(sessionKey, session)
		if ctx.Variant != freighter.VariantStream {
			ctx.Set(SessionCookieParam, SessionCookie{Lifetime: svc.IdleTimeout()})
			return next(ctx)
		}
		return holdSession(ctx, next, svc, session.Key)
	})
}

func holdSession(
	ctx freighter.Context,
	next freighter.Next,
	svc *auth.Service,
	key auth.SessionKey,
) (freighter.Context, error) {
	ended, release, err := svc.Hold(ctx, key)
	if err != nil {
		return freighter.Context{}, err
	}
	defer release()
	streamCtx, cancel := context.WithCancelCause(ctx.Context)
	defer cancel(nil)
	go func() {
		select {
		case <-ended:
			cancel(auth.ErrSessionEnded)
		case <-streamCtx.Done():
		}
	}()
	ctx.Context = streamCtx
	oCtx, err := next(ctx)
	if cause := context.Cause(streamCtx); errors.Is(cause, auth.ErrSessionEnded) {
		return freighter.Context{}, cause
	}
	return oCtx, err
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
	subjectKey = "Subject"
	sessionKey = "Session"
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

func getSession(ctx context.Context) auth.Session {
	s, ok := freighter.MDFromContext(ctx).Get(sessionKey)
	if !ok {
		zap.S().DPanic("[api] - no session found in context")
		return auth.Session{}
	}
	return s.(auth.Session)
}
