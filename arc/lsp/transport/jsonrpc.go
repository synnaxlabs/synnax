// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transport

import (
	"context"
	"io"

	"github.com/synnaxlabs/arc/lsp"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

func ServeJSONRPC(
	ctx context.Context,
	server *lsp.Server,
	comms io.ReadWriteCloser,
) error {
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(comms))
	client := protocol.ClientDispatcher(conn, zap.NewNop())
	server.SetClient(client)
	conn.Go(ctx, protocol.ServerHandler(server, nil))
	<-conn.Done()
	return conn.Err()
}
