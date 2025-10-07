// Copyright 2025 Synnax Labs, Inc.
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
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"go.uber.org/zap"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
)

const tokenRefreshHeader = "Refresh-Token"

func tokenMiddleware(svc *token.Service) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		tk, _err := tryParseToken(ctx.Params)
		if _err != nil {
			return ctx, _err
		}
		userKey, newTK, err := svc.ValidateMaybeRefresh(tk)
		if err != nil {
			return ctx, err
		}
		setSubject(ctx.Params, user.OntologyID(userKey))
		oCtx, err := next(ctx)
		if newTK != "" {
			oCtx.Set(tokenRefreshHeader, newTK)
		}
		return oCtx, err
	})
}

const tokenParamPrefix = "Bearer "

var (
	invalidAuthenticationParam = errors.Wrapf(auth.Error,
		`invalid authorization token. Format should be
		'Authorization: %s <token>'`, tokenParamPrefix,
	)
	noAuthenticationParam = errors.Wrapf(auth.Error, "no authentication token provided")
)

func tryParseToken(p freighter.Params) (string, error) {
	tkParam, ok := p.Get(fiber.HeaderAuthorization)
	if !ok {
		// GRPC sends a lowercase header
		tkParam, ok = p.Get(strings.ToLower(fiber.HeaderAuthorization))
		if !ok {
			return "", noAuthenticationParam
		}
	}
	tkStr, ok := tkParam.(string)
	if !ok {
		return "", noAuthenticationParam
	}
	if !strings.HasPrefix(tkStr, tokenParamPrefix) {
		return "", invalidAuthenticationParam
	}
	tkStr = strings.TrimPrefix(tkStr, tokenParamPrefix)
	if !ok {
		return "", invalidAuthenticationParam
	}
	return tkStr, nil
}

const subjectKey = "Subject"

func setSubject(p freighter.Params, subject ontology.ID) {
	p.Set(subjectKey, subject)
}

func getSubject(ctx context.Context) ontology.ID {
	s, ok := freighter.MDFromContext(ctx).Get(subjectKey)
	if !ok {
		zap.S().DPanic("[api] - no subject found in context")
		return user.OntologyID(uuid.Nil)
	}
	return s.(ontology.ID)
}
