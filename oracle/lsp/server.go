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
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	xlsp "github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Server implements the Language Server Protocol for Oracle schema files.
type Server struct {
	protocol.UnimplementedServer
	capabilities protocol.ServerCapabilities
	documents    map[uri.URI]*Document
	client       protocol.Client
	mu           sync.RWMutex
}

const translateSource = "oracle-analyzer"

// Document represents an open document in the LSP server.
type Document struct {
	Schema      parser.ISchemaContext
	Table       *resolution.Table
	Diagnostics *diagnostics.Diagnostics
	URI         uri.URI
	Content     string
	Version     int32
}

var _ protocol.Server = (*Server)(nil)

// New creates a new Oracle LSP server.
func New() *Server {
	return &Server{
		documents: make(map[uri.URI]*Document),
		capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: new(true),
				Change:    new(protocol.TextDocumentSyncKindFull),
			},
			HoverProvider:              protocol.Boolean(true),
			CompletionProvider:         &protocol.CompletionOptions{},
			DocumentFormattingProvider: protocol.Boolean(true),
			SemanticTokensProvider: &protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{TokenTypes: semanticTokenTypes},
				Full:   protocol.Boolean(true),
			},
		},
	}
}

// gatedStream delays the first Read until ready closes, so the connection cannot
// dispatch a handler before Serve wires the client.
type gatedStream struct {
	jsonrpc2.Stream
	ready chan struct{}
}

func (g *gatedStream) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-g.ready:
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
	return g.Stream.Read(ctx)
}

// Serve starts the LSP server on the given ReadWriteCloser (typically xos.StdIO).
func (s *Server) Serve(ctx context.Context, rwc io.ReadWriteCloser) error {
	stream := &gatedStream{Stream: jsonrpc2.NewStream(rwc), ready: make(chan struct{})}
	conn, client := xlsp.NewConn(ctx, s, stream)
	s.client = client
	close(stream.ready)
	<-conn.Done()
	return conn.Err()
}

// SetClient sets the LSP client for sending notifications.
func (s *Server) SetClient(client protocol.Client) {
	s.client = client
}

// getDocument retrieves a document from the cache by URI.
func (s *Server) getDocument(docURI uri.URI) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[docURI]
	return doc, ok
}

// Initialize handles the initialize request.
func (s *Server) Initialize(
	context.Context,
	*protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo: protocol.ServerInfo{
			Name:    "oracle-lsp",
			Version: protocol.NewOptional("0.1.0"),
		},
	}, nil
}

// Shutdown handles the shutdown request.
func (*Server) Shutdown(context.Context) error { return nil }

// DidOpen handles opening a document.
func (s *Server) DidOpen(
	ctx context.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	docURI := params.TextDocument.URI
	s.mu.Lock()
	s.documents[docURI] = &Document{
		URI:     docURI,
		Version: params.TextDocument.Version,
		Content: params.TextDocument.Text,
	}
	s.mu.Unlock()
	s.publishDiagnostics(ctx, docURI, params.TextDocument.Text)
	return nil
}

// DidChange handles document changes.
func (s *Server) DidChange(
	ctx context.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	docURI := params.TextDocument.URI
	s.mu.Lock()
	if doc, ok := s.documents[docURI]; ok {
		if len(params.ContentChanges) > 0 {
			doc.Version = params.TextDocument.Version
			for _, change := range params.ContentChanges {
				doc.Content = xlsp.ApplyIncrementalChange(doc.Content, change)
			}
		}
	}
	s.mu.Unlock()
	s.mu.RLock()
	content := ""
	if doc, ok := s.documents[docURI]; ok {
		content = doc.Content
	}
	s.mu.RUnlock()
	s.publishDiagnostics(ctx, docURI, content)
	return nil
}

// DidClose handles closing a document.
func (s *Server) DidClose(
	ctx context.Context,
	params *protocol.DidCloseTextDocumentParams,
) error {
	docURI := params.TextDocument.URI
	s.mu.Lock()
	delete(s.documents, docURI)
	s.mu.Unlock()
	// A notification handler error fails the jsonrpc2 v1 connection; drop it instead.
	_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         docURI,
		Diagnostics: []protocol.Diagnostic{},
	})
	return nil
}

// publishDiagnostics parses the document and publishes diagnostics.
func (s *Server) publishDiagnostics(
	ctx context.Context,
	docURI uri.URI,
	content string,
) {
	s.mu.Lock()
	doc, ok := s.documents[docURI]
	s.mu.Unlock()
	if !ok {
		return
	}

	ast, parseDiag := parser.Parse(content)
	if parseDiag != nil && !parseDiag.Ok() {
		doc.Diagnostics = parseDiag
		_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         docURI,
			Diagnostics: xlsp.TranslateDiagnostics(*parseDiag, translateSource),
		})
		return
	}

	doc.Schema = ast
	namespace := deriveNamespaceFromURI(docURI)
	table, analyzeDiag := analyzer.AnalyzeSource(ctx, content, namespace, noopLoader{})
	if analyzeDiag != nil {
		flat := analyzeDiag.Flat()
		doc.Diagnostics = &flat
		doc.Table = table
		_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         docURI,
			Diagnostics: xlsp.TranslateDiagnostics(flat, translateSource),
		})
		return
	}

	doc.Table = table
	doc.Diagnostics = &diagnostics.Diagnostics{}
	_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         docURI,
		Diagnostics: []protocol.Diagnostic{},
	})
}

// deriveNamespaceFromURI extracts a namespace from the document URI.
func deriveNamespaceFromURI(docURI uri.URI) string {
	path := string(docURI)
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

// Formatting handles document formatting requests.
func (s *Server) Formatting(
	_ context.Context,
	params *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	formatted, err := formatter.Format(doc.Content)
	if err != nil {
		return nil, err
	}

	// If no changes, return empty
	if formatted == doc.Content {
		return nil, nil
	}

	// Return a single edit that replaces the entire document
	lines := strings.Split(doc.Content, "\n")
	lastLine := uint32(len(lines) - 1)
	lastChar := uint32(0)
	if len(lines) > 0 {
		lastChar = uint32(len(lines[lastLine]))
	}

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: lastLine, Character: lastChar},
			},
			NewText: formatted,
		},
	}, nil
}
