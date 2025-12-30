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
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/diagnostics"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Server implements the Language Server Protocol for Oracle schema files.
type Server struct {
	client       protocol.Client
	mu           sync.RWMutex
	documents    map[protocol.DocumentURI]*Document
	capabilities protocol.ServerCapabilities
}

// Document represents an open document in the LSP server.
type Document struct {
	URI         protocol.DocumentURI
	Version     int32
	Content     string
	Schema      parser.ISchemaContext
	Table       *resolution.Table
	Diagnostics *diagnostics.Diagnostics
}

var _ protocol.Server = (*Server)(nil)

// New creates a new Oracle LSP server.
func New() *Server {
	return &Server{
		documents: make(map[protocol.DocumentURI]*Document),
		capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
			},
			HoverProvider:      true,
			CompletionProvider: &protocol.CompletionOptions{},
			SemanticTokensProvider: map[string]interface{}{
				"legend": protocol.SemanticTokensLegend{
					TokenTypes: convertToSemanticTokenTypes(semanticTokenTypes),
				},
				"full": true,
			},
		},
	}
}

// Serve starts the LSP server on the given reader and writer (typically stdin/stdout).
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	stream := jsonrpc2.NewStream(&rwCloser{r: r, w: w})
	conn := jsonrpc2.NewConn(stream)
	logger := zap.NewNop() // Use noop logger to avoid nil pointer
	s.client = protocol.ClientDispatcher(conn, logger)
	conn.Go(ctx, protocol.ServerHandler(s, nil))
	<-conn.Done()
	return conn.Err()
}

// rwCloser wraps an io.Reader and io.Writer to implement io.ReadWriteCloser.
type rwCloser struct {
	r io.Reader
	w io.Writer
}

func (rw *rwCloser) Read(p []byte) (int, error)  { return rw.r.Read(p) }
func (rw *rwCloser) Write(p []byte) (int, error) { return rw.w.Write(p) }
func (rw *rwCloser) Close() error                { return nil }

// SetClient sets the LSP client for sending notifications.
func (s *Server) SetClient(client protocol.Client) {
	s.client = client
}

// getDocument retrieves a document from the cache by URI.
func (s *Server) getDocument(uri protocol.DocumentURI) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[uri]
	return doc, ok
}

// Initialize handles the initialize request.
func (s *Server) Initialize(_ context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo:   &protocol.ServerInfo{Name: "oracle-lsp", Version: "0.1.0"},
	}, nil
}

// Initialized handles the initialized notification.
func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error {
	return nil
}

// Shutdown handles the shutdown request.
func (s *Server) Shutdown(_ context.Context) error {
	return nil
}

// DidOpen handles opening a document.
func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := params.TextDocument.URI
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

// DidChange handles document changes.
func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
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

// DidClose handles closing a document.
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.mu.Lock()
	delete(s.documents, uri)
	s.mu.Unlock()
	return s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})
}

// publishDiagnostics parses the document and publishes diagnostics.
func (s *Server) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, content string) {
	s.mu.Lock()
	doc, ok := s.documents[uri]
	s.mu.Unlock()
	if !ok {
		return
	}

	ast, parseDiag := parser.Parse(content)
	if parseDiag != nil && parseDiag.HasErrors() {
		doc.Diagnostics = parseDiag
		_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: translateDiagnostics(*parseDiag),
		})
		return
	}

	doc.Schema = ast
	namespace := deriveNamespaceFromURI(uri)
	table, analyzeDiag := analyzer.AnalyzeSource(ctx, content, namespace, noopLoader{})
	if analyzeDiag != nil {
		doc.Diagnostics = analyzeDiag
		doc.Table = table
		_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: translateDiagnostics(*analyzeDiag),
		})
		return
	}

	doc.Table = table
	doc.Diagnostics = &diagnostics.Diagnostics{}
	_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})
}

// translateDiagnostics converts Oracle diagnostics to LSP diagnostics.
func translateDiagnostics(diag diagnostics.Diagnostics) []protocol.Diagnostic {
	result := make([]protocol.Diagnostic, 0, len(diag))
	for _, d := range diag {
		result = append(result, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(max(0, d.Line-1)),
					Character: uint32(d.Column),
				},
				End: protocol.Position{
					Line:      uint32(max(0, d.Line-1)),
					Character: uint32(d.Column + 10),
				},
			},
			Severity: severity(d.Severity),
			Source:   "oracle-analyzer",
			Message:  d.Message,
		})
	}
	return result
}

func severity(s diagnostics.Severity) protocol.DiagnosticSeverity {
	switch s {
	case diagnostics.Error:
		return protocol.DiagnosticSeverityError
	case diagnostics.Warning:
		return protocol.DiagnosticSeverityWarning
	case diagnostics.Info:
		return protocol.DiagnosticSeverityInformation
	case diagnostics.Hint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}

// deriveNamespaceFromURI extracts a namespace from the document URI.
func deriveNamespaceFromURI(uri protocol.DocumentURI) string {
	path := string(uri)
	path = strings.TrimPrefix(path, "file://")
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base
}

// noopLoader is a FileLoader that doesn't load any files.
// It's used by the LSP server for analyzing single files without import resolution.
type noopLoader struct{}

func (noopLoader) Load(path string) (source, filePath string, err error) {
	return "", path, nil
}

func (noopLoader) RepoRoot() string {
	return ""
}

func convertToSemanticTokenTypes(types []string) []protocol.SemanticTokenTypes {
	result := make([]protocol.SemanticTokenTypes, len(types))
	for i, t := range types {
		result[i] = protocol.SemanticTokenTypes(t)
	}
	return result
}
