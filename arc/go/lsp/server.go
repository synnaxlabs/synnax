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
	"fmt"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Config defines the configuration for opening an arc LSP Server.
type Config struct {
	// GlobalResolver allows the caller to define custom globals that will appear in
	// LSP auto-complete and type checking. Typically used to provide standard library
	// variables and functions as well as channels.
	GlobalResolver symbol.Resolver
	// OnExternalChange is an observable that fires when external state (such as
	// Synnax channels) changes. When this fires, the server will republish diagnostics
	// for all open documents to ensure they reflect the current state.
	OnExternalChange observe.Observable[struct{}]
	// RepublishTimeout is the maximum time to wait for a republish operation to complete.
	// If zero, defaults to 30 seconds.
	RepublishTimeout time.Duration
	// DebounceDelay is the trailing-edge delay after the last keystroke before
	// diagnostics are published. Defaults to 200ms.
	DebounceDelay time.Duration
	// MaxDebounceDelay caps the total delay from the first unprocessed change,
	// ensuring fast typists still get periodic diagnostic updates. Defaults to 1s.
	MaxDebounceDelay time.Duration
	// Instrumentation is used for logging, tracing, metrics, etc.
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = &Config{}
	DefaultConfig                       = Config{
		RepublishTimeout: 30 * time.Second,
		DebounceDelay:    200 * time.Millisecond,
		MaxDebounceDelay: 1 * time.Second,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.GlobalResolver = override.Nil(c.GlobalResolver, other.GlobalResolver)
	c.OnExternalChange = override.Nil(c.OnExternalChange, other.OnExternalChange)
	c.RepublishTimeout = override.Numeric(c.RepublishTimeout, other.RepublishTimeout)
	c.DebounceDelay = override.Numeric(c.DebounceDelay, other.DebounceDelay)
	c.MaxDebounceDelay = override.Numeric(c.MaxDebounceDelay, other.MaxDebounceDelay)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error { return nil }

// Server implements the Language Server Protocol for arc
type Server struct {
	capabilities             protocol.ServerCapabilities
	client                   protocol.Client
	documents                map[protocol.DocumentURI]*Document
	cfg                      Config
	mu                       sync.RWMutex
	republishMu              sync.Mutex
	cancelRepublish          context.CancelFunc
	externalChangeDisconnect observe.Disconnect
}

var _ protocol.Server = (*Server)(nil)

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
				Change:    protocol.TextDocumentSyncKindIncremental,
				Save:      &protocol.SaveOptions{IncludeText: false},
			},
			HoverProvider: true,
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{
					parser.LiteralCOLON,
					parser.LiteralLT,
					parser.LiteralMINUS,
					parser.LiteralGT,
					parser.LiteralLBRACE,
					parser.LiteralEQ,
					parser.LiteralCOMMA,
				},
			},
			DefinitionProvider:              true,
			DocumentSymbolProvider:          true,
			DocumentFormattingProvider:      true,
			DocumentRangeFormattingProvider: true,
			FoldingRangeProvider:            true,
			RenameProvider:                  true,
			SemanticTokensProvider: map[string]any{
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
	if s.cfg.OnExternalChange != nil {
		s.externalChangeDisconnect = s.cfg.OnExternalChange.OnChange(func(ctx context.Context, _ struct{}) {
			s.republishMu.Lock()
			if s.cancelRepublish != nil {
				s.cancelRepublish()
			}
			ctx, cancel := context.WithTimeout(ctx, s.cfg.RepublishTimeout)
			s.cancelRepublish = cancel
			s.republishMu.Unlock()
			go s.republishAllDiagnostics(ctx)
		})
	}
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
	if s.externalChangeDisconnect != nil {
		s.externalChangeDisconnect()
	}
	s.republishMu.Lock()
	if s.cancelRepublish != nil {
		s.cancelRepublish()
	}
	s.republishMu.Unlock()
	s.mu.RLock()
	docs := make([]*Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	s.mu.RUnlock()
	for _, doc := range docs {
		doc.stopDebounce()
	}
	return nil
}

// DidOpen handles opening a document
func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("document opened", zap.String("uri", string(uri)))
	metadata := extractMetadataFromURI(uri)
	s.cfg.L.Debug("file meta-data",
		zap.String("uri", string(uri)),
		zap.Bool("hasMetadata", metadata != nil),
		zap.Bool("isBlock", metadata != nil && metadata.isFunctionBlock))
	s.mu.Lock()
	s.documents[uri] = &Document{
		URI:      uri,
		Version:  params.TextDocument.Version,
		Content:  params.TextDocument.Text,
		metadata: metadata,
	}
	s.mu.Unlock()

	s.publishDiagnostics(ctx, uri, params.TextDocument.Text)

	return nil
}

