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
	"io"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/debounce"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	lsp "github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

// OnRename is invoked by the LSP when a Rename request resolves to a known symbol. It
// runs before the server computes source-text edits and lets the caller propagate the
// rename to the resource the symbol refers to (e.g., the underlying Synnax channel).
// Returning an error aborts the rename and surfaces the error to the client. Callbacks
// should return nil for symbols they do not handle.
type OnRename func(_ context.Context, _ *symbol.Symbol, oldName, newName string) error

// Config defines the configuration for opening an arc LSP Server.
type Config struct {
	// NewRoot builds a fresh analyzer root scope. The LSP calls it once per document
	// analysis (so per-document imports don't bleed across files) and again for
	// completion fallback when the document has no analyzed scope yet. Callers compose
	// STL, their custom globals, and any dynamic resolvers (cluster channels, etc.)
	// into the returned root.
	//
	// [REQUIRED]
	NewRoot func() *symbol.Symbol
	// AllowDashedNames permits '-' inside channel-name identifiers, set when Core runs
	// with channel-name validation disabled.
	//
	// [OPTIONAL]
	AllowDashedNames bool
	// OnRename is invoked when a rename request targets a symbol the LSP itself cannot
	// fully relocate by text edits alone (e.g., a channel). When nil, rename is
	// restricted to source-defined symbols.
	//
	// [OPTIONAL]
	OnRename OnRename
	// OnExternalChange is an observable that fires when external state (such as Synnax
	// channels) changes. When this fires, the server will republish diagnostics for all
	// open documents to ensure they reflect the current state.
	//
	// [OPTIONAL]
	OnExternalChange observe.Observable[struct{}]
	// RepublishTimeout is the maximum time to wait for a republish operation to
	// complete. If zero, defaults to 30 seconds.
	//
	// [OPTIONAL] - Defaults to 30 seconds.
	RepublishTimeout time.Duration
	// DebounceDelay is the trailing-edge delay after the last keystroke before
	// diagnostics are published. Defaults to 200ms.
	//
	// [OPTIONAL] - Defaults to 200ms.
	DebounceDelay time.Duration
	// MaxDebounceDelay caps the total delay from the first unprocessed change, ensuring
	// fast typists still get periodic diagnostic updates. Defaults to 1s.
	//
	// [OPTIONAL] - Defaults to 1s.
	MaxDebounceDelay time.Duration
	// Instrumentation is used for logging, tracing, metrics, etc.
	//
	// [OPTIONAL]
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = &Config{}
	defaultConfig                       = Config{
		RepublishTimeout: 30 * time.Second,
		DebounceDelay:    200 * time.Millisecond,
		MaxDebounceDelay: 1 * time.Second,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.NewRoot = override.Nil(c.NewRoot, other.NewRoot)
	c.OnRename = override.Nil(c.OnRename, other.OnRename)
	c.OnExternalChange = override.Nil(c.OnExternalChange, other.OnExternalChange)
	c.RepublishTimeout = override.Numeric(c.RepublishTimeout, other.RepublishTimeout)
	c.DebounceDelay = override.Numeric(c.DebounceDelay, other.DebounceDelay)
	c.MaxDebounceDelay = override.Numeric(c.MaxDebounceDelay, other.MaxDebounceDelay)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.lsp")
	validate.NotNil(v, "new_root", c.NewRoot)
	return v.Error()
}

const translateSource = "arc-analyzer"

// Server implements the Language Server Protocol for arc
type Server struct {
	protocol.UnimplementedServer
	capabilities             protocol.ServerCapabilities
	client                   protocol.Client
	documents                map[uri.URI]*Document
	cfg                      Config
	mu                       sync.RWMutex
	republishMu              sync.Mutex
	cancelRepublish          context.CancelFunc
	externalChangeDisconnect observe.Disconnect
	republishWG              sync.WaitGroup
}

var _ protocol.Server = (*Server)(nil)

// New creates a new LSP server
func New(cfgs ...Config) (*Server, error) {
	cfg, err := config.New(defaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		documents: make(map[uri.URI]*Document),
		capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: new(true),
				Change:    new(protocol.TextDocumentSyncKindIncremental),
				Save:      &protocol.SaveOptions{IncludeText: new(false)},
			},
			HoverProvider: protocol.Boolean(true),
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{
					parser.LiteralCOLON,
					parser.LiteralLT,
					parser.LiteralMINUS,
					parser.LiteralGT,
					parser.LiteralLBRACE,
					parser.LiteralEQ,
					parser.LiteralCOMMA,
					parser.LiteralDOT,
				},
			},
			DefinitionProvider:              protocol.Boolean(true),
			DocumentFormattingProvider:      protocol.Boolean(true),
			DocumentRangeFormattingProvider: protocol.Boolean(true),
			FoldingRangeProvider:            protocol.Boolean(true),
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{
					protocol.CodeActionKindQuickFix,
				},
			},
			RenameProvider: &protocol.RenameOptions{
				PrepareProvider: new(true),
			},
			SemanticTokensProvider: &protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenTypes: semanticTokenTypes,
				},
				Full: protocol.Boolean(true),
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
			s.republishWG.Go(func() { s.republishAllDiagnostics(ctx) })
		})
	}
}

