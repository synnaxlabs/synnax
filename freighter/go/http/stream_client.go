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
	"go/types"
	"net/http"

	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

var (
	_ freighter.StreamClient[any, types.Nil] = (*streamClient[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil] = (*clientStream[any, types.Nil])(nil)
	_ config.Config[StreamClientConfig]      = StreamClientConfig{}
)

// StreamClientConfig configures a streaming HTTP client built by NewStreamClient.
type StreamClientConfig struct {
	// Codec is used for both encoding outgoing messages and decoding incoming messages
	// on the websocket connection. Websockets negotiate a single codec at upgrade time,
	// so there is no per-direction split.
	//
	// [OPTIONAL] - Defaults to MessagePack.
	Codec Codec
}

// Validate implements config.Config.
func (c StreamClientConfig) Validate() error {
	v := validate.New("http.stream_client")
	validate.NotNil(v, "codec", c.Codec)
	return v.Error()
}

// Override implements config.Config.
func (c StreamClientConfig) Override(other StreamClientConfig) StreamClientConfig {
	c.Codec = override.Nil(c.Codec, other.Codec)
	return c
}

// NewStreamClient builds a freighter.StreamClient using the merged config (left to
// right) layered on top of the default config. Returns an error if the merged config
// fails to validate. The returned client opens a websocket connection per call to
// Stream.
func NewStreamClient[RQ, RS freighter.Payload](
	configs ...StreamClientConfig,
) (freighter.StreamClient[RQ, RS], error) {
	cfg, err := config.New(StreamClientConfig{Codec: MsgPackCodec}, configs...)
	if err != nil {
		return nil, err
	}
	return &streamClient[RQ, RS]{codec: cfg.Codec}, nil
}

type streamClient[RQ, RS freighter.Payload] struct {
	alamos.Instrumentation
	codec  Codec
	dialer ws.Dialer
	freighter.MiddlewareCollector
}

// Report describes the stream client's protocol and the single content type it
// negotiates at websocket upgrade time.
func (s *streamClient[RQ, RS]) Report() alamos.Report {
	return alamos.Report{
		"protocol":  streamProtocol,
		"encodings": []string{s.codec.ContentType()},
	}
}

func (s *streamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	var stream freighter.ClientStream[RQ, RS]
	_, err := s.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   target,
			Protocol: streamProtocol,
			Params:   make(freighter.Params),
		},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			ctx.Params[fiber.HeaderContentType] = s.codec.ContentType()
			conn, res, err := s.dialer.DialContext(
				ctx, "ws://"+target.String(), mdToHeaders(ctx),
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
					codec:           s.codec,
					Instrumentation: s.Instrumentation,
				},
				ctx.Done(),
			)
			msg, err := core.receiveRaw()
			if err != nil {
				return oCtx, err
			}
			if msg.Type != WSMessageTypeOpen {
				return oCtx, errors.Decode(ctx, msg.Err)
			}
			stream = &clientStream[RQ, RS]{streamCore: core}
			return oCtx, nil
		}),
	)
	return stream, err
}

func mdToHeaders(md freighter.Context) http.Header {
	headers := make(http.Header, len(md.Params))
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			headers[k] = []string{vStr}
		}
	}
	return headers
}

type clientStream[RQ, RS freighter.Payload] struct {
	streamCore[RS, RQ]
	sendClosed bool
}

// Send implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Send(req RQ) error {
	if s.peerCloseErr != nil {
		return freighter.EOF
	}
	if s.sendClosed {
		return freighter.ErrStreamClosed
	}
	s.peerCloseErr = s.send(WSMessage[RQ]{Type: WSMessageTypeData, Payload: req})
	return s.peerCloseErr
}

func (s *clientStream[RQ, RS]) Receive() (RS, error) {
	pld, err := s.streamCore.Receive()
	if err != nil {
		return pld, errors.Combine(err, s.close())
	}
	return pld, nil
}

// CloseSend implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) CloseSend() error {
	if s.peerCloseErr != nil || s.sendClosed {
		return nil
	}
	s.sendClosed = true
	return s.send(WSMessage[RQ]{Type: WSMessageTypeClose})
}
