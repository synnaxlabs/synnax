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
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/lsp/protocol"
	"github.com/synnaxlabs/x/lsp/testutil"
	xutil "github.com/synnaxlabs/x/testutil"
)

// defaultNewRoot builds an STL-populated root with no dynamic resolver.
// Used as the default for test servers that don't override NewRoot.
func defaultNewRoot() *symbol.Symbol {
	root := symbol.NewRoot(nil, stl.NewSymbols())
	return root
}

// SetupTestServer creates a new arc LSP server with a MockClient for
// testing. If the supplied configs do not provide a NewRoot, a default
// is applied that builds an STL-populated root with no dynamic resolver.
func SetupTestServer(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI) {
	cfgs = withDefaultNewRoot(cfgs)
	server := xutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	server.SetClient(&testutil.MockClient{})
	return server, uri
}

// SetupTestServerWithClient creates a new arc LSP server and returns
// the server, URI, and the MockClient. NewRoot defaults as in
// SetupTestServer.
func SetupTestServerWithClient(
	cfgs ...lsp.Config,
) (*lsp.Server, protocol.DocumentURI, *testutil.MockClient) {
	cfgs = withDefaultNewRoot(cfgs)
	server := xutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	client := &testutil.MockClient{}
	server.SetClient(client)
	return server, uri, client
}

func withDefaultNewRoot(cfgs []lsp.Config) []lsp.Config {
	for _, c := range cfgs {
		if c.NewRoot != nil {
			return cfgs
		}
	}
	return append(cfgs, lsp.Config{NewRoot: defaultNewRoot})
}

// OpenArcDocument is a helper to open a document in the arc LSP server.
func OpenArcDocument(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
) {
	testutil.OpenDocument(server, ctx, uri, content, "arc")
}
