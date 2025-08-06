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
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
)

// UnaryReadable is an interface that allows for stream reading of a response. If the
// response type implements this interface, then the response can be streamed via http
// body streaming to the client.
type UnaryReadable interface {
	// Read reads the next value from the response to be encoded.
	Read() (any, error)
}

type unaryServer[RQ, RS freighter.Payload] struct {
	serverOptions
	freighter.Reporter
	freighter.MiddlewareCollector
	handle   func(context.Context, RQ) (RS, error)
	internal bool
}

var _ freighter.UnaryServer[any, any] = (*unaryServer[any, any])(nil)

func NewUnaryServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...ServerOption,
) *unaryServer[RQ, RS] {
	so := newServerOptions(opts)
	us := &unaryServer[RQ, RS]{
		serverOptions: so,
		Reporter:      getReporter(so),
	}
	r.register(path, fiber.MethodPost, us, us.fiberHandler)
	return us
}

func (us *unaryServer[RQ, RS]) BindHandler(
	handler func(context.Context, RQ) (RS, error),
) {
	us.handle = handler
}

func (us *unaryServer[RQ, RS]) fiberHandler(fiberCtx *fiber.Ctx) error {
	getCodec, ok := us.reqDecoders[fiberCtx.Get(fiber.HeaderContentType)]
	if !ok {
		return fiber.ErrUnsupportedMediaType
	}
	codec := getCodec()
	var res RS
	oMD, err := us.MiddlewareCollector.Exec(
		parseRequestCtx(fiberCtx.Context(), fiberCtx, address.Address(fiberCtx.Path())),
		func(freighterCtx freighter.Context) (freighter.Context, error) {
			var (
				req RQ
				err error
			)
			if err := codec.Decode(
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
				Context:  freighterCtx.Context,
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
		return us.encodeAndWrite(fiberCtx, res)
	}
	fiberCtx.Status(fiber.StatusBadRequest)
	return us.encodeAndWrite(fiberCtx, fErr)
}

func (us *unaryServer[RQ, RS]) encodeAndWrite(ctx *fiber.Ctx, v any) error {
	contentType := ctx.Accepts(us.resEncoders.Keys()...)
	getCodec, ok := us.resEncoders[contentType]
	if !ok {
		return fiber.ErrNotAcceptable
	}
	codec := getCodec()
	ctx.Set(fiber.HeaderContentType, contentType)
	if uReader, ok := v.(UnaryReadable); ok {
		r, w := io.Pipe()
		reqCtx := ctx.Context()
		go func() {
			for {
				v, err := uReader.Read()
				if err != nil {
					fmt.Println("closing the stream with error", err)
					w.CloseWithError(err)
					return
				}
				if err := codec.EncodeStream(reqCtx, w, v); err != nil {
					w.CloseWithError(err)
					return
				}
			}
		}()
		return ctx.SendStream(r)
	}
	b, err := codec.Encode(ctx.Context(), v)
	if err != nil {
		return err
	}
	_, err = ctx.Write(b)
	return err
}

type unaryClient[RQ, RS freighter.Payload] struct {
	cfg ClientConfig
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ freighter.UnaryClient[any, any] = (*unaryClient[any, any])(nil)

func NewUnaryClient[RQ, RS freighter.Payload](
	cfgs ...ClientConfig,
) (*unaryClient[RQ, RS], error) {
	cfg, err := config.New(DefaultClientConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &unaryClient[RQ, RS]{cfg: cfg}, nil
}

func (uc *unaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (RS, error) {
	var res RS
	if _, err := uc.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Role:     freighter.Client,
			Variant:  freighter.Unary,
			Protocol: "http",
			Target:   target,
			Params:   make(freighter.Params),
		},
		func(inCtx freighter.Context) (freighter.Context, error) {
			b, err := uc.cfg.Encoder.Encode(inCtx, req)
			if err != nil {
				return freighter.Context{}, err
			}
			httpReq, err := http.NewRequestWithContext(
				ctx,
				fiber.MethodPost,
				"http://"+target.String(),
				bytes.NewReader(b),
			)
			if err != nil {
				return freighter.Context{}, err
			}
			setRequestCtx(httpReq, inCtx)
			httpReq.Header.Set(fiber.HeaderContentType, uc.cfg.ContentType)
			httpReq.Header.Set(fiber.HeaderAccept, uc.cfg.Accept)
			httpRes, err := (&http.Client{}).Do(httpReq)
			if err != nil {
				return freighter.Context{}, err
			}
			outCtx := parseResponseCtx(httpRes, target)
			if contentType := httpRes.Header.Get(fiber.HeaderContentType); contentType != uc.cfg.Accept {
				return freighter.Context{}, errors.Newf(
					"unexpected response content type: %s, expected: %s",
					contentType,
					uc.cfg.Accept,
				)
			}
			decoder := uc.cfg.Decoder
			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld errors.Payload
				if err := decoder.DecodeStream(
					inCtx,
					httpRes.Body,
					&pld,
				); err != nil {
					return freighter.Context{}, err
				}
				if err := errors.Decode(ctx, pld); err != nil {
					return freighter.Context{}, err
				}
				return outCtx, nil
			}
			if err := uc.cfg.Decoder.DecodeStream(
				inCtx,
				httpRes.Body,
				&res,
			); err != nil {
				return freighter.Context{}, err
			}
			return outCtx, nil
		},
	); err != nil {
		var r RS
		return r, err
	}
	return res, nil
}
