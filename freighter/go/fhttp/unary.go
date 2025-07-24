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
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
)

var unaryReporter = freighter.Reporter{
	Protocol:  "http",
	Encodings: httputil.SupportedContentTypes(),
}

type unaryServer[RQ, RS freighter.Payload] struct {
	serverOptions
	freighter.Reporter
	freighter.MiddlewareCollector
	handle   func(context.Context, RQ) (RS, error)
	internal bool
}

var _ freighter.UnaryServer[any, any] = &unaryServer[any, any]{}

func NewUnaryServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...ServerOption,
) *unaryServer[RQ, RS] {
	us := &unaryServer[RQ, RS]{
		serverOptions: newServerOptions(opts),
		Reporter:      unaryReporter,
	}
	r.register(path, "POST", us, us.fiberHandler)
	return us
}

func (us *unaryServer[RQ, RS]) BindHandler(
	handler func(context.Context, RQ) (RS, error),
) {
	us.handle = handler
}

func (us *unaryServer[RQ, RS]) fiberHandler(fiberCtx *fiber.Ctx) error {
	reqCodec, err := us.reqCodecResolver(fiberCtx.Get(fiber.HeaderContentType))
	if err != nil {
		return errors.Combine(fiber.ErrUnsupportedMediaType, err)
	}
	resContentType := fiberCtx.Accepts(us.supportedResponseContentTypes...)
	resCodec, err := us.resCodecResolver(resContentType)
	if err != nil {
		return errors.Combine(fiber.ErrNotAcceptable, err)
	}
	fiberCtx.Set(fiber.HeaderContentType, resCodec.ContentType())
	var res RS
	oMD, err := us.MiddlewareCollector.Exec(
		parseRequestCtx(fiberCtx.Context(), fiberCtx, address.Address(fiberCtx.Path())),
		func(freighterCtx freighter.Context) (freighter.Context, error) {
			var req RQ
			if err := reqCodec.Decode(
				freighterCtx,
				fiberCtx.BodyRaw(),
				&req,
			); err != nil {
				return freighter.Context{}, err
			}
			if res, err = us.handle(freighterCtx, req); err != nil {
				return freighter.Context{}, err
			}
			return freighter.Context{
				Protocol: freighterCtx.Protocol,
				Role:     freighter.Server,
				Variant:  freighter.Unary,
				Params:   make(freighter.Params),
			}, nil
		},
	)
	setResponseCtx(fiberCtx, oMD)
	fErr := errors.Encode(fiberCtx.Context(), err, us.internal)
	if fErr.Type == errors.TypeNil {
		return encodeAndWrite(fiberCtx, resCodec, res)
	}
	fiberCtx.Status(fiber.StatusBadRequest)
	return encodeAndWrite(fiberCtx, resCodec, fErr)
}

type unaryClient[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	codec httputil.Codec
}

var _ freighter.UnaryClient[any, any] = &unaryClient[any, any]{}

func NewUnaryClient[RQ, RS freighter.Payload](
	cfgs ...ClientConfig,
) (*unaryClient[RQ, RS], error) {
	cfg, err := config.New(DefaultClientConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &unaryClient[RQ, RS]{codec: cfg.Codec}, nil
}

func (uc *unaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (RS, error) {
	var res RS
	_, err := uc.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Protocol: unaryReporter.Protocol,
			Target:   target,
			Params:   make(freighter.Params),
		},
		func(inCtx freighter.Context) (freighter.Context, error) {
			b, err := uc.codec.Encode(inCtx, req)
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
			httpReq.Header.Set(fiber.HeaderContentType, uc.codec.ContentType())

			httpRes, err := (&http.Client{}).Do(httpReq)
			outCtx := parseResponseCtx(httpRes, target)
			if err != nil {
				return outCtx, err
			}

			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld errors.Payload
				if err := uc.codec.DecodeStream(inCtx, httpRes.Body, &pld); err != nil {
					return outCtx, err
				}
				return outCtx, errors.Decode(ctx, pld)
			}
			if reader, ok := httpRes.Body.(RS); ok {
				res = reader
				return outCtx, nil
			}
			return outCtx, uc.codec.DecodeStream(inCtx, httpRes.Body, &res)
		},
	)
	return res, err
}

func encodeAndWrite(ctx *fiber.Ctx, codec httputil.Codec, v any) error {
	if uReader, ok := v.(freighter.UnaryReadable); ok {
		r, w := io.Pipe()
		go func() {
			for {
				v, err := uReader.Read()
				if err != nil {
					w.CloseWithError(err)
					return
				}
				if err := codec.EncodeStream(ctx.Context(), w, v); err != nil {
					w.CloseWithError(err)
					return
				}
			}
		}()
		return ctx.SendStream(r)
	}
	if r, ok := v.(io.Reader); ok {
		return ctx.SendStream(r)
	}
	b, err := codec.Encode(ctx.Context(), v)
	if err != nil {
		return err
	}
	_, err = ctx.Write(b)
	return err
}

func parseRequestCtx(
	ctx context.Context,
	fiberCtx *fiber.Ctx,
	target address.Address,
) freighter.Context {
	freighterCtx := freighter.Context{
		Context:      ctx,
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
		Protocol: unaryReporter.Protocol,
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
