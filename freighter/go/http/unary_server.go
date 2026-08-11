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
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
)

// unaryServerOptions configures a unary HTTP server. Unary handlers are stateless
// across requests, so the registered decoders and encoders are shared instances rather
// than per-request constructors.
type unaryServerOptions struct {
	compressionOptions
	// requestDecoders is the set of decoders the unary server will consider when
	// resolving the request body codec from the Content-Type header.
	requestDecoders []http.Decoder
	// responseEncoders is the set of encoders the unary server will consider when
	// resolving the response body codec from the Accept header.
	responseEncoders []http.Encoder
}

// UnaryServerOption configures a unary HTTP server.
type UnaryServerOption func(*unaryServerOptions)

// WithRequestDecoders overrides the set of decoders the unary server matches against
// the request's Content-Type header.
func WithRequestDecoders(decoders ...http.Decoder) UnaryServerOption {
	return func(o *unaryServerOptions) { o.requestDecoders = decoders }
}

// WithResponseEncoders overrides the set of encoders the unary server matches against
// the request's Accept header.
func WithResponseEncoders(encoders ...http.Encoder) UnaryServerOption {
	return func(o *unaryServerOptions) { o.responseEncoders = encoders }
}

// WithCompressions overrides the Content-Encodings the unary server accepts on
// requests and offers on responses, ordered by preference. Pass no encodings to
// disable compression entirely.
func WithCompressions(compressions ...http.Compression) UnaryServerOption {
	return func(o *unaryServerOptions) { o.compressions = compressions }
}

// WithMinCompressSize overrides the smallest response body the unary server
// compresses. Bodies below it are sent uncompressed, since compressing them costs more
// CPU than the transmission time it saves.
func WithMinCompressSize(size int) UnaryServerOption {
	return func(o *unaryServerOptions) { o.minCompressSize = size }
}

// WithMaxDecompressedSize overrides how far a compressed request body may expand before
// the unary server rejects it with xhttp.ErrBodyTooLarge.
func WithMaxDecompressedSize(size int) UnaryServerOption {
	return func(o *unaryServerOptions) { o.maxDecompressedSize = size }
}

func newUnaryServerOptions(opts []UnaryServerOption) unaryServerOptions {
	so := unaryServerOptions{
		compressionOptions: defaultCompressionOptions(),
		requestDecoders:    defaultDecoders,
		responseEncoders:   defaultEncoders,
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

type unaryServer[RQ, RS freighter.Payload] struct {
	unaryServerOptions
	handle freighter.UnaryHandler[RQ, RS]
	path   string
	freighter.MiddlewareCollector
}

// Report describes the unary server's protocol, the content types it accepts on
// requests vs emits on responses, and the content encodings it can compress and
// decompress. Accept and emit lists may differ — e.g. an import endpoint that accepts
// JSON|YAML|TOML but emits only JSON|MessagePack.
func (s *unaryServer[RQ, RS]) Report() alamos.Report {
	return alamos.Report{
		"protocol": unaryProtocol,
		"acceptedContentTypes": lo.Map(
			s.requestDecoders,
			func(d http.Decoder, _ int) string {
				return d.ContentType()
			},
		),
		"emittedContentTypes": lo.Map(
			s.responseEncoders,
			func(e http.Encoder, _ int) string {
				return e.ContentType()
			},
		),
		"contentEncodings": s.contentEncodings(),
	}
}

func (s *unaryServer[RQ, RS]) BindHandler(handler freighter.UnaryHandler[RQ, RS]) {
	s.handle = handler
}

func (s *unaryServer[RQ, RS]) fiberHandler(fCtx fiber.Ctx) error {
	decoder, ok := s.resolveRequestDecoder(fCtx.Get(fiber.HeaderContentType))
	if !ok {
		return fCtx.SendStatus(fiber.StatusUnsupportedMediaType)
	}
	encoder, ok := s.resolveResponseEncoder(fCtx)
	if !ok {
		return fCtx.SendStatus(fiber.StatusNotAcceptable)
	}
	fCtx.Set(fiber.HeaderContentType, encoder.ContentType())
	// Responses on this route vary by the request's Accept-Encoding, so caches must key
	// on it whether or not this particular response ends up compressed.
	if len(s.compressions) > 0 {
		fCtx.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
	}
	var res RS
	oMD, err := s.Exec(
		parseRequestCtx(fCtx.RequestCtx(), fCtx, address.Address(fCtx.Path()), false),
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			oCtx := freighter.Context{
				Protocol: ctx.Protocol,
				Params:   make(freighter.Params),
			}
			body, err := s.decompressIncoming(
				fCtx.BodyRaw(), fCtx.Get(fiber.HeaderContentEncoding),
			)
			if err != nil {
				return oCtx, err
			}
			var req RQ
			if err = decoder.Decode(ctx, body, &req); err != nil {
				return oCtx, err
			}
			res, err = s.handle(ctx, req)
			return oCtx, err
		}),
	)
	setResponseCtx(fCtx, oMD)
	fErr := errors.Encode(fCtx.RequestCtx(), err, false)
	if fErr.Type == errors.TypeNil {
		return s.encodeAndWrite(fCtx, encoder, res)
	}
	fCtx.Status(fiber.StatusBadRequest)
	return s.encodeAndWrite(fCtx, encoder, fErr)
}

func (s *unaryServer[RQ, RS]) resolveRequestDecoder(
	contentType string,
) (http.Decoder, bool) {
	for _, d := range s.requestDecoders {
		if d.ContentType() == contentType {
			return d, true
		}
	}
	return nil, false
}

func (s *unaryServer[RQ, RS]) resolveResponseEncoder(
	fCtx fiber.Ctx,
) (http.Encoder, bool) {
	offers := lo.Map(s.responseEncoders, func(e http.Encoder, _ int) string {
		return e.ContentType()
	})
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

func (s *unaryServer[RQ, RS]) encodeAndWrite(
	c fiber.Ctx,
	encoder http.Encoder,
	v any,
) error {
	b, err := encoder.Encode(c.RequestCtx(), v)
	if err != nil {
		return err
	}
	b, contentEncoding, err := s.compressNegotiated(
		b, c.Get(fiber.HeaderAcceptEncoding),
	)
	if err != nil {
		return err
	}
	if contentEncoding != "" {
		c.Set(fiber.HeaderContentEncoding, contentEncoding)
	}
	_, err = c.Write(b)
	return err
}
