// Copyright 2025 Synnax Labs, Inc.
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

// Stub implementations for unimplemented LSP methods

func (s *Server) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, nil
}

func (s *Server) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, nil
}

func (s *Server) CodeLensResolve(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, nil
}

func (s *Server) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, nil
}

func (s *Server) CompletionResolve(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, nil
}

func (s *Server) Declaration(context.Context, *protocol.DeclarationParams) ([]protocol.Location, error) {
	return nil, nil
}

func (s *Server) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return nil
}

func (s *Server) DidChangeWatchedFiles(context.Context, *protocol.DidChangeWatchedFilesParams) error {
	return nil
}

func (s *Server) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return nil
}

func (s *Server) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return nil
}

func (s *Server) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return nil
}

func (s *Server) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return nil
}

func (s *Server) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func (s *Server) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, nil
}

func (s *Server) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, nil
}

func (s *Server) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, nil
}

func (s *Server) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (s *Server) DocumentLinkResolve(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, nil
}

func (s *Server) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (interface{}, error) {
	return nil, nil
}

func (s *Server) Exit(context.Context) error {
	return nil
}

func (s *Server) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, nil
}

func (s *Server) FoldingRanges(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, nil
}

func (s *Server) Formatting(context.Context, *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (s *Server) Implementation(context.Context, *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, nil
}

func (s *Server) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, nil
}

func (s *Server) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, nil
}

func (s *Server) LogTrace(context.Context, *protocol.LogTraceParams) error {
	return nil
}

func (s *Server) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, nil
}

func (s *Server) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (s *Server) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, nil
}

func (s *Server) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, nil
}

func (s *Server) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.Range, error) {
	return nil, nil
}

func (s *Server) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (s *Server) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, nil
}

func (s *Server) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *Server) ResolveCodeAction(context.Context, *protocol.CodeAction) (*protocol.CodeAction, error) {
	return nil, nil
}

func (s *Server) ResolveCompletionItem(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, nil
}

func (s *Server) ResolveDocumentLink(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, nil
}

func (s *Server) SelectionRange(context.Context, *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	return nil, nil
}

// SemanticTokensFull is implemented in semantic.go

func (s *Server) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (interface{}, error) {
	return nil, nil
}

func (s *Server) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, nil
}

func (s *Server) SemanticTokensRefresh(context.Context) error {
	return nil
}

func (s *Server) SetTrace(context.Context, *protocol.SetTraceParams) error {
	return nil
}

func (s *Server) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, nil
}

func (s *Server) Symbols(context.Context, *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, nil
}

func (s *Server) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, nil
}

func (s *Server) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *Server) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *Server) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *Server) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return nil
}

func (s *Server) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (s *Server) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return nil
}

func (s *Server) CodeLensRefresh(context.Context) error {
	return nil
}

func (s *Server) Request(context.Context, string, interface{}) (interface{}, error) {
	return nil, nil
}

func (s *Server) ShowDocument(context.Context, *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, nil
}
