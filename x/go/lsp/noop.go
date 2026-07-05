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

	"go.lsp.dev/protocol"
)

// NoopServer provides stub implementations for all protocol.Server methods.
// Embed this in your server struct and override only the methods you need.
//
// Unimplemented requests answer with a method-not-found error via the embedded
// protocol.UnimplementedServer. Notifications are overridden to succeed
// silently instead: jsonrpc2 treats a handler error on a notification as a
// connection-level failure, so an unhandled notification (e.g. $/setTrace)
// must not surface an error.
type NoopServer struct{ protocol.UnimplementedServer }

func (NoopServer) Initialize(context.Context, *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{}, nil
}

func (NoopServer) Initialized(context.Context, *protocol.InitializedParams) error {
	return nil
}

func (NoopServer) Shutdown(context.Context) error { return nil }

func (NoopServer) Exit(context.Context) error { return nil }

func (NoopServer) SetTrace(context.Context, *protocol.SetTraceParams) error { return nil }

func (NoopServer) Progress(context.Context, *protocol.ProgressParams) error { return nil }

func (NoopServer) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return nil
}

func (NoopServer) DidOpen(context.Context, *protocol.DidOpenTextDocumentParams) error {
	return nil
}

func (NoopServer) DidChange(context.Context, *protocol.DidChangeTextDocumentParams) error {
	return nil
}

func (NoopServer) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return nil
}

func (NoopServer) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func (NoopServer) DidClose(context.Context, *protocol.DidCloseTextDocumentParams) error {
	return nil
}

func (NoopServer) DidOpenNotebookDocument(context.Context, *protocol.DidOpenNotebookDocumentParams) error {
	return nil
}

func (NoopServer) DidChangeNotebookDocument(context.Context, *protocol.DidChangeNotebookDocumentParams) error {
	return nil
}

func (NoopServer) DidSaveNotebookDocument(context.Context, *protocol.DidSaveNotebookDocumentParams) error {
	return nil
}

func (NoopServer) DidCloseNotebookDocument(context.Context, *protocol.DidCloseNotebookDocumentParams) error {
	return nil
}

func (NoopServer) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return nil
}

func (NoopServer) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return nil
}

func (NoopServer) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return nil
}

func (NoopServer) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return nil
}

func (NoopServer) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return nil
}

func (NoopServer) DidChangeWatchedFiles(context.Context, *protocol.DidChangeWatchedFilesParams) error {
	return nil
}

// DocumentSymbol returns an empty result rather than method-not-found so
// embedders can advertise DocumentSymbolProvider without implementing it.
func (NoopServer) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) (protocol.DocumentSymbolResult, error) {
	return nil, nil
}
