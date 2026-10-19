// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	nethttp "net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
)

// SessionCookie is the cookie that carries a browser client's session token. The Core
// sets it HttpOnly, so page scripts never read it.
const SessionCookie = "synnax-session"

func unary[RQ, RS freighter.Payload](
	router *http.Router,
	path string,
	opts ...http.UnaryServerOption,
) freighter.UnaryServer[RQ, RS] {
	s := router.NewUnaryServer[RQ, RS](path, opts...)
	s.Use(sessionCookieMiddleware())
	return s
}

func stream[RQ, RS freighter.Payload](
	router *http.Router,
	path string,
	opts ...http.StreamServerOption,
) freighter.StreamServer[RQ, RS] {
	s := router.NewStreamServer[RQ, RS](path, opts...)
	s.Use(sessionCookieMiddleware())
	return s
}

// sessionCookieMiddleware sends a clear only to a client that authenticated by cookie.
// A token it reads from the cookie never reaches the response headers, even when a
// later middleware fails and returns the request context.
func sessionCookieMiddleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		fromCookie := sessionCookieValue(ctx.Params)
		_, hasHeader := ctx.Get(fiber.HeaderAuthorization)
		injected := !hasHeader && fromCookie != ""
		if injected {
			ctx.Set(fiber.HeaderAuthorization, "Bearer "+fromCookie)
		}
		oCtx, err := next(ctx)
		if injected {
			delete(oCtx.Params, fiber.HeaderAuthorization)
		}
		v, ok := ctx.Get(auth.SessionCookieParam)
		if !ok {
			return oCtx, err
		}
		token := v.(string)
		if token == "" && fromCookie == "" {
			return oCtx, err
		}
		if oCtx.Params == nil {
			oCtx.Params = make(freighter.Params)
		}
		oCtx.Set(
			fiber.HeaderSetCookie,
			sessionCookieHeader(token, ctx.Sec.TLS.Used),
		)
		return oCtx, err
	})
}

func sessionCookieValue(params freighter.Params) string {
	raw, ok := params.Get(fiber.HeaderCookie)
	if !ok {
		return ""
	}
	line, ok := raw.(string)
	if !ok {
		return ""
	}
	cookies, err := nethttp.ParseCookie(line)
	if err != nil {
		return ""
	}
	for _, c := range cookies {
		if c.Name == SessionCookie {
			return c.Value
		}
	}
	return ""
}

// sessionCookieHeader clears the cookie for an empty token.
func sessionCookieHeader(token string, secure bool) string {
	c := nethttp.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: nethttp.SameSiteStrictMode,
	}
	if token == "" {
		c.MaxAge = -1
	}
	return c.String()
}
