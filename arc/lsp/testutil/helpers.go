// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

// SetupTestServer creates a new LSP server configured for testing with a MockClient.
// Returns the server, a background context, and a default test URI.
func SetupTestServer(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI) {
	server := testutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	server.SetClient(&MockClient{})
	return server, uri
}

// OpenDocument is a helper to open a document in the LSP server.
func OpenDocument(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
) error {
	return server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "arc",
			Version:    1,
			Text:       content,
		},
	})
}
