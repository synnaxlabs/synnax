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
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
)

type unaryServer[RQ, RS freighter.Payload] struct {
	serverOptions
	freighter.Reporter
	freighter.MiddlewareCollector
	requestParser func(*fiber.Ctx, httputil.Codec) (RQ, error)
	handle        func(context.Context, RQ) (RS, error)
	internal      bool
	path          string
}

func (s *unaryServer[RQ, RS]) BindHandler(handler func(context.Context, RQ) (RS, error)) {
	s.handle = handler
}

func (s *unaryServer[RQ, RS]) fiberHandler(fCtx *fiber.Ctx) error {
	fCtx.Accepts(httputil.SupportedContentTypes()...)
	codec, err := httputil.ResolveCodec(fCtx.Get(fiber.HeaderContentType))
	if err != nil {
		return err
	}
	fCtx.Set(fiber.HeaderContentType, codec.ContentType())
	var res RS
	oMD, err := s.MiddlewareCollector.Exec(
		parseRequestCtx(fCtx.Context(), fCtx, address.Address(fCtx.Path())),
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var req RQ
			err := codec.Decode(ctx, fCtx.BodyRaw(), &req)
			oCtx := freighter.Context{Protocol: ctx.Protocol, Params: make(freighter.Params)}
			if err != nil {
				return oCtx, err
			}
			res, err = s.handle(ctx, req)
			return oCtx, err
		}),
	)
	setResponseCtx(fCtx, oMD)
	fErr := errors.Encode(fCtx.Context(), err, s.internal)
	if fErr.Type == errors.TypeNil {
		return encodeAndWrite(fCtx, codec, res)
	}
	fCtx.Status(fiber.StatusBadRequest)
	return encodeAndWrite(fCtx, codec, fErr)
}

type unaryClient[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	codec httputil.Codec
}

func (u *unaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (RS, error) {
	var res RS
	_, err := u.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Protocol: unaryReporter.Protocol,
			Target:   target,
		},
		freighter.FinalizerFunc(func(inCtx freighter.Context) (freighter.Context, error) {
			b, err := u.codec.Encode(inCtx, req)
			if err != nil {
				return freighter.Context{}, err
			}
			httpReq, err := http.NewRequestWithContext(
				ctx,
				"POST",
				"http://"+target.String(),
				bytes.NewReader(b),
			)
			if err != nil {
				return freighter.Context{}, err
			}
			setRequestCtx(httpReq, inCtx)
			httpReq.Header.Set(fiber.HeaderContentType, u.codec.ContentType())

			httpRes, err := (&http.Client{}).Do(httpReq)
			outCtx := parseResponseCtx(httpRes, target)
			if err != nil {
				return outCtx, err
			}

			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld errors.Payload
				if err := u.codec.DecodeStream(context.TODO(), httpRes.Body, &pld); err != nil {
					return outCtx, err
				}
				return outCtx, errors.Decode(ctx, pld)
			}
			return outCtx, u.codec.DecodeStream(context.TODO(), httpRes.Body, &res)
		}),
	)
	return res, err
}

func encodeAndWrite(c *fiber.Ctx, codec httputil.Codec, v any) error {
	b, err := codec.Encode(c.Context(), v)
	if err != nil {
		return err
	}
	_, err = c.Write(b)
	return err
}

func parseRequestCtx(socketCtx context.Context, fiberCtx *fiber.Ctx, target address.Address) freighter.Context {
	freighterCtx := freighter.Context{
		Context:      socketCtx,
		Protocol:     unaryReporter.Protocol,
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
		if isFreighterQueryStringParam(k) {
			freighterCtx.Params[strings.TrimPrefix(k, freighterCtxPrefix)] = v
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

func setRequestCtx(c *http.Request, ctx freighter.Context) {
	for k, v := range ctx.Params {
		if vStr, ok := v.(string); ok {
			c.Header.Set(freighterCtxPrefix+k, vStr)
		}
	}
}

func setResponseCtx(fiberCtx *fiber.Ctx, freighterCtx freighter.Context) {
	for k, v := range freighterCtx.Params {
		if vStr, ok := v.(string); ok {
			fiberCtx.Set(freighterCtxPrefix+k, vStr)
		}
	}
}

func parseResponseCtx(c *http.Response, target address.Address) freighter.Context {
	ctx := freighter.Context{
		Role:     freighter.Client,
		Variant:  freighter.Unary,
		Protocol: unaryReporter.Protocol,
		Target:   target,
		Params: lo.Ternary(
			len(c.Header) > 0,
			make(freighter.Params, len(c.Header)),
			nil,
		),
	}
	for k, v := range c.Header {
		ctx.Params[k] = v[0]
	}
	return ctx
}

func parseQueryString(c *fiber.Ctx) map[string]string {
	data := make(map[string]string)
	c.Context().QueryArgs().VisitAll(func(key, val []byte) {
		k := utils.UnsafeString(key)
		v := utils.UnsafeString(val)
		data[k] = v
	})
	return data
}

const freighterCtxPrefix = "freighterctx"

func isFreighterQueryStringParam(k string) bool {
	return strings.HasPrefix(k, freighterCtxPrefix)
}
