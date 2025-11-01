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
	"fmt"
	"sync"

	"github.com/synnaxlabs/alamos"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Config defines the configuration for opening an arc LSP Server.
type Config struct {
	// Instrumentation is used for logging, tracing, metrics, etc.
	alamos.Instrumentation
	// GlobalResolver allows the caller to define custom globals that will appear in
	// LSP auto-complete and type checking. Typically used to provide standard library
	// variables and functions as well as channels.
	GlobalResolver symbol.Resolver
}

var (
	_             config.Config[Config] = &Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.GlobalResolver = override.Nil(c.GlobalResolver, other.GlobalResolver)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error { return nil }

// Server implements the Language Server Protocol for arc
type Server struct {
	cfg          Config
	client       protocol.Client
	mu           sync.RWMutex
	documents    map[protocol.DocumentURI]*Document
	capabilities protocol.ServerCapabilities
}

var _ protocol.Server = (*Server)(nil)

// Document represents an open document
type Document struct {
	URI     protocol.DocumentURI
	Version int32
	Content string
	// IR with symbol table
	IR ir.IR
	// Diagnostics diagnostics
	Diagnostics diagnostics.Diagnostics
	Metadata    *DocumentMetadata // Metadata for calculated channels
}

// New creates a new LSP server
func New(cfgs ...Config) (*Server, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		documents: make(map[protocol.DocumentURI]*Document),
		capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
			},
			HoverProvider: true,
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{".", ":", "<", "-", ">"},
			},
			DefinitionProvider:     true,
			DocumentSymbolProvider: true,
			SemanticTokensProvider: map[string]interface{}{
				"legend": protocol.SemanticTokensLegend{
					TokenTypes: convertToSemanticTokenTypes(semanticTokenTypes),
				},
				"full": true,
			},
		},
	}, nil
}

// SetClient sets the LSP client for sending notifications
func (s *Server) SetClient(client protocol.Client) {
	s.client = client
}

// Logger returns the server's logger
func (s *Server) Logger() *zap.Logger {
	if s.cfg.L == nil {
		return zap.NewNop()
	}
	return s.cfg.L.Zap()
}

// getDocument retrieves a document from the server's cache by URI.
// Returns the document and true if found, or nil and false if not found.
// This method is thread-safe.
func (s *Server) getDocument(uri protocol.DocumentURI) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[uri]
	return doc, ok
}

// Helper functions to convert string slices to protocol types
func convertToSemanticTokenTypes(types []string) []protocol.SemanticTokenTypes {
	result := make([]protocol.SemanticTokenTypes, len(types))
	for i, t := range types {
		result[i] = protocol.SemanticTokenTypes(t)
	}
	return result
}

// Initialize handles the initialize request
func (s *Server) Initialize(_ context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	s.cfg.L.Debug("initializing arc lsp", zap.String("client", params.ClientInfo.Name))
	return &protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo:   &protocol.ServerInfo{Name: "arc-lsp", Version: "0.1.0"},
	}, nil
}

// Initialized handles the initialized notification
func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error {
	s.cfg.L.Debug("arc lsp initialized")
	return nil
}

// Shutdown handles the shutdown request
func (s *Server) Shutdown(_ context.Context) error {
	s.cfg.L.Info("Shutting down server")
	return nil
}

// DidOpen handles opening a document
func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("document opened", zap.String("uri", string(uri)))
	metadata := ExtractMetadataFromURI(uri)
	s.cfg.L.Debug("file meta-data",
		zap.String("uri", string(uri)),
		zap.Bool("hasMetadata", metadata != nil),
		zap.Bool("isBlock", metadata != nil && metadata.IsFunctionBlock))
	s.mu.Lock()
	s.documents[uri] = &Document{
		URI:      uri,
		Version:  params.TextDocument.Version,
		Content:  params.TextDocument.Text,
		Metadata: metadata,
	}
	s.mu.Unlock()

	s.publishDiagnostics(ctx, uri, params.TextDocument.Text)

	return nil
}

// DidChange handles document changes
func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document changed", zap.String("uri", string(uri)))
	s.mu.Lock()
	if doc, ok := s.documents[uri]; ok {
		if len(params.ContentChanges) > 0 {
			doc.Version = params.TextDocument.Version
			doc.Content = params.ContentChanges[0].Text
		}
	}
	s.mu.Unlock()
	s.mu.RLock()
	content := ""
	if doc, ok := s.documents[uri]; ok {
		content = doc.Content
	}
	s.mu.RUnlock()
	s.publishDiagnostics(ctx, uri, content)
	return nil
}

// DidClose handles closing a document
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document closed", zap.String("uri", string(uri)))

	s.mu.Lock()
	delete(s.documents, uri)
	s.mu.Unlock()

	return s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})
}

// publishDiagnostics parses the document and publishes syntax and semantic errors
func (s *Server) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, content string) {
	s.mu.Lock()
	doc, ok := s.documents[uri]
	s.mu.Unlock()
	s.cfg.L.Debug("analyzing program")
	if !ok {
		return
	}

	var pDiagnostics []protocol.Diagnostic
	if doc.Metadata.IsFunctionBlock {
		t, err := parser.ParseBlock(fmt.Sprintf("{%s}", content))
		if err != nil {
			pDiagnostics = translateDiagnostics(*err)
		} else {
			aCtx := acontext.CreateRoot[parser.IBlockContext](
				ctx,
				t,
				s.cfg.GlobalResolver,
			)
			statement.AnalyzeFunctionBody(aCtx)
			doc.IR = ir.IR{Symbols: aCtx.Scope}
			doc.Diagnostics = *aCtx.Diagnostics
			pDiagnostics = translateDiagnostics(*aCtx.Diagnostics)
		}
	} else {
		t, diag := text.Parse(text.Text{Raw: content})
		if diag != nil {
			pDiagnostics = translateDiagnostics(*diag)
		} else {
			analyzedIR, analysisDiag := text.Analyze(ctx, t, s.cfg.GlobalResolver)
			doc.IR = analyzedIR
			if analysisDiag != nil {
				doc.Diagnostics = *analysisDiag
				pDiagnostics = translateDiagnostics(*analysisDiag)
			}
		}
	}

	if err := s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: pDiagnostics,
	}); err != nil {
		s.cfg.L.Error(
			"failed to publish pDiagnostics",
			zap.Error(err),
			zap.String("uri", string(uri)),
		)
	}
}

func severity(in diagnostics.Severity) protocol.DiagnosticSeverity {
	var out protocol.DiagnosticSeverity
	switch in {
	case diagnostics.Warning:
		out = protocol.DiagnosticSeverityWarning
	case diagnostics.Info:
		out = protocol.DiagnosticSeverityInformation
	case diagnostics.Hint:
		out = protocol.DiagnosticSeverityHint
	case diagnostics.Error:
		out = protocol.DiagnosticSeverityError
	}
	return out
}

// translateDiagnostics converts Arc analyzer diagnostics to LSP diagnostics
func translateDiagnostics(analysisDiag diagnostics.Diagnostics) []protocol.Diagnostic {
	oDiagnostics := make([]protocol.Diagnostic, 0, len(analysisDiag))
	for _, diag := range analysisDiag {
		oDiagnostics = append(oDiagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(diag.Line - 1),
					Character: uint32(diag.Column),
				},
				End: protocol.Position{
					Line:      uint32(diag.Line - 1),
					Character: uint32(diag.Column + 10),
				},
			},
			Severity: severity(diag.Severity),
			Source:   "arc-analyzer",
			Message:  diag.Message,
		})
	}
	return oDiagnostics
}
