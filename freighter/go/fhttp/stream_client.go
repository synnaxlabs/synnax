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

	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// StreamClientConfig is a configuration for a streaming HTTP client.
type StreamClientConfig struct{ Codec }

var _ config.Config[StreamClientConfig] = StreamClientConfig{}

// Validate validates the StreamClientConfig.
func (c StreamClientConfig) Validate() error {
	v := validate.New("fhttp.StreamClientConfig")
	validate.NotNil(v, "codec", c.Codec)
	return v.Error()
}

// Override overrides valid fields with the fields in the other config.
func (c StreamClientConfig) Override(other StreamClientConfig) StreamClientConfig {
	c.Codec = override.Nil(c.Codec, other.Codec)
	return c
}

// DefaultStreamClientConfig is a default configuration for a streaming HTTP client,
// using JSON encoding
var DefaultStreamClientConfig = StreamClientConfig{Codec: JSONCodec}

type streamClient[RQ, RS freighter.Payload] struct {
	alamos.Instrumentation
	StreamClientConfig
	ws.Dialer
	freighter.Reporter
	freighter.MiddlewareCollector
}

// NewStreamClient returns a new freighter stream client that uses HTTP as the
// transport.
func NewStreamClient[RQ, RS freighter.Payload](
	cfgs ...StreamClientConfig,
) (freighter.StreamClient[RQ, RS], error) {
	cfg, err := config.New(DefaultStreamClientConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &streamClient[RQ, RS]{
		StreamClientConfig: cfg,
		Reporter: freighter.Reporter{
			Protocol:  "http",
			Encodings: []string{cfg.ContentType()},
		},
	}, nil
}

func (s *streamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	var stream freighter.ClientStream[RQ, RS]
	_, err := s.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   target,
			Protocol: s.Reporter.Protocol,
			Params:   make(freighter.Params),
		},
		func(fCtx freighter.Context) (freighter.Context, error) {
			fCtx.Params[fiber.HeaderContentType] = s.ContentType()
			fCtx.Params[fiber.HeaderAccept] = s.ContentType()
			conn, res, err := s.DialContext(
				fCtx,
				"ws://"+target.String(),
				ctxToHeaders(fCtx),
			)
			oCtx := parseResponseCtx(res, target)
			if err != nil {
				return oCtx, err
			}
			if res.StatusCode != fiber.StatusSwitchingProtocols {
				return oCtx, errors.New("[ws] - unable to upgrade connection")
			}
			core := newStreamCore[RS, RQ](
				coreConfig{
					conn:            conn,
					codec:           s.Codec,
					Instrumentation: s.Instrumentation,
				},
				fCtx.Done(),
			)
			msg, err := core.receiveRaw()
			if err != nil {
				return oCtx, err
			}
			if msg.Type != WSMessageTypeOpen {
				return oCtx, errors.Decode(fCtx, msg.Err)
			}
			stream = &clientStream[RQ, RS]{streamCore: core}
			return oCtx, nil
		},
	)
	return stream, err
}
