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
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// UnaryClientConfig configures a unary HTTP client built by NewUnaryClient.
type UnaryClientConfig struct {
	// Encoder encodes outgoing requests. Sets the Content-Type header.
	//
	// [REQUIRED]
	Encoder xhttp.Encoder
	// Decoders are the codecs the client can decode responses from. Drives the Accept
	// header for content negotiation; the response is decoded with whichever Decoder
	// matches the response Content-Type.
	//
	// [REQUIRED] - At least one decoder must be supplied.
	Decoders []xhttp.Decoder
	// Compressions are the Content-Encodings the client compresses requests with and
	// accepts on responses, ordered by preference. The client compresses a request with
	// the first entry and advertises all of them via the Accept-Encoding header.
	//
	// [OPTIONAL] - Defaults to zstd, brotli, and gzip.
	Compressions []xhttp.Compression
	// DisableCompression stops the client compressing requests and stops it
	// advertising Accept-Encoding, so responses arrive uncompressed too.
	//
	// [OPTIONAL] - Defaults to false.
	DisableCompression bool
	// MinCompressSize is the smallest request body the client compresses. Bodies below
	// it are sent uncompressed, since compressing them costs more CPU than the
	// transmission time it saves.
	//
	// [OPTIONAL] - Defaults to 1 kilobyte.
	MinCompressSize int
	// MaxDecompressedSize bounds how far a compressed response body may expand before
	// the client rejects it with xhttp.ErrBodyTooLarge.
	//
	// [OPTIONAL] - Defaults to 64 megabytes.
	MaxDecompressedSize int
}

// Validate implements config.Config.
func (c UnaryClientConfig) Validate() error {
	v := validate.New("http.unary_client")
	validate.NotNil(v, "encoder", c.Encoder)
	validate.NotEmptySlice(v, "decoders", c.Decoders)
	validate.GreaterThanEq(v, "min_compress_size", c.MinCompressSize, 0)
	validate.GreaterThanEq(v, "max_decompressed_size", c.MaxDecompressedSize, 0)
	return v.Error()
}

// Override implements config.Config.
func (c UnaryClientConfig) Override(other UnaryClientConfig) UnaryClientConfig {
	c.Encoder = override.Nil(c.Encoder, other.Encoder)
	c.Decoders = override.Slice(c.Decoders, other.Decoders)
	c.Compressions = override.Slice(c.Compressions, other.Compressions)
	c.DisableCompression = c.DisableCompression || other.DisableCompression
	c.MinCompressSize = override.Numeric(c.MinCompressSize, other.MinCompressSize)
	c.MaxDecompressedSize = override.Numeric(
		c.MaxDecompressedSize, other.MaxDecompressedSize,
	)
	return c
}

// NewUnaryClient builds a freighter.UnaryClient using the merged config (left to right)
// layered on top of the defaults. Returns an error if the merged config fails to
// validate. The client encodes outgoing requests with Encoder, advertises Decoders via
// the Accept header, and dispatches the response on its Content-Type to pick a decoder.
// Request bodies at or above MinCompressSize are compressed with the first entry in
// Compressions, and responses are decompressed according to their Content-Encoding.
func NewUnaryClient[RQ, RS freighter.Payload](
	configs ...UnaryClientConfig,
) (freighter.UnaryClient[RQ, RS], error) {
	cfg, err := config.New(UnaryClientConfig{
		Encoder:             json.Codec,
		Decoders:            []xhttp.Decoder{json.Codec, msgpack.Codec},
		Compressions:        defaultCompressions,
		MinCompressSize:     defaultMinCompressSize,
		MaxDecompressedSize: defaultMaxDecompressedSize,
	}, configs...)
	if err != nil {
		return nil, err
	}
	compression := compressionOptions{
		compressions:        cfg.Compressions,
		minCompressSize:     cfg.MinCompressSize,
		maxDecompressedSize: cfg.MaxDecompressedSize,
	}
	if cfg.DisableCompression {
		compression.compressions = nil
	}
	return &unaryClient[RQ, RS]{
		compressionOptions:   compression,
		encoder:              cfg.Encoder,
		decoders:             cfg.Decoders,
		acceptHeader:         buildAcceptHeader(cfg.Decoders),
		acceptEncodingHeader: compression.acceptEncodingHeader(),
	}, nil
}

