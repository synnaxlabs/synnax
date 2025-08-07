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
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// UnaryClientConfig is a configuration for a unary HTTP client.
type UnaryClientConfig struct {
	Decoder
	Encoder
}

var _ config.Config[UnaryClientConfig] = UnaryClientConfig{}

// Validate validates the UnaryClientConfig.
func (c UnaryClientConfig) Validate() error {
	v := validate.New("fhttp.UnaryClientConfig")
	validate.NotNil(v, "decoder", c.Decoder)
	validate.NotNil(v, "encoder", c.Encoder)
	return v.Error()
}

// Override overrides valid fields with the fields in the other config.
func (c UnaryClientConfig) Override(other UnaryClientConfig) UnaryClientConfig {
	c.Decoder = override.Nil(c.Decoder, other.Decoder)
	c.Encoder = override.Nil(c.Encoder, other.Encoder)
	return c
}

// DefaultUnaryClientConfig is a default configuration for an HTTP client using JSON for
// both encoding and decoding.
var DefaultUnaryClientConfig = UnaryClientConfig{Decoder: JSONCodec, Encoder: JSONCodec}

type unaryClient[RQ, RS freighter.Payload] struct {
	UnaryClientConfig
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ freighter.UnaryClient[any, any] = (*unaryClient[any, any])(nil)

// NewUnaryClient returns a new freighter unary client that uses HTTP as the transport.
func NewUnaryClient[RQ, RS freighter.Payload](
	cfgs ...UnaryClientConfig,
) (freighter.UnaryClient[RQ, RS], error) {
	cfg, err := config.New(DefaultUnaryClientConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	encodings := []string{cfg.Encoder.ContentType(), cfg.Decoder.ContentType()}
	return &unaryClient[RQ, RS]{
		Reporter:          newReporter(encodings...),
		UnaryClientConfig: cfg,
	}, nil
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
			b, err := uc.Encode(inCtx, req)
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
			httpReq.Header.Set(fiber.HeaderContentType, uc.Encoder.ContentType())
			httpReq.Header.Set(fiber.HeaderAccept, uc.Decoder.ContentType())
			httpRes, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				return freighter.Context{}, err
			}
			if contentType := httpRes.Header.Get(
				fiber.HeaderContentType,
			); contentType != uc.Decoder.ContentType() {
				return freighter.Context{}, errors.Newf(
					"unexpected response content type: %s, expected: %s",
					contentType,
					uc.Decoder.ContentType(),
				)
			}
			outCtx := parseResponseCtx(httpRes, target)
			if httpRes.StatusCode >= http.StatusOK &&
				httpRes.StatusCode < fiber.StatusMultipleChoices {
				if err := uc.DecodeStream(inCtx, httpRes.Body, &res); err != nil {
					return freighter.Context{}, err
				}
				return outCtx, nil
			}
			var pld errors.Payload
			if err := uc.DecodeStream(inCtx, httpRes.Body, &pld); err != nil {
				return freighter.Context{}, err
			}
			if err := errors.Decode(ctx, pld); err != nil {
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
