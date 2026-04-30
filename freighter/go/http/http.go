// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package http provides HTTP and websocket transports for freighter. It includes both
// unary and streaming servers (use in production for communicating with the TypeScript
// and Python clients) and unary and streaming clients for testing.
package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/utils/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

// BindableTransport is a freighter.Transport that knows how to register its routes on a
// fiber.App. The Router and any individual server registered through it satisfy this
// interface.
type BindableTransport interface {
	freighter.Transport
	// BindTo registers the transport's HTTP and websocket routes on the given
	// fiber.App.
	BindTo(*fiber.App)
}

const freighterCtxPrefix = "freighterctx"

func isFreighterQueryStringParam(k string) bool {
	return strings.HasPrefix(k, freighterCtxPrefix)
}

func parseQueryString(c fiber.Ctx) map[string]string {
	data := make(map[string]string)
	for key, val := range c.RequestCtx().QueryArgs().All() {
		k := utils.UnsafeString(key)
		v := utils.UnsafeString(val)
		data[k] = v
	}
	return data
}

func parseSecurityInfo(c fiber.Ctx) (info freighter.SecurityInfo) {
	if c.RequestCtx().IsTLS() {
		info.TLS.Used = true
		info.TLS.ConnectionState = c.RequestCtx().Conn().(*tls.Conn).ConnectionState()
	}
	return info
}

func parseRequestCtx(
	socketCtx context.Context,
	fiberCtx fiber.Ctx,
	target address.Address,
) freighter.Context {
	md := freighter.Context{
		Context:  socketCtx,
		Protocol: unaryProtocol,
		Target:   target,
		Sec:      parseSecurityInfo(fiberCtx),
		Role:     freighter.RoleServer,
		Variant:  freighter.VariantUnary,
	}
	headers := fiberCtx.GetReqHeaders()
	md.Params = make(freighter.Params, len(headers))
	for k, v := range fiberCtx.GetReqHeaders() {
		if len(v) > 0 {
			md.Params[k] = v[0]
		}
	}
	for k, v := range parseQueryString(fiberCtx) {
		if isFreighterQueryStringParam(k) {
			md.Params[strings.TrimPrefix(k, freighterCtxPrefix)] = v
		}
	}
	return md
}

func setRequestCtx(c *http.Request, ctx freighter.Context) {
	for k, v := range ctx.Params {
		if vStr, ok := v.(string); ok {
			c.Header.Set(freighterCtxPrefix+k, vStr)
		}
	}
}

func setResponseCtx(c fiber.Ctx, md freighter.Context) {
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			c.Set(freighterCtxPrefix+k, vStr)
		}
	}
}

func parseResponseCtx(c *http.Response, target address.Address) freighter.Context {
	md := freighter.Context{
		Role:     freighter.RoleClient,
		Variant:  freighter.VariantUnary,
		Protocol: unaryProtocol,
		Target:   target,
		Params: lo.Ternary(
			len(c.Header) > 0,
			make(freighter.Params, len(c.Header)),
			nil,
		),
	}
	for k, v := range c.Header {
		md.Params[k] = v[0]
	}
	return md
}