// getDocument returns an immutable snapshot of the document with the given URI, and
// true if found, or nil and false if not found. The returned *Document is a copy taken
// under the read lock, so callers may read its fields (IR, Content, Diagnostics, ...)
// without further locking: analysis replaces these fields on the live document under
// the write lock, and a reader holding the snapshot is unaffected. Callers must not
// mutate the returned document — writes do not reach the live document and would be
// lost.
func (s *Server) getDocument(uri uri.URI) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[uri]
	if !ok {
		return nil, false
	}
	snapshot := *doc
	return &snapshot, true
}

// Initialize handles the initialize request
func (s *Server) Initialize(
	_ context.Context,
	params *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	s.cfg.L.Debug("initializing Arc LSP", zap.String("client", params.ClientInfo.Name))
	return &protocol.InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo: protocol.ServerInfo{
			Name:    "arc-lsp",
			Version: protocol.NewOptional("0.1.0"),
		},
	}, nil
}

// Initialized handles the initialized notification
func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error {
	s.cfg.L.Debug("Arc LSP initialized")
	return nil
}

// Shutdown handles the shutdown request.
func (s *Server) Shutdown(context.Context) error {
	s.cfg.L.Info("Shutting down server")
	if s.externalChangeDisconnect != nil {
		s.externalChangeDisconnect()
	}
	s.republishMu.Lock()
	if s.cancelRepublish != nil {
		s.cancelRepublish()
	}
	s.republishMu.Unlock()
	s.republishWG.Wait()
	s.mu.RLock()
	docs := make([]*Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	s.mu.RUnlock()
	for _, doc := range docs {
		doc.debouncer.Stop()
	}
	return nil
}

// DidOpen handles opening a document
func (s *Server) DidOpen(
	ctx context.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("document opened", zap.String("uri", string(uri)))
	metadata := extractMetadataFromURI(uri)
	s.cfg.L.Debug("file meta-data",
		zap.String("uri", string(uri)),
		zap.Bool("hasMetadata", metadata != nil),
		zap.Bool("isBlock", metadata != nil && metadata.isFunctionBlock))
	doc := &Document{
		URI:      uri,
		Version:  params.TextDocument.Version,
		Content:  params.TextDocument.Text,
		metadata: metadata,
	}
	deb, err := debounce.New(debounce.Config{
		Delay:    s.cfg.DebounceDelay,
		MaxDelay: s.cfg.MaxDebounceDelay,
		Callback: func(ctx context.Context) { s.runAnalysis(ctx, doc, uri) },
	})
	if err != nil {
		return errors.Wrap(err, "create document debouncer")
	}
	doc.debouncer = deb
	s.mu.Lock()
	s.documents[uri] = doc
	s.mu.Unlock()

	s.publishDiagnostics(ctx, uri, params.TextDocument.Text)

	return nil
}

// DidChange handles document changes
func (s *Server) DidChange(
	_ context.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document changed", zap.String("uri", string(uri)))

	s.mu.Lock()
	doc, ok := s.documents[uri]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	for _, change := range params.ContentChanges {
		doc.Content = lsp.ApplyIncrementalChange(doc.Content, change)
	}
	doc.Version = params.TextDocument.Version
	s.mu.Unlock()

	doc.debouncer.Trigger()
	return nil
}

// DidSave handles document save - force-flushes any pending analysis.
func (s *Server) DidSave(
	ctx context.Context,
	params *protocol.DidSaveTextDocumentParams,
) error {
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

	doc.debouncer.Stop()
	s.publishDiagnostics(ctx, uri, content)
	return nil
}

// DidClose handles closing a document
func (s *Server) DidClose(
	ctx context.Context,
	params *protocol.DidCloseTextDocumentParams,
) error {
	uri := params.TextDocument.URI
	s.cfg.L.Debug("Document closed", zap.String("uri", string(uri)))

	s.mu.Lock()
	doc, ok := s.documents[uri]
	delete(s.documents, uri)
	s.mu.Unlock()

	if ok {
		doc.debouncer.Stop()
	}

	return s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{},
	})
}

