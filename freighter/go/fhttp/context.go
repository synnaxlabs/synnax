// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

func parseRequestCtx(
	ctx context.Context,
	fiberCtx *fiber.Ctx,
	target address.Address,
) freighter.Context {
	freighterCtx := freighter.Context{
		Context:      ctx,
		Protocol:     "http",
		Target:       target,
		SecurityInfo: parseSecurityInfo(fiberCtx),
		Role:         freighter.Server,
		Variant:      freighter.Unary,
	}
	headers := fiberCtx.GetReqHeaders()
	freighterCtx.Params = make(freighter.Params, len(headers))
	for k, v := range fiberCtx.GetReqHeaders() {
		if len(v) > 0 {
			freighterCtx.Params[k] = v[0]
		}
	}
	for k, v := range parseQueryString(fiberCtx) {
		if cut, found := strings.CutPrefix(k, "freighterctx"); found {
			freighterCtx.Params[cut] = v
		}
	}
	return freighterCtx
}

func parseSecurityInfo(ctx *fiber.Ctx) freighter.SecurityInfo {
	var info freighter.SecurityInfo
	if ctx.Context().IsTLS() {
		info.TLS.Used = true
		info.TLS.ConnectionState = ctx.Context().Conn().(*tls.Conn).ConnectionState()
	}
	return info
}

func setRequestCtx(req *http.Request, ctx freighter.Context) {
	for k, v := range ctx.Params {
		if vStr, ok := v.(string); ok {
			req.Header.Set(k, vStr)
		}
	}
}

func setResponseCtx(fiberCtx *fiber.Ctx, freighterCtx freighter.Context) {
	for k, v := range freighterCtx.Params {
		if vStr, ok := v.(string); ok {
			fiberCtx.Set(k, vStr)
		}
	}
}

func parseResponseCtx(res *http.Response, target address.Address) freighter.Context {
	ctx := freighter.Context{
		Role:     freighter.Client,
		Variant:  freighter.Unary,
		Protocol: "http",
		Target:   target,
		Params: lo.Ternary(
			len(res.Header) > 0,
			make(freighter.Params, len(res.Header)),
			nil,
		),
	}
	for k, v := range res.Header {
		ctx.Params[k] = v[0]
	}
	return ctx
}

func parseQueryString(ctx *fiber.Ctx) map[string]string {
	data := make(map[string]string)
	ctx.Context().QueryArgs().VisitAll(func(key, val []byte) {
		k := utils.UnsafeString(key)
		v := utils.UnsafeString(val)
		data[k] = v
	})
	return data
}

func ctxToHeaders(ctx freighter.Context) http.Header {
	headers := make(http.Header, len(ctx.Params))
	for k, v := range ctx.Params {
		if vStr, ok := v.(string); ok {
			headers[k] = []string{vStr}
		}
	}
	return headers
}
