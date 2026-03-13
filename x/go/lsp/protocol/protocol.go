// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package protocol

import (
	"context"

	"go.uber.org/zap"

	"go.lsp.dev/jsonrpc2"
)

// NewServer returns the context in which client is embedded, jsonrpc2.Conn, and the Client.
func NewServer(ctx context.Context, server Server, stream jsonrpc2.Stream, logger *zap.Logger) (context.Context, jsonrpc2.Conn, Client) {
	conn := jsonrpc2.NewConn(stream)
	cliint := ClientDispatcher(conn, logger.Named("client"))
	ctx = WithClient(ctx, cliint)

	conn.Go(ctx,
		Handlers(
			ServerHandler(server, jsonrpc2.MethodNotFoundHandler),
		),
	)

	return ctx, conn, cliint
}

// NewClient returns the context in which Client is embedded, jsonrpc2.Conn, and the Server.
func NewClient(ctx context.Context, client Client, stream jsonrpc2.Stream, logger *zap.Logger) (context.Context, jsonrpc2.Conn, Server) {
	ctx = WithClient(ctx, client)

	conn := jsonrpc2.NewConn(stream)
	conn.Go(ctx,
		Handlers(
			ClientHandler(client, jsonrpc2.MethodNotFoundHandler),
		),
	)
	server := ServerDispatcher(conn, logger.Named("server"))

	return ctx, conn, server
}
