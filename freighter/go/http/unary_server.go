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
	"bufio"
	"compress/gzip"
	"context"
	"io"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// unaryServerOptions configures a unary HTTP server. Unary handlers are stateless
// across requests, so the registered decoders and encoders are shared instances rather
// than per-request constructors.
type unaryServerOptions struct {
	// requestDecoders is the set of decoders the unary server will consider when
	// resolving the request body codec from the Content-Type header.
	requestDecoders []http.Decoder
	// responseEncoders is the set of encoders the unary server will consider when
	// resolving the response body codec from the Accept header.
	responseEncoders []http.Encoder
	// streaming writes the response through Encoder.EncodeStream as a chunked body
	// instead of buffering it. See WithStreamingResponse.
	streaming bool
	// compressed holds the content types whose bodies are worth gzipping. See
	// WithStreamingResponse.
	compressed set.Set[string]
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

// WithStreamingResponse writes the response body as a chunked stream driven by
// Encoder.EncodeStream. An encoder that fails before it writes is still an ordinary
// error response. Once it writes, the status is committed, so the encoder owns the
// in-band failure contract. Only use it for responses too large to buffer.
//
// Bodies from the given encoders are gzipped when the request accepts that encoding;
// every other body is written as is. Name only encoders whose output compression
// meaningfully shrinks, since a body that does not compress pays CPU on both ends for
// nothing.
func WithStreamingResponse(compressed ...http.Encoder) UnaryServerOption {
	return func(o *unaryServerOptions) {
		o.streaming = true
		o.compressed = set.New[string]()
		for _, e := range compressed {
			o.compressed.Add(e.ContentType())
		}
	}
}

func newUnaryServerOptions(opts []UnaryServerOption) unaryServerOptions {
	so := unaryServerOptions{
		requestDecoders:  defaultDecoders,
		responseEncoders: defaultEncoders,
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

type unaryServer[RQ, RS freighter.Payload] struct {
	alamos.Instrumentation
	unaryServerOptions
	// serverCtx outlives the request, so a streamed body can keep using it after the
	// fiber context is recycled.
	serverCtx context.Context
	handle    freighter.UnaryHandler[RQ, RS]
	path      string
	freighter.MiddlewareCollector
}

// Report describes the unary server's protocol and the content types it accepts on
// requests vs emits on responses. Accept and emit lists may differ — e.g. an import
// endpoint that accepts JSON|YAML|TOML but emits only JSON|MessagePack.
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
	var (
		res     RS
		handled bool
	)
	oMD, err := s.Exec(
		parseRequestCtx(fCtx.RequestCtx(), fCtx, address.Address(fCtx.Path()), false),
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
			handled = err == nil
			return oCtx, err
		}),
	)
	// A middleware can fail after the handler succeeds. No encoder then receives res,
	// so nothing else releases what it holds.
	if closer, ok := any(res).(io.Closer); ok && handled && err != nil {
		err = errors.Combine(err, closer.Close())
	}
	fErr := errors.Encode(fCtx.RequestCtx(), err, false)
	if fErr.Type == errors.TypeNil {
		setResponseCtx(fCtx, oMD)
		if s.streaming {
			return s.streamAndWrite(fCtx, encoder, res)
		}
		return encodeAndWrite(fCtx, encoder, res)
	}
	return s.writeError(fCtx, fErr)
}

// writeError answers the request with fErr as a 400.
func (s *unaryServer[RQ, RS]) writeError(fCtx fiber.Ctx, fErr errors.Payload) error {
	errEncoder := resolveErrorEncoder(fCtx)
	fCtx.Set(fiber.HeaderContentType, errEncoder.ContentType())
	fCtx.Status(fiber.StatusBadRequest)
	return encodeAndWrite(fCtx, errEncoder, fErr)
}

// Errors skip the route's response encoders, which may carry only one payload shape.
func resolveErrorEncoder(fCtx fiber.Ctx) http.Encoder {
	offers := lo.Map(defaultEncoders, func(e http.Encoder, _ int) string {
		return e.ContentType()
	})
	if matched := fCtx.Accepts(offers...); matched != "" {
		for _, e := range defaultEncoders {
			if e.ContentType() == matched {
				return e
			}
		}
	}
	return defaultEncoders[0]
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

// streamAndWrite writes v as a chunked body, gzipping it when the route names the
// resolved encoder as compressed and the request accepts that encoding. It commits the
// status when the first bytes reach the body, so an encoder that fails before then gets
// a 400. The writer can outlive the handler, which recycles c on return, so the closure
// only captures values read here.
func (s *unaryServer[RQ, RS]) streamAndWrite(
	c fiber.Ctx,
	encoder http.Encoder,
	v any,
) error {
	ctx := s.serverCtx
	var gz *gzip.Writer
	if s.compressed.Contains(encoder.ContentType()) && acceptsGzip(c) {
		var err error
		// Compression competes with the encoder for CPU on a body large enough to need
		// streaming, so take the fastest level over the smallest output.
		if gz, err = gzip.NewWriterLevel(nil, gzip.BestSpeed); err != nil {
			return err
		}
	}
	var (
		pr, pw  = io.Pipe()
		started = make(chan struct{})
		done    = make(chan error, 1)
	)
	go func() {
		w := bufio.NewWriter(&startWriter{Writer: pw, started: started})
		var dst io.Writer = w
		if gz != nil {
			gz.Reset(w)
			dst = gz
		}
		err := encoder.EncodeStream(ctx, dst, v)
		if gz != nil && err == nil {
			err = gz.Close()
		}
		if err == nil {
			err = w.Flush()
		}
		if err != nil {
			select {
			case <-started:
				s.L.Error("failed to write streamed response", zap.Error(err))
			default:
				// The handler sends the failure to the client as a 400.
			}
		}
		done <- err
		// Once bytes reach the body, the status is committed. A pipe closed with an
		// error makes the server drop the connection before the chunk that ends the
		// body, so the client sees the body cut short instead of complete.
		if cErr := pw.CloseWithError(err); cErr != nil {
			s.L.Error("failed to close streamed response", zap.Error(cErr))
		}
	}()
	select {
	case <-started:
	case err := <-done:
		if err != nil {
			return s.writeError(c, errors.Encode(c.RequestCtx(), err, false))
		}
	}
	if gz != nil {
		c.Set(fiber.HeaderContentEncoding, gzipEncoding)
	}
	return c.SendStream(pr)
}

// startWriter closes started on the first write that carries bytes.
type startWriter struct {
	io.Writer
	started chan struct{}
	once    sync.Once
}

// Write implements io.Writer.
func (w *startWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		w.once.Do(func() { close(w.started) })
	}
	return w.Writer.Write(p)
}

const gzipEncoding = "gzip"

// acceptsGzip reports whether the request's Accept-Encoding accepts gzip. Fiber weighs
// q-values and wildcards, but it accepts any coding when the header is missing, and a
// client that never asked for gzip may not decode it.
func acceptsGzip(c fiber.Ctx) bool {
	return c.Get(fiber.HeaderAcceptEncoding) != "" &&
		c.AcceptsEncodings(gzipEncoding) == gzipEncoding
}

// encodeAndWrite encodes v straight into the response rather than into a buffer the
// caller then copies, so a large response body is written once.
func encodeAndWrite(c fiber.Ctx, encoder http.Encoder, v any) error {
	return encoder.EncodeStream(c.RequestCtx(), c, v)
}
