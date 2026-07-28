// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"context"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// codec matches the connection codec protocol.NewServer installs: raw messages pass
// through verbatim, typed payloads use the union-aware protocol marshaling.
type codec struct{}

var _ jsonrpc2.Codec = codec{}

// Marshal implements jsonrpc2.Codec.
func (codec) Marshal(v any) ([]byte, error) {
	switch m := v.(type) {
	case jsonrpc2.RawMessage:
		if m == nil {
			return []byte("null"), nil
		}
		return m, nil
	case *jsonrpc2.RawMessage:
		if m == nil || *m == nil {
			return []byte("null"), nil
		}
		return *m, nil
	}
	return protocol.Marshal(v)
}

// Unmarshal implements jsonrpc2.Codec.
func (codec) Unmarshal(data []byte, v any) error {
	if p, ok := v.(*jsonrpc2.RawMessage); ok {
		b := make(jsonrpc2.RawMessage, len(data))
		copy(b, data)
		*p = b
		return nil
	}
	return protocol.Unmarshal(data, v)
}

// asyncCalls releases calls for concurrent handling; notifications run inline on the
// read loop, in wire order.
func asyncCalls(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if req.IsCall() {
			jsonrpc2.Async(ctx)
		}
		return handler(ctx, req)
	}
}

// NewConn wires server onto stream and returns the serving connection and a client
// for server-initiated messages. Unlike protocol.NewServer, whose AsyncHandler
// releases every message for concurrent handling, notifications are handled inline
// in wire order — an incremental didChange must splice against the text produced by
// its predecessor — while calls still run concurrently.
func NewConn(
	ctx context.Context,
	server protocol.Server,
	stream jsonrpc2.Stream,
) (jsonrpc2.Conn, protocol.Client) {
	conn := jsonrpc2.NewConn(stream, jsonrpc2.WithCodec(codec{}))
	client := protocol.ClientDispatcher(conn)
	conn.Go(ctx, protocol.CancelHandler(
		asyncCalls(protocol.ServerHandler(server, jsonrpc2.MethodNotFoundHandler)),
	))
	return conn, client
}
