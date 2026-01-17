// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	arclsp "github.com/synnaxlabs/arc/lsp"
	arctransport "github.com/synnaxlabs/arc/lsp/transport"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

type Arc = arc.Arc

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *arc.Service
	status   *status.Service
	alamos.Instrumentation
}

func NewService(cfg config.Config) *Service {
	return &Service{
		Instrumentation: cfg.Instrumentation,
		db:              cfg.Distribution.DB,
		access:          cfg.Service.RBAC,
		internal:        cfg.Service.Arc,
		status:          cfg.Service.Status,
	}
}

type (
	CreateRequest struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
	}
	CreateResponse = CreateRequest
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: arc.OntologyIDsFromArcs(req.Arcs),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, a := range req.Arcs {
			if err = w.Create(ctx, &a); err != nil {
				return err
			}
			req.Arcs[i] = a
		}
		res.Arcs = req.Arcs
		return nil
	})
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: arc.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	RetrieveRequest struct {
		SearchTerm    string      `json:"search_term" msgpack:"search_term"`
		Keys          []uuid.UUID `json:"keys" msgpack:"keys"`
		Names         []string    `json:"names" msgpack:"names"`
		Limit         int         `json:"limit" msgpack:"limit"`
		Offset        int         `json:"offset" msgpack:"offset"`
		IncludeStatus bool        `json:"include_status" msgpack:"include_status"`
		Compile       bool        `json:"compile" msgpack:"compile"`
	}
	RetrieveResponse struct {
		Arcs []Arc `json:"arcs" msgpack:"arcs"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	var (
		svcArcs   []arc.Arc
		q         = s.internal.NewRetrieve().Entries(&svcArcs)
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasSearch = req.SearchTerm != ""
	)

	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	if err = q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}

	res.Arcs = svcArcs

	// Compile Arcs to modules if requested
	if req.Compile {
		for i := range res.Arcs {
			if err = s.compileArc(ctx, &res.Arcs[i]); err != nil {
				return RetrieveResponse{}, err
			}
		}
	}

	if req.IncludeStatus {
		statuses := make([]status.Status[arc.StatusDetails], 0, len(res.Arcs))
		uuidStrings := lo.Map(res.Arcs, func(a Arc, _ int) string {
			return a.Key.String()
		})
		if err = status.NewRetrieve[arc.StatusDetails](s.status).
			WhereKeys(uuidStrings...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return RetrieveResponse{}, err
		}
		for i, stat := range statuses {
			res.Arcs[i].Status = &stat
		}
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: arc.OntologyIDsFromArcs(svcArcs),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// LSPMessage represents a single JSON-RPC message for the LSP
type LSPMessage = arctransport.JSONRPCMessage

// LSP handles LSP protocol messages over a Freighter stream
func (s *Service) LSP(ctx context.Context, stream freighter.ServerStream[LSPMessage, LSPMessage]) error {
	// Create a new LSP server instance for this connection with a no-op logger
	// to avoid nil pointer panics
	lspServer, err := arclsp.New(arclsp.Config{
		Instrumentation: s.Child("arc").Child("lsp"),
		GlobalResolver:  s.internal.SymbolResolver(),
	})
	if err != nil {
		return err
	}
	return arctransport.ServeFreighter(ctx, lspServer, stream)
}

// compileArc compiles the Arc text to a module containing IR and WASM bytecode.
// Returns an error if parsing, analysis, or compilation fails.
func (s *Service) compileArc(ctx context.Context, arc *Arc) error {
	// Step 1: Parse the Arc text
	parsed, diag := arctext.Parse(arc.Text)
	if diag != nil && !diag.Ok() {
		return errors.Newf("parse failed for arc %s: %w", arc.Key, diag)
	}

	// Step 2: Analyze the parsed text to produce IR
	ir, diag := arctext.Analyze(ctx, parsed, s.internal.SymbolResolver())
	if diag != nil && !diag.Ok() {
		return errors.Newf("analysis failed for arc %s: %w", arc.Key, diag)
	}

	// Step 3: Compile IR to WebAssembly module
	mod, err := arctext.Compile(ctx, ir)
	if err != nil {
		return errors.Newf("compilation failed for arc %s: %w", arc.Key, err)
	}

	// Step 4: Attach compiled module to Arc
	arc.Module = &mod
	return nil
}