// DidChange handles document changes
func (s *Server) DidChange(_ context.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document changed", zap.String("uri", string(uri)))

	s.mu.Lock()
	doc, ok := s.documents[uri]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	for _, change := range params.ContentChanges {
		if IsFullReplacement(change) {
			doc.Content = change.Text
		} else {
			doc.Content = ApplyIncrementalChange(doc.Content, change)
		}
	}
	doc.Version = params.TextDocument.Version
	s.mu.Unlock()

	s.scheduleDiagnostics(doc, uri)
	return nil
}

// DidSave handles document save - force-flushes any pending analysis.
func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document saved", zap.String("uri", string(uri)))

	s.mu.RLock()
	doc, ok := s.documents[uri]
	if !ok {
		s.mu.RUnlock()
		return nil
	}
	content := doc.Content
	s.mu.RUnlock()

	doc.stopDebounce()
	s.publishDiagnostics(ctx, uri, content)
	return nil
}

// DidClose handles closing a document
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document closed", zap.String("uri", string(uri)))

	s.mu.Lock()
	doc, ok := s.documents[uri]
	delete(s.documents, uri)
	s.mu.Unlock()

	if ok {
		doc.stopDebounce()
	}

	return s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})
}

// scheduleDiagnostics debounces analysis for a document. Each keystroke
// resets the trailing-edge timer, but a max-delay cap ensures periodic
// updates during sustained typing.
func (s *Server) scheduleDiagnostics(
	doc *Document,
	uri protocol.DocumentURI,
) {
	doc.dMu.Lock()
	defer doc.dMu.Unlock()

	if doc.cancelAnalysis != nil {
		doc.cancelAnalysis()
		doc.cancelAnalysis = nil
	}
	if doc.debounceTimer != nil {
		doc.debounceTimer.Stop()
	}

	now := time.Now()
	if doc.firstChangeAt.IsZero() {
		doc.firstChangeAt = now
	}

	delay := s.cfg.DebounceDelay
	elapsed := now.Sub(doc.firstChangeAt)
	if maxRemaining := s.cfg.MaxDebounceDelay - elapsed; maxRemaining < delay {
		delay = maxRemaining
	}
	if delay <= 0 {
		doc.firstChangeAt = time.Time{}
		s.startAnalysis(doc, uri)
		return
	}

	doc.debounceTimer = time.AfterFunc(delay, func() {
		doc.dMu.Lock()
		doc.firstChangeAt = time.Time{}
		doc.dMu.Unlock()
		s.startAnalysis(doc, uri)
	})
}

func (s *Server) startAnalysis(
	doc *Document,
	uri protocol.DocumentURI,
) {
	ctx, cancel := context.WithCancel(context.Background())
	doc.cancelAnalysis = cancel
	go s.runAnalysis(ctx, doc, uri)
}

func (s *Server) runAnalysis(
	ctx context.Context,
	doc *Document,
	uri protocol.DocumentURI,
) {
	if ctx.Err() != nil {
		return
	}

	s.mu.RLock()
	content := doc.Content
	isBlock := doc.metadata.isFunctionBlock
	s.mu.RUnlock()

	pDiagnostics, docIR, docDiag := s.analyze(ctx, content, isBlock)
	if ctx.Err() != nil {
		return
	}

	s.mu.Lock()
	if _, ok := s.documents[uri]; ok {
		doc.IR = docIR
		doc.Diagnostics = docDiag
	}
	s.mu.Unlock()

	if err := s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: pDiagnostics,
	}); err != nil {
		s.cfg.L.Error(
			"failed to publish diagnostics",
			zap.Error(err),
			zap.String("uri", string(uri)),
		)
	}
}

