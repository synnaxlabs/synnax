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
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
)

var unaryReporter = freighter.Reporter{
	Protocol:  "http",
	Encodings: SupportedContentTypes(),
}

// unaryServerOptions configures a unary HTTP server. Unary handlers are stateless
// across requests, so the registered decoders and encoders are shared instances rather
// than per-request constructors.
type unaryServerOptions struct {
	// requestDecoders is the set of decoders the unary server will consider when
	// resolving the request body codec from the Content-Type header.
	requestDecoders []RequestDecoder
	// responseEncoders is the set of encoders the unary server will consider when
	// resolving the response body codec from the Accept header.
	responseEncoders []ResponseEncoder
}

// UnaryServerOption configures a unary HTTP server.
type UnaryServerOption func(*unaryServerOptions)

// WithRequestDecoders overrides the set of decoders the unary server matches against
// the request's Content-Type header.
func WithRequestDecoders(decoders ...RequestDecoder) UnaryServerOption {
	return func(o *unaryServerOptions) { o.requestDecoders = decoders }
}

// WithResponseEncoders overrides the set of encoders the unary server matches against
// the request's Accept header.
func WithResponseEncoders(encoders ...ResponseEncoder) UnaryServerOption {
	return func(o *unaryServerOptions) { o.responseEncoders = encoders }
}

func newUnaryServerOptions(opts []UnaryServerOption) unaryServerOptions {
	so := unaryServerOptions{
		requestDecoders:  DefaultRequestDecoders(),
		responseEncoders: DefaultResponseEncoders(),
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

type unaryServer[RQ, RS freighter.Payload] struct {
	unaryServerOptions
	handle func(context.Context, RQ) (RS, error)
	path   string
	freighter.Reporter
	freighter.MiddlewareCollector
	internal bool
}

func (s *unaryServer[RQ, RS]) BindHandler(
	handle func(context.Context, RQ) (RS, error),
) {
	s.handle = handle
}

func (s *unaryServer[RQ, RS]) fiberHandler(fCtx fiber.Ctx) error {
	decoder, err := s.resolveRequestDecoder(fCtx.Get(fiber.HeaderContentType))
	if err != nil {
		return err
	}
	encoder, ok := s.resolveResponseEncoder(fCtx)
	if !ok {
		return fCtx.Status(fiber.StatusNotAcceptable).SendString("Not Acceptable")
	}
	fCtx.Set(fiber.HeaderContentType, encoder.ContentType())
	var res RS
	oMD, err := s.Exec(
		parseRequestCtx(fCtx.RequestCtx(), fCtx, address.Address(fCtx.Path())),
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var req RQ
			err := decoder.Decode(ctx, fCtx.BodyRaw(), &req)
			oCtx := freighter.Context{
				Protocol: ctx.Protocol,
				Params:   make(freighter.Params),
			}
			if err != nil {
				return oCtx, err
			}
			res, err = s.handle(ctx, req)
			return oCtx, err
		}),
	)
	setResponseCtx(fCtx, oMD)
	fErr := errors.Encode(fCtx.RequestCtx(), err, s.internal)
	if fErr.Type == errors.TypeNil {
		return encodeAndWrite(fCtx, encoder, res)
	}
	fCtx.Status(fiber.StatusBadRequest)
	return encodeAndWrite(fCtx, encoder, fErr)
}

func (s *unaryServer[RQ, RS]) resolveRequestDecoder(
	contentType string,
) (RequestDecoder, error) {
	for _, d := range s.requestDecoders {
		if d.ContentType() == contentType {
			return d, nil
		}
	}
	return nil, errors.Newf(
		"[encoding] - unable to determine encoding type for %s",
		contentType,
	)
}

func (s *unaryServer[RQ, RS]) resolveResponseEncoder(
	fCtx fiber.Ctx,
) (ResponseEncoder, bool) {
	offers := make([]string, len(s.responseEncoders))
	for i, e := range s.responseEncoders {
		offers[i] = e.ContentType()
	}
	matched := fCtx.Accepts(offers...)
	if matched == "" {
		return nil, false
	}
	for _, e := range s.responseEncoders {
		if e.ContentType() == matched {
			return e, true
		}
	}
	return nil, false
}

func encodeAndWrite(c fiber.Ctx, encoder ResponseEncoder, v any) error {
	b, err := encoder.Encode(c.RequestCtx(), v)
	if err != nil {
		return err
	}
	_, err = c.Write(b)
	return err
}
