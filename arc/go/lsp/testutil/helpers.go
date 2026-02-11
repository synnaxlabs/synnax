// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/x/lsp/testutil"
	xutil "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

// SetupTestServer creates a new arc LSP server with a MockClient for testing.
func SetupTestServer(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI) {
	server := xutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	server.SetClient(&MockClient{})
	return server, uri
}

// SetupTestServerWithClient creates a new arc LSP server and returns
// the server, URI, and the MockClient.
func SetupTestServerWithClient(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI, *MockClient) {
	server := xutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	client := &MockClient{}
	server.SetClient(client)
	return server, uri, client
}

// OpenArcDocument is a helper to open a document in the arc LSP server.
func OpenArcDocument(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
) {
	OpenDocument(server, ctx, uri, content, "arc")
}
