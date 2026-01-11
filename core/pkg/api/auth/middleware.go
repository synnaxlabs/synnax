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

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/uuid"
	"go.uber.org/zap"
)

// TokenMiddleware creates a middleware that validates JWT tokens.
func TokenMiddleware(svc *token.Service) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		tk, err := tryParseToken(ctx.Params)
		if err != nil {
			return ctx, err
		}
		userKey, newTK, err := svc.ValidateMaybeRefresh(tk)
		if err != nil {
			return ctx, err
		}
		ctx.Set(subjectKey, user.OntologyID(userKey))
		oCtx, err := next(ctx)
		if newTK != "" {
			oCtx.Set("Refresh-Token", newTK)
		}
		return oCtx, err
	})
}

const tokenParamPrefix = "Bearer "

var (
	errInvalidAuthenticationParam = errors.Wrapf(
		svcauth.Error,
		`invalid authorization token. Format should be
		'Authorization: %s <token>'`,
		tokenParamPrefix,
	)
	errNoAuthenticationParam = errors.Wrapf(
		svcauth.Error,
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

const subjectKey = "Subject"

// GetSubject retrieves the authenticated subject's ontology ID from the context.
func GetSubject(ctx context.Context) ontology.ID {
	s, ok := freighter.MDFromContext(ctx).Get(subjectKey)
	if !ok {
		zap.S().DPanic("[api] - no subject found in context")
		return user.OntologyID(uuid.Nil)
	}
	return s.(ontology.ID)
}
