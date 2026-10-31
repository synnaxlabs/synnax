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
	"net"
	nethttp "net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
)

// sessionCookiePrefix starts the name of the cookie that carries a browser client's
// session token. The port the browser reached the Core on ends the name, since cookies
// do not separate Cores that share a host. The Core sets it HttpOnly, so page scripts
// never read it. Browsers send it to every port on the host, so a browser-served Core
// needs a hostname it does not share with untrusted services.
const sessionCookiePrefix = "synnax-session-"

func sessionCookieName(host string, tls bool) string {
	_, port, err := net.SplitHostPort(host)
	if err != nil || port == "" {
		port = lo.Ternary(tls, "443", "80")
	}
	return sessionCookiePrefix + port
}

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

// sessionCookieMiddleware refreshes or clears the cookie only for a client that sent
// one. A token it reads from the cookie never reaches the Authorization header of the
// response, even when a later middleware fails and returns the request context.
func sessionCookieMiddleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		// Request headers and query parameters become params, so a client could set
		// the value a handler leaves for this middleware.
		delete(ctx.Params, auth.SessionCookieParam)
		host, _ := ctx.GetDefault(fiber.HeaderHost, "").(string)
		name := sessionCookieName(host, ctx.Sec.TLS.Used)
		fromCookie := sessionCookieValue(ctx.Params, name)
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
		cookie := v.(auth.SessionCookie)
		if cookie.Token == "" {
			cookie.Token = fromCookie
		}
		if cookie.Token == "" {
			return oCtx, err
		}
		if oCtx.Params == nil {
			oCtx.Params = make(freighter.Params)
		}
		oCtx.Set(
			fiber.HeaderSetCookie,
			sessionCookieHeader(name, cookie, ctx.Sec.TLS.Used),
		)
		return oCtx, err
	})
}

func sessionCookieValue(params freighter.Params, name string) string {
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
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// sessionCookieHeader clears the cookie for a zero lifetime.
func sessionCookieHeader(name string, cookie auth.SessionCookie, secure bool) string {
	c := nethttp.Cookie{
		Name:     name,
		Value:    cookie.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: nethttp.SameSiteStrictMode,
		MaxAge:   int(cookie.Lifetime / time.Second),
	}
	if cookie.Lifetime == 0 {
		c.Value = ""
		c.MaxAge = -1
	}
	return c.String()
}
