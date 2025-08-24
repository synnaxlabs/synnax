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
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
)

type unaryServerOptions struct {
	decoders map[string]binary.Decoder
	encoders map[string]binary.Encoder
}

type UnaryServerOption func(*unaryServerOptions)

func WithRequestDecoders(decoders map[string]binary.Decoder) UnaryServerOption {
	return func(o *unaryServerOptions) { o.decoders = decoders }
}

func WithResponseEncoders(encoders map[string]binary.Encoder) UnaryServerOption {
	return func(o *unaryServerOptions) { o.encoders = encoders }
}

func newUnaryServerOptions(opts []UnaryServerOption) unaryServerOptions {
	o := unaryServerOptions{
		decoders: map[string]binary.Decoder{
			MIMEApplicationJSON:    binary.JSONCodec,
			MIMEApplicationMsgPack: binary.MsgPackCodec,
		},
		encoders: map[string]binary.Encoder{
			MIMEApplicationJSON:    binary.JSONCodec,
			MIMEApplicationMsgPack: binary.MsgPackCodec,
		},
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// UnaryReadable is an interface that allows for stream reading of a response. If the
// response type implements this interface, then the response can be streamed via http
// response body streaming to the client.
type UnaryReadable interface {
	// Read reads the next value from the response to be encoded.
	Read(context.Context) (any, error)
}

type unaryServer[RQ, RS freighter.Payload] struct {
	unaryServerOptions
	offers []string
	freighter.Reporter
	freighter.MiddlewareCollector
	handle   func(context.Context, RQ) (RS, error)
	internal bool
}

var _ freighter.UnaryServer[any, any] = (*unaryServer[any, any])(nil)

// NewUnaryServer creates a new unary server that uses HTTP as the transport, connected
// to the given router at the given path.
func NewUnaryServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...UnaryServerOption,
) freighter.UnaryServer[RQ, RS] {
	so := newUnaryServerOptions(opts)
	offers := lo.Keys(so.encoders)
	allEncodings := append(lo.Keys(so.decoders), offers...)
	us := &unaryServer[RQ, RS]{
		unaryServerOptions: so,
		Reporter:           newReporter(allEncodings...),
		offers:             offers,
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
	decoder, ok := us.decoders[fiberCtx.Get(fiber.HeaderContentType)]
	if !ok {
		return fiber.ErrUnsupportedMediaType
	}
	var res RS
	oMD, err := us.MiddlewareCollector.Exec(
		parseRequestCtx(fiberCtx.Context(), fiberCtx, address.Address(fiberCtx.Path())),
		func(freighterCtx freighter.Context) (freighter.Context, error) {
			var (
				req RQ
				err error
			)
			if err := decoder.Decode(
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
	contentType := ctx.Accepts(us.offers...)
	encoder, ok := us.encoders[contentType]
	if !ok {
		return fiber.ErrNotAcceptable
	}
	ctx.Set(fiber.HeaderContentType, contentType)
	reqCtx := ctx.Context()
	if uReader, ok := v.(UnaryReadable); ok {
		r, w := io.Pipe()
		go func() {
			defer w.Close()
			for {
				select {
				case <-reqCtx.Done():
					w.CloseWithError(reqCtx.Err())
					return
				default:
					v, err := uReader.Read(reqCtx)
					if err != nil {
						w.CloseWithError(err)
						return
					}
					if err := encoder.EncodeStream(reqCtx, w, v); err != nil {
						w.CloseWithError(err)
						return
					}
				}
			}
		}()
		return ctx.SendStream(r)
	}
	b, err := encoder.Encode(reqCtx, v)
	if err != nil {
		return err
	}
	_, err = ctx.Write(b)
	return err
}
