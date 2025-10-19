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
	"strings"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/analyzer"
	arcdiag "github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/ir"
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
	URI      protocol.DocumentURI
	Version  int32
	Content  string
	IR       ir.IR                // Analyzed IR with symbol table
	Analysis analyzer.Diagnostics // Cached analysis diagnostics
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
					TokenTypes:     convertToSemanticTokenTypes(semanticTokenTypes),
					TokenModifiers: convertToSemanticTokenModifiers(SemanticTokenModifiers),
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
	return s.cfg.L.Zap()
}

// Helper functions to convert string slices to protocol types
func convertToSemanticTokenTypes(types []string) []protocol.SemanticTokenTypes {
	result := make([]protocol.SemanticTokenTypes, len(types))
	for i, t := range types {
		result[i] = protocol.SemanticTokenTypes(t)
	}
	return result
}

func convertToSemanticTokenModifiers(modifiers []string) []protocol.SemanticTokenModifiers {
	result := make([]protocol.SemanticTokenModifiers, len(modifiers))
	for i, m := range modifiers {
		result[i] = protocol.SemanticTokenModifiers(m)
	}
	return result
}

// Initialize handles the initialize request
func (s *Server) Initialize(_ context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	s.cfg.L.Info("Initializing arc LSP server", zap.String("client_name", params.ClientInfo.Name))
	return &protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo:   &protocol.ServerInfo{Name: "arc-lsp", Version: "0.1.0"},
	}, nil
}

// Initialized handles the initialized notification
func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error {
	s.cfg.L.Info("Server initialized")
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
	s.cfg.L.Debug("Document opened", zap.String("uri", string(uri)))

	s.mu.Lock()
	s.documents[uri] = &Document{
		URI:     uri,
		Version: params.TextDocument.Version,
		Content: params.TextDocument.Text,
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
		// We use full document sync, so there should be exactly one change
		if len(params.ContentChanges) > 0 {
			doc.Version = params.TextDocument.Version
			doc.Content = params.ContentChanges[0].Text
		}
	}
	s.mu.Unlock()

	// Resolve the updated content
	s.mu.RLock()
	content := ""
	if doc, ok := s.documents[uri]; ok {
		content = doc.Content
	}
	s.mu.RUnlock()

	// Run diagnostics
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
	t, err := text.Parse(text.Text{Raw: content})
	var diagnostics []protocol.Diagnostic

	if err != nil {
		diagnostics, err = extractParseErrors(err)
		if err != nil {
			s.cfg.L.Error("error parsing diagnostics", zap.Error(err))
		}
	} else {
		analyzedIR, analysisDiag := text.Analyze(ctx, t, s.cfg.GlobalResolver)

		s.mu.Lock()
		if doc, ok := s.documents[uri]; ok {
			doc.IR = analyzedIR
			doc.Analysis = analysisDiag
		}
		s.mu.Unlock()

		diagnostics = convertAnalysisDiagnostics(analysisDiag)
	}

	if err := s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}); err != nil {
		s.cfg.L.Error(
			"failed to publish diagnostics",
			zap.Error(err),
			zap.String("uri", string(uri)),
		)
	}
}

// extractParseErrors extracts LSP diagnostics from Arc parser errors
func extractParseErrors(err error) ([]protocol.Diagnostic, error) {
	var diagnostics []protocol.Diagnostic
	errMsg := err.Error()

	if !strings.Contains(errMsg, "parse errors:") {
		return diagnostics, nil
	}

	parts := strings.Split(errMsg, "[")
	for i := 1; i < len(parts); i++ {
		part := strings.TrimSuffix(parts[i], "]")
		if !strings.HasPrefix(part, "line ") {
			continue
		}

		var line, col int
		var msg string
		if _, err := fmt.Sscanf(part, "line %d:%d %s", &line, &col, &msg); err != nil {
			return nil, err
		}

		msgStart := strings.Index(part, fmt.Sprintf("%d:%d ", line, col))
		if msgStart >= 0 {
			msg = part[msgStart+len(fmt.Sprintf("%d:%d ", line, col)):]
		}

		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(line - 1),
					Character: uint32(col),
				},
				End: protocol.Position{
					Line:      uint32(line - 1),
					Character: uint32(col + 1),
				},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "arc-parser",
			Message:  msg,
		})
	}

	return diagnostics, nil
}

func severity(s arcdiag.Severity) protocol.DiagnosticSeverity {
	var severity protocol.DiagnosticSeverity
	switch s {
	case arcdiag.Warning:
		severity = protocol.DiagnosticSeverityWarning
	case arcdiag.Info:
		severity = protocol.DiagnosticSeverityInformation
	case arcdiag.Hint:
		severity = protocol.DiagnosticSeverityHint
	case arcdiag.Error:
		severity = protocol.DiagnosticSeverityError
	}
	return severity
}

// convertAnalysisDiagnostics converts Arc analyzer diagnostics to LSP diagnostics
func convertAnalysisDiagnostics(analysisDiag analyzer.Diagnostics) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0, len(analysisDiag))
	for _, diag := range analysisDiag {
		diagnostics = append(diagnostics, protocol.Diagnostic{
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
	return diagnostics
}
