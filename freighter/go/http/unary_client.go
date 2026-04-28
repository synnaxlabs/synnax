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

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
)

// NewUnaryClient builds a freighter.UnaryClient using the merged config (left to right)
// layered on top of defaultClientConfig. Returns an error if the merged config fails to
// validate. The client uses the same codec for both Content-Type and the response body
// (no Accept negotiation); use a server-side configuration to negotiate alternate
// response codecs.
func NewUnaryClient[RQ, RS freighter.Payload](
	configs ...ClientConfig,
) (freighter.UnaryClient[RQ, RS], error) {
	cfg, err := config.New(defaultClientConfig, configs...)
	if err != nil {
		return nil, err
	}
	return &unaryClient[RQ, RS]{codec: cfg.Codec}, nil
}

type unaryClient[RQ, RS freighter.Payload] struct {
	codec Codec
	freighter.Reporter
	freighter.MiddlewareCollector
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
			Protocol: unaryReporter.Protocol,
			Target:   target,
		},
		freighter.FinalizerFunc(func(
			inCtx freighter.Context,
		) (freighter.Context, error) {
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
				if err := u.codec.DecodeStream(outCtx, httpRes.Body, &pld); err != nil {
					return outCtx, err
				}
				return outCtx, errors.Decode(ctx, pld)
			}
			return outCtx, u.codec.DecodeStream(outCtx, httpRes.Body, &res)
		}),
	)
	return res, err
}
