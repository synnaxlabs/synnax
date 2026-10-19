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

// unary builds a unary server on router that accepts a session cookie in place of the
// Authorization header and sets or clears the cookie when a handler asks.
func unary[RQ, RS freighter.Payload](
	router *http.Router,
	path string,
	opts ...http.UnaryServerOption,
) freighter.UnaryServer[RQ, RS] {
	s := router.NewUnaryServer[RQ, RS](path, opts...)
	s.Use(sessionCookieMiddleware())
	return s
}

// stream builds a stream server on router that accepts a session cookie in place of
// the Authorization header.
func stream[RQ, RS freighter.Payload](
	router *http.Router,
	path string,
	opts ...http.StreamServerOption,
) freighter.StreamServer[RQ, RS] {
	s := router.NewStreamServer[RQ, RS](path, opts...)
	s.Use(sessionCookieMiddleware())
	return s
}

// sessionCookieMiddleware lets a request authenticate with the session cookie when it
// carries no Authorization header, and turns a handler's [auth.SessionCookieParam] into
// a Set-Cookie header. A clear is sent only to a client that authenticated by cookie.
func sessionCookieMiddleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		fromCookie := sessionCookieValue(ctx.Params)
		if _, ok := ctx.Get(fiber.HeaderAuthorization); !ok && fromCookie != "" {
			ctx.Set(fiber.HeaderAuthorization, "Bearer "+fromCookie)
		}
		oCtx, err := next(ctx)
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

// sessionCookieValue returns the session cookie's value from the request's Cookie
// header, or an empty string when there is none.
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

// sessionCookieHeader renders the Set-Cookie value for token. An empty token clears
// the cookie.
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
