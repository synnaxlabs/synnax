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
	"github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

type Config struct {
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = &Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
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
	Analysis analyzer.Diagnostics // Cached analysis results
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
					TokenTypes:     convertToSemanticTokenTypes(SemanticTokenTypes),
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
func (s *Server) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	s.cfg.L.Info("Initializing arc LSP server",
		zap.String("rootURI", string(params.RootURI)),
		zap.String("clientName", params.ClientInfo.Name),
	)

	return &protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo:   &protocol.ServerInfo{Name: "arc-lsp", Version: "0.1.0"},
	}, nil
}

// Initialized handles the initialized notification
func (s *Server) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	s.cfg.L.Info("Server initialized")
	return nil
}

// Shutdown handles the shutdown request
func (s *Server) Shutdown(ctx context.Context) error {
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

	// Run diagnostics
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

	// Clear diagnostics
	s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})

	return nil
}

// publishDiagnostics parses the document and publishes syntax and semantic errors
func (s *Server) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, content string) {
	oDiagnostics := []protocol.Diagnostic{}

	// Parse the document
	t, err := text.Parse(text.Text{Raw: content})
	if err != nil {
		// Extract parse errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "parse errors:") {
			// Parse error format: "parse errors: [line X:Y message]"
			parts := strings.Split(errMsg, "[")
			for i := 1; i < len(parts); i++ {
				part := strings.TrimSuffix(parts[i], "]")
				if strings.HasPrefix(part, "line ") {
					// Extract line, column, and message
					var line, col int
					var msg string
					fmt.Sscanf(part, "line %d:%d %s", &line, &col, &msg)

					// Resolve the rest of the message
					msgStart := strings.Index(part, fmt.Sprintf("%d:%d ", line, col))
					if msgStart >= 0 {
						msg = part[msgStart+len(fmt.Sprintf("%d:%d ", line, col)):]
					}

					diagnostic := protocol.Diagnostic{
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      uint32(line - 1), // LSP is 0-based
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
					}
					oDiagnostics = append(oDiagnostics, diagnostic)
				}
			}
		}
	} else {
		// Run semantic analysis if parsing succeeded
		_, diag := text.Analyze(ctx, t, nil)

		// Store analysis results for other features (hover, completion, etc.)
		s.mu.Lock()
		if doc, ok := s.documents[uri]; ok {
			doc.Analysis = diag
		}
		s.mu.Unlock()

		// Convert semantic oDiagnostics to LSP oDiagnostics
		for _, diag := range diag {
			severity := protocol.DiagnosticSeverityError
			switch diag.Severity {
			case diagnostics.Warning:
				severity = protocol.DiagnosticSeverityWarning
			case diagnostics.Info:
				severity = protocol.DiagnosticSeverityInformation
			case diagnostics.Hint:
				severity = protocol.DiagnosticSeverityHint
			}

			diagnostic := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(diag.Line - 1), // Convert to 0-based
						Character: uint32(diag.Column),
					},
					End: protocol.Position{
						Line:      uint32(diag.Line - 1),
						Character: uint32(diag.Column + 10), // Approximate end
					},
				},
				Severity: severity,
				Source:   "arc-analyzer",
				Message:  diag.Message,
			}
			oDiagnostics = append(oDiagnostics, diagnostic)
		}
	}

	// Publish oDiagnostics
	s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: oDiagnostics,
	})
}
