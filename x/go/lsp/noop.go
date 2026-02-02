// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package lsp provides shared utilities for implementing Language Server Protocol servers.
package lsp

import (
	"context"

	"go.lsp.dev/protocol"
)

// NoopServer provides stub implementations for all LSP methods that return nil.
// Embed this struct in your server implementation to inherit all stubs, then
// override only the methods you need.
//
// Example:
//
//	type MyServer struct {
//	    lsp.NoopServer
//	    // your fields...
//	}
//
//	func (s *MyServer) Hover(...) (*protocol.Hover, error) {
//	    // your implementation
//	}
type NoopServer struct{}

func (NoopServer) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, nil
}

func (NoopServer) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, nil
}

func (NoopServer) CodeLensResolve(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, nil
}

func (NoopServer) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, nil
}

func (NoopServer) CompletionResolve(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, nil
}

func (NoopServer) Declaration(context.Context, *protocol.DeclarationParams) ([]protocol.Location, error) {
	return nil, nil
}

func (NoopServer) Definition(context.Context, *protocol.DefinitionParams) ([]protocol.Location, error) {
	return nil, nil
}

func (NoopServer) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return nil
}

func (NoopServer) DidChangeWatchedFiles(context.Context, *protocol.DidChangeWatchedFilesParams) error {
	return nil
}

func (NoopServer) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return nil
}

func (NoopServer) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return nil
}

func (NoopServer) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return nil
}

func (NoopServer) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return nil
}

func (NoopServer) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func (NoopServer) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, nil
}

func (NoopServer) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, nil
}

func (NoopServer) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, nil
}

func (NoopServer) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (NoopServer) DocumentLinkResolve(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, nil
}

func (NoopServer) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (interface{}, error) {
	return nil, nil
}

func (NoopServer) Exit(context.Context) error {
	return nil
}

func (NoopServer) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, nil
}

func (NoopServer) FoldingRanges(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, nil
}

func (NoopServer) Formatting(context.Context, *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (NoopServer) Implementation(context.Context, *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, nil
}

func (NoopServer) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, nil
}

func (NoopServer) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, nil
}

func (NoopServer) LogTrace(context.Context, *protocol.LogTraceParams) error {
	return nil
}

func (NoopServer) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, nil
}

func (NoopServer) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (NoopServer) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, nil
}

func (NoopServer) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, nil
}

func (NoopServer) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.Range, error) {
	return nil, nil
}

func (NoopServer) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (NoopServer) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, nil
}

func (NoopServer) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (NoopServer) ResolveCodeAction(context.Context, *protocol.CodeAction) (*protocol.CodeAction, error) {
	return nil, nil
}

func (NoopServer) ResolveCompletionItem(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, nil
}

func (NoopServer) ResolveDocumentLink(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, nil
}

func (NoopServer) SelectionRange(context.Context, *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	return nil, nil
}

func (NoopServer) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (interface{}, error) {
	return nil, nil
}

func (NoopServer) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, nil
}

func (NoopServer) SemanticTokensRefresh(context.Context) error {
	return nil
}

func (NoopServer) SetTrace(context.Context, *protocol.SetTraceParams) error {
	return nil
}

func (NoopServer) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, nil
}

func (NoopServer) Symbols(context.Context, *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, nil
}

func (NoopServer) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, nil
}

func (NoopServer) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (NoopServer) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (NoopServer) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (NoopServer) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return nil
}

func (NoopServer) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (NoopServer) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return nil
}

func (NoopServer) CodeLensRefresh(context.Context) error {
	return nil
}

func (NoopServer) Request(context.Context, string, interface{}) (interface{}, error) {
	return nil, nil
}

func (NoopServer) ShowDocument(context.Context, *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, nil
}