func (s *Server) runAnalysis(
	ctx context.Context,
	doc *Document,
	uri uri.URI,
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

	s.refreshSemanticTokens(ctx, uri)
}

func (s *Server) refreshSemanticTokens(ctx context.Context, uri uri.URI) {
	if s.client == nil {
		return
	}
	if err := s.client.SemanticTokensRefresh(ctx); err != nil &&
		!errors.IsAny(err, io.ErrClosedPipe, context.Canceled) {
		s.cfg.L.Warn(
			"failed to refresh semantic tokens",
			zap.Error(err),
			zap.String("uri", string(uri)),
		)
	}
}

// analyze performs parse+analyze on the given content and returns protocol diagnostics,
// the resulting IR, and the raw diagnostics. It does NOT mutate any Document fields.
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
		cfg          = parser.Config{AllowDashedNames: s.cfg.AllowDashedNames}
	)

	if isBlock {
		wrappedContent := fmt.Sprintf("{%s}", content)
		t, err := parser.ParseBlock(wrappedContent, cfg)
		if err != nil {
			pDiagnostics = lsp.TranslateDiagnostics(*err, translateSource)
		} else {
			aCtx := acontext.NewRoot(ctx, t, s.cfg.NewRoot()).WithConfig(cfg)
			statement.AnalyzeFunctionBody(aCtx)
			docIR = ir.IR{Symbols: aCtx.Scope}
			docDiag = *aCtx.Diagnostics
			pDiagnostics = lsp.TranslateDiagnostics(docDiag, translateSource)
		}
	} else {
		t, diag := text.Parse(text.Text{Raw: content}, cfg)
		if diag != nil {
			pDiagnostics = lsp.TranslateDiagnostics(*diag, translateSource)
		} else {
			analyzedIR, analysisDiag := text.Analyze(ctx, t, s.cfg.NewRoot(), cfg)
			docIR = analyzedIR
			if analysisDiag != nil {
				docDiag = *analysisDiag
				pDiagnostics = lsp.TranslateDiagnostics(docDiag, translateSource)
			}
		}
	}
	return pDiagnostics, docIR, docDiag
}

// publishDiagnostics synchronously parses and publishes diagnostics. Used for DidOpen,
// DidSave, and republish where immediate feedback is expected.
func (s *Server) publishDiagnostics(
	ctx context.Context,
	uri uri.URI,
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
	docs := make(map[uri.URI]string, len(s.documents))
	for uri, doc := range s.documents {
		docs[uri] = doc.Content
	}
	s.mu.RUnlock()
	for uri, content := range docs {
		s.publishDiagnostics(ctx, uri, content)
	}
	s.refreshSemanticTokens(ctx, "")
}