// analyze performs parse+analyze on the given content and returns protocol
// diagnostics, the resulting IR, and the raw diagnostics. It does NOT
// mutate any Document fields.
func (s *Server) analyze(
	ctx context.Context,
	content string,
	isBlock bool,
) ([]protocol.Diagnostic, ir.IR, diagnostics.Diagnostics) {
	s.cfg.L.Debug("analyzing program")
	var (
		pDiagnostics []protocol.Diagnostic
		docIR        ir.IR
		docDiag      diagnostics.Diagnostics
	)

	if isBlock {
		wrappedContent := fmt.Sprintf("{%s}", content)
		t, err := parser.ParseBlock(wrappedContent)
		if err != nil {
			pDiagnostics = translateDiagnostics(*err)
		} else {
			aCtx := acontext.CreateRoot[parser.IBlockContext](
				ctx, t, s.cfg.GlobalResolver,
			)
			statement.AnalyzeFunctionBody(aCtx)
			docIR = ir.IR{Symbols: aCtx.Scope}
			docDiag = *aCtx.Diagnostics
			pDiagnostics = translateDiagnostics(docDiag)
		}
	} else {
		t, diag := text.Parse(text.Text{Raw: content})
		if diag != nil {
			pDiagnostics = translateDiagnostics(*diag)
		} else {
			analyzedIR, analysisDiag := text.Analyze(ctx, t, s.cfg.GlobalResolver)
			docIR = analyzedIR
			if analysisDiag != nil {
				docDiag = *analysisDiag
				pDiagnostics = translateDiagnostics(docDiag)
			}
		}
	}
	return pDiagnostics, docIR, docDiag
}

// publishDiagnostics synchronously parses and publishes diagnostics.
// Used for DidOpen, DidSave, and republish where immediate feedback is expected.
func (s *Server) publishDiagnostics(
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
) {
	s.mu.RLock()
	doc, ok := s.documents[uri]
	if !ok {
		s.mu.RUnlock()
		return
	}
	isBlock := doc.metadata.isFunctionBlock
	s.mu.RUnlock()

	pDiagnostics, docIR, docDiag := s.analyze(ctx, content, isBlock)

	s.mu.Lock()
	if _, stillOpen := s.documents[uri]; stillOpen {
		doc.IR = docIR
		doc.Diagnostics = docDiag
	}
	s.mu.Unlock()

	if err := s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: pDiagnostics,
	}); err != nil {
		s.cfg.L.Error(
			"failed to publish diagnostics",
			zap.Error(err),
			zap.String("uri", string(uri)),
		)
	}
}

// republishAllDiagnostics re-analyzes and publishes diagnostics for all open documents.
// This is called when external state changes (e.g., channels are created or deleted).
func (s *Server) republishAllDiagnostics(ctx context.Context) {
	s.mu.RLock()
	docs := make(map[protocol.DocumentURI]string, len(s.documents))
	for uri, doc := range s.documents {
		docs[uri] = doc.Content
	}
	s.mu.RUnlock()
	for uri, content := range docs {
		s.publishDiagnostics(ctx, uri, content)
	}
}

func severity(in diagnostics.Severity) protocol.DiagnosticSeverity {
	var out protocol.DiagnosticSeverity
	switch in {
	case diagnostics.SeverityWarning:
		out = protocol.DiagnosticSeverityWarning
	case diagnostics.SeverityInfo:
		out = protocol.DiagnosticSeverityInformation
	case diagnostics.SeverityHint:
		out = protocol.DiagnosticSeverityHint
	case diagnostics.SeverityError:
		out = protocol.DiagnosticSeverityError
	}
	return out
}

func translateDiagnostics(analysisDiag diagnostics.Diagnostics) []protocol.Diagnostic {
	oDiagnostics := make([]protocol.Diagnostic, 0, len(analysisDiag))
	for _, diag := range analysisDiag {
		end := diag.End
		if end.Line == 0 && end.Col == 0 {
			end.Line = diag.Start.Line
			end.Col = diag.Start.Col + 1
		}

		pDiag := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(diag.Start.Line - 1),
					Character: uint32(diag.Start.Col),
				},
				End: protocol.Position{
					Line:      uint32(end.Line - 1),
					Character: uint32(end.Col),
				},
			},
			Severity: severity(diag.Severity),
			Source:   "arc",
			Message:  diag.Message,
		}

		if diag.Code != "" {
			pDiag.Code = string(diag.Code)
		}

		if len(diag.Notes) > 0 {
			related := make([]protocol.DiagnosticRelatedInformation, 0, len(diag.Notes))
			for _, note := range diag.Notes {
				loc := protocol.Location{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      uint32(note.Start.Line - 1),
							Character: uint32(note.Start.Col),
						},
						End: protocol.Position{
							Line:      uint32(note.Start.Line - 1),
							Character: uint32(note.Start.Col + 1),
						},
					},
				}
				if note.Start.Line == 0 {
					loc.Range = pDiag.Range
				}
				related = append(related, protocol.DiagnosticRelatedInformation{
					Location: loc,
					Message:  note.Message,
				})
			}
			pDiag.RelatedInformation = related
		}

		oDiagnostics = append(oDiagnostics, pDiag)
	}
	return oDiagnostics
}