type unaryClient[RQ, RS freighter.Payload] struct {
	compressionOptions
	encoder      xhttp.Encoder
	decoders     []xhttp.Decoder
	acceptHeader string
	// acceptEncodingHeader is the rendered Accept-Encoding value, empty when
	// compression is disabled.
	acceptEncodingHeader string
	freighter.MiddlewareCollector
}

// Report describes the unary client's protocol, the content type it sends on requests,
// the content types it can decode from responses, and the content encodings it can
// compress and decompress.
func (u *unaryClient[RQ, RS]) Report() alamos.Report {
	return alamos.Report{
		"protocol":        unaryProtocol,
		"sentContentType": u.encoder.ContentType(),
		"acceptedContentTypes": lo.Map(u.decoders, func(d xhttp.Decoder, _ int) string {
			return d.ContentType()
		}),
		"contentEncodings": u.contentEncodings(),
	}
}

func (u *unaryClient[RQ, RS]) resolveResponseDecoder(
	contentType string,
) (xhttp.Decoder, error) {
	for _, d := range u.decoders {
		if d.ContentType() == contentType {
			return d, nil
		}
	}
	return nil, errors.Newf("no decoder for response content type %q", contentType)
}

func buildAcceptHeader(decoders []xhttp.Decoder) string {
	cts := lo.Map(decoders, func(d xhttp.Decoder, _ int) string {
		return d.ContentType()
	})
	return strings.Join(cts, ", ")
}

func (u *unaryClient[RQ, RS]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (RS, error) {
	var res RS
	_, err := u.Exec(
		freighter.Context{
			Context:  ctx,
			Protocol: unaryProtocol,
			Target:   target,
		},
		freighter.FinalizerFunc(func(
			inCtx freighter.Context,
		) (_ freighter.Context, err error) {
			b, err := u.encoder.Encode(inCtx, req)
			if err != nil {
				return freighter.Context{}, err
			}
			b, contentEncoding, err := u.compressPreferred(b)
			if err != nil {
				return freighter.Context{}, err
			}
			httpReq, err := http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				"http://"+target.String(),
				bytes.NewReader(b),
			)
			if err != nil {
				return freighter.Context{}, err
			}
			setRequestCtx(httpReq, inCtx)
			// This client is only used in tests; disable connection keep-alive so the
			// underlying connection (and its net/http reader/writer goroutines) is torn
			// down when the response is read rather than lingering in the idle pool.
			httpReq.Close = true
			httpReq.Header.Set(fiber.HeaderContentType, u.encoder.ContentType())
			httpReq.Header.Set(fiber.HeaderAccept, u.acceptHeader)
			if contentEncoding != "" {
				httpReq.Header.Set(fiber.HeaderContentEncoding, contentEncoding)
			}
			// Setting Accept-Encoding explicitly stops net/http adding its own and
			// transparently decompressing the response, leaving the body for
			// decompressIncoming below.
			if u.acceptEncodingHeader != "" {
				httpReq.Header.Set(fiber.HeaderAcceptEncoding, u.acceptEncodingHeader)
			}

			httpRes, err := (&http.Client{}).Do(httpReq)
			if err != nil {
				return freighter.Context{Target: target}, err
			}
			defer func() { err = errors.Combine(err, httpRes.Body.Close()) }()
			outCtx := parseResponseCtx(httpRes, target, false)

			decoder, err := u.resolveResponseDecoder(
				httpRes.Header.Get(fiber.HeaderContentType),
			)
			if err != nil {
				return outCtx, err
			}
			body, err := io.ReadAll(httpRes.Body)
			if err != nil {
				return outCtx, err
			}
			body, err = u.decompressIncoming(
				body, httpRes.Header.Get(fiber.HeaderContentEncoding),
			)
			if err != nil {
				return outCtx, err
			}

			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld errors.Payload
				if err := decoder.Decode(outCtx, body, &pld); err != nil {
					return outCtx, err
				}
				return outCtx, errors.Decode(ctx, pld)
			}
			return outCtx, decoder.Decode(outCtx, body, &res)
		}),
	)
	return res, err
}
