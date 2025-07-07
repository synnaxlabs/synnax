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
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
)

var pipelineReporter = freighter.Reporter{
	Protocol:  "http",
	Encodings: httputil.SupportedContentTypes(),
}

type pipelineServer[RQ freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	serverOptions
	path   string
	handle func(context.Context, RQ) (io.Reader, error)
}

var _ freighter.PipelineServer[any] = (*pipelineServer[any])(nil)

func (ps *pipelineServer[RQ]) BindHandler(handler func(context.Context, RQ) (io.Reader, error)) {
	ps.handle = handler
}

func (ps *pipelineServer[RQ]) fiberHandler(fiberCtx *fiber.Ctx) error {
	fiberCtx.Accepts(httputil.SupportedContentTypes()...)
	codec, err := httputil.ResolveCodec(fiberCtx.Get(fiber.HeaderContentType))
	if err != nil {
		return err
	}
	fiberCtx.Set(fiber.HeaderContentType, codec.ContentType())
	var reader io.Reader
	oMD, err := ps.MiddlewareCollector.Exec(
		parseRequestCtx(fiberCtx.Context(), fiberCtx, address.Address(fiberCtx.Path())),
		freighter.FinalizerFunc(func(inCtx freighter.Context) (freighter.Context, error) {
			var req RQ
			err := codec.Decode(inCtx, fiberCtx.BodyRaw(), &req)
			if err != nil {
				return inCtx, err
			}
			oCtx := freighter.Context{Protocol: inCtx.Protocol, Params: make(freighter.Params)}
			reader, err = ps.handle(inCtx, req)
			if err != nil {
				return oCtx, err
			}
			return oCtx, err
		}),
	)
	setResponseCtx(fiberCtx, oMD)
	fErr := errors.Encode(fiberCtx.Context(), err, false)
	if fErr.Type == errors.TypeNil {
		return fiberCtx.SendStream(reader)
	}
	fiberCtx.Status(fiber.StatusBadRequest)
	return encodeAndWrite(fiberCtx, codec, fErr)
}

type pipelineClient[RQ freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	codec httputil.Codec
}

var _ freighter.PipelineClient[any] = (*pipelineClient[any])(nil)

func (pc *pipelineClient[RQ]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (io.ReadCloser, error) {
	var reader io.ReadCloser
	if _, err := pc.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Protocol: pipelineReporter.Protocol,
			Target:   target,
		},
		freighter.FinalizerFunc(func(inCtx freighter.Context) (freighter.Context, error) {
			encoded, err := pc.codec.Encode(inCtx, req)
			if err != nil {
				return freighter.Context{}, err
			}
			httpReq, err := http.NewRequestWithContext(
				ctx,
				"POST",
				"http://"+target.String(),
				bytes.NewReader(encoded),
			)
			if err != nil {
				return freighter.Context{}, err
			}
			setRequestCtx(httpReq, inCtx)
			httpReq.Header.Set(fiber.HeaderContentType, pc.codec.ContentType())
			httpRes, err := (&http.Client{}).Do(httpReq)
			if err != nil {
				return freighter.Context{}, err
			}
			outCtx := parseResponseCtx(httpRes, target)
			if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
				var pld errors.Payload
				if err := pc.codec.DecodeStream(context.TODO(), httpRes.Body, &pld); err != nil {
					return outCtx, err
				}
				return outCtx, errors.Decode(ctx, pld)
			}
			reader = httpRes.Body
			return outCtx, nil
		}),
	); err != nil {
		return nil, err
	}
	return reader, nil
}
