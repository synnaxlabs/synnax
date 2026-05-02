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
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// UnaryClientConfig configures a unary HTTP client built by NewUnaryClient.
type UnaryClientConfig struct {
	// Encoder encodes outgoing requests. Sets the Content-Type header.
	//
	// [OPTIONAL] - Defaults to MessagePack.
	Encoder Encoder
	// Decoders are the codecs the client can decode responses from. Drives the Accept
	// header for content negotiation; the response is decoded with whichever Decoder
	// matches the response Content-Type.
	//
	// [OPTIONAL] - Defaults to JSON and MessagePack.
	Decoders []Decoder
}

// Validate implements config.Config.
func (c UnaryClientConfig) Validate() error {
	v := validate.New("http.unary_client")
	validate.NotNil(v, "encoder", c.Encoder)
	v.Ternary("decoders", len(c.Decoders) == 0, "at least one decoder is required")
	return v.Error()
}

// Override implements config.Config.
func (c UnaryClientConfig) Override(other UnaryClientConfig) UnaryClientConfig {
	c.Encoder = override.Nil(c.Encoder, other.Encoder)
	c.Decoders = override.Slice(c.Decoders, other.Decoders)
	return c
}

// NewUnaryClient builds a freighter.UnaryClient using the merged config (left to right)
// layered on top of the defaults. Returns an error if the merged config fails to
// validate. The client encodes outgoing requests with Encoder, advertises Decoders via
// the Accept header, and dispatches the response on its Content-Type to pick a decoder.
func NewUnaryClient[RQ, RS freighter.Payload](
	configs ...UnaryClientConfig,
) (freighter.UnaryClient[RQ, RS], error) {
	cfg, err := config.New(UnaryClientConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	return &unaryClient[RQ, RS]{
		encoder:      cfg.Encoder,
		decoders:     cfg.Decoders,
		acceptHeader: buildAcceptHeader(cfg.Decoders),
	}, nil
}

type unaryClient[RQ, RS freighter.Payload] struct {
	encoder      Encoder
	decoders     []Decoder
	acceptHeader string
	freighter.MiddlewareCollector
}

// Report describes the unary client's protocol, the content type it sends on requests,
// and the content types it can decode from responses.
func (u *unaryClient[RQ, RS]) Report() alamos.Report {
	return alamos.Report{
		"protocol":             unaryProtocol,
		"sentContentType":      u.encoder.ContentType(),
		"acceptedContentTypes": lo.Map(u.decoders, func(d Decoder, _ int) string { return d.ContentType() }),
	}
}

func (u *unaryClient[RQ, RS]) resolveResponseDecoder(contentType string) (Decoder, error) {
	for _, d := range u.decoders {
		if d.ContentType() == contentType {
			return d, nil
		}
	}
	return nil, errors.Newf(
		"[encoding] - no decoder for response content type %q",
		contentType,
	)
}

func buildAcceptHeader(decoders []Decoder) string {
	cts := make([]string, len(decoders))
	for i, d := range decoders {
		cts[i] = d.ContentType()
	}
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
		) (freighter.Context, error) {
			b, err := u.encoder.Encode(inCtx, req)
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
			httpReq.Header.Set(fiber.HeaderContentType, u.encoder.ContentType())
			httpReq.Header.Set(fiber.HeaderAccept, u.acceptHeader)

			httpRes, err := (&http.Client{}).Do(httpReq)
			outCtx := parseResponseCtx(httpRes, target)
			if err != nil {
				return outCtx, err
			}

			decoder, err := u.resolveResponseDecoder(httpRes.Header.Get(fiber.HeaderContentType))
			if err != nil {
				return outCtx, err
			}

			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld errors.Payload
				if err := decoder.DecodeStream(outCtx, httpRes.Body, &pld); err != nil {
					return outCtx, err
				}
				return outCtx, errors.Decode(ctx, pld)
			}
			return outCtx, decoder.DecodeStream(outCtx, httpRes.Body, &res)
		}),
	)
	return res, err
}
